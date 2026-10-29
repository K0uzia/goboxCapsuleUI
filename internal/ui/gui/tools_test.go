package gui

import (
	"os"
	"testing"
	"time"

	"fyne.io/fyne/v2"
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
	path, err := writeTone("both")
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
	s.Results["son"] = result{Status: statusFail, Detail: "Micro interne nul", Data: map[string]string{
		"card": statusData(statusOK), "speakers": statusData(statusOK), "jack": statusData(statusNA),
		"mic_int": statusData(statusFail), "mic_in": statusData(statusNA)}}
	s.Results["ports"] = result{Status: statusOK, Data: map[string]string{"usb": "3 / 9", "HDMI": "0 / 3", "DisplayPort": "1 / 3"}}
	s.Form.Tech = "Alex"
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

// Chaque ligne de la fiche suit la partie du test qui la renseigne : son
// (une ligne par sortie et par micro) et ports (testés / détectés par type).
func TestItemState(t *testing.T) {
	s := newSession()
	s.Results["son"] = result{Status: statusFail, Data: map[string]string{
		"speakers": statusData(statusOK), "jack": statusData(statusNA), "mic_int": statusData(statusFail)}}
	s.Results["ports"] = result{Status: statusWarn, Data: map[string]string{"usb": "0 / 4", "HDMI": "1 / 3"}}
	cases := []struct {
		item ctrlItem
		want status
	}{
		{ctrlItem{TestID: "son", Key: "speakers"}, statusOK},
		{ctrlItem{TestID: "son", Key: "jack"}, statusNA},
		{ctrlItem{TestID: "son", Key: "mic_int"}, statusFail},
		{ctrlItem{TestID: "ports", Key: "HDMI"}, statusOK},
		{ctrlItem{TestID: "ports", Key: "VGA"}, statusNA},   // aucun connecteur VGA
		{ctrlItem{TestID: "ports", Key: "usb"}, statusWarn}, // aucun testé : verdict du test
	}
	for _, c := range cases {
		if got, _ := s.itemState(c.item); got != c.want {
			t.Errorf("%s/%s : %s, attendu %s", c.item.TestID, c.item.Key, got.style().label, c.want.style().label)
		}
	}
}

// Pendant le test clavier, le focus reste sur le piège à touches : Tab ne
// doit pas le déplacer sur un bouton, et un clic ailleurs le lui rend.
func TestKeyboardKeepsFocus(t *testing.T) {
	manageSuperKey = false
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(true))
	w := a.NewWindow("test")
	s := newSession()
	content, tools := mainContent(s, w)
	w.SetContent(content)
	w.Resize(fyne.NewSize(1280, 900))

	test.Tap(tools.buttons["clavier"])
	trapOf := func() *keyTrap {
		k, _ := w.Canvas().Focused().(*keyTrap)
		return k
	}
	deadline := time.Now().Add(2 * time.Second)
	for trapOf() == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	trap := trapOf()
	if trap == nil {
		t.Fatalf("le piège à touches n'a pas le focus : %T", w.Canvas().Focused())
	}
	if !trap.AcceptsTab() {
		t.Fatal("le piège doit accepter Tab")
	}
	w.Canvas().FocusNext() // ce que ferait Tab sans le piège
	time.Sleep(100 * time.Millisecond)
	if trapOf() == nil {
		t.Fatalf("le focus est parti sur %T", w.Canvas().Focused())
	}
	tools.close(false)
}
