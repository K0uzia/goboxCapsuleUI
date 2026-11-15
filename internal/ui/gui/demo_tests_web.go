//go:build demo && js

package gui

// Tests de la démo en ligne : chaque test garde le déroulé de l'application
// (consigne, attente, question au technicien, résultat), sur le matériel
// fictif. Ce qui demanderait le vrai PC (branchements, réseau, disque, mise à
// jour) est simulé et le dit dans le journal.

import (
	"fmt"
	"syscall/js"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// demoTests remplace les tests qui liraient le système par leur version simulée.
// Clavier et souris restent ceux de l'application (clavier et souris du
// visiteur), son aussi (tonalités jouées par le navigateur).
func demoTests(s *session) {
	runs := map[string]func(*testCtx) result{
		"stress":    demoStress,
		"ports":     demoPorts,
		"reseau":    demoNetwork,
		"webcam":    demoWebcam,
		"sd":        demoSD,
		"wifi":      demoWifi,
		"bluetooth": demoBluetooth,
		"batterie":  demoBattery,
		"chargeur":  demoCharger,
		"hdd":       demoSmart,
		"ssd":       demoNoDisk("ssd"),
		"nvme":      demoNoDisk("nvme"),
		"sim":       demoSlot("Modem SIM", "Aucun modem sur cette machine", "Pas de modem"),
		"lecteur":   demoSlot("Lecteur de puce", "Aucun lecteur de carte à puce sur cette machine", "Pas de lecteur de carte à puce"),
		"maj":       demoUpdate,
	}
	for _, def := range s.Tests {
		if run, ok := runs[def.ID]; ok {
			def.Run = run
		}
	}
}

// demoLog journal ligne à ligne, au rythme du vrai test ; faux si annulé
func demoLog(t *testCtx, step time.Duration, lines ...string) bool {
	for _, l := range lines {
		if !t.sleep(step) {
			return false
		}
		t.log("%s", l)
	}
	return true
}

// ---- Ports : branchements simulés -------------------------------------------

func demoPorts(t *testCtx) result {
	t.progress(-1)
	t.markSimulated()
	type plug struct{ kind, ico, log string }
	totals := []struct {
		kind, ico string
		n         int
	}{{usbKind2, "usb", 1}, {usbKind3, "usb", 2}, {usbKindC, "usb", 1}, {"HDMI", "display", 1}, {"VGA", "display", 1}}
	plugs := []plug{
		{usbKind3, "usb", "USB : clé USB (simulée), prise USB 3.x testée"},
		{usbKind3, "usb", "USB : clé USB (simulée), prise USB 3.x testée"},
		{usbKind2, "usb", "USB : souris (simulée), prise USB 2.0 testée"},
		{usbKindC, "usb", "USB : clé USB-C (simulée), prise USB-C testée"},
		{"HDMI", "display", "Écran HDMI détecté sur HDMI-A-1 (1920x1080, simulé)"},
		{"VGA", "display", "Écran VGA détecté sur VGA-1 (1280x1024, simulé)"},
	}
	tiles := map[string]*portTile{}
	done := map[string]int{}
	t.log("Démo : les branchements sont simulés, rien n'est lu sur votre ordinateur")
	t.log("4 prise(s) USB, dont 0 déjà occupée(s)")
	t.log("HDMI : 1 prise(s)")
	t.log("VGA : 1 prise(s)")
	t.panel(func() fyne.CanvasObject {
		grid := container.New(&respGrid{minCol: 130, cols: []int{4, 3, 2}, gap: 8})
		for _, k := range totals {
			tl := newPortTile(k.ico, k.kind, k.n)
			tiles[k.kind] = tl
			grid.Add(tl.root)
		}
		return container.NewVBox(
			instruction("plug", "Branchez une clé USB sur chaque port, puis un écran sur chaque sortie vidéo. Un port déjà occupé compte comme testé."),
			grid,
			mutedNote("Démo : les branchements arrivent tout seuls, une prise après l'autre."),
			listRow(container.NewHBox(iconOf("computer", cMuted, 14), text(internalScreen, 13, false, cText)), text("connecté 1366x768", 13, true, cText)),
		)
	})
	answer := t.ask("Tous les ports présents ont été testés ?", verdictChoices...)
	next := 0
	tick := time.NewTicker(1500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-t.ctx.Done():
			return result{}
		case s := <-answer:
			t.clearAsk()
			usb := done[usbKind2] + done[usbKind3] + done[usbKindC]
			data := map[string]string{"usb": portCount(usb, 4)}
			for _, k := range totals {
				data[k.kind] = portCount(done[k.kind], k.n)
			}
			data["screen"] = "connecté 1366x768"
			return result{Status: s, Detail: fmt.Sprintf("USB %s · HDMI %s · VGA %s", data["usb"], data["HDMI"], data["VGA"]), Data: data}
		case <-tick.C:
			if next < len(plugs) {
				p := plugs[next]
				next++
				done[p.kind]++
				t.log("%s", p.log)
				n := done[p.kind]
				t.ui(func() { tiles[p.kind].set(n) })
			}
		}
	}
}

// ---- Réseau ----------------------------------------------------------------

func demoNetwork(t *testCtx) result {
	t.progress(-1)
	t.markSimulated()
	t.log("Démo : rien n'est envoyé sur le réseau, les mesures sont simulées")
	if !demoLog(t, 500*time.Millisecond,
		"enp1s0 (ethernet) : lien actif 1000 Mb/s 192.168.1.42",
		"wlp2s0 (wifi) : pas de lien",
		"DNS : OK",
		"Internet : OK (38ms)",
		"Ping de 1.1.1.1…",
	) || !t.sleep(1500*time.Millisecond) {
		return result{}
	}
	ping := pingResult{Avg: 12, ok: true}
	return result{Status: statusOK, Detail: "ethernet + wifi · internet OK (38ms) · " + ping.text()}
}

// ---- Webcam ------------------------------------------------------------------

func demoWebcam(t *testCtx) result {
	t.progress(-1)
	t.markSimulated()
	t.log("Caméra : HP HD Camera (/dev/video0)")
	t.log("Démo : la caméra de votre ordinateur n'est pas ouverte")
	t.panel(func() fyne.CanvasObject {
		return instruction("camera", "Sur le PC, l'aperçu de la caméra s'ouvre dans une autre fenêtre. Dans la démo, rien n'est filmé : répondez comme le ferait le technicien.")
	})
	s := t.choose("L'image de la webcam est-elle correcte ?", verdictChoices...)
	if s == statusIdle {
		return result{}
	}
	return result{Status: s, Detail: "1 caméra(s) · " + s.style().label}
}

// ---- Carte SD : carte insérée au bout de quelques secondes -------------------

func demoSD(t *testCtx) result {
	t.progress(-1)
	t.markSimulated()
	t.panel(func() fyne.CanvasObject {
		return container.NewVBox(
			instruction("sd-card", "Insérez une carte SD dans le lecteur : elle est détectée toute seule."),
			mutedNote("Démo : une carte est « insérée » au bout de quelques secondes."),
		)
	})
	answer := t.ask("Sans détection :", choice{"Pas de lecteur", statusNA}, choice{"Nul", statusFail})
	select {
	case <-t.ctx.Done():
		return result{}
	case s := <-answer:
		t.clearAsk()
		return result{Status: s, Detail: map[status]string{statusNA: "Pas de lecteur", statusFail: "Carte non détectée"}[s]}
	case <-time.After(5 * time.Second):
		t.clearAsk()
		t.answered = false
		t.log("Carte détectée : mmcblk0 (16 Go, simulée)")
		return result{Status: statusOK, Detail: "Carte détectée : mmcblk0"}
	}
}

// ---- Wi-Fi et Bluetooth -----------------------------------------------------

func demoWifi(t *testCtx) result {
	t.progress(-1)
	t.markSimulated()
	if !demoLog(t, 600*time.Millisecond,
		"Carte : Intel Wireless 3165 (wlp2s0)",
		"Recherche des réseaux visibles…",
		"5 réseau(x) vu(s) (simulés)",
	) {
		return result{}
	}
	return result{Status: statusOK, Detail: "Carte reconnue : 5 réseau(x) vu(s)"}
}

func demoBluetooth(t *testCtx) result {
	t.progress(-1)
	t.markSimulated()
	if !demoLog(t, 700*time.Millisecond,
		"Adaptateur : Intel Bluetooth (hci0)",
		"Recherche des appareils (8 s)…",
		"2 appareil(s) vu(s) (simulés)",
	) {
		return result{}
	}
	return result{Status: statusOK, Detail: "Adaptateur reconnu : 2 appareil(s) vu(s)"}
}

// ---- Chargeur et charge : chargeur « branché » au bout de quelques secondes --

func demoWaitCharger(t *testCtx, ico, prompt string, opts ...choice) (status, bool) {
	t.panel(func() fyne.CanvasObject {
		return container.NewVBox(
			instruction(ico, "Branchez le chargeur : il est détecté tout seul."),
			mutedNote("Démo : le chargeur est « branché » au bout de quelques secondes."),
		)
	})
	answer := t.ask(prompt, opts...)
	select {
	case <-t.ctx.Done():
		return statusIdle, false
	case s := <-answer:
		t.clearAsk()
		return s, false
	case <-time.After(4 * time.Second):
		t.clearAsk()
		t.answered = false
		t.setPanelEmpty()
		return statusIdle, true
	}
}

func demoCharger(t *testCtx) result {
	t.progress(-1)
	t.markSimulated()
	s, online := demoWaitCharger(t, "bolt", "Sans détection :", choice{"Pas de chargeur", statusNA}, choice{"Nul", statusFail})
	switch {
	case t.cancelled():
		return result{}
	case !online && s == statusNA:
		return result{Status: statusNA, Detail: "Pas de chargeur fourni", Data: map[string]string{"pill": pillNoCharger}}
	case !online:
		return result{Status: statusFail, Detail: "Chargeur branché, non détecté"}
	}
	t.log("Chargeur détecté (65 W, simulé)")
	return result{Status: statusOK, Detail: "Chargeur détecté (65 W)"}
}

func demoBattery(t *testCtx) result {
	t.progress(-1)
	t.markSimulated()
	s, online := demoWaitCharger(t, "battery-half", "Sans chargeur, la charge ne peut pas être testée :", choice{"Pas de chargeur", statusNA}, choice{"Nul", statusFail})
	switch {
	case t.cancelled():
		return result{}
	case !online && s == statusNA:
		return result{Status: statusNA, Detail: "Non testée : pas de chargeur", Data: map[string]string{"pill": pillNoCharger}}
	case !online:
		return result{Status: statusFail, Detail: "La batterie ne charge pas"}
	}
	t.log("Chargeur détecté, mesure de la charge…")
	if !t.sleep(3 * time.Second) {
		return result{}
	}
	t.log("La batterie charge (25,0 W, simulé)")
	return result{Status: statusOK, Detail: "La batterie charge (25,0 W)"}
}

// ---- SMART du disque dur -------------------------------------------------------

func demoSmart(t *testCtx) result {
	t.progress(-1)
	t.markSimulated()
	if !demoLog(t, 500*time.Millisecond,
		"sda : Toshiba MK3261GSYN · HDD · 298,1 Go",
		"Lecture SMART (smartctl, simulée)…",
		"État général : PASSED",
		"Secteurs réalloués : 0 · en attente : 0",
		"Heures de fonctionnement : 8 412 h · température 34 °C",
	) {
		return result{}
	}
	return result{Status: statusOK, Detail: "SMART : disque sain"}
}

// demoNoDisk SMART d'un type de disque absent du PC fictif
func demoNoDisk(kind string) func(*testCtx) result {
	return func(t *testCtx) result {
		t.progress(-1)
		t.markSimulated()
		if !demoLog(t, 500*time.Millisecond, "sda : Toshiba MK3261GSYN · HDD · 298,1 Go", "Aucun "+kindNames[kind]+" sur cette machine") {
			return result{}
		}
		return result{Status: statusNA, Detail: "Aucun " + kindNames[kind]}
	}
}

// demoSlot modem SIM ou lecteur de puce : rien n'est vu, le technicien dit ce
// qu'il y a sur le PC (même déroulé que l'application)
func demoSlot(what, logLine, detail string) func(*testCtx) result {
	return func(t *testCtx) result {
		t.progress(-1)
		t.markSimulated()
		if !demoLog(t, 600*time.Millisecond, logLine) {
			return result{}
		}
		switch t.choose(what+" non vu par le système. Sur ce PC :",
			choice{"Présent, non testable", statusOK}, choice{"En panne", statusFail}, choice{"Absent", statusNA}) {
		case statusOK:
			return result{Status: statusOK, Detail: "Présent, non vu par le système", Data: map[string]string{"pill": pillPresent}}
		case statusFail:
			return result{Status: statusFail, Detail: "Présent, en panne"}
		case statusNA:
			return result{Status: statusNA, Detail: detail}
		}
		return result{}
	}
}

// ---- Mise à jour : l'accord du technicien, puis une mise à jour simulée --------

func demoUpdate(t *testCtx) result {
	t.markSimulated()
	t.waiting()
	t.log("En attente : la mise à jour ne démarre qu'après « Lancer la mise à jour »")
	t.panel(func() fyne.CanvasObject {
		return container.NewVBox(
			instruction("arrow-pointer", "En attente de votre accord : rien n'est lancé. Cliquez sur « Lancer la mise à jour » ci-dessous pour démarrer."),
			mutedNote("Met à jour Linux Mint 21.3 et ses paquets (apt-get). La modification est définitive. Ne pas éteindre le PC pendant l'opération."),
			mutedNote("Démo : rien n'est téléchargé ni installé."),
		)
	})
	if t.choose("Démarrer la mise à jour ?", choice{"Lancer la mise à jour", statusOK}, choice{"Ne pas faire", statusNA}) != statusOK {
		if t.cancelled() {
			return result{}
		}
		return result{Status: statusNA, Detail: "Non réalisée"}
	}
	t.progress(-1)
	t.setPanelEmpty()
	t.log("Mise à jour lancée")
	t.answered = false
	t.lockCancel()
	demoLog(t, 700*time.Millisecond,
		"Mise à jour de la liste des paquets…",
		"Lecture des listes de paquets… Fait",
		"Installation des mises à jour…",
		"0 mis à jour, 0 nouvellement installés, 0 à enlever (simulé)",
		"Mise à jour terminée",
	)
	return result{Status: statusOK, Detail: "Linux Mint 21.3 : système et paquets à jour"}
}

// ---- Son : tonalités jouées par le navigateur ----------------------------------

var audioCtx js.Value

// writeTone joue la tonalité de test dans le navigateur (gauche 440 Hz, droite
// 660 Hz) au lieu d'écrire un fichier : le test son garde son déroulé
func writeTone(channel string) (string, error) {
	if audioCtx.IsUndefined() || audioCtx.IsNull() {
		ctor := js.Global().Get("AudioContext")
		if ctor.IsUndefined() {
			return "", fmt.Errorf("son indisponible dans ce navigateur")
		}
		audioCtx = ctor.New()
	}
	audioCtx.Call("resume")
	parts := []float64{-1, 1}
	switch channel {
	case "left":
		parts = []float64{-1}
	case "right":
		parts = []float64{1}
	}
	start := audioCtx.Get("currentTime").Float()
	for i, pan := range parts {
		freq := 440.0
		if pan > 0 {
			freq = 660.0
		}
		at := start + float64(i)
		osc := audioCtx.Call("createOscillator")
		osc.Get("frequency").Set("value", freq)
		gain := audioCtx.Call("createGain")
		gain.Get("gain").Call("setValueAtTime", 0.15, at)
		gain.Get("gain").Call("setValueAtTime", 0, at+0.95)
		node := osc
		if panner := audioCtx.Get("createStereoPanner"); !panner.IsUndefined() {
			p := audioCtx.Call("createStereoPanner")
			p.Get("pan").Set("value", pan)
			osc.Call("connect", p)
			node = p
		}
		node.Call("connect", gain)
		gain.Call("connect", audioCtx.Get("destination"))
		osc.Call("start", at)
		osc.Call("stop", at+1)
	}
	return "", nil
}
