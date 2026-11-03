package gui

import (
	"os"
	"path/filepath"
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
