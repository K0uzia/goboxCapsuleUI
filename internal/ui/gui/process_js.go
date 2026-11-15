//go:build js

package gui

import "os/exec"

// Navigateur : aucun processus ne peut être lancé
func setProcessGroup(*exec.Cmd) {}

func setpgidOnly(*exec.Cmd) {}

func stopProcess(*exec.Cmd) {}
