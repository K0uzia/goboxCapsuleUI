//go:build windows

package gui

import "os/exec"

func setProcessGroup(*exec.Cmd) {}

func stopProcess(c *exec.Cmd) {
	_ = c.Process.Kill()
	_ = c.Wait()
}
