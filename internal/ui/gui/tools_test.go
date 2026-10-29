package gui

import (
	"os"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
)

// Parcours complet d'un test : clic sur le bouton, résultat, confirmation,
// puis vérification que la fiche a enregistré le résultat.
func TestToolFlow(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(true))
	w := a.NewWindow("test")

	s := newSession()
	s.Tests = []*testDef{{ID: "fake", Title: "Faux", Category: catPeripherals, Color: cPeripheral, Soft: cPeripheralBg,
		Run: func(tc *testCtx) result {
			tc.log("étape 1")
			return result{Status: statusWarn, Detail: "réserve"}
		}}}
	fiche := newFichePanel(s)
	tools := newToolsPanel(s, w)
	_ = fiche

	test.Tap(tools.buttons["fake"])
	if tools.active == nil || tools.menu.Visible() {
		t.Fatal("le test devrait être affiché à la place du menu")
	}

	deadline := time.Now().Add(2 * time.Second)
	for !tools.view.finished.Visible() {
		if time.Now().After(deadline) {
			t.Fatal("le résultat n'est jamais affiché")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if s.status("fake") != statusIdle {
		t.Fatal("le résultat ne doit pas être enregistré avant confirmation")
	}

	tools.close(true)
	if got := s.status("fake"); got != statusWarn {
		t.Fatalf("statut enregistré = %v, attendu MOYEN", got)
	}
	if !tools.menu.Visible() {
		t.Fatal("le menu devrait être réaffiché")
	}
	if v := s.verdict(); v.Label != "Apte à la vente" {
		t.Fatalf("verdict = %q", v.Label)
	}
}

func TestWAVGeneration(t *testing.T) {
	path, err := writeStereoTones()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
}

func TestFicheTemplate(t *testing.T) {
	s := newSession()
	s.Machine = machineInfo{Vendor: "Dell", Product: "Latitude 7480", Serial: "7XK2Q9A"}
	s.Inventory = collectInventory()
	s.Results["stress"] = result{Status: statusOK, Data: map[string]string{"before": "41 °C", "after": "78 °C"}}
	s.Results["sd"] = result{Status: statusFail, Detail: "Carte non détectée"}
	out := os.Getenv("GOBOX_FICHE_OUT")
	if out == "" {
		out = os.DevNull
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := ficheTemplate.Execute(f, newPrintData(s)); err != nil {
		t.Fatal(err)
	}
}
