package gui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// Mise à jour du système et des paquets du PC : le système en cours d'exécution, ou
// celui du disque depuis une image live (update_target.go)

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

// isLiveSession vrai sur une image live : la session disparaît au redémarrage
func isLiveSession() bool {
	if info, err := os.Stat("/run/live"); err == nil && info.IsDir() {
		return true
	}
	data, _ := os.ReadFile("/proc/mounts")
	return rootIsVolatile(string(data))
}

// rootIsVolatile vrai si la racine est un système de fichiers qui ne survit pas
func rootIsVolatile(mounts string) bool {
	for _, line := range strings.Split(mounts, "\n") {
		f := strings.Fields(line)
		if len(f) >= 3 && f[1] == "/" {
			switch f[2] {
			case "overlay", "aufs", "tmpfs", "squashfs":
				return true
			}
		}
	}
	return false
}

// chooseSystem le système à mettre à jour s'il y en a plusieurs ; faux si annulé
func chooseSystem(t *testCtx, systems []diskSystem) (diskSystem, bool) {
	if len(systems) == 1 {
		return systems[0], true
	}
	picked := make(chan diskSystem, 1)
	t.waiting()
	t.panel(func() fyne.CanvasObject {
		list := container.NewVBox(instruction("arrows-rotate", "Plusieurs systèmes sur le disque : lequel mettre à jour ?"))
		for _, sys := range systems {
			list.Add(secondaryButton(sys.Name+" ("+sys.Device+")", nil, func() { picked <- sys }))
		}
		return list
	})
	select {
	case s := <-picked:
		t.progress(-1)
		return s, true
	case <-t.ctx.Done():
		return diskSystem{}, false
	}
}

func runUpdate(t *testCtx) result {
	t.progress(-1)
	if simulated {
		t.markSimulated()
		t.log("Données simulées : aucune mise à jour réelle")
		return result{Status: statusOK, Detail: "Système et paquets à jour"}
	}
	if !isAdmin() {
		t.log("La mise à jour nécessite les droits administrateur")
		return unsupported("droits administrateur nécessaires")
	}
	if isLiveSession() {
		return updateDiskSystem(t)
	}
	manager := firstAvailable("apt-get", "dnf")
	if len(updateSteps(manager)) == 0 {
		t.log("Aucun gestionnaire de paquets pris en charge (apt-get, dnf)")
		return unsupported("gestionnaire de paquets non pris en charge")
	}
	name, _ := osIdentity("/")
	return applyUpdate(t, orUnknown(name, "ce système"), manager, nil)
}

// updateDiskSystem image live : met à jour le système installé sur le disque du PC
func updateDiskSystem(t *testCtx) result {
	t.log("Session live : recherche du système installé sur le disque…")
	systems, encrypted := findDiskSystems(t.ctx, t.log)
	if t.cancelled() {
		return result{}
	}
	if len(systems) == 0 {
		detail := "aucun système Linux sur le disque"
		if encrypted > 0 {
			detail = "disque chiffré, système illisible"
		}
		t.log("Rien à mettre à jour : %s", detail)
		return unsupported(detail)
	}
	sys, ok := chooseSystem(t, systems)
	if !ok {
		return result{}
	}
	return applyUpdate(t, sys.Name+" sur le disque ("+sys.Device+")", sys.Manager, &sys)
}

// applyUpdate demande confirmation puis lance la mise à jour ; disk nil : le système
// en cours d'exécution
func applyUpdate(t *testCtx, name, manager string, disk *diskSystem) result {
	// Rien ne démarre sans le technicien : la barre reste arrêtée et la consigne
	// le dit, sinon il attend devant un écran qui semble travailler
	t.waiting()
	t.log("En attente : la mise à jour ne démarre qu'après « Lancer la mise à jour »")
	t.panel(func() fyne.CanvasObject {
		return container.NewVBox(
			instruction("arrow-pointer", "En attente de votre accord : rien n'est lancé. Cliquez sur « Lancer la mise à jour » ci-dessous pour démarrer."),
			mutedNote("Met à jour "+name+" et ses paquets ("+manager+"). La modification est définitive. Ne pas éteindre le PC pendant l'opération."),
		)
	})
	if t.choose("Démarrer la mise à jour ?", choice{"Lancer la mise à jour", statusOK}, choice{"Ne pas faire", statusNA}) != statusOK {
		if t.cancelled() {
			return result{}
		}
		return result{Status: statusNA, Detail: "Non réalisée"}
	}
	t.progress(-1)
	t.setPanelEmpty()
	t.log("Mise à jour lancée")
	t.answered = false // la suite est longue : le résultat reste affiché
	t.lockCancel()     // interrompre un paquet en cours d'installation casserait le système

	var prefix []string
	if disk != nil {
		t.log("Montage de %s…", disk.Device)
		m, err := mountForUpdate(context.Background(), *disk)
		if err != nil {
			t.log("Erreur : %v", err)
			return result{Status: statusFail, Detail: "Disque non monté : " + err.Error()}
		}
		defer func() {
			if err := m.unmount(); err != nil {
				t.log("Erreur : démontage incomplet (%v)", err)
			} else {
				t.log("Disque démonté")
			}
		}()
		chroot := toolPath("chroot")
		if chroot == "" {
			return result{Status: statusFail, Detail: "chroot absent"}
		}
		prefix = []string{chroot, m.root}
	}
	for _, step := range updateSteps(manager) {
		t.log("%s…", step.Label)
		if err := runLogged(t, append(append([]string{}, prefix...), step.Args...)); err != nil {
			t.log("Erreur : %v", err)
			return result{Status: statusFail, Detail: step.Label + " : échec"}
		}
	}
	t.log("Mise à jour terminée")
	return result{Status: statusOK, Detail: name + " : système et paquets à jour"}
}

// runLogged lance une commande et écrit sa sortie dans le journal du test
func runLogged(t *testCtx, args []string) error {
	// sans contexte : une fois lancée, la mise à jour va à son terme
	cmd := exec.Command(args[0], args[1:]...)
	setpgidOnly(cmd)
	cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive", "LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin")
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
