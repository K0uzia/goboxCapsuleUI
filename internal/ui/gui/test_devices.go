package gui

import (
	"bytes"
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
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"gobox/internal/probe"
)

// ---- Ports USB et sorties vidéo -------------------------------------------

// portTile compteur « testés / détectés » d'un type de port
type portTile struct {
	root          fyne.CanvasObject
	bg            *tRect
	count, status *tText
	bar           *thinBar
	tested, total int
}

func newPortTile(ico, label string, total int) *portTile {
	p := &portTile{total: total, bg: newRect(cSurface2, cLine, 8), bar: newThinBar(cAccent, 5)}
	p.count = text("", 24, true, cText)
	p.count.Mono = true
	p.status = text("", 11, false, cMuted)
	head := container.NewHBox(iconOf(ico, cMuted, 15), container.NewCenter(caption(label)))
	p.root = container.NewStack(p.bg, container.New(&padLayout{h: 12, v: 10}, container.NewVBox(head, p.count, p.bar.root, p.status)))
	p.set(0)
	return p
}

func (p *portTile) set(tested int) {
	p.tested = tested
	p.count.SetText(fmt.Sprintf("%d / %d", tested, p.total))
	p.bar.set(float32(tested) / float32(max(p.total, 1)))
	switch {
	case tested >= p.total:
		p.bg.setRoles(cSuccessBg, cSuccess)
		p.count.Role, p.status.Text = cSuccess, "Tous testés"
	case tested > 0:
		p.bg.setRoles(cSurface2, cAccent)
		p.count.Role, p.status.Text = cText, fmt.Sprintf("%d restant(s)", p.total-tested)
	default:
		p.bg.setRoles(cSurface2, cLine)
		p.count.Role, p.status.Text = cText, "Aucun testé"
	}
	p.count.Refresh()
	p.status.Refresh()
}

func runPorts(t *testCtx) result {
	t.progress(-1)
	usb := readUSBPorts()
	conns := videoConnectors()
	if simulated {
		t.markSimulated()
		usb = usbPorts{group: map[string]string{}, total: 4}
		conns = []videoConnector{{Name: "HDMI-A-1", Type: "HDMI"}, {Name: "VGA-1", Type: "VGA"}, {Name: "eDP-1", Type: internalScreen, Connected: true, Mode: "1920x1080"}}
	}

	// un port déjà occupé (clavier, souris, écran) est un port qui marche
	usbTested := map[string]bool{}
	known := usbDevices()
	for dev := range known {
		if port := usb.portOf(dev); port != "" {
			usbTested[port] = true
		}
	}
	videoTotal, videoTested := map[string]int{}, map[string]map[string]bool{}
	var screen string
	for _, c := range conns {
		if c.Type == internalScreen {
			screen = "non connecté"
			if c.Connected {
				screen = "connecté " + c.Mode
			}
			continue
		}
		videoTotal[c.Type]++
		if videoTested[c.Type] == nil {
			videoTested[c.Type] = map[string]bool{}
		}
		if c.Connected {
			videoTested[c.Type][c.Name] = true
		}
	}
	t.log("%d port(s) USB détecté(s), dont %d déjà occupé(s)", usb.total, len(usbTested))
	for _, typ := range videoTypes {
		if videoTotal[typ] > 0 {
			t.log("%s : %d connecteur(s) déclaré(s) par la carte graphique", typ, videoTotal[typ])
		}
	}

	tiles := map[string]*portTile{}
	t.panel(func() fyne.CanvasObject {
		var objs []fyne.CanvasObject
		if usb.total > 0 {
			tiles["USB"] = newPortTile("usb", "USB", usb.total)
			objs = append(objs, tiles["USB"].root)
		}
		for _, typ := range videoTypes {
			if videoTotal[typ] > 0 {
				tiles[typ] = newPortTile("display", typ, videoTotal[typ])
				objs = append(objs, tiles[typ].root)
			}
		}
		content := container.NewVBox(
			instruction("plug", "Branchez une clé USB sur chaque port, puis un écran sur chaque sortie vidéo. Un port déjà occupé compte comme testé."),
			container.New(&respGrid{minCol: 130, cols: []int{len(objs), 3, 2}, gap: 8}, objs...),
		)
		if screen != "" {
			content.Add(listRow(container.NewHBox(iconOf("computer", cMuted, 14), text(internalScreen, 13, false, cText)), text(screen, 13, true, cText)))
		}
		content.Add(mutedNote("La carte graphique peut déclarer plus de connecteurs qu'il n'y en a sur la machine : seuls ceux présents sont à tester."))
		return content
	})
	update := func() {
		t.ui(func() {
			if tl := tiles["USB"]; tl != nil {
				tl.set(len(usbTested))
			}
			for typ, set := range videoTested {
				if tl := tiles[typ]; tl != nil {
					tl.set(len(set))
				}
			}
		})
	}
	update()

	answer := t.ask("Tous les ports présents ont été testés ?", verdictChoices...)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-t.ctx.Done():
			return result{}
		case s := <-answer:
			t.clearAsk()
			data := map[string]string{"usb": fmt.Sprintf("%d / %d", len(usbTested), usb.total)}
			parts := []string{"USB " + data["usb"]}
			for _, typ := range videoTypes {
				if n := videoTotal[typ]; n > 0 {
					data[typ] = fmt.Sprintf("%d / %d", len(videoTested[typ]), n)
					parts = append(parts, typ+" "+data[typ])
				}
			}
			if screen != "" {
				data["screen"] = screen
			}
			return result{Status: s, Detail: strings.Join(parts, " · "), Data: data}
		case <-tick.C:
			for dev, name := range usbDevices() {
				if _, ok := known[dev]; ok {
					continue
				}
				known[dev] = name
				port := usb.portOf(dev)
				switch {
				case port == "":
					t.log("USB : %s (port non identifié)", name)
				case usbTested[port]:
					t.log("USB : %s, port déjà testé", name)
				default:
					usbTested[port] = true
					t.log("USB : %s, port %d / %d", name, len(usbTested), usb.total)
				}
			}
			for _, c := range videoConnectors() {
				if set := videoTested[c.Type]; set != nil && c.Connected && !set[c.Name] {
					set[c.Name] = true
					t.log("Écran détecté sur %s (%s)", c.Name, c.Mode)
				}
			}
			update()
		}
	}
}

// instruction consigne mise en avant, avec une icône
func instruction(ico, s string) fyne.CanvasObject {
	l := wrapLabel(s)
	l.TextStyle.Bold = true
	return container.NewStack(newRect(cAccentSoft, cAccent, 8), container.New(&padLayout{h: 10, v: 2},
		container.NewBorder(nil, nil, container.NewVBox(container.New(&padLayout{v: 9}, iconOf(ico, cAccent, 16))), nil, l)))
}

// ---- Son ------------------------------------------------------------------

// soundCards cartes son reconnues par le système (ALSA)
func soundCards() []string {
	var out []string
	for _, line := range strings.Split(readTrim("/proc/asound/cards"), "\n") {
		// « 0 [PCH ]: HDA-Intel - HDA Intel PCH »
		if _, name, ok := strings.Cut(line, " - "); ok {
			out = append(out, strings.TrimSpace(name))
		}
	}
	return out
}

// soundParts parties du test son, chacune reportée sur sa ligne de la fiche
var soundParts = []struct{ key, label string }{
	{"speakers", "Haut-parleurs"}, {"jack", "Prise casque"}, {"mic_int", "Micro interne"}, {"mic_in", "Entrée micro"},
}

// statusData valeur d'un statut dans Result.Data (« st:ok »), lue par la fiche
func statusData(s status) string { return "st:" + strings.ToLower(s.style().label) }

func runSound(t *testCtx) result {
	t.progress(0)
	cards := soundCards()
	data := map[string]string{}
	if len(cards) == 0 && !simulated {
		// le matériel est là (PCI) mais le système ne l'a pas pris en charge
		var hw []string
		for _, slot := range pciDevices("0x0403") {
			hw = append(hw, orUnknown(pciName(slot), slot))
		}
		if len(hw) == 0 {
			t.log("Aucune carte son sur cette machine")
			for _, p := range soundParts {
				data[p.key] = statusData(statusNA)
			}
			data["card"] = statusData(statusNA)
			return result{Status: statusNA, Detail: "Aucune carte son", Data: data}
		}
		t.log("Carte son présente : %s", strings.Join(hw, ", "))
		t.log("Erreur : aucun pilote son chargé, le système ne peut rien jouer ni enregistrer")
		t.log("Cause probable : noyau sans ses modules (le noyau démarré ne correspond pas à l'image)")
		t.panel(func() fyne.CanvasObject {
			return instruction("volume-xmark", "Carte son présente mais non reconnue par le système : aucun son ne peut être joué ni enregistré. Ce n'est pas forcément une panne de la carte.")
		})
		s := t.choose("Résultat à inscrire :", choice{"Nul", statusFail}, choice{"Non testable", statusNA})
		for _, p := range soundParts {
			data[p.key] = statusData(s)
		}
		data["card"] = statusData(s)
		return result{Status: s, Detail: "Carte son sans pilote : " + strings.Join(hw, ", "), Data: data}
	}
	if simulated {
		t.markSimulated()
	}
	player := firstAvailable("pw-play", "paplay", "aplay")
	if player == "" && !simulated {
		t.log("Erreur : aucun lecteur audio trouvé (pw-play, paplay, aplay)")
		return result{Status: statusWarn, Detail: "Lecteur audio absent"}
	}
	t.log("Carte(s) son : %s", orUnknown(strings.Join(cards, ", "), "simulée"))

	play := func(path, what string) {
		t.log("Lecture : %s", what)
		if simulated {
			return
		}
		out, err := userCommand(t.ctx, player, path).CombinedOutput()
		if err != nil && !t.cancelled() {
			t.log("Erreur de lecture (%s) : %v %s", player, err, strings.TrimSpace(string(out)))
		}
	}
	tone := func(channel string) {
		wav, err := writeTone(channel)
		if err != nil {
			t.log("Erreur : son de test impossible à créer (%v)", err)
			return
		}
		defer os.Remove(wav)
		play(wav, map[string]string{"left": "gauche (440 Hz)", "right": "droite (660 Hz)", "both": "gauche puis droite"}[channel])
	}
	toneButtons := func() fyne.CanvasObject {
		btn := func(label, ico, ch string) fyne.CanvasObject {
			return secondaryButton(label, buttonIcon(ico), func() { go tone(ch) })
		}
		return container.New(&respGrid{minCol: 120, cols: []int{3, 1}, gap: 8},
			btn("Gauche", "volume-high", "left"), btn("Droite", "volume-high", "right"), btn("Gauche puis droite", "headphones", "both"))
	}
	step := func(n int, title string, body ...fyne.CanvasObject) {
		t.progress(float64(n-1) / float64(len(soundParts)))
		t.panel(func() fyne.CanvasObject {
			head := container.NewHBox(text(fmt.Sprintf("Étape %d / %d", n, len(soundParts)), 12, false, cMuted), text(title, 15, true, cText))
			return container.NewVBox(append([]fyne.CanvasObject{head}, body...)...)
		})
	}
	withAbsent := append(append([]choice{}, verdictChoices...), choice{"Absent", statusNA})
	results := map[string]status{}

	// 1. Haut-parleurs
	step(1, "Haut-parleurs", instruction("volume-high", "La tonalité doit sortir uniquement du côté indiqué, sans grésillement."), toneButtons())
	go tone("both")
	if results["speakers"] = t.choose("Haut-parleurs :", verdictChoices...); results["speakers"] == statusIdle {
		return result{}
	}
	// 2. Prise casque
	step(2, "Prise casque", instruction("headphones", "Branchez un casque sur la prise jack, puis écoutez chaque côté."), toneButtons())
	if results["jack"] = t.choose("Sortie audio (prise casque) :", withAbsent...); results["jack"] == statusIdle {
		return result{}
	}
	// 3 et 4. Micros : enregistrement, niveau capté, réécoute
	for i, part := range []struct{ key, title, hint string }{
		{"mic_int", "Micro interne", "Débranchez le casque, cliquez sur Enregistrer puis parlez 3 secondes près de l'écran ou du clavier."},
		{"mic_in", "Entrée micro", "Branchez un micro (ou un casque-micro) sur la prise, cliquez sur Enregistrer puis parlez 3 secondes."},
	} {
		level := newThinBar(cAccent, 6)
		levelText := text("Aucun enregistrement", 12, false, cMuted)
		var last string
		recording := false
		var recBtn, playBtn *widget.Button
		recBtn = primaryButton("Enregistrer 3 s", buttonIcon("microphone"), func() {
			if recording {
				return
			}
			recording = true
			recBtn.Disable()
			levelText.SetText("Enregistrement…")
			go func() {
				path, peak, err := recordMic(t, 3*time.Second)
				t.ui(func() {
					recording = false
					recBtn.Enable()
					if err != nil {
						levelText.SetText("Échec de l'enregistrement")
						return
					}
					if last != "" {
						os.Remove(last)
					}
					last = path
					level.set(float32(peak))
					levelText.SetText(fmt.Sprintf("Niveau maximal capté : %.0f %%", peak*100))
					playBtn.Enable()
				})
				if err != nil {
					t.log("Erreur d'enregistrement : %v", err)
				} else if peak < 0.02 {
					t.log("Attention : presque aucun son capté (%.0f %%)", peak*100)
				} else {
					t.log("Son capté : niveau maximal %.0f %%", peak*100)
				}
			}()
		})
		playBtn = secondaryButton("Réécouter", buttonIcon("volume-high"), func() {
			if last != "" {
				go play(last, "enregistrement")
			}
		})
		playBtn.Disable()
		step(3+i, part.title, instruction("microphone", part.hint),
			container.NewHBox(recBtn, playBtn), container.NewBorder(nil, nil, nil, levelText, container.NewVBox(layout.NewSpacer(), level.root, layout.NewSpacer())))
		results[part.key] = t.choose(part.title+" : la voix est-elle captée et audible à la réécoute ?", withAbsent...)
		if last != "" {
			os.Remove(last)
		}
		if results[part.key] == statusIdle {
			return result{}
		}
	}
	t.progress(1)

	// verdict : la pire des parties présentes
	overall, parts := statusNA, []string{}
	rank := map[status]int{statusNA: 0, statusOK: 1, statusWarn: 2, statusFail: 3}
	for _, p := range soundParts {
		s := results[p.key]
		data[p.key] = statusData(s)
		parts = append(parts, p.label+" "+strings.ToLower(s.style().label))
		if rank[s] > rank[overall] {
			overall = s
		}
	}
	// la carte fonctionne dès qu'une sortie produit du son
	card := statusOK
	if results["speakers"] == statusFail && results["jack"] != statusOK && results["jack"] != statusWarn {
		card = statusFail
	}
	data["card"] = statusData(card)
	return result{Status: overall, Detail: strings.Join(parts, " · "), Data: data}
}

// recordMic enregistre le micro par défaut de la session et rend le fichier
// et le niveau maximal capté (0 à 1)
func recordMic(t *testCtx, d time.Duration) (string, float64, error) {
	if simulated {
		time.Sleep(d)
		return "", 0.4, nil
	}
	// dossier ouvert à tous : l'enregistreur tourne en utilisateur de la
	// session quand gobox tourne en root
	dir, err := os.MkdirTemp("", "gobox-micro-*")
	if err != nil {
		return "", 0, err
	}
	_ = os.Chmod(dir, 0o777)
	path := filepath.Join(dir, "micro.wav")
	ctx, cancel := context.WithTimeout(t.ctx, d+3*time.Second)
	defer cancel()
	var cmd *exec.Cmd
	switch {
	case firstAvailable("arecord") != "":
		cmd = userCommand(ctx, "arecord", "-q", "-d", fmt.Sprint(int(d.Seconds())), "-f", "S16_LE", "-r", "44100", "-c", "1", path)
	case firstAvailable("pw-record") != "":
		ctx2, cancel2 := context.WithTimeout(t.ctx, d)
		defer cancel2()
		cmd = userCommand(ctx2, "pw-record", "--rate", "44100", "--channels", "1", "--format", "s16", path)
		cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
		cmd.WaitDelay = 2 * time.Second
	default:
		return "", 0, fmt.Errorf("aucun enregistreur (arecord, pw-record)")
	}
	out, err := cmd.CombinedOutput()
	b, rerr := os.ReadFile(path)
	if rerr != nil {
		if err == nil {
			err = rerr
		}
		return "", 0, fmt.Errorf("%v %s", err, strings.TrimSpace(string(out)))
	}
	_ = os.Chmod(path, 0o644)
	return path, wavPeak(b), nil
}

// wavPeak niveau maximal (0 à 1) des échantillons 16 bits d'un fichier WAV
func wavPeak(b []byte) float64 {
	i := bytes.Index(b, []byte("data"))
	if i < 0 || i+8 > len(b) {
		return 0
	}
	peak := 0
	for j := i + 8; j+1 < len(b); j += 2 {
		v := int(int16(binary.LittleEndian.Uint16(b[j:])))
		if v < 0 {
			v = -v
		}
		peak = max(peak, v)
	}
	return float64(peak) / 32768
}

// writeTone fichier WAV de test : left (1 s, 440 Hz à gauche), right (1 s,
// 660 Hz à droite) ou both (les deux à la suite)
func writeTone(channel string) (string, error) {
	const rate = 44100
	f, err := os.CreateTemp("", "gobox-son-*.wav")
	if err != nil {
		return "", err
	}
	defer f.Close()

	var parts []bool // true : gauche
	switch channel {
	case "left":
		parts = []bool{true}
	case "right":
		parts = []bool{false}
	default:
		parts = []bool{true, false}
	}
	samples := make([]int16, 0, rate*2*len(parts))
	for _, left := range parts {
		freq := 440.0
		if !left {
			freq = 660.0
		}
		for i := 0; i < rate; i++ {
			v := int16(math.Sin(2*math.Pi*freq*float64(i)/rate) * 12000)
			if left {
				samples = append(samples, v, 0)
			} else {
				samples = append(samples, 0, v)
			}
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

// mouseActions actions à faire dans la zone de test, dans l'ordre affiché
var mouseActions = []struct{ key, label, icon string }{
	{"move", "Déplacer", "arrow-pointer"},
	{"left", "Clic gauche", "computer-mouse"},
	{"middle", "Clic molette", "computer-mouse"},
	{"right", "Clic droit", "computer-mouse"},
	{"up", "Molette haut", "arrows-up-down"},
	{"down", "Molette bas", "arrows-up-down"},
}

func runMouse(t *testCtx) result {
	t.progress(0)
	var zone *mouseZone
	t.panel(func() fyne.CanvasObject {
		zone = newMouseZone(func(done, total int) {
			t.progress(float64(done) / float64(total))
			if done == total {
				t.log("Toutes les actions ont été détectées")
			}
		})
		return zone
	})
	s := t.choose("Souris / pavé tactile :", verdictChoices...)
	if s == statusIdle {
		return result{}
	}
	var missing []string
	fyne.DoAndWait(func() { missing = zone.missing() })
	detail := "Déplacement, 3 clics et molette"
	if len(missing) > 0 {
		detail = "Non détecté : " + strings.ToLower(strings.Join(missing, ", "))
	}
	return result{Status: s, Detail: detail}
}

// mouseZone zone de test : chaque action détectée passe sa case au vert
type mouseZone struct {
	widget.BaseWidget
	root    fyne.CanvasObject
	area    *tRect
	hint    *tText
	counter *tText
	tiles   map[string]*mouseTile
	onDone  func(done, total int)
}

type mouseTile struct {
	bg    *tRect
	label *tText
	state *tText
	icon  *widget.Icon
	ico   string
	done  bool
}

func newMouseZone(onDone func(done, total int)) *mouseZone {
	z := &mouseZone{tiles: map[string]*mouseTile{}, onDone: onDone}
	var objs []fyne.CanvasObject
	for _, a := range mouseActions {
		mt := &mouseTile{bg: newRect(cSurface, cLine, 8), label: text(a.label, 12, true, cText), state: text("À faire", 11, false, cMuted), icon: widget.NewIcon(coloredIcon(a.icon, cMuted)), ico: a.icon}
		mt.label.Alignment, mt.state.Alignment = fyne.TextAlignCenter, fyne.TextAlignCenter
		z.tiles[a.key] = mt
		objs = append(objs, container.NewStack(mt.bg, container.New(&padLayout{h: 6, v: 8},
			container.NewVBox(sized(mt.icon, 18), mt.label, mt.state))))
	}
	z.area = newRect(cSurface2, cBorder, 10)
	z.area.MinSz = fyne.NewSize(0, 150)
	z.area.StrokeWidth = 2
	z.hint = text("Faites chaque action avec le pointeur dans ce cadre", 15, true, cText)
	z.hint.Alignment = fyne.TextAlignCenter
	z.counter = text("", 13, false, cMuted)
	z.counter.Alignment = fyne.TextAlignCenter
	z.root = container.NewVBox(
		container.NewStack(z.area, container.NewVBox(layout.NewSpacer(), z.hint, z.counter, layout.NewSpacer())),
		container.New(&respGrid{minCol: 120, cols: []int{6, 3, 2}, gap: 6}, objs...),
	)
	z.ExtendBaseWidget(z)
	z.paint()
	return z
}

func (z *mouseZone) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(z.root) }

func (z *mouseZone) mark(k string) {
	if mt := z.tiles[k]; mt != nil && !mt.done {
		mt.done = true
		z.paint()
	}
}

func (z *mouseZone) paint() {
	done := 0
	for _, mt := range z.tiles {
		if mt.done {
			done++
			mt.bg.setRoles(cSuccessBg, cSuccess)
			mt.state.Text, mt.state.Role = "Détecté", cSuccess
			mt.icon.SetResource(coloredIcon("circle-check", cSuccess))
		} else {
			mt.bg.setRoles(cSurface, cLine)
			mt.state.Text, mt.state.Role = "À faire", cMuted
			mt.icon.SetResource(coloredIcon(mt.ico, cMuted))
		}
		mt.state.Refresh()
	}
	total := len(z.tiles)
	z.counter.SetText(fmt.Sprintf("%d / %d actions détectées", done, total))
	if done == total {
		z.area.setRoles(cSuccessBg, cSuccess)
		z.hint.Text, z.hint.Role = "Tout fonctionne : choisissez OK ci-dessous", cSuccess
		z.hint.Refresh()
	}
	if z.onDone != nil {
		z.onDone(done, total)
	}
}

// missing actions pas encore détectées
func (z *mouseZone) missing() []string {
	var out []string
	for _, a := range mouseActions {
		if !z.tiles[a.key].done {
			out = append(out, a.label)
		}
	}
	return out
}

func (z *mouseZone) MouseDown(e *desktop.MouseEvent) {
	switch e.Button {
	case desktop.MouseButtonPrimary:
		z.mark("left")
	case desktop.MouseButtonSecondary:
		z.mark("right")
	case desktop.MouseButtonTertiary:
		z.mark("middle")
	}
}
func (z *mouseZone) MouseUp(*desktop.MouseEvent) {}
func (z *mouseZone) Scrolled(e *fyne.ScrollEvent) {
	if e.Scrolled.DY > 0 {
		z.mark("up")
	} else if e.Scrolled.DY < 0 {
		z.mark("down")
	}
}
func (z *mouseZone) MouseIn(*desktop.MouseEvent)    {}
func (z *mouseZone) MouseMoved(*desktop.MouseEvent) { z.mark("move") }
func (z *mouseZone) MouseOut()                      {}

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
		t.log("Aucun visualiseur (ffplay, mpv, cheese) : installer ffmpeg")
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
	t.panel(func() fyne.CanvasObject {
		return instruction("sd-card", "Insérez une carte SD dans le lecteur : elle est détectée toute seule.")
	})
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
// (son, vidéo, navigateur), même quand gobox tourne en root via sudo. La
// session graphique (X11 ou Wayland) et son bus D-Bus lui sont transmis.
func userCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	user, uid := sessionUser()
	if user == "" {
		return exec.CommandContext(ctx, name, args...)
	}
	runtime := "/run/user/" + uid
	env := []string{
		"-u", user, "env",
		"DISPLAY=" + os.Getenv("DISPLAY"),
		"XAUTHORITY=" + os.Getenv("XAUTHORITY"),
		"WAYLAND_DISPLAY=" + os.Getenv("WAYLAND_DISPLAY"),
		"XDG_RUNTIME_DIR=" + runtime,
		"DBUS_SESSION_BUS_ADDRESS=unix:path=" + runtime + "/bus",
		name,
	}
	return exec.CommandContext(ctx, "sudo", append(env, args...)...)
}
