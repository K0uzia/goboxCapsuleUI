package gui

import (
	"context"
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
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
	if code, done := relaunchAsAdmin(); done {
		os.Exit(code)
	}
	cleanup := prepareTools()
	defer cleanup()
	defer removeTraces()
	a := app.NewWithID(appID)
	a.Settings().SetTheme(newTheme(true))
	w := a.NewWindow("gobox · Toolbox La Capsule")
	w.Resize(fyne.NewSize(1280, 800))

	s := newSession()
	content, tools := mainContent(s, w)
	w.SetContent(content)

	// Inventaire en tâche de fond (dmidecode, sysfs… peuvent prendre 1 à 2 s)
	a.Lifecycle().SetOnStarted(func() {
		go func() {
			m, inv, ports, absent := readMachine(), collectInventory(), readPhysicalPorts(), absentHardware()
			fyne.Do(func() {
				s.Machine, s.Inventory, s.Ports = m, inv, ports
				s.markAbsent(absent)
				s.changed()
			})
			// SMART : rien à demander au technicien, lu dès l'ouverture
			if simulated {
				return
			}
			for _, auto := range autoScans {
				r := auto.scan(context.Background(), func(string, ...any) {})
				fyne.Do(func() {
					if s.status(auto.id) == statusIdle || s.Results[auto.id].Data["auto"] == "1" {
						s.setResult(auto.id, r)
					}
				})
			}
			for _, kind := range []string{"hdd", "ssd", "nvme"} {
				r := smartScan(context.Background(), kind, func(string, ...any) {})
				fyne.Do(func() {
					if s.status(kind) == statusIdle {
						s.setResult(kind, r)
					}
				})
			}
		}()
	})
	w.ShowAndRun()
	if tools.cancel != nil { // gobox fermé pendant un test : rien ne reste ouvert (aperçu caméra…)
		tools.cancel()
		time.Sleep(500 * time.Millisecond)
	}
	restoreSuperKey() // si gobox est fermé pendant le test clavier
}

// mainContent contenu de la fenêtre principale : en-tête, fiche, outils,
// barre d'étapes
func mainContent(s *session, w fyne.Window) (fyne.CanvasObject, *toolsPanel) {
	printNow := func() {
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
	return container.NewBorder(
		headerBar(s, w, fiche, tools, onPrint),
		nil, nil, nil,
		container.New(&padLayout{h: 16, v: 12}, body),
	), tools
}

// showPrinted fiche générée : ouverte dans le navigateur, ou chemin à ouvrir
// à la main si aucun navigateur n'a répondu
func showPrinted(w fyne.Window, path string, err error) {
	copyPath := secondaryButton("Copier le chemin", buttonIcon("copy"), func() {
		fyne.CurrentApp().Clipboard().SetContent(path)
	})
	pathLabel := wrapLabel(path)
	pathLabel.TextStyle.Monospace = true
	title, msg := "Fiche ouverte", "La fiche est ouverte dans le navigateur : imprimez-la ou enregistrez-la en PDF avec Ctrl+P."
	var extra fyne.CanvasObject = container.NewVBox()
	if err != nil {
		title, msg = "Navigateur non ouvert", "La fiche est créée mais aucun navigateur n'a pu l'ouvrir. Ouvrez ce fichier dans un navigateur :"
		detail := mutedNote(err.Error())
		extra = detail
	}
	content := container.NewVBox(wrapLabel(msg), pathLabel, container.NewHBox(copyPath), extra)
	dialog.NewCustom(title, "Fermer", container.New(&widthLayout{w: 460}, content), w).Show()
}

// headerBar bandeau du haut : marque, étapes au centre, puis durée, température,
// impression, à propos et réglages
func headerBar(s *session, w fyne.Window, fiche *fichePanel, tools *toolsPanel, onPrint func()) fyne.CanvasObject {
	brand := container.NewCenter(text("gobox", 20, true, cAccent))
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
	right := container.NewHBox()
	if !isAdmin() {
		// sans droits : RAM, SMART et n° de série incomplets
		limited := text("sans droits admin", 13, true, cWarning)
		right.Add(chip("triangle-exclamation", "Mode limité", limited))
	}
	right.Objects = append(right.Objects,
		chip("clock", "Durée", chrono),
		chip("temperature-half", "CPU", cpu),
		container.New(&padLayout{h: 4}),
		primaryButton("PRINT", coloredIcon("print", cAccentText), onPrint),
		iconButton(buttonIcon("circle-info"), func() { showInfo(s, w) }),
		iconButton(buttonIcon("gear"), func() { showSettings(w, fiche) }),
	)
	bar := container.New(&headerLayout{gap: 12}, brand, stepBar(s, tools.setStep), right)
	return container.NewVBox(
		container.NewStack(newRect(cSurface, "", 0), container.New(&padLayout{h: 16, v: 8}, bar)),
		hRule(cLine),
	)
}

// headerLayout trois blocs : le deuxième centré sur la fenêtre tant qu'il y a la
// place, et réduit plutôt que de chevaucher ses voisins
type headerLayout struct{ gap float32 }

func (l *headerLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var h float32
	for _, o := range objs {
		h = max(h, o.MinSize().Height)
	}
	return fyne.NewSize(objs[0].MinSize().Width+objs[2].MinSize().Width+2*l.gap+160, h)
}

func (l *headerLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	left, mid, right := objs[0], objs[1], objs[2]
	lw, rw := left.MinSize().Width, right.MinSize().Width
	left.Resize(fyne.NewSize(lw, size.Height))
	left.Move(fyne.NewPos(0, 0))
	right.Resize(fyne.NewSize(rw, size.Height))
	right.Move(fyne.NewPos(size.Width-rw, 0))

	from, to := lw+l.gap, size.Width-rw-l.gap
	w := min(mid.MinSize().Width, to-from)
	x := max(from, min((size.Width-w)/2, to-w))
	h := mid.MinSize().Height
	mid.Resize(fyne.NewSize(w, h))
	mid.Move(fyne.NewPos(x, (size.Height-h)/2))
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
// segments faits, étape en cours mise en avant dans les tests
func stepBar(s *session, onStep func(category string)) fyne.CanvasObject {
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
	}
	s.onChange(update)
	update()

	head := container.NewBorder(nil, nil, container.NewHBox(counter, label), container.NewCenter(hint))
	content := container.NewVBox(head, container.New(&respGrid{minCol: 10, cols: []int{len(steps)}, gap: 4}, segObjs...))
	bar := container.NewStack(newRect(cSurface, cLineStrong, 8), container.New(&padLayout{h: 14, v: 8}, content))
	return container.New(&widthLayout{w: 460}, bar)
}

// showInfo à propos, et raccourcis clavier
func showInfo(s *session, w fyne.Window) {
	keys := container.NewVBox(
		formRow("Échap", wrapLabel("Annuler le test en cours ou revenir au menu")),
		formRow("Entrée", wrapLabel("Revenir au menu une fois le résultat affiché")),
		formRow("Tab", wrapLabel("Passer d'un bouton à l'autre, Entrée ou Espace pour lancer")),
	)
	content := container.NewVBox(
		text("gobox", 20, true, cAccent),
		wrapLabel("Toolbox de reconditionnement de La Capsule : chaque bouton lance un test, le résultat s'inscrit dans la fiche, puis PRINT ouvre la fiche à enregistrer ou imprimer."),
		formRow("Fiche en cours", text(s.Ref, 13, true, cText)),
		sectionTitle("Raccourcis clavier"),
		keys,
		container.NewHBox(secondaryButton("Diagnostic matériel", buttonIcon("microchip"), func() { showDiagnostic(w) })),
		mutedNote("Icônes : Font Awesome Free (CC BY 4.0). Polices : Exo 2 et Share Tech Mono (SIL OFL 1.1)."),
	)
	showPopup(w, "À propos", content)
}

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

// showSettings réglages de la session (perdus au redémarrage du PC)
func showSettings(w fyne.Window, f *fichePanel) {
	choice := widget.NewRadioGroup([]string{"Sombre", "Clair"}, nil)
	choice.Horizontal = true
	if isDark() {
		choice.SetSelected("Sombre")
	} else {
		choice.SetSelected("Clair")
	}
	// après la sélection initiale : sinon chaque ouverture réapplique le thème et
	// redessine toute l'application
	choice.OnChanged = func(v string) {
		themeApplied()
		fyne.CurrentApp().Settings().SetTheme(newTheme(v != "Clair"))
	}
	detail := widget.NewCheck("Afficher le détail des tests confirmés", f.setDetailed)
	detail.SetChecked(f.detailed)
	content := container.NewVBox(
		formRow("Thème", choice),
		formRow("Fiche", detail),
		mutedNote("Valables jusqu'au redémarrage du PC : chaque lancement repart des réglages par défaut."),
	)
	showPopup(w, "Réglages", content)
}

// themeApplied observé par les tests
var themeApplied = func() {}

// openPopup dernière fenêtre ouverte : un clic en dehors la masque (PopUp de Fyne non
// modal, ce que ses dialogues ne permettent pas), Échap et son bouton aussi
var openPopup *widget.PopUp

func closePopup() {
	if openPopup != nil {
		openPopup.Hide()
	}
}

// showPopup fenêtre centrée, qui se ferme par un clic en dehors
func showPopup(w fyne.Window, title string, content fyne.CanvasObject) {
	closePopup()
	head := container.NewBorder(nil, nil, nil, iconButton(theme.CancelIcon(), closePopup), container.NewCenter(sectionTitle(title)))
	body := container.New(&padLayout{h: 12, v: 8}, container.New(&widthLayout{w: 460}, container.NewVBox(head, content)))
	openPopup = widget.NewPopUp(body, w.Canvas())
	size := openPopup.MinSize()
	cs := w.Canvas().Size()
	openPopup.ShowAtPosition(fyne.NewPos((cs.Width-size.Width)/2, max((cs.Height-size.Height)/2, 0)))
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
