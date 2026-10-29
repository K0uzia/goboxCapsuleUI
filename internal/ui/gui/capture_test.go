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
			s.Results["ports"] = result{Status: statusOK, Detail: "USB 6 / 6 · HDMI 1 / 1 · VGA 0 / 1",
				Data: map[string]string{"usb": "6 / 6", "HDMI": "1 / 1", "VGA": "0 / 1"}}
			s.Results["stress"] = result{Status: statusOK, Data: map[string]string{"before": "41 °C", "after": "78 °C"}}
			s.Results["son"] = result{Status: statusWarn, Data: map[string]string{"card": statusData(statusOK), "speakers": statusData(statusWarn),
				"jack": statusData(statusOK), "mic_int": statusData(statusOK), "mic_in": statusData(statusNA)}}
			s.Form.Checks["Autre"] = statusOK
			s.Form.Texts["Autre"] = "Lecteur d'empreintes"
			s.Machine = machineInfo{Vendor: "HP", Product: "ProBook 450 G3", Serial: "5CD70729PR"}
			s.Inventory = demoInventory()
			s.changed()
		})
	}
	for _, id := range []string{"souris", "ports", "son"} {
		capture(t, dir, "sombre-test-"+id, big, true, func(tools *toolsPanel, s *session, w fyne.Window) {
			test.Tap(tools.buttons[id])
			time.Sleep(400 * time.Millisecond)
			if id == "souris" {
				z := findMouseZone(tools.view.panel)
				z.MouseMoved(nil)
				z.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.Delta{DY: 1}})
				time.Sleep(100 * time.Millisecond)
			}
		})
	}
	// test son en données simulées, jusqu'à l'étape du micro interne
	simulated = true
	capture(t, dir, "sombre-test-son-micro", big, true, func(tools *toolsPanel, s *session, w fyne.Window) {
		test.Tap(tools.buttons["son"])
		for i := 0; i < 2; i++ { // haut-parleurs puis prise casque : OK
			time.Sleep(300 * time.Millisecond)
			answerAsk(tools.view, 0)
		}
		time.Sleep(300 * time.Millisecond)
	})
	simulated = false
	captureInventory(t, dir)
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

func findMouseZone(c *fyne.Container) *mouseZone {
	for _, o := range c.Objects {
		if z, ok := o.(*mouseZone); ok {
			return z
		}
	}
	return newMouseZone(nil)
}

// demoInventory inventaire d'un portable, pour les captures
func demoInventory() []Component {
	return []Component{
		{Name: "Processeur", Icon: "microchip", Summary: "Intel(R) Core(TM) i3-6100U CPU @ 2.30GHz", Fields: []Field{{"Architecture", "x86_64"}, {"Cœurs", "4"}, {"Fréquence", "400 à 2300 MHz"}}},
		{Name: "Mémoire RAM", Icon: "memory", Summary: "4 Go DDR4", Fields: []Field{{"Total", "4,0 Go"}, {"Bottom-Slot 2", "4096 Mo DDR4 SODIMM @ 2133 MHz · Samsung"}}},
		{Name: "Stockage", Icon: "hard-drive", Summary: "1 disque(s) · 298,1 Go", Fields: []Field{{"sda", "Toshiba MK3261GSYN · HDD · 298,1 Go"}}},
		{Name: "Batterie", Icon: "battery-half", Summary: "Santé 100 % · 112 cycles", Fields: []Field{{"Charge", "8 % (sur batterie)"}, {"Temps restant", "1:11:57"}, {"Capacité réelle", "28,6 Wh, soit 100 %"}, {"Fabricant", "Hewlett-Packard"}, {"Tension", "16,29 V"}}},
		{Name: "Cartes réseau", Icon: "wifi", Summary: "2 interface(s) · 1 connectée(s)", Fields: []Field{{"enp1s0", "Realtek RTL8111/8168 · ethernet · connecté"}, {"wlp2s0", "Intel Wireless 3165 · wifi · déconnecté"}}},
		{Name: "Carte graphique", Icon: "display", Summary: "Intel Skylake GT2 [HD Graphics 520]", Fields: []Field{{"Modèle", "HD Graphics 520"}, {"Pilote", "i915"}}},
		{Name: "Affichage", Icon: "tv", Summary: "1 écran(s) connecté(s)", Fields: []Field{{"Écran intégré", "connecté 1366x768"}, {"HDMI (HDMI-A-1)", "déconnecté"}, {"VGA (VGA-1)", "déconnecté"}}},
	}
}

// captureInventory bas de la fiche : l'inventaire en cartes
func captureInventory(t *testing.T, dir string) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(true))
	s := newSession()
	s.Machine = machineInfo{Vendor: "HP", Product: "ProBook 450 G3", Serial: "5CD70729PR"}
	s.Inventory = demoInventory()
	f := newFichePanel(s)
	w := test.NewWindow(f.root)
	w.Resize(fyne.NewSize(600, 1000))
	s.changed()
	f.scroll.ScrollToBottom()
	save(t, dir, "sombre-fiche-inventaire", w)
}

// answerAsk touche le bouton de réponse n de la question affichée
func answerAsk(v *testView, n int) {
	var tap func(o fyne.CanvasObject) bool
	count := 0
	tap = func(o fyne.CanvasObject) bool {
		if b, ok := o.(*verdictButton); ok {
			if count == n {
				test.Tap(b)
				return true
			}
			count++
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, child := range c.Objects {
				if tap(child) {
					return true
				}
			}
		}
		return false
	}
	fyne.DoAndWait(func() { tap(v.askButtons) })
}
