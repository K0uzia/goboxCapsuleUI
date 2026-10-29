package gui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// La molette au-dessus d'un champ fait défiler la fiche, et un clic sur ce
// champ lui donne toujours le focus.
func TestFicheScrollOverField(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(true))

	s := newSession()
	f := newFichePanel(s)
	w := test.NewWindow(f.root)
	defer w.Close()
	w.Resize(fyne.NewSize(620, 500))

	// amène le champ Observations dans la vue, sous le pointeur
	f.scroll.ScrollToOffset(fyne.NewPos(0, 0))
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(f.notes)
	f.scroll.ScrollToOffset(fyne.NewPos(0, pos.Y-200))
	pos = fyne.CurrentApp().Driver().AbsolutePositionForObject(f.notes).Add(fyne.NewPos(20, 10))

	before := f.scroll.Offset.Y
	test.Scroll(w.Canvas(), pos, 0, 60) // vers le haut : la fiche est en bas
	if f.scroll.Offset.Y >= before {
		t.Fatalf("la fiche n'a pas défilé sous le champ : %v puis %v", before, f.scroll.Offset.Y)
	}

}

// La couche de défilement ne prend que la molette : un clic atteint le widget
// dessous. Le pilote de test ne donne jamais le focus au clic (il ne fait que
// le retirer), le focus d'un champ se vérifie donc dans la vraie fenêtre.
func TestScrollPassLetsTapsThrough(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	tapped := false
	b := widget.NewButton("Bouton", func() { tapped = true })
	w := test.NewWindow(container.NewStack(b, newScrollPass(func(*fyne.ScrollEvent) {})))
	defer w.Close()
	w.Resize(fyne.NewSize(200, 60))

	test.TapCanvas(w.Canvas(), fyne.NewPos(100, 30))
	if !tapped {
		t.Fatal("le clic doit traverser la couche de défilement")
	}
}

// La barre d'étapes suit la fiche sans action du technicien.
func TestCurrentStep(t *testing.T) {
	s := newSession()
	if step, _ := currentStep(s); step != 0 {
		t.Fatalf("sans technicien : étape %d, attendu 0", step)
	}
	s.Form.Tech = "Camille"
	if step, _ := currentStep(s); steps[step].category != catPeripherals {
		t.Fatalf("technicien saisi : %q, attendu les tests périphériques", steps[step].label)
	}
	for _, d := range s.Tests {
		if d.Category == catPeripherals {
			s.Results[d.ID] = result{Status: statusOK}
		}
	}
	if step, _ := currentStep(s); steps[step].category != catStorage {
		t.Fatalf("périphériques faits : %q, attendu le stockage", steps[step].label)
	}
	for _, d := range s.Tests {
		s.Results[d.ID] = result{Status: statusOK}
	}
	if step, _ := currentStep(s); step != len(steps)-1 {
		t.Fatalf("tout fait : %q, attendu l'impression", steps[step].label)
	}
}
