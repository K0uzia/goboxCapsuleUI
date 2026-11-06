package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeVideo nœud /sys/class/video4linux relié à une interface d'appareil
func fakeVideo(t *testing.T, root, node, iface, name, index string) {
	t.Helper()
	dev := filepath.Join(root, "devices", iface)
	cls := filepath.Join(root, "class", node)
	for _, d := range []string{dev, cls} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(dev, filepath.Join(cls, "device")); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(cls, "name"), []byte(name+"\n"), 0o644)
	_ = os.WriteFile(filepath.Join(cls, "index"), []byte(index+"\n"), 0o644)
}

// Une webcam USB crée deux nœuds (image et métadonnées), et son capteur
// infrarouge un troisième : une seule caméra
func TestCamerasGroupNodes(t *testing.T) {
	root := t.TempDir()
	fakeVideo(t, root, "video0", "usb1/1-5/1-5:1.0", "Integrated Camera: Integrated C", "0")
	fakeVideo(t, root, "video1", "usb1/1-5/1-5:1.0", "Integrated Camera: Integrated C", "1")
	fakeVideo(t, root, "video2", "usb1/1-5/1-5:1.2", "Integrated Camera: Integrated I", "0")
	fakeVideo(t, root, "video3", "usb1/1-5/1-5:1.2", "Integrated Camera: Integrated I", "1")
	fakeVideo(t, root, "video4", "usb1/1-6/1-6:1.0", "Integrated IR Camera", "0")
	got := cameras(filepath.Join(root, "class"))
	if len(got) != 1 || got[0].Dev != "/dev/video0" {
		t.Fatalf("caméras = %+v, attendu une seule (video0)", got)
	}
}

func TestCamerasTwoDevices(t *testing.T) {
	root := t.TempDir()
	fakeVideo(t, root, "video0", "usb1/1-5/1-5:1.0", "Integrated Camera", "0")
	fakeVideo(t, root, "video2", "usb1/1-2/1-2:1.0", "Logitech C270", "0")
	if got := cameras(filepath.Join(root, "class")); len(got) != 2 {
		t.Fatalf("caméras = %+v, attendu deux", got)
	}
}

// Une clé branchée sur un port que le microprogramme dit inutilisé est bien
// sur une prise : elle compte, au lieu de « port non identifié »
func TestUSBAdoptUnlistedPort(t *testing.T) {
	root := t.TempDir()
	port := filepath.Join(root, "usb1", "1-0:1.0", "usb1-port7")
	if err := os.MkdirAll(port, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(root, "usb1", "speed"), []byte("480\n"), 0o644)
	_ = os.WriteFile(filepath.Join(port, "connect_type"), []byte("not used\n"), 0o644)
	p := usbPorts{group: map[string]string{"usb1-port1": "usb1-port1"}, possible: map[string]string{}, kinds: map[string]string{}}
	p.finish()
	if got := p.adoptPort(root, "1-7"); got != "usb1-port7" {
		t.Fatalf("port adopté %q, attendu usb1-port7", got)
	}
	if p.total != 2 || p.portOf("1-7") != "usb1-port7" || p.kindOf("usb1-port7") != usbKind2 {
		t.Fatalf("prise non comptée : total %d, %+v", p.total, p)
	}
	if p.adoptPort(root, "1-9") != "" {
		t.Fatal("port inexistant adopté")
	}
}

// usbTree construit sous root les ports d'un relevé réel (testdata/*.tsv) ;
// change remplace le connect_type de certains ports
func usbTree(t *testing.T, file string, change map[string]string) string {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	dir := func(name string) string {
		hub, _, _ := strings.Cut(name, "-port")
		return filepath.Join(root, hub, strings.TrimPrefix(hub, "usb")+"-0:1.0", name)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		name, speed, ctype, peer := f[0], f[1], f[2], f[3]
		if c, ok := change[name]; ok {
			ctype = c
		}
		d := dir(name)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		hub, _, _ := strings.Cut(name, "-port")
		_ = os.WriteFile(filepath.Join(root, hub, "speed"), []byte(speed+"\n"), 0o644)
		_ = os.WriteFile(filepath.Join(d, "connect_type"), []byte(ctype+"\n"), 0o644)
		if peer != "" {
			_ = os.Symlink(dir(peer), filepath.Join(d, "peer"))
		}
	}
	return root
}

// Relevé réel d'un Dell OptiPlex 3050 (PC de bureau) : 6 prises USB 3 déclarées
// par l'ACPI. Un port USB 2.0 seul déclaré branchable (connecteur interne ou
// façade non câblée sur une tour) ne compte sur un PC de bureau qu'une fois
// utilisé ; sur un portable il compte tout de suite.
func TestUSBDesktopUSB2(t *testing.T) {
	const file = "testdata/usb-optiplex-3050.tsv"
	for _, desktop := range []bool{true, false} {
		p := readUSBPortsAt(usbTree(t, file, nil), desktop)
		if got := p.kindTotals(); got[usbKind3] != 6 || got[usbKind2] != 0 || got[usbKindC] != 0 || p.total != 6 {
			t.Errorf("bureau %v : %v, total %d", desktop, got, p.total)
		}
	}

	root := usbTree(t, file, map[string]string{"usb1-port6": "hotplug", "usb1-port8": "hotplug"})
	if p := readUSBPortsAt(root, false); p.kindTotals()[usbKind2] != 2 || p.total != 8 {
		t.Errorf("portable : %v, total %d", p.kindTotals(), p.total)
	}
	p := readUSBPortsAt(root, true)
	if got := p.kindTotals(); got[usbKind2] != 0 || got[usbKind3] != 6 || p.total != 6 {
		t.Fatalf("bureau : %v, total %d", got, p.total)
	}
	if p.adoptUnsure("1-10") != "" {
		t.Error("port non renseigné adopté sans branchement")
	}
	if got := p.adoptUnsure("1-6"); got != "usb1-port6" || p.total != 7 || p.kindTotals()[usbKind2] != 1 {
		t.Errorf("prise occupée à l'ouverture : %q, total %d, %v", got, p.total, p.kindTotals())
	}
	if got := p.adopt("1-8"); got != "usb1-port8" || p.total != 8 {
		t.Errorf("prise branchée pendant le test : %q, total %d", got, p.total)
	}
}
