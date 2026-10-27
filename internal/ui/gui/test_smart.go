package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gobox/internal/probe"
)

// smartReport champs utiles de `smartctl --json`
type smartReport struct {
	SmartStatus *struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	PowerOnTime struct {
		Hours int `json:"hours"`
	} `json:"power_on_time"`
	NVMeLog *struct {
		PercentageUsed int `json:"percentage_used"`
	} `json:"nvme_smart_health_information_log"`
	ATA *struct {
		Table []struct {
			ID  int `json:"id"`
			Raw struct {
				Value int `json:"value"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
	Smartctl struct {
		Messages []struct {
			String string `json:"string"`
		} `json:"messages"`
	} `json:"smartctl"`
}

// diskKind classe un disque : nvme, ssd (SATA) ou hdd
func diskKind(d *probe.DiskInfo) string {
	switch {
	case strings.HasPrefix(d.Name, "nvme"):
		return "nvme"
	case d.Type == "HDD":
		return "hdd"
	}
	return "ssd"
}

var kindNames = map[string]string{"hdd": "disque dur (HDD)", "ssd": "SSD SATA", "nvme": "SSD NVMe"}

// runSmart vérifie l'état SMART de tous les disques d'un type
func runSmart(kind string) func(t *testCtx) result {
	return func(t *testCtx) result {
		t.progress(-1)
		names, err := probe.ListDisks()
		if err != nil {
			return result{Status: statusFail, Detail: err.Error()}
		}
		var disks []*probe.DiskInfo
		for _, n := range names {
			if d, err := probe.GetDiskInfo(n); err == nil && diskKind(d) == kind {
				disks = append(disks, d)
			}
		}
		if len(disks) == 0 {
			t.log("Aucun %s sur cette machine", kindNames[kind])
			return result{Status: statusNA, Detail: "Aucun " + kindNames[kind]}
		}
		if firstAvailable("smartctl") == "" {
			t.log("smartctl introuvable : sudo apt install smartmontools")
			return result{Status: statusWarn, Detail: "smartctl non installé"}
		}
		if os.Geteuid() != 0 {
			t.log("La lecture SMART nécessite les droits root (make dev)")
		}

		worst := statusOK
		var details []string
		for _, d := range disks {
			s, detail := smartCheck(t, d)
			details = append(details, detail)
			if s > worst {
				worst = s // ordre : OK < Moyen < Nul
			}
		}
		joined := strings.Join(details, " | ")
		return result{Status: worst, Detail: joined, Data: map[string]string{"state": joined}}
	}
}

func smartCheck(t *testCtx, d *probe.DiskInfo) (status, string) {
	dev := "/dev/" + d.Name
	model := strings.TrimSpace(d.Vendor + " " + d.Model)
	t.log("%s : %s", dev, model)

	// smartctl renvoie un code non nul pour de simples avertissements :
	// on lit le JSON quoi qu'il arrive.
	out, _ := exec.CommandContext(t.ctx, "smartctl", "--json", "-H", "-A", dev).Output()
	var r smartReport
	if err := json.Unmarshal(out, &r); err != nil {
		t.log("  Lecture SMART impossible : %v", err)
		return statusWarn, d.Name + " : SMART illisible"
	}
	for _, m := range r.Smartctl.Messages {
		t.log("  %s", m.String)
	}
	if r.SmartStatus == nil {
		return statusWarn, d.Name + " : SMART indisponible"
	}

	parts := []string{fmt.Sprintf("%d h", r.PowerOnTime.Hours)}
	s := statusOK
	if r.NVMeLog != nil {
		parts = append(parts, fmt.Sprintf("usure %d %%", r.NVMeLog.PercentageUsed))
		if r.NVMeLog.PercentageUsed >= 80 {
			s = statusWarn
		}
	}
	if r.ATA != nil {
		for _, a := range r.ATA.Table {
			if a.ID == 5 { // secteurs réalloués
				parts = append(parts, fmt.Sprintf("%d secteur(s) réalloué(s)", a.Raw.Value))
				if a.Raw.Value > 0 {
					s = statusWarn
				}
			}
		}
	}
	state := "SMART OK"
	if !r.SmartStatus.Passed {
		state, s = "SMART ÉCHEC", statusFail
	}
	detail := fmt.Sprintf("%s · %s · %s", d.Name, state, strings.Join(parts, " · "))
	t.log("  %s", detail)
	return s, detail
}
