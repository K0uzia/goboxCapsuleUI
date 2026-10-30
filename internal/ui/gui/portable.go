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

// CheckTools diagnostic de `gobox --verifier-outils` : chaque outil est-il trouvé
// et se lance-t-il ? Rend le code de sortie.
func CheckTools() int {
	cleanup := prepareTools()
	defer cleanup()
	code := 0
	for _, t := range []struct {
		name string
		args []string
	}{{"smartctl", []string{"--version"}}, {"dmidecode", []string{"--version"}}, {"lspci", nil}, {"iw", []string{"--version"}}} {
		path := toolPath(t.name)
		if path == "" {
			fmt.Printf("%-10s introuvable\n", t.name)
			code = 1
			continue
		}
		out, err := exec.Command(path, t.args...).CombinedOutput()
		first, _, _ := strings.Cut(string(out), "\n")
		if err != nil {
			fmt.Printf("%-10s ERREUR %v (%s) : %s\n", t.name, err, path, first)
			code = 1
			continue
		}
		fmt.Printf("%-10s OK (%s) : %s\n", t.name, path, first)
	}
	reportUSB()
	return code
}

// reportUSB détail des prises USB, pour comprendre une détection de type incorrecte
func reportUSB() {
	loadTypeCDriver()
	phys := readPhysicalPorts()
	ports := readUSBPorts(max(phys.USB, 0))
	ports.typeC = max(ports.typeC, phys.USBC)
	fmt.Printf("USB : %v, total %d\n", ports.kindTotals(), ports.total)
	typeC, _ := filepath.Glob("/sys/class/typec/*")
	thunderbolt, _ := filepath.Glob("/sys/bus/thunderbolt/devices/domain*")
	fmt.Printf("Type-C : %d entrée(s) dans /sys/class/typec, BIOS %d, thunderbolt %d\n", len(typeC), phys.USBC, len(thunderbolt))
	for name, key := range ports.group {
		fmt.Printf("  %s : prise %s, %s\n", name, key, ports.kindOf(key))
	}
	for name := range ports.possible {
		fmt.Printf("  %s : non renseigné par le BIOS\n", name)
	}
	if tool := toolPath("dmidecode"); tool != "" {
		out, err := exec.Command(tool, "-t", "8").CombinedOutput()
		if err != nil {
			fmt.Printf("dmidecode : %v (droits administrateur ?)\n", err)
			return
		}
		fmt.Printf("%s\n", out)
	}
}
