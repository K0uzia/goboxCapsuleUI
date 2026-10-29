package gui

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

// Captures de la fenêtre principale, rendues en mémoire (aucune fenêtre
// ouverte) : GOBOX_CAPTURES=<dossier> go test -run TestCaptures ./internal/ui/gui
func TestCaptures(t *testing.T) {
	dir := os.Getenv("GOBOX_CAPTURES")
	if dir == "" {
		t.Skip("GOBOX_CAPTURES non défini")
	}
	sizes := []fyne.Size{{Width: 1366, Height: 768}, {Width: 1280, Height: 1000}, {Width: 1920, Height: 1080}, {Width: 2560, Height: 1440}}
	big := fyne.NewSize(1280, 1000)

	for _, dark := range []bool{true, false} {
		mode := "sombre"
		if !dark {
			mode = "clair"
		}
		for _, sz := range sizes {
			capture(t, dir, fmt.Sprintf("%s-accueil-%.0fx%.0f", mode, sz.Width, sz.Height), sz, dark, nil)
		}
		capture(t, dir, mode+"-test-en-cours", big, dark, func(tools *toolsPanel, s *session, w fyne.Window) {
			def := s.Tests[0]
			def.Run = func(tc *testCtx) result {
				tc.log("Préparation de la charge sur 8 threads")
				tc.log("Pas de capteur de consommation : RAPL illisible sans droits root")
				tc.log("Charge sur tous les cœurs, relevés toutes les 2 s")
				tc.log("Erreur de calcul détectée sur le cœur 3")
				tc.progress(0.4)
				tc.sleep(time.Minute)
				return result{}
			}
			test.Tap(tools.buttons[def.ID])
			time.Sleep(200 * time.Millisecond)
		})
		capture(t, dir, mode+"-test-resultat", big, dark, func(tools *toolsPanel, s *session, w fyne.Window) {
			def := s.Tests[0]
			def.Run = func(tc *testCtx) result {
				tc.log("Terminé")
				return result{Status: statusWarn, Detail: "41 °C au repos, 87 °C au maximum en 03:00, 0 erreur(s) de calcul", Advice: stressVerdict{hasTemp: true, maxTemp: 87}.advice()}
			}
			test.Tap(tools.buttons[def.ID])
			deadline := time.Now().Add(2 * time.Second)
			for !tools.view.finished.Visible() && time.Now().Before(deadline) {
				time.Sleep(10 * time.Millisecond)
			}
		})
		capture(t, dir, mode+"-fiche-remplie", big, dark, func(tools *toolsPanel, s *session, w fyne.Window) {
			for i, def := range s.Tests {
				st := []status{statusOK, statusOK, statusWarn, statusOK, statusFail, statusNA}[i%6]
				s.Results[def.ID] = result{Status: st, Detail: "Résultat de démonstration pour la capture"}
			}
			s.changed()
		})
	}
	captureStress(t, dir)
	capture(t, dir, "sombre-stress-reglages-1366", fyne.NewSize(1366, 768), true, func(tools *toolsPanel, s *session, w fyne.Window) {
		test.Tap(tools.buttons["stress"])
		time.Sleep(300 * time.Millisecond)
	})
	capture(t, dir, "sombre-reglages", big, true, func(tools *toolsPanel, s *session, w fyne.Window) { showSettings(w, newFichePanel(s)) })
	capture(t, dir, "sombre-a-propos", big, true, func(tools *toolsPanel, s *session, w fyne.Window) { showInfo(s, w) })
}

func capture(t *testing.T, dir, name string, sz fyne.Size, dark bool, act func(*toolsPanel, *session, fyne.Window)) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(dark))
	w := a.NewWindow("capture")

	s := newSession()
	s.Machine = machineInfo{Vendor: "Dell", Product: "Latitude 7480", Serial: "7XK2Q9A-DL"}
	content, tools := mainContent(s, w)
	w.SetContent(content)
	w.Resize(sz)
	s.changed()
	if act != nil {
		act(tools, s, w)
	}

	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, w.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}

// captureStress écran de suivi du stress test, nourri par la source simulée
// en accéléré, puis l'écran du verdict
func captureStress(t *testing.T, dir string) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(true))

	sim := newSimSensors()
	sim.plateau = 88 // passe le seuil de réserve et déclenche le throttling
	first := sim.sample()
	m := newStressMonitor(len(first.CoreLoad), first.HasPower)
	var temp, freq, load, power stat
	for i := 0; i < 90; i++ {
		sim.start = sim.start.Add(-2 * time.Second) // 2 s simulées par relevé
		s := sim.sample()
		temp.add(s.Temp)
		freq.add(s.Freq)
		load.add(s.Load)
		power.add(s.Power)
		m.update(s, &temp, &freq, &load, &power, 0, time.Duration(i*2)*time.Second, 10*time.Minute)
	}
	w := test.NewWindow(container.NewVScroll(m.root))
	w.Resize(fyne.NewSize(660, 900))
	save(t, dir, "sombre-stress-suivi", w)

	v := stressVerdict{hasTemp: true, before: 41, maxTemp: temp.max, throttled: true,
		ran: 3 * time.Minute, planned: 10 * time.Minute, avgFreq: freq.avg(), maxFreq: freq.max}
	w2 := test.NewWindow(v.summary(m.tempChart))
	w2.Resize(fyne.NewSize(660, 420))
	save(t, dir, "sombre-stress-verdict", w2)
}

func save(t *testing.T, dir, name string, w fyne.Window) {
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, w.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
}
