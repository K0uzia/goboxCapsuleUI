//go:build demo && js

package gui

// Démo en ligne : la vraie application, compilée pour le navigateur
// (GOOS=js GOARCH=wasm, étiquette demo), sur un PC portable fictif. Rien n'est
// lu sur la machine du visiteur, rien n'est écrit, imprimé, mis à jour ni
// envoyé : les tests qui ont besoin du vrai matériel rendent un résultat
// simulé, le stress test tourne en accéléré, le test clavier écoute le clavier
// du visiteur et PRINT télécharge la fiche PDF dans le navigateur.

import (
	"fmt"
	"io"
	"log"
	"sync"
	"syscall/js"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/test"
)

const demoUnavailable = "Indisponible dans la démo"

// RunDemo ouvre GoBox dans la page, sur une fiche vierge : chaque test se
// lance et suit son déroulé, sur un matériel simulé
func RunDemo() {
	// le navigateur n'a ni disque ni exécutable : les avertissements de Fyne
	// (cache, métadonnées) n'apprennent rien au visiteur
	log.SetOutput(io.Discard)
	a := app.NewWithID(appID)
	a.Settings().SetTheme(newTheme(true))
	w := a.NewWindow("gobox · Toolbox La Capsule")

	s := newSession()
	s.Machine = machineInfo{Vendor: "HP", Product: "ProBook 450 G3", Serial: "DEMO-000451"}
	s.Inventory = demoWebInventory()
	s.markAbsent(map[string]string{"sim": "Pas de modem", "lecteur": "Pas de lecteur de carte à puce", "ssd": "Pas de SSD SATA", "nvme": "Pas de SSD NVMe"})
	demoTests(s)

	content, tools := mainContent(s, w)
	w.SetContent(content)
	s.changed()
	slowMarchFrames(content)

	// puce CPU de l'en-tête : relevé simulé, la page ne lit aucun capteur
	walk(content, func(o fyne.CanvasObject) {
		if t, ok := o.(*tText); ok && t.Text == "non lue" {
			demoCPU = t
		}
	})
	setDemoCPU(43)
	go func() {
		for range time.Tick(3 * time.Second) {
			demoCPUMu.Lock()
			busy := demoStressing
			demoCPUMu.Unlock()
			if !busy {
				setDemoCPU(42 + float64(time.Now().Second()%3))
			}
		}
	}()

	listenPhysicalKeys()
	// une fois par test clavier, quand le piège à touches prend le focus
	var relabeled *keyTrap
	go func() {
		for range time.Tick(300 * time.Millisecond) {
			fyne.Do(func() {
				if trap, ok := w.Canvas().Focused().(*keyTrap); ok && trap != relabeled {
					relabeled = trap
					relabelKeys(tools.view.root)
				}
				// le contour animé de la tuile du test en cours est dans le menu,
				// masqué pendant le test : inutile de redessiner pour lui
				if !tools.menu.Visible() {
					for _, b := range tools.buttons {
						if b.anim != nil {
							b.anim.Stop()
						}
					}
				}
			})
		}
	}()
	w.ShowAndRun()
}

// slowMarchFrames les cadres en pointillés tournent à 12 images/s au lieu de
// chaque image : en WebAssembly, chaque image redessine toute la fenêtre et
// produit des déchets mémoire, dont le ramassage fige la page
func slowMarchFrames(root fyne.CanvasObject) {
	var frames []*marchFrame
	walk(root, func(o fyne.CanvasObject) {
		if m, ok := o.(*marchFrame); ok {
			running := m.running
			m.anim.Stop()
			m.anim = fyne.NewAnimation(time.Second, func(float32) {})
			if running {
				m.anim.Start()
			}
			frames = append(frames, m)
		}
	})
	const fps = 12
	step := float32(marchDash+marchGap) / (1.2 * fps) // un tour de motif en 1,2 s, comme l'application
	go func() {
		for range time.Tick(time.Second / fps) {
			fyne.Do(func() {
				for _, m := range frames {
					if m.running && m.Visible() {
						m.offset += step
						if m.offset >= marchDash+marchGap {
							m.offset -= marchDash + marchGap
						}
						m.draw()
					}
				}
			})
		}
	}()
}

// walk parcourt les objets affichés
func walk(root fyne.CanvasObject, f func(fyne.CanvasObject)) {
	for _, o := range test.LaidOutObjects(root) {
		f(o)
	}
}

// Le navigateur n'a pas les polices du système : les symboles absents des
// polices de l'application prennent un libellé en toutes lettres
var demoKeyLabels = map[string]string{"⇪": "Verr", "⊞": "Win", "⏎": "Entrée"}

func relabelKeys(root fyne.CanvasObject) {
	walk(root, func(o fyne.CanvasObject) {
		k, ok := o.(*keyCapW)
		if !ok {
			return
		}
		if r, ok := demoKeyLabels[k.def.Azerty]; ok {
			k.def.Azerty, k.def.Qwerty = r, r
			k.label.Text = r
			k.label.Refresh()
		}
	})
}

// ---- Stress test accéléré ---------------------------------------------------

var (
	demoCPU       *tText
	demoCPUMu     sync.Mutex
	demoStressing bool
)

func setDemoCPU(v float64) {
	if demoCPU == nil {
		return
	}
	fyne.Do(func() { demoCPU.SetText(fmt.Sprintf("%.0f °C", v)) })
}

// demoStress le vrai stress test (réglages, suivi en direct, verdict) sur les
// mesures simulées d'un i3 de portable, le temps accéléré : 3 min en 9 s
func demoStress(t *testCtx) result {
	t.markSimulated()
	t.log("Démo : aucune charge réelle, mesures d'une machine fictive en accéléré")
	duration, variable, ok := stressSettings(t)
	if !ok {
		return result{}
	}
	demoCPUMu.Lock()
	demoStressing = true
	demoCPUMu.Unlock()
	defer func() {
		demoCPUMu.Lock()
		demoStressing = false
		demoCPUMu.Unlock()
		setDemoCPU(52)
	}()

	sim := newSimSensors()
	sim.plateau = 79
	first := sim.sample()
	before := first.Temp
	mon := newStressMonitor(len(first.CoreLoad), first.HasPower)
	t.panel(func() fyne.CanvasObject { return mon.root })
	mode := "constante"
	if variable {
		mode = "variable"
	}
	t.log("Température au repos : %.0f °C", before)
	t.log("Charge %s sur %d threads pendant %s", mode, sim.cores, formatClock(duration))

	n := int(duration / stressInterval)
	step := min(100*time.Millisecond, max(30*time.Millisecond, 12*time.Second/time.Duration(n)))
	var temp, freq, load, power stat
	for i := 1; i <= n; i++ {
		if !t.sleep(step) {
			return result{}
		}
		elapsed := time.Duration(i) * stressInterval
		sim.start = time.Now().Add(-elapsed)
		smp := sim.sample()
		temp.add(smp.Temp)
		freq.add(smp.Freq)
		load.add(smp.Load)
		power.add(smp.Power)
		t.ui(func() { mon.update(smp, &temp, &freq, &load, &power, 0, elapsed, duration) })
		setDemoCPU(smp.Temp)
		t.progress(elapsed.Seconds() / duration.Seconds())
		if i%15 == 0 {
			t.log("%s · %.0f °C · %s · charge %.0f %%", formatClock(elapsed), smp.Temp, ghz(smp.Freq), smp.Load)
		}
	}
	t.log("Arrêt de la charge")
	v := stressVerdict{hasTemp: true, before: before, maxTemp: temp.max, ran: duration, planned: duration,
		avgFreq: freq.avg(), maxFreq: freq.max}
	r := v.result()
	t.panel(func() fyne.CanvasObject { return v.summary(mon.tempChart) })
	return r
}

// ---- Impression : la fiche PDF téléchargée par le navigateur -----------------

func printFiche(s *session, w fyne.Window) {
	data, err := fichePDF(s)
	if err != nil {
		dialog.ShowError(fmt.Errorf("création de la fiche : %w", err), w)
		return
	}
	name := "Fiche-" + s.Ref + ".pdf"
	buf := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(buf, data)
	blob := js.Global().Get("Blob").New([]any{buf}, map[string]any{"type": "application/pdf"})
	url := js.Global().Get("URL").Call("createObjectURL", blob)
	doc := js.Global().Get("document")
	link := doc.Call("createElement", "a")
	link.Set("href", url)
	link.Set("download", name)
	doc.Get("body").Call("appendChild", link)
	link.Call("click")
	link.Call("remove")
	js.Global().Set("goboxDemoFiche", map[string]any{"name": name, "size": len(data)})

	msg := wrapLabel("La fiche PDF est téléchargée par votre navigateur sous le nom " + name + ". Sur le PC de l'atelier, GoBox l'ouvre directement pour l'imprimer.")
	note := mutedNote(demoUnavailable + " : aucune impression réelle, rien n'est envoyé.")
	dialog.NewCustom("Fiche téléchargée", "Fermer", container.New(&widthLayout{w: 460}, container.NewVBox(msg, note)), w).Show()
}

// showDiagnostic le rapport lance des commandes du système : neutralisé
func showDiagnostic(w fyne.Window) {
	showPopup(w, "Diagnostic matériel", container.NewVBox(
		wrapLabel(demoUnavailable+" : sur le PC, ce rapport liste les outils de lecture du matériel (smartctl, dmidecode, lspci, iw) et leur version."),
	))
}

// ---- Clavier : code physique de la touche ------------------------------------

// Le navigateur ne donne pas le code matériel au pilote de Fyne (toujours -1) :
// la page relève KeyboardEvent.code, indépendant de la disposition, et le
// traduit en code X11 comme sous Linux
var keyCodes = map[string]int{
	"Escape": 9, "Digit1": 10, "Digit2": 11, "Digit3": 12, "Digit4": 13, "Digit5": 14, "Digit6": 15, "Digit7": 16,
	"Digit8": 17, "Digit9": 18, "Digit0": 19, "Minus": 20, "Equal": 21, "Backspace": 22, "Tab": 23,
	"KeyQ": 24, "KeyW": 25, "KeyE": 26, "KeyR": 27, "KeyT": 28, "KeyY": 29, "KeyU": 30, "KeyI": 31, "KeyO": 32, "KeyP": 33,
	"BracketLeft": 34, "BracketRight": 35, "Enter": 36, "ControlLeft": 37,
	"KeyA": 38, "KeyS": 39, "KeyD": 40, "KeyF": 41, "KeyG": 42, "KeyH": 43, "KeyJ": 44, "KeyK": 45, "KeyL": 46,
	"Semicolon": 47, "Quote": 48, "Backquote": 49, "ShiftLeft": 50, "Backslash": 51,
	"KeyZ": 52, "KeyX": 53, "KeyC": 54, "KeyV": 55, "KeyB": 56, "KeyN": 57, "KeyM": 58,
	"Comma": 59, "Period": 60, "Slash": 61, "ShiftRight": 62, "NumpadMultiply": 63, "AltLeft": 64, "Space": 65, "CapsLock": 66,
	"F1": 67, "F2": 68, "F3": 69, "F4": 70, "F5": 71, "F6": 72, "F7": 73, "F8": 74, "F9": 75, "F10": 76,
	"NumLock": 77, "ScrollLock": 78, "Numpad7": 79, "Numpad8": 80, "Numpad9": 81, "NumpadSubtract": 82,
	"Numpad4": 83, "Numpad5": 84, "Numpad6": 85, "NumpadAdd": 86, "Numpad1": 87, "Numpad2": 88, "Numpad3": 89,
	"Numpad0": 90, "NumpadDecimal": 91, "IntlBackslash": 94, "F11": 95, "F12": 96, "NumpadEnter": 104,
	"ControlRight": 105, "NumpadDivide": 106, "PrintScreen": 107, "AltRight": 108, "Home": 110, "ArrowUp": 111,
	"PageUp": 112, "ArrowLeft": 113, "ArrowRight": 114, "End": 115, "ArrowDown": 116, "PageDown": 117,
	"Insert": 118, "Delete": 119, "AudioVolumeMute": 121, "AudioVolumeDown": 122, "AudioVolumeUp": 123,
	"Pause": 127, "MetaLeft": 133, "OSLeft": 133, "MetaRight": 134, "OSRight": 134, "ContextMenu": 135,
	"Sleep": 150, "MediaTrackNext": 171, "MediaPlayPause": 172, "MediaTrackPrevious": 173,
}

type pendingKey struct {
	code int
	at   time.Time
}

var (
	keyMu    sync.Mutex
	keyQueue []pendingKey
)

// listenPhysicalKeys relève chaque appui avant le pilote de Fyne (phase de
// capture), dans l'ordre où le pilote les transmet
func listenPhysicalKeys() {
	js.Global().Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) any {
		ev := args[0]
		if ev.Get("repeat").Bool() {
			return nil
		}
		code := -1
		if c, ok := keyCodes[ev.Get("code").String()]; ok {
			code = c
		}
		keyMu.Lock()
		keyQueue = append(keyQueue, pendingKey{code, time.Now()})
		if len(keyQueue) > 64 {
			keyQueue = keyQueue[len(keyQueue)-64:]
		}
		keyMu.Unlock()
		return nil
	}), true)
}

// keyScanCode code de l'appui relevé par la page ; les appuis trop anciens
// (hors du test clavier) sont ignorés
func keyScanCode(ev *fyne.KeyEvent) int {
	keyMu.Lock()
	defer keyMu.Unlock()
	for len(keyQueue) > 0 {
		k := keyQueue[0]
		keyQueue = keyQueue[1:]
		if time.Since(k.at) < 2*time.Second {
			return k.code
		}
	}
	return ev.Physical.ScanCode
}

// ---- Machine fictive ----------------------------------------------------------

// demoWebInventory HP ProBook 450 G3 de la vidéo de présentation
func demoWebInventory() []Component {
	return []Component{
		{Name: "Processeur", Icon: "microchip", Summary: "Intel(R) Core(TM) i3-6100U CPU @ 2.30GHz", Fields: []Field{{"Architecture", "x86_64"}, {"Cœurs", "4"}, {"Fréquence", "400 à 2300 MHz"}}},
		{Name: "Mémoire RAM", Icon: "memory", Summary: "8 Go DDR4", Fields: []Field{{"Total", "8,0 Go"}, {"Bottom-Slot 1", "4096 Mo DDR4 SODIMM @ 2133 MHz · Samsung"}, {"Bottom-Slot 2", "4096 Mo DDR4 SODIMM @ 2133 MHz · Samsung"}}},
		{Name: "Stockage", Icon: "hard-drive", Summary: "1 disque(s) · 298,1 Go", Fields: []Field{{"sda", "Toshiba MK3261GSYN · HDD · 298,1 Go"}}},
		{Name: "Batterie", Icon: "battery-half", Summary: "Santé 100 % · 112 cycles", Fields: []Field{{"Charge", "86 % (en charge)"}, {"Capacité réelle", "28,6 Wh, soit 100 %"}, {"Fabricant", "Hewlett-Packard"}, {"Tension", "16,29 V"}}},
		{Name: "Cartes réseau", Icon: "wifi", Summary: "2 interface(s) · 1 connectée(s)", Fields: []Field{{"enp1s0", "Realtek RTL8111/8168 · ethernet · connecté"}, {"wlp2s0", "Intel Wireless 3165 · wifi · déconnecté"}}},
		{Name: "Carte graphique", Icon: "display", Summary: "Intel Skylake GT2 [HD Graphics 520]", Fields: []Field{{"Modèle", "HD Graphics 520"}, {"Pilote", "i915"}}},
		{Name: "Affichage", Icon: "tv", Summary: "1 écran(s) connecté(s)", Fields: []Field{{"Écran intégré", "connecté 1366x768"}}},
	}
}
