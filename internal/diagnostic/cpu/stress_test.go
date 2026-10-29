package cpu

import (
	"os/exec"
)

// LaunchStressTest lance stress-ng avec sudo en arrière-plan.
func LaunchStressTest() error {
	cmd := exec.Command("sudo", "stress-ng",
		"--matrix", "0",
		"--ignite-cpu",
		"--log-brief",
		"--metrics-brief",
		"--times",
		"--tz",
		"--verify",
		"--timeout", "190",
		"-q",
	)

	// Redirige la sortie vers /dev/null
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Exécute en tâche de fond (non bloquant)
	if err := cmd.Start(); err != nil {
		return err
	}

	return nil
}
