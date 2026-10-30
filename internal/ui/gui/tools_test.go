package gui

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// Parcours complet d'un test sans question : clic sur le bouton, résultat affiché
// et déjà inscrit dans la fiche.
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
	for !tools.view.resultBox.Visible() {
		if time.Now().After(deadline) {
			t.Fatal("le résultat n'est jamais affiché")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// aucune confirmation : le résultat est déjà dans la fiche
	if got := s.status("fake"); got != statusWarn {
		t.Fatalf("statut enregistré = %v, attendu MOYEN", got)
	}

	tools.close(false)
	if got := s.status("fake"); got != statusWarn {
		t.Fatalf("le retour au menu ne doit pas effacer le résultat : %v", got)
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

func TestFichePDF(t *testing.T) {
	s := newSession()
	s.Machine = machineInfo{Vendor: "HP", Product: "HP EliteBook 850 G3", Serial: "7XK2Q9A"}
	s.Inventory = demoInventory()
	s.Results["stress"] = result{Status: statusWarn, Detail: "41 °C au repos, 87 °C au maximum"}
	s.Results["sd"] = result{Status: statusFail, Detail: "Carte non détectée"}
	s.Results["ports"] = result{Status: statusOK, Data: map[string]string{"usb": "3 / 4", "HDMI": "1 / 1", "DisplayPort": "0 / 1"}}
	s.Form.Tech = "Alex"
	data, err := fichePDF(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("ce n'est pas un PDF")
	}
	if n := bytes.Count(data, []byte("/Type /Page\n")); n != 1 {
		t.Fatalf("%d page(s), attendu 1", n)
	}
	if out := os.Getenv("GOBOX_FICHE_OUT"); out != "" {
		if err := os.WriteFile(out, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// Le verdict donné par le technicien inscrit le résultat et ramène au menu, sans
// étape de plus ; une correction depuis l'écran de résultat fait de même.
func TestAnswerReturnsToMenu(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(true))
	w := a.NewWindow("test")
	s := newSession()
	s.Tests = []*testDef{{ID: "q", Title: "Question", Category: catPeripherals, Color: cPeripheral, Soft: cPeripheralBg,
		Run: func(tc *testCtx) result { return result{Status: tc.choose("Résultat ?", verdictChoices...)} }},
		{ID: "auto", Title: "Auto", Category: catPeripherals, Color: cPeripheral, Soft: cPeripheralBg,
			Run: func(tc *testCtx) result { return result{Status: statusOK} }}}
	tools := newToolsPanel(s, w)
	wait := func(cond func() bool, what string) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for !cond() {
			if time.Now().After(deadline) {
				t.Fatal(what)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	test.Tap(tools.buttons["q"])
	time.Sleep(50 * time.Millisecond)
	answerAsk(tools.view, 2) // Nul
	wait(func() bool { return tools.active == nil }, "pas de retour au menu après le verdict")
	if s.status("q") != statusFail || !tools.menu.Visible() {
		t.Fatalf("statut %v, menu visible %v", s.status("q"), tools.menu.Visible())
	}
	test.Tap(tools.buttons["auto"])
	wait(func() bool { return tools.view.resultBox.Visible() }, "résultat non affiché")
	tools.view.pickVerdict(statusWarn)
	if s.status("auto") != statusWarn || tools.active != nil {
		t.Fatalf("correction : %v, actif %v", s.status("auto"), tools.active)
	}
}

// Un écran analogique se reconnaît à son EDID, même vu comme DisplayPort.
func TestEDIDAnalog(t *testing.T) {
	edid := make([]byte, 128)
	copy(edid, []byte{0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0})
	if !edidAnalog(edid) {
		t.Error("bit 7 à 0 : analogique")
	}
	edid[20] = 0xa5
	if edidAnalog(edid) {
		t.Error("bit 7 à 1 : numérique")
	}
	if edidAnalog(nil) || edidAnalog(make([]byte, 128)) {
		t.Error("EDID absent ou invalide : pas analogique")
	}
}

// Les prises USB sont comptées par type ; les ports Type-C non reliés sont
// retirés des USB 3.
func TestUSBKinds(t *testing.T) {
	if betterKind(usbKind2, usbKind3) != usbKind3 || betterKind(usbKindC, usbKind3) != usbKindC || betterKind("", usbKind2) != usbKind2 {
		t.Error("type le plus capable attendu")
	}
	p := usbPorts{group: map[string]string{"a": "a", "a2": "a", "b": "b", "c": "c"}, kinds: map[string]string{"a": usbKind3, "b": usbKind3, "c": usbKind2}}
	got := p.kindTotals()
	if got[usbKind3] != 2 || got[usbKind2] != 1 || got[usbKindC] != 0 {
		t.Errorf("totaux : %v", got)
	}
	p.typeC = 1
	got = p.kindTotals()
	if got[usbKind3] != 1 || got[usbKindC] != 1 {
		t.Errorf("Type-C : %v", got)
	}

	s := newSession()
	s.Results["ports"] = result{Status: statusOK, Data: map[string]string{"usb": "3 / 4", usbKind2: "2 / 2", usbKind3: "1 / 2"}}
	if r := findRow(t, s, "Ports et périphériques", usbKind2); r.Status != statusOK {
		t.Errorf("USB 2 : %+v", r)
	}
	if r := findRow(t, s, "Ports et périphériques", usbKind3); r.pillLabel() != pillUntested {
		t.Errorf("USB 3 incomplet : %+v", r)
	}
}

// Les outils du système passent avant ceux embarqués.
func TestWithTools(t *testing.T) {
	if got := withTools("/usr/bin", "/tmp/x"); got != "/usr/bin:/tmp/x" {
		t.Errorf("PATH : %q", got)
	}
	if got := withTools("", "/tmp/x"); got != "/tmp/x" {
		t.Errorf("PATH vide : %q", got)
	}
}

// L'archive des outils s'extrait, droits compris, et un chemin qui sort du dossier
// temporaire est refusé.
func TestExtractTools(t *testing.T) {
	build := func(name string) []byte {
		var b bytes.Buffer
		gz := gzip.NewWriter(&b)
		tw := tar.NewWriter(gz)
		if err := tw.WriteHeader(&tar.Header{Name: "./bin/", Typeflag: tar.TypeDir, Mode: 0o755}); err != nil {
			t.Fatal(err)
		}
		if err := tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o755, Size: 2}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte("ok"))
		tw.Close()
		gz.Close()
		return b.Bytes()
	}
	dir := t.TempDir()
	if err := extractTools(build("./bin/outil"), dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "bin", "outil"))
	if err != nil || info.Mode()&0o100 == 0 {
		t.Fatalf("outil extrait : %v, %v", info, err)
	}
	if err := extractTools(build("../sorti"), dir); err == nil {
		t.Fatal("chemin hors du dossier accepté")
	}
}

// À la fermeture, gobox efface les dossiers de Fyne et ses fiches, sans toucher à
// ce qui ne lui appartient pas.
func TestRemoveTraces(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "c"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "g"))
	own := filepath.Join(home, "c", "fyne", appID)
	other := filepath.Join(home, "g", "fyne", "autre.appli")
	ownConfig := filepath.Join(home, "g", "fyne", appID)
	for _, d := range []string{own, other, ownConfig} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	fiche := filepath.Join(home, "Fiche.pdf")
	if err := os.WriteFile(fiche, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ficheFiles = []string{fiche}
	defer func() { ficheFiles = nil }()
	removeTraces()
	for _, gone := range []string{own, filepath.Join(home, "c", "fyne"), ownConfig, fiche} {
		if _, err := os.Stat(gone); err == nil {
			t.Errorf("%s reste sur le disque", gone)
		}
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("le dossier d'une autre application a été effacé : %v", err)
	}
}

// Les erreurs du noyau désignent la prise défectueuse.
func TestUSBKernelErrors(t *testing.T) {
	log := `[  12.1] usb 1-2: device descriptor read/64, error -71
[  13.0] usb usb1-port3: unable to enumerate USB device
[  14.0] usb 1-4: new high-speed USB device number 5 using xhci_hcd
[  15.0] usb 1-5.1: device not accepting address 7, error -71
[  16.0] usb usb2-port1: over-current condition
`
	got := parseUSBErrors(log)
	want := []string{"1-2", "usb1-port3", "1-5.1", "usb2-port1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("erreurs : %v", got)
	}
	p := usbPorts{group: map[string]string{"usb1-port2": "usb1-port2", "usb1-port3": "usb1-port3", "usb2-port1": "usb1-port1", "usb1-port5": "usb1-port5"}, possible: map[string]string{}}
	for name, port := range map[string]string{"1-2": "usb1-port2", "usb1-port3": "usb1-port3", "usb2-port1": "usb1-port1", "1-5.1": "usb1-port5", "9-9": ""} {
		if got := p.portOfError(name); got != port {
			t.Errorf("%s : %q, attendu %q", name, got, port)
		}
	}
	s := newSession()
	s.Results["ports"] = result{Status: statusOK, Data: map[string]string{"usb": "1 / 2", "usb_bad": "1 : USB 3.x"}}
	if r := findRow(t, s, "Ports et périphériques", "USB défectueux"); r.Status != statusFail || r.Value != "1 : USB 3.x" {
		t.Errorf("fiche : %+v", r)
	}
}

// Un aperçu lancé par sudo reçoit SIGTERM (sudo le relaie) et son groupe s'arrête :
// l'enfant ne doit pas survivre, sinon la caméra reste ouverte.
func TestStopProcessStopsChildren(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "enfant")
	cmd := exec.CommandContext(context.Background(), "sh", "-c", "sleep 60 & echo $! >"+pidFile+"; wait")
	setProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var pid int
	deadline := time.Now().Add(2 * time.Second)
	for pid == 0 && time.Now().Before(deadline) {
		b, _ := os.ReadFile(pidFile)
		pid, _ = strconv.Atoi(strings.TrimSpace(string(b)))
		time.Sleep(10 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("enfant non lancé")
	}
	stopProcess(cmd)
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat("/proc/" + strconv.Itoa(pid)); err != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("l'enfant %d tourne encore", pid)
}

// Charge et chargeur sont distincts : sans chargeur, la charge n'est pas notée.
func TestPower(t *testing.T) {
	root := t.TempDir()
	write := func(path, v string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(v+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if readCharger(root).Online || hasBattery(root) {
		t.Fatal("dossier vide : ni chargeur ni batterie")
	}
	write("BAT0/type", "Battery")
	write("AC/type", "Mains")
	write("AC/online", "0")
	write("ucsi/type", "USB")
	write("ucsi/online", "1")
	write("ucsi/voltage_max", "20000000")
	write("ucsi/current_max", "3250000")
	c := readCharger(root)
	if !hasBattery(root) || !c.Online || c.Watts != 65 {
		t.Fatalf("chargeur : %+v", c)
	}
	for state, want := range map[string]status{"Charging": statusOK, "Full": statusOK, "Not charging": statusWarn, "Discharging": statusFail} {
		if got, _ := chargeVerdict(state, 20); got != want {
			t.Errorf("%s : %v", state, got)
		}
	}
	s := newSession()
	s.Results["batterie"] = result{Status: statusNA, Detail: "Non testée : pas de chargeur", Data: map[string]string{"pill": pillNoCharger}}
	s.Results["chargeur"] = result{Status: statusNA, Data: map[string]string{"pill": pillNoCharger}}
	if r := findRow(t, s, "Batterie", "Test de charge"); r.pillLabel() != pillNoCharger {
		t.Errorf("charge : %+v", r)
	}
	findRow(t, s, "Batterie", "Chargeur")
	if v := s.verdict(); strings.Contains(v.Sub, "échec") {
		t.Errorf("verdict : %+v", v)
	}
}

func TestUpdateSteps(t *testing.T) {
	for manager, want := range map[string]string{"apt-get": "upgrade", "dnf": "upgrade"} {
		steps := updateSteps(manager)
		if len(steps) != 2 || steps[1].Args[len(steps[1].Args)-1] != want {
			t.Errorf("%s : %+v", manager, steps)
		}
	}
	if updateSteps("pacman") != nil {
		t.Error("gestionnaire inconnu : non pris en charge")
	}
}

func TestParsePing(t *testing.T) {
	out := "4 packets transmitted, 3 received, 25% packet loss, time 3004ms\nrtt min/avg/max/mdev = 10.1/12.6/15.2/2.0 ms\n"
	p := parsePing(out)
	if !p.ok || p.Loss != 25 || p.Avg != 12.6 || p.text() != "ping 13 ms, 25 % perdus" {
		t.Errorf("ping : %+v, %s", p, p.text())
	}
	if parsePing("ping: connect: Network is unreachable").ok {
		t.Error("sans résumé : pas de résultat")
	}
}

// Sans liste du BIOS, seules les cartes non Intel décrivent la machine.
func TestGPUPorts(t *testing.T) {
	conns := []videoConnector{{Type: "DisplayPort"}, {Type: "HDMI"}, {Type: "HDMI"}, {Type: internalScreen}}
	if got := gpuPorts(conns, map[string]string{"card0": "amdgpu"}); got["HDMI"] != 2 || got["DisplayPort"] != 1 || got[internalScreen] != 0 {
		t.Errorf("AMD : %v", got)
	}
	if got := gpuPorts(conns, map[string]string{"card0": "i915", "card1": "nouveau"}); len(got) != 0 {
		t.Errorf("Intel : %v", got)
	}
}

// Un SMART refusé s'écrit « non supporté » et ne pèse pas sur le verdict.
func TestSmartUnsupported(t *testing.T) {
	s := newSession()
	s.Results["ssd"] = unsupported("sda : SMART non supporté")
	if r := findRow(t, s, "Stockage", "SMART SSD"); r.pillLabel() != pillUnsupported {
		t.Errorf("ligne : %+v", r)
	}
	done, ok, warn, fail := s.counts()
	if done != 1 || ok+warn+fail != 0 {
		t.Errorf("compte : %d %d %d %d", done, ok, warn, fail)
	}
}

// Une fiche chargée (4 barrettes, longues observations) reste sur une page : le
// texte se réduit. Au-delà du plus petit texte lisible, elle continue sur une
// deuxième page sans rien perdre.
func TestFichePDFFitsOnePage(t *testing.T) {
	load := func(rows, notes int) []byte {
		t.Helper()
		s := newSession()
		s.Inventory = demoInventory()
		s.Form.Notes = strings.Repeat("Observation longue du technicien sur l'état de la machine. ", notes)
		for i := range s.Inventory {
			for j := 0; j < rows; j++ {
				s.Inventory[i].Fields = append(s.Inventory[i].Fields, Field{"Ligne", "Valeur de test assez longue pour passer sur deux lignes dans la colonne"})
			}
		}
		data, err := fichePDF(s)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	if n := bytes.Count(load(2, 6), []byte("/Type /Page\n")); n != 1 {
		t.Errorf("fiche chargée : %d page(s), attendu 1", n)
	}
	if n := bytes.Count(load(12, 60), []byte("/Type /Page\n")); n < 2 {
		t.Errorf("fiche démesurée : %d page(s), attendu au moins 2", n)
	}
}

// Chaque ligne de la fiche suit la partie du test qui la renseigne.
func TestItemState(t *testing.T) {
	s := newSession()
	s.Results["son"] = result{Status: statusFail, Data: map[string]string{
		"speakers": statusData(statusOK), "jack": statusData(statusNA), "mic_int": statusData(statusFail)}}
	cases := []struct {
		key  string
		want status
	}{{"speakers", statusOK}, {"jack", statusNA}, {"mic_int", statusFail}, {"card", statusFail}}
	for _, c := range cases {
		if got := s.itemState(ctrlItem{TestID: "son", Key: c.key}); got != c.want {
			t.Errorf("son/%s : %s, attendu %s", c.key, got.style().label, c.want.style().label)
		}
	}
}

// findRow ligne d'une section de la fiche
func findRow(t *testing.T, s *session, section, label string) ficheRow {
	t.Helper()
	for _, sec := range s.ficheSections() {
		for _, r := range sec.Rows {
			if sec.Name == section && r.Label == label {
				return r
			}
		}
	}
	t.Fatalf("ligne %q absente de la section %q", label, section)
	return ficheRow{}
}

// Le résultat d'un test est écrit avec le composant qu'il concerne, en face de
// chaque prise vidéo.
func TestFicheSectionsMergeResults(t *testing.T) {
	s := newSession()
	s.Results["stress"] = result{Status: statusOK, Detail: "42 °C au repos, 63 °C au maximum"}
	s.Results["ports"] = result{Status: statusOK, Data: map[string]string{"usb": "2 / 4", "HDMI": "1 / 2"}}
	s.Form.Paste = "oui"

	if r := findRow(t, s, "Processeur", "Stress test"); r.Status != statusOK || r.Value == "" {
		t.Errorf("stress : %+v", r)
	}
	if r := findRow(t, s, "Processeur", "Pâte thermique changée"); r.Value != "oui" {
		t.Errorf("pâte : %+v", r)
	}
	if r := findRow(t, s, "Affichage", "HDMI 1"); r.Status != statusOK {
		t.Errorf("HDMI 1 : %+v", r)
	}
	if r := findRow(t, s, "Affichage", "HDMI 2"); r.pillLabel() != pillUntested {
		t.Errorf("HDMI 2 non branché, verdict OK : %+v", r)
	}
	if r := findRow(t, s, "Ports et périphériques", "Ports USB"); r.Value != "2 / 4 testé(s)" {
		t.Errorf("USB : %+v", r)
	}
	// verdict négatif : la prise non branchée est celle qui ne marche pas
	s.Results["ports"] = result{Status: statusFail, Data: map[string]string{"HDMI": "1 / 2"}}
	if r := findRow(t, s, "Affichage", "HDMI 2"); r.Status != statusFail {
		t.Errorf("HDMI 2, verdict Nul : %+v", r)
	}
	// nombre de prises inconnu : seules les branchées sont nommées
	s.Results["ports"] = result{Status: statusOK, Data: map[string]string{"DisplayPort": "1 / ?"}}
	if r := findRow(t, s, "Affichage", "DisplayPort"); r.Status != statusOK {
		t.Errorf("DisplayPort : %+v", r)
	}
	findRow(t, s, "Affichage", "Autres sorties vidéo")
}

// Chaque icône de la fiche se trace jusqu'au bout, arcs compris : le tracé rendu
// reste dans le carré de l'icône.
func TestIconPathsInBounds(t *testing.T) {
	for _, name := range []string{"microchip", "memory", "hard-drive", "display", "battery-half", "tv", "plug", "volume-high", "wifi", "rocket", "triangle-exclamation"} {
		b, err := iconFiles.ReadFile("icons/" + name + ".svg")
		if err != nil {
			t.Fatal(err)
		}
		box := svgViewBox.FindSubmatch(b)
		path := svgPath.FindSubmatch(b)
		if box == nil || path == nil {
			t.Fatalf("%s : tracé illisible", name)
		}
		w, _ := strconv.ParseFloat(string(box[1]), 64)
		h, _ := strconv.ParseFloat(string(box[2]), 64)
		pen := &boundsPen{}
		drawSVGPath(pen, string(path[1]), 0, 0, 1)
		if pen.points < 4 || pen.minX < -1 || pen.minY < -1 || pen.maxX > w+1 || pen.maxY > h+1 {
			t.Errorf("%s : %+v pour %v x %v", name, pen, w, h)
		}
	}
}

type boundsPen struct {
	points                 int
	minX, minY, maxX, maxY float64
}

func (b *boundsPen) add(x, y float64) {
	b.points++
	b.minX, b.minY = min(b.minX, x), min(b.minY, y)
	b.maxX, b.maxY = max(b.maxX, x), max(b.maxY, y)
}
func (b *boundsPen) MoveTo(x, y float64)                         { b.add(x, y) }
func (b *boundsPen) LineTo(x, y float64)                         { b.add(x, y) }
func (b *boundsPen) CurveBezierCubicTo(_, _, _, _, x, y float64) { b.add(x, y) }
func (b *boundsPen) ClosePath()                                  {}

// La fiche produit nomme le produit, sans adresse réseau ni nom système.
func TestFicheSheet(t *testing.T) {
	s := newSession()
	s.Machine = machineInfo{Vendor: "HP", Product: "HP EliteBook 850 G3", Serial: "SN1"}
	s.Inventory = []Component{
		{Name: "Processeur", Summary: "Intel(R) Core(TM) i5-6200U CPU @ 2.30GHz"},
		{Name: "Stockage", Fields: []Field{{"sda", "Samsung 870 · SSD · 232.9 Go · 2 partition(s)"}}},
		{Name: "Cartes réseau", Fields: []Field{{"enp0s31f6", "Intel I219-V · ethernet · connecté · ac:e2:d3:96:29:dd · 1000 Mbps · 192.168.1.5"}}},
	}
	s.Results["sd"] = result{Status: statusFail, Detail: "Carte non détectée"}
	sh := newFicheSheet(s)
	if sh.Name != "HP EliteBook 850 G3" {
		t.Errorf("nom : %q", sh.Name)
	}
	if sh.Specs[0].Value != "Intel Core i5-6200U" || sh.Specs[1].Value != "232.9 Go SSD" {
		t.Errorf("essentiel : %+v", sh.Specs)
	}
	if strings.Contains(fmt.Sprintf("%+v", sh), "À signaler") {
		t.Error("le résultat ne doit apparaître qu'avec son composant")
	}
	all := fmt.Sprintf("%+v", sh)
	for _, absent := range []string{s.Ref, "ac:e2", "192.168", "enp0s31f6", "partition", "Tests gobox"} {
		if strings.Contains(all, absent) {
			t.Errorf("la fiche contient %q", absent)
		}
	}
}

// Le verdict donne la note, et les colonnes s'équilibrent.
func TestFicheGradeAndColumns(t *testing.T) {
	s := newSession()
	for _, d := range s.Tests {
		s.Results[d.ID] = result{Status: statusOK}
	}
	s.Results["sd"] = result{Status: statusWarn}
	if sh := newFicheSheet(s); sh.Grade != "B" || sh.Kind != verdictOK {
		t.Errorf("note : %q, verdict %v", sh.Grade, sh.Kind)
	}
	if k := splitColumns([]float64{10, 10, 10, 30}); k != 3 {
		t.Errorf("coupure : %d, attendu 3", k)
	}
}

// Le navigateur par défaut vient des préférences, sans bureau dans
// l'environnement (pkexec, sudo) ; une entrée sans fichier .desktop est ignorée.
func TestFindDefaultBrowser(t *testing.T) {
	home, data := t.TempDir(), t.TempDir()
	apps := filepath.Join(data, "applications")
	if err := os.MkdirAll(apps, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entry := func(cmd string) string { return "[Desktop Entry]\nExec=" + cmd + " %u\n" }
	write(filepath.Join(apps, "brave-browser.desktop"), entry("brave"))
	write(filepath.Join(apps, "firefox.desktop"), entry("firefox"))
	write(filepath.Join(apps, "mimeapps.list"), "[Default Applications]\nx-scheme-handler/http=absent.desktop\n")
	write(filepath.Join(apps, "gnome-mimeapps.list"), "[Default Applications]\nx-scheme-handler/http=firefox.desktop\n")

	got := findDefaultBrowser(home, []string{data}, nil)
	if len(got) != 1 || got[0] != "firefox" {
		t.Fatalf("sans bureau : %v, attendu firefox", got)
	}
	write(filepath.Join(home, ".config", "mimeapps.list"), "[Default Applications]\nx-scheme-handler/https=brave-browser.desktop;firefox.desktop\n")
	got = findDefaultBrowser(home, []string{data}, []string{"GNOME"})
	if len(got) != 1 || got[0] != "brave" {
		t.Fatalf("choix de l'utilisateur : %v, attendu brave", got)
	}
}

// Le navigateur par défaut est lancé depuis son .desktop, sans ses codes.
func TestDesktopExec(t *testing.T) {
	d := "[Desktop Entry]\nName=Brave\nExec=/usr/bin/flatpak run com.brave.Browser @@u %U @@\n[Desktop Action new]\nExec=/usr/bin/autre\n"
	got := strings.Join(desktopExec(d), " ")
	if got != "/usr/bin/flatpak run com.brave.Browser" {
		t.Fatalf("commande : %q", got)
	}
	if desktopExec("[Desktop Entry]\nExec=\"/opt/mon navigateur\" %u\n") != nil {
		t.Fatal("guillemets : repli sur xdg-open attendu")
	}
	if desktopExec("[Desktop Action new]\nExec=/usr/bin/autre\n") != nil {
		t.Fatal("seule l'entrée principale compte")
	}
}

// Le nombre de prises vient du BIOS : deux machines, deux résultats.
func TestParseDMIConnectors(t *testing.T) {
	block := func(extName, extType, port string) string {
		return "Handle 0x0008, DMI type 8, 9 bytes\nPort Connector Information\n\tInternal Reference Designator: J1\n\tInternal Connector Type: None\n\tExternal Reference Designator: " + extName + "\n\tExternal Connector Type: " + extType + "\n\tPort Type: " + port + "\n"
	}
	out := strings.Join([]string{
		block("HDMI", "Other", "Video Port"), block("DisplayPort 1", "Other", "Video Port"), block("Mini DP", "Other", "Other"),
		block("VGA", "DB-15 female", "Video Port"), block("DVI-D", "Other", "Video Port"),
		block("USB1", "Access Bus (USB)", "USB"), block("USB2", "Access Bus (USB)", "USB"),
		block("Serial", "DB-9 male", "Serial Port 16550A Compatible"), block("Interne HDMI", "None", "Video Port"),
	}, "\n")
	video, usb, usbC := parseDMIConnectors(out)
	want := map[string]int{"HDMI": 1, "DisplayPort": 2, "VGA": 1, "DVI": 1}
	for typ, n := range want {
		if video[typ] != n {
			t.Errorf("%s : %d, attendu %d", typ, video[typ], n)
		}
	}
	if usb != 2 || usbC != 0 {
		t.Errorf("USB : %d, USB-C %d", usb, usbC)
	}
	for _, name := range []string{"USB Type-C", "USB-C 1", "Thunderbolt 3"} {
		if _, u, c := parseDMIConnectors(block(name, "Other", "USB")); u != 1 || c != 1 {
			t.Errorf("%s : USB %d, USB-C %d", name, u, c)
		}
	}
	if v, u, _ := parseDMIConnectors(""); len(v) != 0 || u != 0 {
		t.Errorf("sortie vide : %v %d", v, u)
	}
}

// Les ports que le contrôleur déclare sans prise ne comptent pas : seul
// « hotplug » est une prise, une moitié USB 3 non renseignée rejoint sa prise.
func TestUSBPortsCount(t *testing.T) {
	p := usbPorts{group: map[string]string{"usb1-port1": "usb1-port1", "usb2-port1": "usb1-port1", "usb1-port3": "usb1-port3"},
		possible: map[string]string{"usb2-port6": "usb1-port3", "usb1-port8": "usb1-port8"}}
	p.finish(0)
	if p.total != 2 {
		t.Fatalf("total %d, attendu 2", p.total)
	}
	if p.portOf("1-8") != "" {
		t.Fatal("un port non renseigné n'est pas une prise tant qu'on n'y branche rien")
	}
	if p.portOf("2-6") != "usb1-port3" {
		t.Fatal("la moitié USB 3 doit rejoindre sa prise")
	}
	if p.adopt("1-8") != "usb1-port8" || p.total != 3 {
		t.Fatalf("branchement sur un port non renseigné : total %d", p.total)
	}
	// aucun port renseigné : BIOS, sinon nombre inconnu
	none := usbPorts{group: map[string]string{}, possible: map[string]string{"usb1-port1": "usb1-port1"}}
	none.finish(4)
	if none.total != 4 {
		t.Fatalf("repli BIOS : %d", none.total)
	}
	none.finish(0)
	if none.total != -1 {
		t.Fatalf("inconnu : %d", none.total)
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
	// Fyne remet les touches au widget qui a le focus (window.go, processKeyPressed)
	trap.KeyDown(&fyne.KeyEvent{Physical: fyne.HardwareKey{ScanCode: 38}})
	trap.KeyDown(&fyne.KeyEvent{Physical: fyne.HardwareKey{ScanCode: 38}})
	trap.KeyDown(&fyne.KeyEvent{Physical: fyne.HardwareKey{ScanCode: 23}}) // Tab
	w.Canvas().FocusNext()                                                 // ce que ferait Tab sans le piège
	time.Sleep(100 * time.Millisecond)
	if trapOf() == nil {
		t.Fatalf("le focus est parti sur %T", w.Canvas().Focused())
	}
	time.Sleep(100 * time.Millisecond) // la question de fin arrive après la prise de focus
	answerAsk(tools.view, 0)
	deadline = time.Now().Add(2 * time.Second)
	for tools.active != nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := s.Results["clavier"].Detail; !strings.HasPrefix(got, "2 / ") {
		t.Fatalf("touches comptées : %q, attendu 2 touches", got)
	}
	if w.Canvas().Focused() != nil {
		t.Fatalf("le piège doit rendre le focus à la fin du test : %T", w.Canvas().Focused())
	}
}
