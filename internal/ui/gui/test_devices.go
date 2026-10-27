package gui

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"gobox/internal/probe"
)

// ---- Ports USB et sorties vidéo -------------------------------------------

func runPorts(t *testCtx) result {
	t.progress(-1)
	t.log("Branchez une clé USB sur chaque port, puis un écran sur chaque sortie vidéo.")
	usbSeen, videoSeen := usbDevices(), connectedOutputs()
	t.log("%d périphérique(s) USB déjà présent(s), %d sortie(s) vidéo connectée(s)", len(usbSeen), len(videoSeen))

	var counter *canvas.Text
	t.panel(func() fyne.CanvasObject {
		counter = text("0 USB · 0 vidéo", 22, true, colInk)
		return box(container.NewVBox(caption("Branchements détectés"), counter), colLine, colPanel)
	})

	answer := t.ask("Tous les ports ont été testés ?", verdictChoices...)
	usbCount, videoCount := 0, 0
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-t.ctx.Done():
			return result{}
		case s := <-answer:
			t.clearAsk()
			return result{Status: s, Detail: fmt.Sprintf("%d branchement(s) USB · %d sortie(s) vidéo", usbCount, videoCount)}
		case <-tick.C:
			for path, name := range usbDevices() {
				if _, ok := usbSeen[path]; !ok {
					usbSeen[path] = name
					usbCount++
					t.log("USB : %s", name)
				}
			}
			for out := range connectedOutputs() {
				if _, ok := videoSeen[out]; !ok {
					videoSeen[out] = struct{}{}
					videoCount++
					t.log("Écran connecté sur %s", out)
				}
			}
			u, v := usbCount, videoCount
			t.ui(func() {
				counter.Text = fmt.Sprintf("%d USB · %d vidéo", u, v)
				counter.Refresh()
			})
		}
	}
}

// usbDevices périphériques USB présents (chemin sysfs → nom)
func usbDevices() map[string]string {
	devs := map[string]string{}
	dirs, _ := filepath.Glob("/sys/bus/usb/devices/*")
	for _, d := range dirs {
		vendor, err := os.ReadFile(filepath.Join(d, "idVendor"))
		if err != nil || strings.HasPrefix(filepath.Base(d), "usb") {
			continue // pas un périphérique, ou hub racine
		}
		name := readTrim(filepath.Join(d, "product"))
		if name == "" {
			name = strings.TrimSpace(string(vendor))
		}
		if m := readTrim(filepath.Join(d, "manufacturer")); m != "" {
			name = m + " " + name
		}
		devs[d] = name
	}
	return devs
}

// connectedOutputs connecteurs vidéo avec un écran branché (HDMI-A-1…)
func connectedOutputs() map[string]struct{} {
	outs := map[string]struct{}{}
	files, _ := filepath.Glob("/sys/class/drm/card*-*/status")
	for _, f := range files {
		if readTrim(f) == "connected" {
			name := filepath.Base(filepath.Dir(f))
			_, name, _ = strings.Cut(name, "-") // card0-HDMI-A-1 → HDMI-A-1
			outs[name] = struct{}{}
		}
	}
	return outs
}

func readTrim(path string) string {
	b, _ := os.ReadFile(path)
	return strings.TrimSpace(string(b))
}

// ---- Son ------------------------------------------------------------------

func runSound(t *testCtx) result {
	t.progress(-1)
	wav, err := writeStereoTones()
	if err != nil {
		t.log("Impossible de créer le son de test : %v", err)
		return result{Status: statusFail, Detail: err.Error()}
	}
	defer os.Remove(wav)

	player := firstAvailable("paplay", "pw-play", "aplay")
	if player == "" {
		t.log("Aucun lecteur audio trouvé (paplay, pw-play, aplay)")
		return result{Status: statusWarn, Detail: "Lecteur audio absent"}
	}

	for {
		t.log("Lecture : 440 Hz à gauche, puis 660 Hz à droite (%s)", player)
		out, err := userCommand(t.ctx, player, wav).CombinedOutput()
		if err != nil && !t.cancelled() {
			t.log("Erreur de lecture : %v %s", err, strings.TrimSpace(string(out)))
		}
		s := t.choose("Le son est-il sorti à gauche puis à droite ?",
			append([]choice{{"Rejouer", statusRunning}}, verdictChoices...)...)
		switch s {
		case statusIdle:
			return result{}
		case statusRunning:
			continue
		}
		detail := map[status]string{statusOK: "Haut-parleurs gauche + droite", statusWarn: "Son dégradé", statusFail: "Pas de son"}[s]
		return result{Status: s, Detail: detail}
	}
}

// writeStereoTones fichier WAV : 1 s à gauche (440 Hz) puis 1 s à droite (660 Hz)
func writeStereoTones() (string, error) {
	const rate = 44100
	f, err := os.CreateTemp("", "gobox-son-*.wav")
	if err != nil {
		return "", err
	}
	defer f.Close()

	n := rate * 2
	samples := make([]int16, 0, n*2)
	for i := 0; i < n; i++ {
		freq, left := 440.0, true
		if i >= rate {
			freq, left = 660.0, false
		}
		v := int16(math.Sin(2*math.Pi*freq*float64(i)/rate) * 12000)
		if left {
			samples = append(samples, v, 0)
		} else {
			samples = append(samples, 0, v)
		}
	}
	size := uint32(len(samples) * 2)
	header := []any{
		[4]byte{'R', 'I', 'F', 'F'}, 36 + size, [4]byte{'W', 'A', 'V', 'E'},
		[4]byte{'f', 'm', 't', ' '}, uint32(16), uint16(1), uint16(2), uint32(rate), uint32(rate * 4), uint16(4), uint16(16),
		[4]byte{'d', 'a', 't', 'a'}, size,
	}
	for _, h := range header {
		if err := binary.Write(f, binary.LittleEndian, h); err != nil {
			return "", err
		}
	}
	if err := binary.Write(f, binary.LittleEndian, samples); err != nil {
		return "", err
	}
	// lisible par l'utilisateur de la session si gobox tourne en root
	_ = f.Chmod(0o644)
	return f.Name(), nil
}

// ---- Réseau ---------------------------------------------------------------

func runNetwork(t *testCtx) result {
	t.progress(-1)
	ifaces, err := probe.ListNetworkInterfaces()
	if err != nil {
		t.log("Erreur : %v", err)
		return result{Status: statusFail, Detail: err.Error()}
	}
	linked := 0
	var cards []string
	for _, n := range ifaces {
		state := "pas de lien"
		if n.IsUp && n.Carrier {
			state = "lien actif"
			linked++
		}
		t.log("%s (%s) : %s %s %s", n.Name, n.Type, state, n.Speed, n.IPAddress)
		cards = append(cards, n.Type)
	}
	if linked == 0 {
		return result{Status: statusFail, Detail: fmt.Sprintf("%d interface(s), aucun lien", len(ifaces))}
	}

	ctx, cancel := context.WithTimeout(t.ctx, 5*time.Second)
	defer cancel()
	if _, err := net.DefaultResolver.LookupHost(ctx, "www.google.fr"); err != nil {
		t.log("DNS : échec (%v)", err)
		return result{Status: statusWarn, Detail: "Lien OK, pas de résolution DNS"}
	}
	t.log("DNS : OK")

	start := time.Now()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://connectivitycheck.gstatic.com/generate_204", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.log("Internet : échec (%v)", err)
		return result{Status: statusWarn, Detail: "Lien OK, pas d'accès internet"}
	}
	resp.Body.Close()
	latency := time.Since(start).Round(time.Millisecond)
	t.log("Internet : OK (%s)", latency)
	return result{Status: statusOK, Detail: fmt.Sprintf("%s · internet OK (%s)", strings.Join(unique(cards), " + "), latency)}
}

func unique(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// ---- Souris / pavé tactile ------------------------------------------------

func runMouse(t *testCtx) result {
	t.progress(-1)
	t.log("Dans la zone grise : déplacez le pointeur, clic gauche, clic droit, défilement.")
	t.panel(func() fyne.CanvasObject { return newMouseZone() })
	s := t.choose("Souris / pavé tactile :", verdictChoices...)
	detail := map[status]string{statusOK: "Déplacement, clics et défilement", statusWarn: "Fonctionne avec réserve", statusFail: "Défectueux"}[s]
	return result{Status: s, Detail: detail}
}

type mouseZone struct {
	widget.BaseWidget
	checks map[string]*widget.Check
	root   fyne.CanvasObject
}

func newMouseZone() *mouseZone {
	z := &mouseZone{checks: map[string]*widget.Check{}}
	row := container.NewHBox()
	for _, k := range []string{"Déplacement", "Clic gauche", "Clic droit", "Défilement"} {
		c := widget.NewCheck(k, nil)
		c.Disable()
		z.checks[k] = c
		row.Add(c)
	}
	bg := canvas.NewRectangle(colPanel)
	bg.CornerRadius = 6
	bg.SetMinSize(fyne.NewSize(0, 160))
	z.root = container.NewVBox(container.NewStack(bg, container.NewCenter(text("Zone de test", 14, true, colMuted))), row)
	z.ExtendBaseWidget(z)
	return z
}

func (z *mouseZone) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(z.root) }
func (z *mouseZone) mark(k string)                       { z.checks[k].SetChecked(true) }
func (z *mouseZone) Tapped(*fyne.PointEvent)             { z.mark("Clic gauche") }
func (z *mouseZone) TappedSecondary(*fyne.PointEvent)    { z.mark("Clic droit") }
func (z *mouseZone) Scrolled(*fyne.ScrollEvent)          { z.mark("Défilement") }
func (z *mouseZone) MouseIn(*desktop.MouseEvent)         {}
func (z *mouseZone) MouseMoved(*desktop.MouseEvent)      { z.mark("Déplacement") }
func (z *mouseZone) MouseOut()                           {}

// ---- Webcam ---------------------------------------------------------------

func runWebcam(t *testCtx) result {
	t.progress(-1)
	cams, _ := filepath.Glob("/dev/video*")
	if len(cams) == 0 {
		t.log("Aucun périphérique /dev/video* détecté")
		s := t.choose("Aucune webcam détectée :", choice{"Pas de webcam", statusNA}, choice{"Nul", statusFail})
		return result{Status: s, Detail: "Aucune caméra détectée"}
	}
	t.log("Caméra(s) : %s", strings.Join(cams, ", "))

	var viewer *exec.Cmd
	switch {
	case firstAvailable("ffplay") != "":
		viewer = userCommand(t.ctx, "ffplay", "-loglevel", "quiet", "-f", "v4l2", "-i", cams[0])
	case firstAvailable("mpv") != "":
		viewer = userCommand(t.ctx, "mpv", "av://v4l2:"+cams[0])
	case firstAvailable("cheese") != "":
		viewer = userCommand(t.ctx, "cheese")
	}
	if viewer == nil {
		t.log("Aucun visualiseur (ffplay, mpv, cheese) : sudo apt install ffmpeg")
	} else if err := viewer.Start(); err != nil {
		t.log("Impossible d'ouvrir l'aperçu : %v", err)
	} else {
		t.log("Aperçu ouvert dans une autre fenêtre")
		defer func() { _ = viewer.Process.Kill(); _ = viewer.Wait() }()
	}

	s := t.choose("L'image de la webcam est-elle correcte ?", verdictChoices...)
	return result{Status: s, Detail: fmt.Sprintf("%d caméra(s) · %s", len(cams), s.style().label)}
}

// ---- Carte SD -------------------------------------------------------------

func runSD(t *testCtx) result {
	t.progress(-1)
	before := sdCards()
	if len(before) > 0 {
		t.log("Carte déjà présente : %s", strings.Join(before, ", "))
		return result{Status: statusOK, Detail: "Carte détectée : " + before[0]}
	}
	t.log("Insérez une carte SD dans le lecteur…")
	answer := t.ask("Sans détection :", choice{"Pas de lecteur", statusNA}, choice{"Nul", statusFail})
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-t.ctx.Done():
			return result{}
		case s := <-answer:
			t.clearAsk()
			return result{Status: s, Detail: map[status]string{statusNA: "Pas de lecteur", statusFail: "Carte non détectée"}[s]}
		case <-tick.C:
			if cards := sdCards(); len(cards) > 0 {
				t.clearAsk()
				t.log("Carte détectée : %s", cards[0])
				return result{Status: statusOK, Detail: "Carte détectée : " + cards[0]}
			}
		}
	}
}

// sdCards cartes SD/MMC présentes (lecteurs PCI mmcblk)
func sdCards() []string {
	devs, _ := filepath.Glob("/sys/block/mmcblk*")
	var out []string
	for _, d := range devs {
		out = append(out, filepath.Base(d))
	}
	return out
}

// ---- Utilitaires ----------------------------------------------------------

func firstAvailable(names ...string) string {
	for _, n := range names {
		if _, err := exec.LookPath(n); err == nil {
			return n
		}
	}
	return ""
}

// userCommand lance un programme dans la session de l'utilisateur
// (son, vidéo, navigateur), même quand gobox tourne en root via sudo.
func userCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	user, uid := os.Getenv("SUDO_USER"), os.Getenv("SUDO_UID")
	if os.Geteuid() != 0 || user == "" {
		return exec.CommandContext(ctx, name, args...)
	}
	env := []string{
		"-u", user, "env",
		"DISPLAY=" + os.Getenv("DISPLAY"),
		"XAUTHORITY=" + os.Getenv("XAUTHORITY"),
		"XDG_RUNTIME_DIR=/run/user/" + uid,
		name,
	}
	return exec.CommandContext(ctx, "sudo", append(env, args...)...)
}
