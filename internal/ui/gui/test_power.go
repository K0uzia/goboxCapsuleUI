package gui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"

	"gobox/internal/probe"
)

// Batterie et chargeur : deux tests distincts, car un PC arrive souvent sans son
// chargeur. Sans chargeur, le test de charge ne compte ni pour ni contre.

const (
	sysPower       = "/sys/class/power_supply"
	pillNoCharger  = "SANS CHARGEUR"
	chargerMaxWait = 500 * time.Millisecond
)

// charger état de l'alimentation externe
type charger struct {
	Online bool
	Watts  float64 // puissance annoncée, 0 si le système ne la donne pas
}

// readCharger lit les sources Mains et USB de root
func readCharger(root string) charger {
	var c charger
	dirs, _ := filepath.Glob(filepath.Join(root, "*"))
	for _, d := range dirs {
		typ := readTrim(filepath.Join(d, "type"))
		if typ != "Mains" && !strings.HasPrefix(typ, "USB") {
			continue
		}
		if readTrim(filepath.Join(d, "online")) != "1" {
			continue
		}
		c.Online = true
		num := func(f string) float64 {
			v, _ := strconv.ParseFloat(readTrim(filepath.Join(d, f)), 64)
			return v
		}
		if w := num("voltage_max") * num("current_max") / 1e12; w > c.Watts { // µV × µA
			c.Watts = w
		}
	}
	return c
}

func hasBattery(root string) bool { return batteryDir(root) != "" }

// batteryDir dossier de la batterie du PC dans root, "" sans batterie
func batteryDir(root string) string {
	dirs, _ := filepath.Glob(filepath.Join(root, "*"))
	for _, d := range dirs {
		if probe.IsSystemBattery(d) {
			return d
		}
	}
	return ""
}

// chargeVerdict résultat de la charge, chargeur branché
func chargeVerdict(state string, watts float64) (status, string) {
	switch state {
	case "Charging":
		if watts > 0 {
			return statusOK, fmt.Sprintf("la batterie charge (%.1f W)", watts)
		}
		return statusOK, "la batterie charge"
	case "Full":
		return statusOK, "batterie pleine"
	case "Not charging":
		return statusWarn, "chargeur branché, la batterie ne charge pas (seuil de charge ou batterie à contrôler)"
	}
	return statusFail, "chargeur branché, la batterie se décharge"
}

// waitCharger attend le branchement du chargeur ; statusIdle si le chargeur est
// branché, sinon la réponse du technicien
func waitCharger(t *testCtx, ico, prompt string, opts ...choice) (status, bool) {
	if readCharger(sysPower).Online || simulated {
		return statusIdle, true
	}
	t.panel(func() fyne.CanvasObject {
		return instruction(ico, "Branchez le chargeur : il est détecté tout seul.")
	})
	answer := t.ask(prompt, opts...)
	tick := time.NewTicker(chargerMaxWait)
	defer tick.Stop()
	for {
		select {
		case <-t.ctx.Done():
			return statusIdle, false
		case s := <-answer:
			t.clearAsk()
			return s, false
		case <-tick.C:
			if readCharger(sysPower).Online {
				t.clearAsk()
				t.answered = false
				t.setPanelEmpty()
				return statusIdle, true
			}
		}
	}
}

// runCharger le chargeur est-il présent et reconnu ?
func runCharger(t *testCtx) result {
	t.progress(-1)
	if simulated {
		t.markSimulated()
		return result{Status: statusOK, Detail: "Chargeur détecté (65 W)"}
	}
	s, online := waitCharger(t, "bolt", "Sans détection :",
		choice{"Pas de chargeur", statusNA}, choice{"Nul", statusFail})
	switch {
	case t.cancelled():
		return result{}
	case !online && s == statusNA:
		return result{Status: statusNA, Detail: "Pas de chargeur fourni", Data: map[string]string{"pill": pillNoCharger}}
	case !online:
		return result{Status: statusFail, Detail: "Chargeur branché, non détecté"}
	}
	c := readCharger(sysPower)
	detail := "Chargeur détecté"
	if c.Watts > 0 {
		detail = fmt.Sprintf("Chargeur détecté (%.0f W)", c.Watts)
	}
	t.log("%s", detail)
	return result{Status: statusOK, Detail: detail}
}

// runBatteryCharge la batterie charge-t-elle, chargeur branché ?
func runBatteryCharge(t *testCtx) result {
	t.progress(-1)
	if simulated {
		t.markSimulated()
		return result{Status: statusOK, Detail: "La batterie charge (25,0 W)"}
	}
	if !hasBattery(sysPower) {
		return result{Status: statusNA, Detail: "Pas de batterie"}
	}
	s, online := waitCharger(t, "battery-half", "Sans chargeur, la charge ne peut pas être testée :",
		choice{"Pas de chargeur", statusNA}, choice{"Nul", statusFail})
	switch {
	case t.cancelled():
		return result{}
	case !online && s == statusNA:
		return result{Status: statusNA, Detail: "Non testée : pas de chargeur", Data: map[string]string{"pill": pillNoCharger}}
	case !online:
		return result{Status: statusFail, Detail: "La batterie ne charge pas"}
	}
	t.log("Chargeur détecté, mesure de la charge…")
	if !t.sleep(5 * time.Second) { // l'état met quelques secondes à se stabiliser
		return result{}
	}
	dir := batteryDir(sysPower)
	if dir == "" {
		return result{Status: statusNA, Detail: "Pas de batterie"}
	}
	st, detail := chargeVerdict(readTrim(filepath.Join(dir, "status")), readBatterySys().power)
	detail = strings.ToUpper(detail[:1]) + detail[1:]
	t.log("%s", detail)
	return result{Status: st, Detail: detail}
}
