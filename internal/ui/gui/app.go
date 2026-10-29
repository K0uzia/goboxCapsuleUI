package gui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"gobox/internal/probe"
)

// Étapes du reconditionnement et catégorie de tests mise en avant
var steps = []struct {
	label, category string
}{
	{"Diagnostic initial", ""},
	{"Tests périphériques", catPeripherals},
	{"Tests stockage", catStorage},
	{"Système", catSystem},
	{"Imprimer la fiche", ""},
}

// Run ouvre la fenêtre principale et bloque jusqu'à sa fermeture.
// Réglages valables pour la session seulement (image live partagée) : thème
// sombre à chaque lancement.
func Run() {
	a := app.NewWithID("fr.lacapsule.gobox")
	a.Settings().SetTheme(newTheme(true))
	w := a.NewWindow("gobox · Toolbox La Capsule")
	w.Resize(fyne.NewSize(1280, 800))

	s := newSession()
	content, _ := mainContent(s, w)
	w.SetContent(content)

	// Inventaire en tâche de fond (dmidecode, sysfs… peuvent prendre 1 à 2 s)
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

// mainContent contenu de la fenêtre principale : en-tête, fiche, outils,
// barre d'étapes
func mainContent(s *session, w fyne.Window) (fyne.CanvasObject, *toolsPanel) {
	printNow := func() {
		if path, err := printFiche(s); err != nil {
			dialog.ShowError(fmt.Errorf("impression de la fiche : %w", err), w)
		} else {
			dialog.ShowInformation("Fiche générée", "Fiche ouverte dans le navigateur :\n"+path, w)
		}
	}
	// Fiche incomplète : on demande avant d'imprimer
	onPrint := func() {
		done, _, _, _ := s.counts()
		left := len(s.Tests) - done
		if left == 0 {
			printNow()
			return
		}
		msg := wrapLabel(fmt.Sprintf("Il reste %d test(s) à faire. La fiche imprimée les indiquera « À faire ».", left))
		dialog.NewCustomConfirm("Fiche incomplète", "Imprimer quand même", "Annuler",
			container.New(&widthLayout{w: 380}, msg), func(ok bool) {
				if ok {
					printNow()
				}
			}, w).Show()
	}
	fiche := newFichePanel(s)
	tools := newToolsPanel(s, w)
	w.Canvas().SetOnTypedKey(tools.typedKey)

	body := container.New(&ratioLayout{left: 0.46, gap: 24}, fiche.root, tools.root)
	bottom := container.NewHBox(layout.NewSpacer(),
		stepBar(s, tools.setStep, onPrint),
		iconButton(buttonIcon("circle-info"), func() { showInfo(s, w) }),
		iconButton(buttonIcon("gear"), func() { showSettings(w, fiche) }),
	)
	return container.NewBorder(
		headerBar(s, onPrint),
		container.New(&padLayout{h: 16, v: 8}, bottom),
		nil, nil,
		container.New(&padLayout{h: 16, v: 12}, body),
	), tools
}

// headerBar bandeau du haut : marque, durée de la session, impression
func headerBar(s *session, onPrint func()) fyne.CanvasObject {
	brand := container.NewHBox(
		text("gobox", 20, true, cAccent),
		container.NewVBox(layout.NewSpacer(), text("Toolbox de reconditionnement · La Capsule", 13, false, cMuted), layout.NewSpacer()),
	)
	chrono := text(s.elapsed(), 13, true, cText)
	cpu := text("non lue", 13, true, cText)
	go func() {
		for range time.Tick(20 * time.Second) {
			fyne.Do(func() { chrono.SetText(s.elapsed()) })
		}
	}()
	go func() {
		for {
			if t, err := probe.CPUTemperature(); err == nil {
				fyne.Do(func() { cpu.SetText(fmt.Sprintf("%.0f °C", t)) })
			}
			time.Sleep(3 * time.Second)
		}
	}()
	right := container.NewHBox(
		chip("clock", "Durée", chrono),
		chip("temperature-half", "CPU", cpu),
		container.New(&padLayout{h: 4}),
		primaryButton("PRINT", coloredIcon("print", cAccentText), onPrint),
	)
	bar := container.NewBorder(nil, nil, brand, right)
	return container.NewVBox(
		container.NewStack(newRect(cSurface, "", 0), container.New(&padLayout{h: 16, v: 8}, bar)),
		hRule(cLine),
	)
}

// chip indicateur de l'en-tête : libellé discret, valeur
func chip(ico, label string, value *tText) fyne.CanvasObject {
	return container.NewCenter(container.NewStack(newRect(cSurface2, cLine, 8),
		container.New(&padLayout{h: 10, v: 4}, container.NewHBox(iconOf(ico, cMuted, 15), container.NewCenter(text(label, 12, false, cMuted)), container.NewCenter(value)))))
}

// currentStep étape déduite de la fiche : technicien à saisir, puis la
// première catégorie qui a encore des tests à faire, puis l'impression
func currentStep(s *session) (step int, hint string) {
	if s.Form.Tech == "" {
		return 0, "Saisir le nom du technicien"
	}
	for i, st := range steps {
		if st.category == "" {
			continue
		}
		left := 0
		for _, t := range s.Tests {
			if t.Category == st.category && s.status(t.ID) == statusIdle {
				left++
			}
		}
		if left > 0 {
			return i, fmt.Sprintf("%d test(s) restant(s)", left)
		}
	}
	return len(steps) - 1, "Tous les tests sont faits"
}

// stepBar barre d'étapes, mise à jour seule à chaque changement de la fiche :
// segments faits, étape en cours mise en avant dans les tests, impression à
// la fin
func stepBar(s *session, onStep func(category string), onPrint func()) fyne.CanvasObject {
	label := text("", 14, true, cText)
	counter := text("", 12, false, cMuted)
	hint := text("", 12, false, cMuted)
	hint.Alignment = fyne.TextAlignTrailing
	segs := make([]*tRect, len(steps))
	segObjs := make([]fyne.CanvasObject, len(steps))
	for i := range segs {
		segs[i] = newRect(cSurface2, "", 2)
		segs[i].MinSz = fyne.NewSize(0, 4)
		segObjs[i] = segs[i]
	}
	printBtn := primaryButton("Imprimer la fiche", coloredIcon("print", cAccentText), onPrint)

	update := func() {
		step, h := currentStep(s)
		label.SetText(steps[step].label)
		counter.SetText(fmt.Sprintf("Étape %d / %d", step+1, len(steps)))
		hint.SetText(h)
		for i, r := range segs {
			switch {
			case i < step:
				r.setRoles(cSuccess, "")
			case i == step:
				r.setRoles(cAccent, "")
			default:
				r.setRoles(cSurface2, "")
			}
		}
		onStep(steps[step].category)
		if step == len(steps)-1 {
			printBtn.Show()
		} else {
			printBtn.Hide()
		}
	}
	s.onChange(update)
	update()

	head := container.NewBorder(nil, nil, container.NewHBox(counter, label), container.NewCenter(hint))
	content := container.NewVBox(head, container.New(&respGrid{minCol: 10, cols: []int{len(steps)}, gap: 4}, segObjs...))
	bar := container.NewStack(newRect(cSurface, cLineStrong, 8), container.New(&padLayout{h: 14, v: 8}, content))
	return container.NewHBox(container.New(&widthLayout{w: 480}, bar), container.NewCenter(printBtn))
}

// showInfo à propos, et raccourcis clavier
func showInfo(s *session, w fyne.Window) {
	keys := container.NewVBox(
		formRow("Échap", wrapLabel("Annuler le test en cours")),
		formRow("Entrée", wrapLabel("Confirmer le résultat affiché")),
		formRow("Tab", wrapLabel("Passer d'un bouton à l'autre, Entrée ou Espace pour lancer")),
	)
	content := container.NewVBox(
		text("gobox", 20, true, cAccent),
		wrapLabel("Toolbox de reconditionnement de La Capsule : chaque bouton lance un test, le résultat s'inscrit dans la fiche après confirmation, puis PRINT ouvre la fiche à enregistrer ou imprimer."),
		formRow("Fiche en cours", text(s.Ref, 13, true, cText)),
		sectionTitle("Raccourcis clavier"),
		keys,
		mutedNote("Icônes : Font Awesome Free (CC BY 4.0). Polices : Exo 2 et Share Tech Mono (SIL OFL 1.1)."),
	)
	d := dialog.NewCustom("À propos", "Fermer", container.New(&widthLayout{w: 460}, content), w)
	d.Show()
}

// showSettings réglages de la session (perdus au redémarrage du PC)
func showSettings(w fyne.Window, f *fichePanel) {
	choice := widget.NewRadioGroup([]string{"Sombre", "Clair"}, func(v string) {
		fyne.CurrentApp().Settings().SetTheme(newTheme(v != "Clair"))
	})
	choice.Horizontal = true
	if isDark() {
		choice.SetSelected("Sombre")
	} else {
		choice.SetSelected("Clair")
	}
	detail := widget.NewCheck("Afficher le détail des tests confirmés", f.setDetailed)
	detail.SetChecked(f.detailed)
	content := container.NewVBox(
		formRow("Thème", choice),
		formRow("Fiche", detail),
		mutedNote("Valables jusqu'au redémarrage du PC : chaque lancement repart des réglages par défaut."),
	)
	d := dialog.NewCustom("Réglages", "Fermer", container.New(&widthLayout{w: 460}, content), w)
	d.Show()
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

// fracLayout remplit une fraction de la largeur (barres d'avancement)
type fracLayout struct{ frac float32 }

func (l *fracLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(0, 0) }

func (l *fracLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Resize(fyne.NewSize(size.Width*l.frac, size.Height))
		o.Move(fyne.NewPos(0, 0))
	}
}
