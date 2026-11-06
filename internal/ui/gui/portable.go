package gui

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"gobox/internal/probe"
)

// Rien ne reste sur le PC, qui part ensuite en vente : gobox n'installe rien, ses
// outils sont extraits dans un dossier temporaire le temps de la session, et ce
// qu'il a écrit est effacé à sa fermeture.

const appID = "fr.lacapsule.gobox"

// extractTools extrait l'archive des outils dans dir ; refuse un chemin qui en sort
func extractTools(archive []byte, dir string) error {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if !filepath.IsLocal(h.Name) {
			return fmt.Errorf("chemin refusé dans l'archive : %s", h.Name)
		}
		path := filepath.Join(dir, h.Name)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode)&0o755)
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tr)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return err
			}
		}
	}
}

// prepareTools rend les outils embarqués utilisables : après ceux du système dans
// le PATH. /tmp peut interdire l'exécution : /dev/shm prend alors le relais. La
// fonction rendue efface tout.
func prepareTools() (cleanup func()) {
	if len(embeddedToolsArchive) == 0 {
		return func() {}
	}
	for _, base := range []string{"", "/dev/shm"} {
		dir, err := os.MkdirTemp(base, "gobox-")
		if err != nil {
			continue
		}
		err = extractTools(embeddedToolsArchive, dir)
		if err == nil {
			err = exec.Command(filepath.Join(dir, "bin", "dmidecode"), "--version").Run()
		}
		if err != nil {
			os.RemoveAll(dir)
			continue
		}
		os.Setenv("PATH", withTools(os.Getenv("PATH"), filepath.Join(dir, "bin")))
		return func() { os.RemoveAll(dir) }
	}
	return func() {}
}

// withTools ajoute des outils au PATH, après ceux du système
func withTools(path, dir string) string {
	if path == "" {
		return dir
	}
	return path + ":" + dir
}

// ficheFiles fiches écrites pendant la session
var ficheFiles []string

// removeTraces efface ce que gobox laisse sur le disque : fiches, cache et réglages
// de Fyne (les dossiers fyne seulement s'ils restent vides)
func removeTraces() {
	for _, f := range ficheFiles {
		os.Remove(f)
	}
	for _, get := range []func() (string, error){os.UserCacheDir, os.UserConfigDir} {
		if base, err := get(); err == nil {
			os.RemoveAll(filepath.Join(base, "fyne", appID))
			os.Remove(filepath.Join(base, "fyne"))
		}
	}
}

// CheckTools diagnostic de `gobox --verifier-outils` ; rend le code de sortie
func CheckTools() int {
	cleanup := prepareTools()
	defer cleanup()
	report, ok := diagnosticReport()
	fmt.Print(report)
	if !ok {
		return 1
	}
	return 0
}

// diagnosticReport état des outils et des prises USB, à transmettre pour comprendre
// une détection incorrecte ; ok faux si un outil manque ou échoue
func diagnosticReport() (string, bool) {
	var b strings.Builder
	ok := true
	m := readMachine()
	fmt.Fprintf(&b, "Machine : %s\nDroits administrateur : %v\nSession live : %v\n\n", m.Name(), isAdmin(), isLiveSession())
	for _, t := range []struct {
		name string
		args []string
	}{{"smartctl", []string{"--version"}}, {"dmidecode", []string{"--version"}}, {"lspci", nil}, {"iw", []string{"--version"}}} {
		path := toolPath(t.name)
		if path == "" {
			fmt.Fprintf(&b, "%-10s introuvable\n", t.name)
			ok = false
			continue
		}
		out, err := exec.Command(path, t.args...).CombinedOutput()
		first, _, _ := strings.Cut(string(out), "\n")
		if err != nil {
			fmt.Fprintf(&b, "%-10s ERREUR %v (%s) : %s\n", t.name, err, path, first)
			ok = false
			continue
		}
		fmt.Fprintf(&b, "%-10s OK (%s)\n", t.name, path)
	}
	b.WriteString("\n" + powerReport(sysPower) + "\n" + diskReport("/sys/block") + "\n" + keyboardReport() + "\n" + usbReport())
	return b.String(), ok
}

// powerReport valeurs brutes de chaque source d'alimentation (batterie, secteur)
func powerReport(root string) string {
	var b strings.Builder
	b.WriteString("Alimentation (/sys/class/power_supply) :\n")
	dirs, _ := filepath.Glob(filepath.Join(root, "*"))
	if len(dirs) == 0 {
		b.WriteString("  aucune entrée\n")
	}
	for _, d := range dirs {
		fmt.Fprintf(&b, "  %s : batterie du PC %v\n", filepath.Base(d), probe.IsSystemBattery(d))
		for _, f := range []string{"type", "scope", "status", "capacity", "cycle_count", "energy_full", "energy_full_design",
			"charge_full", "charge_full_design", "voltage_min_design", "voltage_now", "online"} {
			if v, err := os.ReadFile(filepath.Join(d, f)); err == nil {
				fmt.Fprintf(&b, "    %s = %s\n", f, strings.TrimSpace(string(v)))
			}
		}
	}
	if d := batteryDir(root); d != "" {
		info, err := probe.ReadBattery(d)
		fmt.Fprintf(&b, "  lecture : %+v, erreur %v\n", info, err)
	}
	return b.String()
}

// diskReport chaque disque de /sys/block, son chemin réel et s'il est retenu
func diskReport(root string) string {
	var b strings.Builder
	b.WriteString("Disques (/sys/block) :\n")
	dirs, _ := filepath.Glob(filepath.Join(root, "*"))
	for _, d := range dirs {
		if strings.HasPrefix(filepath.Base(d), "loop") {
			continue
		}
		real, _ := filepath.EvalSymlinks(d)
		fmt.Fprintf(&b, "  %s : interne %v, removable %s, rotatif %s, %s\n", filepath.Base(d), probe.IsInternalDisk(d),
			readTrim(filepath.Join(d, "removable")), readTrim(filepath.Join(d, "queue", "rotational")), real)
	}
	return b.String()
}

// keyboardReport ce qui sert à reconnaître le clavier : châssis, capot, claviers
// vus par le noyau
func keyboardReport() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Clavier : châssis %q, capot %v, portable %v\n", readTrim("/sys/class/dmi/id/chassis_type"), hasLid(sysInput), isLaptop("/sys/class/dmi/id", sysPower))
	for _, k := range inputKeyboards(sysInput) {
		fmt.Fprintf(&b, "  %s (%s)\n", k.Name, k.Phys)
	}
	return b.String()
}

// portLocation emplacement ACPI d'un port (usb1-port3) : les deux moitiés d'une
// prise USB 3 ont le même
func portLocation(name string) string {
	found, _ := filepath.Glob(filepath.Join(sysUSB, "*", "*:1.0", name, "location"))
	if len(found) == 0 {
		return "?"
	}
	return readTrim(found[0])
}

// usbReport détail des prises USB, du Type-C et de la liste du BIOS
func usbReport() string {
	var b strings.Builder
	phys := readPhysicalPorts()
	ports := readUSBPorts()
	ports.typeC = max(ports.typeC, phys.USBC)
	fmt.Fprintf(&b, "PC de bureau : %v (châssis %q)\n", isDesktop(), readTrim("/sys/class/dmi/id/chassis_type"))
	fmt.Fprintf(&b, "USB : %v, total %d (BIOS : %d)\n", ports.kindTotals(), ports.total, phys.BIOSUSB)
	typeC, _ := filepath.Glob("/sys/class/typec/*")
	thunderbolt, _ := filepath.Glob("/sys/bus/thunderbolt/devices/domain*")
	fmt.Fprintf(&b, "Type-C : %d entrée(s) dans /sys/class/typec, BIOS %d, thunderbolt %d\n", len(typeC), phys.USBC, len(thunderbolt))
	var lines []string
	for name, key := range ports.group {
		lines = append(lines, fmt.Sprintf("  %s : prise %s, %s (location %s)", name, key, ports.kindOf(key), portLocation(name)))
	}
	for name, key := range ports.possible {
		why := "non renseigné par le BIOS"
		if ports.unsure[key] {
			why = "USB 2.0 de PC de bureau, compté au branchement"
		}
		lines = append(lines, fmt.Sprintf("  %s : %s (%s, location %s)", name, why, key, portLocation(name)))
	}
	sort.Strings(lines)
	b.WriteString(strings.Join(lines, "\n") + "\n\n")
	b.WriteString("Périphériques USB :\n" + usbInventory(sysUSB) + "\n")
	fmt.Fprintf(&b, "Lecteur de puce : %v\nModem : %v\nWi-Fi : %v\nBluetooth : %v\n", cardReaders(), modemNames("/"), wifiInterfaces(sysNet), bluetoothAdapters(sysBluetooth))
	rf, _ := filepath.Glob(filepath.Join(sysRfkill, "rfkill*"))
	for _, d := range rf {
		fmt.Fprintf(&b, "  rfkill %s : soft %s, hard %s\n", readTrim(filepath.Join(d, "type")), readTrim(filepath.Join(d, "soft")), readTrim(filepath.Join(d, "hard")))
	}
	if tool := toolPath("lspci"); tool != "" {
		if out, err := exec.Command(tool).Output(); err == nil {
			b.WriteString("\nPCI :\n" + string(out))
		}
	}
	conns := videoConnectors()
	b.WriteString("\nConnecteurs vidéo :\n")
	for _, c := range conns {
		fmt.Fprintf(&b, "  %s (%s) connecté %v\n", c.Name, c.Plug, c.Connected)
	}
	fmt.Fprintf(&b, "\nPrises vidéo retenues : %v (%s)\nAnnoncées par le BIOS ou la table Intel : %v\nTable Intel : %v\n", phys.Video, phys.Why, phys.Declared, readVBTPorts())
	if tool := toolPath("dmidecode"); tool != "" {
		out, err := exec.Command(tool, "-t", "8").CombinedOutput()
		if err != nil {
			fmt.Fprintf(&b, "dmidecode -t 8 : %v (droits administrateur ?)\n", err)
		} else {
			b.WriteString("\n" + string(out))
		}
	}
	return b.String()
}
