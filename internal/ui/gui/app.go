package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var steps = []string{"Diagnostic initial", "Tests périphériques", "Tests stockage", "Système", "Imprimer la fiche"}

// Run ouvre la fenêtre principale et bloque jusqu'à sa fermeture
func Run() {
	a := app.NewWithID("fr.lacapsule.gobox")
	a.Settings().SetTheme(atelierTheme{})
	w := a.NewWindow("gobox — Toolbox La Capsule")
	w.Resize(fyne.NewSize(1280, 800))

	s := newSession()
	fiche := newFichePanel(s, func() {
		if path, err := printFiche(s); err != nil {
			dialog.ShowError(fmt.Errorf("impression de la fiche : %w", err), w)
		} else {
			dialog.ShowInformation("Fiche générée", "Fiche ouverte dans le navigateur :\n"+path, w)
		}
	})
	tools := newToolsPanel(s, w)

	body := container.New(&ratioLayout{left: 0.46, gap: 24}, fiche.root, tools.root)
	w.SetContent(container.NewBorder(nil, stepBar(), nil, nil, container.NewPadded(body)))

	// Inventaire en tâche de fond (dmidecode, sysfs… peuvent prendre 1–2 s)
	a.Lifecycle().SetOnStarted(func() {
		go func() {
			m, inv := readMachine(), collectInventory()
			fyne.Do(func() {
				s.Machine, s.Inventory = m, inv
				s.changed()
			})
		}()
	})
	w.ShowAndRun()
}

// stepBar barre d'étapes du bas : ‹ étape (n / 5) ›
func stepBar() fyne.CanvasObject {
	step := 0
	label := widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	counter := text("", 11, false, colMuted)
	fill := canvas.NewRectangle(colLine)
	frac := &fracLayout{}
	track := container.New(frac, fill)

	update := func() {
		label.SetText(steps[step])
		counter.Text = fmt.Sprintf("%d / %d", step+1, len(steps))
		counter.Refresh()
		frac.frac = float32(step+1) / float32(len(steps))
		track.Refresh()
	}
	prev := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if step > 0 {
			step--
			update()
		}
	})
	next := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		if step < len(steps)-1 {
			step++
			update()
		}
	})
	update()

	border := canvas.NewRectangle(nil)
	border.StrokeColor, border.StrokeWidth, border.CornerRadius = colInk, 1.5, 4
	center := container.NewStack(track, label, container.NewHBox(layout.NewSpacer(), counter))
	bar := container.NewStack(border, container.NewBorder(nil, nil, prev, next, center))
	return container.NewPadded(container.NewHBox(layout.NewSpacer(), container.NewGridWrap(fyne.NewSize(460, 44), bar)))
}

// ratioLayout deux colonnes : gauche = left × largeur
type ratioLayout struct {
	left, gap float32
}

func (l *ratioLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(900, 500)
}

func (l *ratioLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	lw := (size.Width - l.gap) * l.left
	objs[0].Resize(fyne.NewSize(lw, size.Height))
	objs[0].Move(fyne.NewPos(0, 0))
	objs[1].Resize(fyne.NewSize(size.Width-lw-l.gap, size.Height))
	objs[1].Move(fyne.NewPos(lw+l.gap, 0))
}

// fracLayout remplit une fraction de la largeur (barre d'étapes)
type fracLayout struct{ frac float32 }

func (l *fracLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(0, 0) }

func (l *fracLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Resize(fyne.NewSize(size.Width*l.frac, size.Height))
		o.Move(fyne.NewPos(0, 0))
	}
}
