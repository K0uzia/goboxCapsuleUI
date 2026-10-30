package gui

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
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

// Sur image live, seules les partitions internes non montées, en système de fichiers
// Linux, sont candidates ; une partition sans système est écartée.
func TestDiskUpdateTarget(t *testing.T) {
	lsblk := `{"blockdevices":[
	 {"name":"loop0","path":"/dev/loop0","fstype":"squashfs","type":"loop","mountpoint":"/usr/lib/live","rm":false},
	 {"name":"sda","path":"/dev/sda","fstype":null,"type":"disk","mountpoint":null,"rm":false,"children":[
	  {"name":"sda1","path":"/dev/sda1","fstype":"vfat","type":"part","mountpoint":null,"rm":false},
	  {"name":"sda2","path":"/dev/sda2","fstype":"ext4","type":"part","mountpoint":null,"rm":false},
	  {"name":"sda3","path":"/dev/sda3","fstype":"crypto_LUKS","type":"part","mountpoint":null,"rm":false},
	  {"name":"sda4","path":"/dev/sda4","fstype":"ext4","type":"part","mountpoint":"/mnt","rm":false}]},
	 {"name":"sdb","path":"/dev/sdb","fstype":null,"type":"disk","mountpoint":null,"rm":true,"children":[
	  {"name":"sdb1","path":"/dev/sdb1","fstype":"ext4","type":"part","mountpoint":null,"rm":false}]}]}`
	paths, fs, enc, err := rootCandidates([]byte(lsblk))
	if err != nil || len(paths) != 1 || paths[0] != "/dev/sda2" || fs["/dev/sda2"] != "ext4" || enc != 1 {
		t.Fatalf("candidats : %v %v %d %v", paths, fs, enc, err)
	}
	root := t.TempDir()
	if name, m := osIdentity(root); name != "" || m != "" {
		t.Errorf("partition vide : %q %q", name, m)
	}
	os.MkdirAll(filepath.Join(root, "usr/bin"), 0o755)
	os.MkdirAll(filepath.Join(root, "etc"), 0o755)
	os.WriteFile(filepath.Join(root, "usr/bin/apt-get"), nil, 0o755)
	os.WriteFile(filepath.Join(root, "etc/os-release"), []byte("NAME=x\nPRETTY_NAME=\"Linux Mint 22\"\n"), 0o644)
	if name, m := osIdentity(root); name != "Linux Mint 22" || m != "apt-get" {
		t.Errorf("système : %q %q", name, m)
	}
	got := bootMounts("# c\nUUID=a / ext4 defaults 0 1\nUUID=b /boot ext4 defaults 0 2\nUUID=c /boot/efi vfat umask=0077 0 1\nUUID=d /home ext4 defaults 0 2\n")
	if len(got) != 2 || got[1].Target != "/boot/efi" || got[1].FSType != "vfat" {
		t.Errorf("fstab : %+v", got)
	}
	if !rootIsVolatile("overlay / overlay rw 0 0\n") || rootIsVolatile("/dev/sda2 / ext4 rw 0 0\n") {
		t.Error("racine volatile")
	}
	if got := mountArgs(diskSystem{Device: "/dev/sda2", Subvol: "@"}, "ro", "/m"); strings.Join(got, " ") != "mount -o ro,subvol=@ /dev/sda2 /m" {
		t.Errorf("mount : %v", got)
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

// Wi-Fi, Bluetooth, modem et lecteur de puce : présents ou absents selon le système,
// « DÉTECTÉ » tant qu'aucune carte n'a été lue, et rien sur la fiche si absents.
func TestRadioDetection(t *testing.T) {
	root := t.TempDir()
	mk := func(path string, v string) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(v+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("net/wlp2s0/wireless/x", "")
	mk("net/enp1s0/type", "1")
	if got := wifiInterfaces(filepath.Join(root, "net")); len(got) != 1 || got[0] != "wlp2s0" {
		t.Errorf("Wi-Fi : %v", got)
	}
	mk("bt/hci0/x", "")
	mk("bt/hci0:256/x", "")
	if got := bluetoothAdapters(filepath.Join(root, "bt")); len(got) != 1 || got[0] != "hci0" {
		t.Errorf("Bluetooth : %v", got)
	}
	mk("rf/rfkill0/type", "wlan")
	mk("rf/rfkill0/soft", "0")
	mk("rf/rfkill0/hard", "1")
	soft, hard := rfkillState(filepath.Join(root, "rf"), "wlan")
	if soft || !hard || blockedDetail(soft, hard) == "" {
		t.Errorf("rfkill : %v %v", soft, hard)
	}
	if s, h := rfkillState(filepath.Join(root, "rf"), "bluetooth"); s || h || blockedDetail(s, h) != "" {
		t.Errorf("rfkill bluetooth : %v %v", s, h)
	}
	mk("usb/1-4/product", "Lecteur Test")
	mk("usb/1-4:1.0/bInterfaceClass", "0b")
	mk("usb/1-5:1.0/bInterfaceClass", "08")
	if got := usbCardReaders(filepath.Join(root, "usb")); len(got) != 1 || got[0] != "Lecteur Test" {
		t.Errorf("lecteur USB : %v", got)
	}
	got := parseSmartCardPCI("00:1f.3 Audio device: Intel\n03:00.0 Unassigned class: O2 Micro SmartCard Reader (rev 01)\n")
	if len(got) != 1 || !strings.Contains(got[0], "SmartCard") {
		t.Errorf("lecteur PCI : %v", got)
	}
	mk("sys/class/wwan/wwan0/x", "")
	if len(modemNames(root)) != 1 {
		t.Errorf("modem : %v", modemNames(root))
	}
	if countNetworks("iw", "BSS aa(on wlan0)\n\tfreq: 1\nBSS bb(on wlan0)\n") != 2 || countNetworks("nmcli", "aa\nbb\n\n") != 2 {
		t.Error("nombre de réseaux")
	}
	s := newSession()
	s.Results["wifi"] = result{Status: statusOK, Detail: "x"}
	s.Results["sim"] = result{Status: statusOK, Data: map[string]string{"pill": pillDetected}}
	s.Results["lecteur"] = result{Status: statusNA}
	findRow(t, s, "Cartes réseau", "Wi-Fi")
	if r := findRow(t, s, "Cartes réseau", "Carte SIM"); r.pillLabel() != pillDetected {
		t.Errorf("SIM : %+v", r)
	}
	for _, sec := range s.ficheSections() {
		for _, r := range sec.Rows {
			if r.Label == "Lecteur de puce" {
				t.Error("un lecteur absent ne paraît pas sur la fiche")
			}
		}
	}
}

// La puce pulse tant que le technicien et le nettoyage ne sont pas remplis.
func TestPulseDots(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(true))
	s := newSession()
	f := newFichePanel(s)
	if !f.techDot.root.Visible() || !f.cleanDot.root.Visible() {
		t.Fatal("les deux puces doivent être visibles au départ")
	}
	s.Form.Tech = "Alex"
	f.showTechHint()
	if f.techDot.root.Visible() {
		t.Error("technicien rempli : puce masquée")
	}
}

// Un clic hors de la fenêtre de réglages la ferme, et l'ouvrir ne réapplique pas le
// thème (toute l'application se redessinait à chaque ouverture).
func TestPopupClosesOnOutsideTap(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(true))
	w := a.NewWindow("test")
	s := newSession()
	f := newFichePanel(s)
	w.SetContent(f.root)
	w.Resize(fyne.NewSize(1000, 800))
	applied := 0
	themeApplied = func() { applied++ }
	defer func() { themeApplied = func() {} }()
	showSettings(w, f)
	if applied != 0 {
		t.Errorf("thème réappliqué %d fois à l'ouverture", applied)
	}
	if openPopup == nil {
		t.Fatal("réglages non affichés")
	}
	test.TapCanvas(w.Canvas(), fyne.NewPos(500, 400)) // dans la fenêtre : reste ouverte
	if openPopup == nil {
		t.Fatal("un clic dans la fenêtre ne doit pas la fermer")
	}
	test.TapCanvas(w.Canvas(), fyne.NewPos(3, 3))
	if openPopup != nil || len(w.Canvas().Overlays().List()) != 0 {
		t.Fatal("un clic en dehors doit fermer la fenêtre")
	}
}

// Le diagnostic décrit les outils et les prises, sans commande à taper.
func TestDiagnosticReport(t *testing.T) {
	report, _ := diagnosticReport()
	for _, want := range []string{"Machine :", "smartctl", "USB :", "Type-C :"} {
		if !strings.Contains(report, want) {
			t.Errorf("rapport sans %q", want)
		}
	}
}

// Une prise seulement côté USB 3 est une USB-C ; les autres ont leur moitié USB 2.
func TestPortKindUSBC(t *testing.T) {
	root := t.TempDir()
	hub := filepath.Join(root, "usb2")
	mk := func(path, v string) {
		t.Helper()
		full := filepath.Join(hub, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(v+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("speed", "5000")
	mk("usb2-port5/connect_type", "hotplug")
	mk("usb2-port1/connect_type", "hotplug")
	mk("usb2-port1/peer", "x")
	mk("usb2-port6/connect_type", "unknown")
	for port, want := range map[string]string{"usb2-port5": usbKindC, "usb2-port1": usbKind3, "usb2-port6": usbKind3} {
		if got := portKind(hub, filepath.Join(hub, port)); got != want {
			t.Errorf("%s : %s, attendu %s", port, got, want)
		}
	}
}

// Les lecteurs de portable qui ne se déclarent pas en CCID et les modems sans pilote
// se reconnaissent à leur nom.
func TestReaderAndModemNames(t *testing.T) {
	root := t.TempDir()
	mk := func(path, v string) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(v+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk("1-3/product", "EMV Smartcard Reader")
	mk("1-4/product", "USB Optical Mouse")
	mk("1-5/product", "HP lt4120 Snapdragon X5 LTE")
	if got := usbCardReaders(root); len(got) != 1 || got[0] != "EMV Smartcard Reader" {
		t.Errorf("lecteur : %v", got)
	}
	if got := usbModems(root); len(got) != 1 || !strings.Contains(got[0], "Snapdragon") {
		t.Errorf("modem : %v", got)
	}
}

// Non vu par le système : le technicien dit ce qu'il voit, la fiche le reprend.
func TestSlotScanAsksTechnician(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(true))
	w := a.NewWindow("test")
	s := newSession()
	s.Tests = []*testDef{{ID: "lecteur", Title: "Puce", Category: catPeripherals, Color: cPeripheral, Soft: cPeripheralBg,
		Run: runSlotScan("Lecteur de puce", func(context.Context, func(string, ...any)) result { return result{Status: statusNA} }, "")}}
	wasSim := simulated
	simulated = false
	defer func() { simulated = wasSim }()
	tools := newToolsPanel(s, w)
	test.Tap(tools.buttons["lecteur"])
	time.Sleep(50 * time.Millisecond)
	answerAsk(tools.view, 0)
	deadline := time.Now().Add(2 * time.Second)
	for tools.active != nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if r := findRow(t, s, "Ports et périphériques", "Lecteur de puce"); r.pillLabel() != pillPresent {
		t.Errorf("fiche : %+v", r)
	}
}

// Table vidéo réelle d'un HP EliteBook 850 G3 : l'écran interne ne compte pas, et ses
// 2 sorties déclarées DP et HDMI comptent sans type précis.
func TestParseVBT(t *testing.T) {
	raw, err := os.ReadFile("testdata/vbt-hp-elitebook.b64")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimSpace(string(raw))
	data, err := base64.RawStdEncoding.DecodeString(strings.TrimRight(text, "="))
	if err != nil {
		t.Fatal(err)
	}
	got := parseVBT(data)
	if len(got) != 1 || got[videoAny] != 2 {
		t.Fatalf("sorties : %v", got)
	}
	if parseVBT(nil) != nil || parseVBT([]byte("pas une table")) != nil || parseVBT(data[:40]) != nil {
		t.Error("table illisible : rien")
	}
	for typ, want := range map[uint16]string{0x1806: "", 0x60d6: videoAny, 0x68c6: "DisplayPort", 0x60d2: videoAny, 0x6001: "VGA", 0: ""} {
		if got := vbtKind(typ); got != want {
			t.Errorf("%04x : %q, attendu %q", typ, got, want)
		}
	}
}

// Sortie sans type précis : la fiche dit ce qui a été branché dessus.
func TestPortRowsAnyOutput(t *testing.T) {
	s := newSession()
	s.Results["ports"] = result{Status: statusOK, Data: map[string]string{videoAny: "1 / 2", "video_plugs": "VGA"}}
	if r := findRow(t, s, "Affichage", videoAny+" 1"); r.Status != statusOK || r.Value != "écran VGA détecté" {
		t.Errorf("sortie 1 : %+v", r)
	}
	if r := findRow(t, s, "Affichage", videoAny+" 2"); r.pillLabel() != pillUntested {
		t.Errorf("sortie 2 : %+v", r)
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
