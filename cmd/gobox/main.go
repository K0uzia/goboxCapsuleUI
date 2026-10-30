package main

import (
	"os"

	"gobox/internal/ui/gui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--verifier-outils" {
		os.Exit(gui.CheckTools())
	}
	gui.Run()
}
