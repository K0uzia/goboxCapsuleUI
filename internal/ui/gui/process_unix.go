//go:build !windows

package gui

import (
	"os/exec"
	"syscall"
	"time"
)

// setProcessGroup place la commande dans son propre groupe pour l'arrêter avec ce
// qu'elle a lancé. L'annulation envoie SIGTERM : sudo le relaie à la commande,
// alors que SIGKILL le tuerait seul et laisserait la commande ouverte.
func setProcessGroup(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error { return syscall.Kill(-c.Process.Pid, syscall.SIGTERM) }
	c.WaitDelay = 2 * time.Second
}

// setpgidOnly place la commande dans son groupe, sans annulation
func setpgidOnly(c *exec.Cmd) { c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// stopProcess arrête la commande et ce qu'elle a lancé, puis attend sa fin ; SIGKILL
// seulement si elle n'a pas répondu
func stopProcess(c *exec.Cmd) {
	_ = syscall.Kill(-c.Process.Pid, syscall.SIGTERM)
	done := make(chan struct{})
	go func() { _ = c.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
		<-done
	}
}
