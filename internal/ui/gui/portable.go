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
	b.WriteString("\n" + usbReport())
	return b.String(), ok
}

// usbReport détail des prises USB, du Type-C et de la liste du BIOS
func usbReport() string {
	var b strings.Builder
	phys := readPhysicalPorts()
	ports := readUSBPorts(max(phys.USB, 0))
	ports.typeC = max(ports.typeC, phys.USBC)
	fmt.Fprintf(&b, "USB : %v, total %d\n", ports.kindTotals(), ports.total)
	typeC, _ := filepath.Glob("/sys/class/typec/*")
	thunderbolt, _ := filepath.Glob("/sys/bus/thunderbolt/devices/domain*")
	fmt.Fprintf(&b, "Type-C : %d entrée(s) dans /sys/class/typec, BIOS %d, thunderbolt %d\n", len(typeC), phys.USBC, len(thunderbolt))
	var lines []string
	for name, key := range ports.group {
		lines = append(lines, fmt.Sprintf("  %s : prise %s, %s", name, key, ports.kindOf(key)))
	}
	for name := range ports.possible {
		lines = append(lines, fmt.Sprintf("  %s : non renseigné par le BIOS", name))
	}
	sort.Strings(lines)
	b.WriteString(strings.Join(lines, "\n") + "\n\n")
	if tool := toolPath("lspci"); tool != "" {
		if out, err := exec.Command(tool).Output(); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if l := strings.ToLower(line); strings.Contains(l, "usb") || strings.Contains(l, "thunderbolt") {
					b.WriteString(line + "\n")
				}
			}
		}
	}
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
