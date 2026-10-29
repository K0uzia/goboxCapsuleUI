package gui

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"runtime"
)

// Les sondes ont besoin des droits administrateur : détail des barrettes de
// RAM (dmidecode), SMART des disques, n° de série, consommation du CPU.
// Lancé sans ces droits, gobox se relance lui-même par pkexec (fenêtre de mot
// de passe du système). Refusé ou impossible : il continue en mode limité.

// isAdmin vrai si gobox a les droits administrateur
func isAdmin() bool { return runtime.GOOS != "linux" || os.Geteuid() == 0 }

// relaunchAsAdmin relance gobox en administrateur et attend sa fin ; faux si
// la relance n'a pas eu lieu (déjà root, refus, pkexec absent)
func relaunchAsAdmin() (exitCode int, relaunched bool) {
	if isAdmin() || os.Getenv("GOBOX_NO_ELEVATE") != "" || os.Getenv("GOBOX_ELEVATED") != "" || firstAvailable("pkexec") == "" {
		return 0, false
	}
	exe, err := os.Executable()
	if err != nil {
		return 0, false
	}
	// pkexec vide l'environnement : on redonne l'accès à l'écran de la session
	args := []string{"env", "GOBOX_ELEVATED=1"}
	for _, k := range []string{"DISPLAY", "XAUTHORITY", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR", "XDG_SESSION_TYPE", "LANG"} {
		if v := os.Getenv(k); v != "" {
			args = append(args, k+"="+v)
		}
	}
	args = append(append(args, exe), os.Args[1:]...)
	cmd := exec.Command("pkexec", args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err = cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0, true
	case errors.As(err, &exit) && (exit.ExitCode() == 126 || exit.ExitCode() == 127):
		return 0, false // mot de passe refusé ou annulé : mode limité
	case errors.As(err, &exit):
		return exit.ExitCode(), true
	}
	return 0, false
}

// sessionUser utilisateur de la session graphique quand gobox tourne en root
// (lancé par sudo ou par pkexec) : nom et uid ; vides sinon
func sessionUser() (name, uid string) {
	if os.Geteuid() != 0 {
		return "", ""
	}
	if name, uid = os.Getenv("SUDO_USER"), os.Getenv("SUDO_UID"); name != "" && uid != "" {
		return name, uid
	}
	if uid = os.Getenv("PKEXEC_UID"); uid != "" {
		if u, err := user.LookupId(uid); err == nil {
			return u.Username, uid
		}
	}
	return "", ""
}
