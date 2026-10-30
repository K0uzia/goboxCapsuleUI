package gui

import (
	"context"
	"encoding/json"
	"fmt"
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

// Un SMART illisible n'est pas une panne : « non supporté » sur la fiche, sans
// peser sur le verdict
const pillUnsupported = "NON SUPPORTÉ"

func unsupported(detail string) result {
	return result{Status: statusNA, Detail: detail, Data: map[string]string{"pill": pillUnsupported}}
}

// runSmart vérifie l'état SMART de tous les disques d'un type
func runSmart(kind string) func(t *testCtx) result {
	return func(t *testCtx) result {
		t.progress(-1)
		return smartScan(t.ctx, kind, t.log)
	}
}

// smartScan lit le SMART des disques d'un type ; aussi lancé seul à l'ouverture
// de gobox, le test ne demande rien au technicien
func smartScan(ctx context.Context, kind string, logf func(string, ...any)) result {
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
		logf("Aucun %s sur cette machine", kindNames[kind])
		return result{Status: statusNA, Detail: "Aucun " + kindNames[kind]}
	}
	tool := toolPath("smartctl")
	if tool == "" {
		logf("smartctl introuvable : installer le paquet smartmontools")
		return unsupported("smartctl non installé")
	}
	if !isAdmin() {
		logf("La lecture SMART nécessite les droits administrateur")
		return unsupported("droits administrateur nécessaires")
	}

	worst, read := statusOK, false
	var details []string
	for _, d := range disks {
		s, detail, ok := smartCheck(ctx, tool, d, logf)
		details = append(details, detail)
		if ok {
			read = true
			if s > worst {
				worst = s // ordre : OK < Moyen < Nul
			}
		}
	}
	joined := strings.Join(details, " | ")
	if !read {
		return unsupported(joined)
	}
	return result{Status: worst, Detail: joined, Data: map[string]string{"state": joined}}
}

// smartCheck ok faux si le disque refuse le SMART : on réessaie en SAT, le mode
// des disques derrière un pont USB ou un contrôleur
func smartCheck(ctx context.Context, tool string, d *probe.DiskInfo, logf func(string, ...any)) (status, string, bool) {
	dev := "/dev/" + d.Name
	logf("%s : %s", dev, strings.TrimSpace(d.Vendor+" "+d.Model))

	var r smartReport
	read := false
	for _, mode := range [][]string{nil, {"-d", "sat"}} {
		// smartctl renvoie un code non nul pour de simples avertissements :
		// on lit le JSON quoi qu'il arrive.
		args := append(append([]string{"--json", "-H", "-A"}, mode...), dev)
		out, _ := exec.CommandContext(ctx, tool, args...).Output()
		r = smartReport{}
		if json.Unmarshal(out, &r) == nil && r.SmartStatus != nil {
			read = true
			break
		}
	}
	for _, m := range r.Smartctl.Messages {
		logf("  %s", m.String)
	}
	if !read {
		logf("  SMART non supporté par ce disque")
		return statusNA, d.Name + " : SMART non supporté", false
	}
	s, detail := smartVerdict(d.Name, r)
	logf("  %s", detail)
	return s, detail, true
}

// smartVerdict état et résumé d'un rapport lu
func smartVerdict(name string, r smartReport) (status, string) {
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
	return s, fmt.Sprintf("%s · %s · %s", name, state, strings.Join(parts, " · "))
}
