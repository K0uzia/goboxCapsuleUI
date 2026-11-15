//go:build !(demo && js)

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// showDiagnostic rapport à copier pour transmettre l'état de la machine, sans passer
// par une commande
func showDiagnostic(w fyne.Window) {
	report := widget.NewLabel("Lecture du matériel…")
	report.Wrapping = fyne.TextWrapWord
	report.TextStyle.Monospace = true
	view := container.NewVScroll(report)
	view.SetMinSize(fyne.NewSize(440, 380))
	text := ""
	copyBtn := primaryButton("Copier le diagnostic", coloredIcon("copy", cAccentText), func() {
		fyne.CurrentApp().Clipboard().SetContent(text)
	})
	showPopup(w, "Diagnostic matériel", container.NewVBox(view, container.NewHBox(copyBtn)))
	go func() {
		out, _ := diagnosticReport()
		fyne.Do(func() {
			text = out
			report.SetText(out)
		})
	}()
}
