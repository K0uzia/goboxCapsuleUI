package gui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

// writeFiche génère la fiche en PDF, lisible par le navigateur de l'utilisateur
// même quand gobox tourne en root
func writeFiche(s *session) (string, error) {
	data, err := fichePDF(s)
	if err != nil {
		return "", err
	}
	path := filepath.Join(os.TempDir(), "Fiche-"+s.Ref+".pdf")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// browserKeys types associés au navigateur, du plus au moins sûr
var browserKeys = []string{"x-scheme-handler/https", "x-scheme-handler/http", "text/html"}

// mimeDefaults navigateurs d'un fichier mimeapps.list, dans l'ordre de browserKeys
func mimeDefaults(content string) []string {
	values := map[string]string{}
	inDefaults := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			inDefaults = line == "[Default Applications]"
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok && inDefaults {
			values[k] = v
		}
	}
	var out []string
	for _, k := range browserKeys {
		for _, id := range strings.Split(values[k], ";") {
			if id != "" {
				out = append(out, id)
			}
		}
	}
	return out
}

// mimeFiles fichiers mimeapps.list par ordre de priorité (norme XDG). Bureau
// inconnu (pkexec et sudo le retirent) : ceux de tous les bureaux, après les
// fichiers communs, plutôt que le premier navigateur installé.
func mimeFiles(dirs, desktops []string) []string {
	var out []string
	for _, d := range dirs {
		for _, name := range desktops {
			out = append(out, filepath.Join(d, strings.ToLower(name)+"-mimeapps.list"))
		}
		out = append(out, filepath.Join(d, "mimeapps.list"))
	}
	if len(desktops) == 0 {
		for _, d := range dirs {
			found, _ := filepath.Glob(filepath.Join(d, "*-mimeapps.list"))
			out = append(out, found...)
		}
	}
	return out
}

// findDefaultBrowser commande du navigateur par défaut, lue dans les préférences
// de l'utilisateur : xdg-settings répond autre chose sans bureau dans
// l'environnement. Le fichier .desktop doit exister (les listes du système en
// nomment parfois d'absents).
func findDefaultBrowser(home string, dataDirs, desktops []string) []string {
	apps := []string{filepath.Join(home, ".local/share/applications")}
	for _, d := range dataDirs {
		apps = append(apps, filepath.Join(d, "applications"))
	}
	dirs := append([]string{filepath.Join(home, ".config"), "/etc/xdg"}, apps...)
	for _, f := range mimeFiles(dirs, desktops) {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, id := range mimeDefaults(string(b)) {
			if strings.ContainsAny(id, "/\\") {
				continue
			}
			for _, app := range apps {
				if d, err := os.ReadFile(filepath.Join(app, id)); err == nil {
					if cmd := desktopExec(string(d)); len(cmd) > 0 {
						return cmd
					}
				}
			}
		}
	}
	return nil
}

// defaultBrowserCommand commande du navigateur par défaut de la session
func defaultBrowserCommand() []string {
	home, _ := os.UserHomeDir()
	if name, _ := sessionUser(); name != "" {
		if found, err := user.Lookup(name); err == nil {
			home = found.HomeDir
		}
	}
	dataDirs := []string{"/usr/local/share", "/usr/share", "/var/lib/flatpak/exports/share"}
	if v := os.Getenv("XDG_DATA_DIRS"); v != "" {
		dataDirs = strings.Split(v, ":")
	}
	var desktops []string
	if v := os.Getenv("XDG_CURRENT_DESKTOP"); v != "" {
		desktops = strings.Split(v, ":")
	}
	return findDefaultBrowser(home, dataDirs, desktops)
}

// desktopExec commande d'un fichier .desktop, sans ses codes (%u, @@) ; vide si
// elle contient des guillemets, qu'on ne sait pas découper sans risque
func desktopExec(desktop string) []string {
	inEntry := false
	for _, line := range strings.Split(desktop, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			inEntry = line == "[Desktop Entry]"
			continue
		}
		v, ok := strings.CutPrefix(line, "Exec=")
		if !inEntry || !ok {
			continue
		}
		if strings.Contains(v, `"`) {
			return nil
		}
		var cmd []string
		for _, f := range strings.Fields(v) {
			if !strings.HasPrefix(f, "%") && !strings.HasPrefix(f, "@@") {
				cmd = append(cmd, f)
			}
		}
		return cmd
	}
	return nil
}

// openInBrowser ouvre un fichier dans une nouvelle fenêtre du navigateur par
// défaut : un onglet ajouté à une fenêtre existante reste derrière gobox, et
// xdg-open donnerait au PDF le lecteur du système. Repli sur xdg-open, gio puis
// Firefox, avec l'erreur de chacun si aucun n'y arrive.
func openInBrowser(path string) error {
	var errs []error
	attempts := [][]string{{"xdg-open", path}, {"gio", "open", path}, {"firefox", "--new-window", path}}
	if cmd := defaultBrowserCommand(); len(cmd) > 0 {
		attempts = append([][]string{append(cmd, "--new-window", "file://"+path)}, attempts...)
	}
	for _, args := range attempts {
		if firstAvailable(args[0]) == "" {
			continue
		}
		var out bytes.Buffer
		cmd := userCommand(context.Background(), args[0], args[1:]...)
		cmd.Stdout, cmd.Stderr = &out, &out
		if err := cmd.Start(); err != nil {
			errs = append(errs, fmt.Errorf("%s : %w", args[0], err))
			continue
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			if err == nil {
				return nil
			}
			errs = append(errs, fmt.Errorf("%s : %v %s", args[0], err, strings.TrimSpace(out.String())))
		case <-time.After(4 * time.Second):
			return nil // le navigateur tourne : il a été lancé par cette commande
		}
	}
	if len(errs) == 0 {
		return errors.New("aucun outil pour ouvrir un navigateur (xdg-open, gio, firefox)")
	}
	return errors.Join(errs...)
}
