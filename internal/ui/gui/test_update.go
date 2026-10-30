package gui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"fyne.io/fyne/v2"
)

// Mise à jour du système et des paquets du système en cours d'exécution

// updateStep une commande de la mise à jour
type updateStep struct {
	Label string
	Args  []string
}

// updateSteps update puis upgrade selon le gestionnaire de paquets ; vide s'il n'est
// pas pris en charge
func updateSteps(manager string) []updateStep {
	switch manager {
	case "apt-get":
		return []updateStep{
			{"Mise à jour de la liste des paquets", []string{"apt-get", "update"}},
			{"Installation des mises à jour", []string{"apt-get", "-y", "-o", "Dpkg::Options::=--force-confold", "upgrade"}},
		}
	case "dnf":
		return []updateStep{
			{"Mise à jour de la liste des paquets", []string{"dnf", "-y", "makecache"}},
			{"Installation des mises à jour", []string{"dnf", "-y", "upgrade"}},
		}
	}
	return nil
}

// isLiveSession vrai sur une image live : la mise à jour disparaît au redémarrage
func isLiveSession() bool {
	info, err := os.Stat("/run/live")
	return err == nil && info.IsDir()
}

func runUpdate(t *testCtx) result {
	t.progress(-1)
	if simulated {
		t.markSimulated()
		t.log("Données simulées : aucune mise à jour réelle")
		return result{Status: statusOK, Detail: "Système et paquets à jour"}
	}
	manager := firstAvailable("apt-get", "dnf")
	steps := updateSteps(manager)
	if len(steps) == 0 {
		t.log("Aucun gestionnaire de paquets pris en charge (apt-get, dnf)")
		return unsupported("gestionnaire de paquets non pris en charge")
	}
	if !isAdmin() {
		t.log("La mise à jour nécessite les droits administrateur")
		return unsupported("droits administrateur nécessaires")
	}
	message := "Met à jour le système et les paquets de la session en cours (" + manager + ")."
	if isLiveSession() {
		message = "Session live : la mise à jour s'applique à la session et disparaît au redémarrage, le disque du PC n'est pas modifié."
	}
	t.panel(func() fyne.CanvasObject { return instruction("arrows-rotate", message) })
	if t.choose("Lancer la mise à jour ?", choice{"Lancer", statusOK}, choice{"Ne pas faire", statusNA}) != statusOK {
		if t.cancelled() {
			return result{}
		}
		return result{Status: statusNA, Detail: "Non réalisée"}
	}
	t.answered = false // la suite est longue : le résultat reste affiché
	for _, step := range steps {
		t.log("%s…", step.Label)
		if err := runLogged(t, step.Args); err != nil {
			if t.cancelled() {
				return result{}
			}
			t.log("Erreur : %v", err)
			return result{Status: statusFail, Detail: step.Label + " : échec"}
		}
	}
	t.log("Mise à jour terminée")
	return result{Status: statusOK, Detail: "Système et paquets à jour (" + manager + ")"}
}

// runLogged lance une commande et écrit sa sortie dans le journal du test
func runLogged(t *testCtx, args []string) error {
	cmd := exec.CommandContext(t.ctx, args[0], args[1:]...)
	setProcessGroup(cmd)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive", "LC_ALL=C")
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			t.log("%s", line)
		}
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s : %w", args[0], err)
	}
	return nil
}
