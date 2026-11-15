//go:build !(demo && js)

package gui

import "fyne.io/fyne/v2"

// keyScanCode code X11 de la touche appuyée, donné par le pilote
func keyScanCode(ev *fyne.KeyEvent) int { return ev.Physical.ScanCode }
