package gui

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// Captures de la démo adaptée à la largeur (téléphone, tablette), rendues en
// mémoire : GOBOX_RESP=<dossier> go test -run TestResponsiveCaptures ./internal/ui/gui
func TestResponsiveCaptures(t *testing.T) {
	dir := os.Getenv("GOBOX_RESP")
	if dir == "" {
		t.Skip("GOBOX_RESP non défini")
	}
	responsive, touchUI = true, true
	defer func() { responsive, touchUI = false, false }()
	simulated = true
	defer func() { simulated = false }()
	manageSuperKey = false

	sizes := []fyne.Size{{Width: 360, Height: 650}, {Width: 390, Height: 760}, {Width: 430, Height: 850}, {Width: 768, Height: 960},
		{Width: 820, Height: 1110}, {Width: 1024, Height: 700}, {Width: 1180, Height: 760}, {Width: 1280, Height: 660}}
	only := os.Getenv("GOBOX_RESP_ONLY")
	for _, sz := range sizes {
		tag := fmt.Sprintf("%.0fx%.0f", sz.Width, sz.Height)
		if only != "" && !strings.Contains(only, tag) {
			continue
		}
		respCapture(t, dir, tag+"-accueil", sz, nil)
		respCapture(t, dir, tag+"-fiche", sz, func(tools *toolsPanel, s *session, w fyne.Window) { pickTab(w, "Fiche") })
		respCapture(t, dir, tag+"-fiche-remplie", sz, func(tools *toolsPanel, s *session, w fyne.Window) {
			fillFiche(s)
			pickTab(w, "Fiche")
			scrollFiche(w, 900)
		})
		for _, id := range strings.Split(os.Getenv("GOBOX_RESP_TESTS"), ",") {
			if id == "" {
				continue
			}
			respCapture(t, dir, tag+"-test-"+id, sz, func(tools *toolsPanel, s *session, w fyne.Window) {
				test.Tap(tools.buttons[id])
				time.Sleep(600 * time.Millisecond)
			})
		}
		respCapture(t, dir, tag+"-resultat", sz, func(tools *toolsPanel, s *session, w fyne.Window) {
			def := s.Tests[0]
			def.Run = func(tc *testCtx) result {
				return result{Status: statusNA, Detail: "41 °C au repos, 87 °C au maximum en 03:00, 0 erreur(s) de calcul", Advice: stressVerdict{hasTemp: true, maxTemp: 87}.advice()}
			}
			test.Tap(tools.buttons[def.ID])
			time.Sleep(300 * time.Millisecond)
		})
		respCapture(t, dir, tag+"-a-propos", sz, func(tools *toolsPanel, s *session, w fyne.Window) { showInfo(s, w) })
		respCapture(t, dir, tag+"-reglages", sz, func(tools *toolsPanel, s *session, w fyne.Window) { showSettings(w, newFichePanel(s)) })
	}
}

func fillFiche(s *session) {
	for i, def := range s.Tests {
		st := []status{statusOK, statusOK, statusWarn, statusOK, statusFail, statusNA}[i%6]
		s.Results[def.ID] = result{Status: st, Detail: "Résultat de démonstration pour la capture"}
	}
	s.Form.Tech = "Alexandre"
	s.Machine = machineInfo{Vendor: "HP", Product: "ProBook 450 G3", Serial: "5CD70729PR"}
	s.Inventory = demoInventory()
	s.changed()
}

// respBodyOf corps adaptatif de la fenêtre
func respBodyOf(o fyne.CanvasObject) *respBody {
	c, ok := o.(*fyne.Container)
	if !ok {
		return nil
	}
	if b, ok := c.Layout.(*respBody); ok {
		return b
	}
	for _, child := range c.Objects {
		if b := respBodyOf(child); b != nil {
			return b
		}
	}
	return nil
}

func pickTab(w fyne.Window, tab string) {
	if b := respBodyOf(w.Content()); b != nil {
		fyne.DoAndWait(func() { b.tabs.set(tab) })
		time.Sleep(100 * time.Millisecond)
	}
}

func scrollFiche(w fyne.Window, dy float32) {
	if b := respBodyOf(w.Content()); b != nil {
		fyne.DoAndWait(func() {
			b.fiche.scroll.Offset.Y = dy
			b.fiche.scroll.Refresh()
		})
	}
}

func respCapture(t *testing.T, dir, name string, sz fyne.Size, act func(*toolsPanel, *session, fyne.Window)) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(true))
	w := a.NewWindow("capture")
	s := newSession()
	s.Machine = machineInfo{Vendor: "HP", Product: "ProBook 450 G3", Serial: "DEMO-000451"}
	content, tools := mainContent(s, w)
	w.SetContent(content)
	w.Resize(sz)
	s.changed()
	time.Sleep(150 * time.Millisecond) // mises en page différées (respGrid, onglets)
	if act != nil {
		act(tools, s, w)
	}
	time.Sleep(150 * time.Millisecond)
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, w.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}
