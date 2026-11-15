//go:build !(demo && js)

package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// printFiche génère la fiche PDF et l'ouvre dans le navigateur de la session
func printFiche(s *session, w fyne.Window) {
	path, err := writeFiche(s)
	if err != nil {
		dialog.ShowError(fmt.Errorf("création de la fiche : %w", err), w)
		return
	}
	// l'ouverture attend quelques secondes la réponse du navigateur
	go func() {
		err := openInBrowser(path)
		fyne.Do(func() { showPrinted(w, path, err) })
	}()
}
