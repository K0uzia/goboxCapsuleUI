package gui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"gobox/internal/probe"
)

// ctrlRow ligne de contrôle de la fiche, reliée au résultat d'un test
type ctrlRow struct {
	Label  string
	TestID string
}

var (
	ctrlVideo = []ctrlRow{
		{"HDMI", "ports"}, {"DisplayPort / USB-C", "ports"}, {"VGA · DVI", "ports"},
		{"Webcam", "webcam"}, {"Port carte SD", "sd"}, {"Ports USB", "ports"},
	}
	ctrlAudio = []ctrlRow{
		{"Clavier", "clavier"}, {"Pad / souris", "souris"}, {"Carte son", "son"},
		{"Haut-parleurs", "son"}, {"Sortie audio (jack)", "son"}, {"Micro interne", "webcam"},
		{"Réseau Ethernet / Wi-Fi", "reseau"},
	}
	ctrlSystem = []ctrlRow{
		{"Stress test CPU", "stress"}, {"SMART disque HDD", "hdd"}, {"SMART disque SSD", "ssd"},
		{"SMART disque NVMe", "nvme"}, {"Clonage système", "clone"},
		{"Déploiement post-install", "deploy"}, {"Mises à jour", "maj"},
	}
)

// fichePanel colonne de gauche : la fiche, mise à jour à chaque test confirmé
type fichePanel struct {
	s    *session
	root fyne.CanvasObject

	counts      *widget.Label
	progress    *thinBar
	machineName *canvas.Text
	machineSN   *canvas.Text
	verdictBg   *canvas.Rectangle
	verdictCap  *canvas.Text
	verdictText *canvas.Text
	verdictSub  *canvas.Text
	pills       []struct {
		p  *pill
		id string
	}
	stressBefore, stressAfter *widget.Label
	detailBox                 *fyne.Container
	inventoryBox              *fyne.Container
	cpuTemp                   *canvas.Text
	detailed                  bool
}

func newFichePanel(s *session, onPrint func()) *fichePanel {
	f := &fichePanel{s: s, detailed: true}

	// En-tête : bouton PRINT
	printRow := container.NewBorder(nil, nil, solidButton("PRINT", colInk, onPrint), nil,
		wrapLabel("Ouvre la fiche dans le navigateur (enregistrer en PDF ou imprimer)"))

	// Titre, technicien, avancement
	tech := widget.NewEntry()
	tech.SetPlaceHolder("Nom du technicien")
	tech.OnChanged = func(v string) { s.Form.Tech = v }
	f.counts = widget.NewLabel("")
	f.progress = newThinBar()
	header := container.NewVBox(
		caption(s.Ref+" · page 1-2"),
		text("Fiche de reconditionnement", 20, true, colInk),
		widget.NewLabel("Fait le "+s.Started.Format("02/01/2006 15:04")),
		container.NewBorder(nil, nil, widget.NewLabel("Fait par"), nil, tech),
		container.NewBorder(nil, nil, f.counts, nil, f.progress.root),
	)

	// Synthèse : machine, aspect, nettoyage, verdict
	f.machineName = text("Détection…", 13, true, colInk)
	f.machineSN = text("", 11, false, colGrayInk)
	f.machineSN.TextStyle.Monospace = true
	machine := box(container.NewVBox(caption("Machine"), f.machineName, f.machineSN), colLine, color.White)

	aspect := widget.NewRadioGroup([]string{"Neuf", "Occasion", "Nul"}, func(v string) { s.Form.Aspect = v })
	aspect.SetSelected("Occasion")
	aspectBox := box(container.NewVBox(caption("Aspect"), aspect), colLine, color.White)

	cleanIn := widget.NewCheck("Intérieur", func(b bool) { s.Form.CleanIn = b })
	cleanOut := widget.NewCheck("Extérieur", func(b bool) { s.Form.CleanOut = b })
	cleanBox := box(container.NewVBox(caption("Nettoyage"), cleanIn, cleanOut), colLine, color.White)

	f.verdictCap = caption("Verdict")
	f.verdictText = text("", 13, true, colInk)
	f.verdictSub = text("", 11, false, colGrayInk)
	f.verdictBg = canvas.NewRectangle(colPanel)
	f.verdictBg.StrokeWidth = 1.5
	f.verdictBg.CornerRadius = 5
	verdictBox := container.NewStack(f.verdictBg, container.NewPadded(container.NewVBox(f.verdictCap, f.verdictText, f.verdictSub)))

	summary := container.NewGridWithColumns(2, machine, verdictBox, aspectBox, cleanBox)

	// Contrôles
	f.stressBefore, f.stressAfter = widget.NewLabel("—"), widget.NewLabel("—")
	controls := container.NewVBox(
		container.NewGridWithColumns(2,
			f.ctrlSection("Sorties vidéo & ports", ctrlVideo),
			f.ctrlSection("Saisie & audio", ctrlAudio),
		),
		f.ctrlSection("Système & stockage", ctrlSystem),
		f.maintenanceSection(),
	)

	notes := widget.NewMultiLineEntry()
	notes.SetPlaceHolder("Observations du technicien (reportées sur la fiche)")
	notes.SetMinRowsVisible(3)
	notes.OnChanged = func(v string) { s.Form.Notes = v }

	f.detailBox = container.NewVBox()
	f.inventoryBox = container.NewVBox(widget.NewLabel("Détection du matériel…"))

	// Pied : température CPU + case DÉTAILLÉ
	f.cpuTemp = text("CPU : —", 12, true, colInk)
	detailedCheck := widget.NewCheck("DÉTAILLÉ", func(b bool) { f.detailed = b; f.refresh() })
	detailedCheck.SetChecked(true)
	footer := container.NewHBox(f.cpuTemp, layout.NewSpacer(), detailedCheck)

	paper := container.NewVBox(
		header, summary, controls,
		sectionTitle("Observations du technicien"), notes,
		f.detailBox,
		sectionTitle("Inventaire matériel"), f.inventoryBox,
	)
	f.root = container.NewBorder(printRow, footer, nil, nil,
		container.NewVScroll(box(paper, colLine, color.White)))

	s.onChange(f.refresh)
	f.refresh()
	go f.watchTemperature()
	return f
}

func (f *fichePanel) ctrlSection(title string, rows []ctrlRow) fyne.CanvasObject {
	list := container.NewVBox(sectionTitle(title))
	for _, r := range rows {
		p := newPill(statusIdle)
		f.pills = append(f.pills, struct {
			p  *pill
			id string
		}{p, r.TestID})
		list.Add(container.NewBorder(nil, nil, nil, p.root, widget.NewLabel(r.Label)))
	}
	return list
}

func (f *fichePanel) maintenanceSection() fyne.CanvasObject {
	s := &f.s.Form
	sel := func(opts []string, v *string) *widget.Select {
		// construit à la main : NewSelect cherche une traduction absente en français
		w := &widget.Select{Options: opts, PlaceHolder: "—", OnChanged: func(x string) { *v = x }}
		w.ExtendBaseWidget(w)
		w.SetSelected(*v)
		return w
	}
	yesNo := []string{"oui", "non"}
	usb := widget.NewEntry()
	usb.SetText(s.USBBad)
	usb.OnChanged = func(v string) { s.USBBad = v }
	otherVideo := widget.NewEntry()
	otherVideo.SetPlaceHolder("—")
	otherVideo.OnChanged = func(v string) { s.OtherVideo = v }
	other := widget.NewEntry()
	other.SetPlaceHolder("—")
	other.OnChanged = func(v string) { s.Other = v }

	formOf := func(objs ...fyne.CanvasObject) fyne.CanvasObject {
		return container.New(layout.NewFormLayout(), objs...)
	}
	form := container.NewGridWithColumns(2,
		formOf(
			widget.NewLabel("Temp. avant stress"), f.stressBefore,
			widget.NewLabel("Temp. après stress"), f.stressAfter,
			widget.NewLabel("Pâte thermique"), sel(yesNo, &s.Paste),
			widget.NewLabel("Shred disques"), sel(yesNo, &s.Shred),
			widget.NewLabel("Chargeur OK"), sel(yesNo, &s.Charger),
		),
		formOf(
			widget.NewLabel("Pile setup changée"), sel(yesNo, &s.CMOS),
			widget.NewLabel("Lecteur DVD"), sel([]string{"oui", "non", "absent"}, &s.DVD),
			widget.NewLabel("USB défectueux"), usb,
			widget.NewLabel("Vidéo autre"), otherVideo,
			widget.NewLabel("Autre"), other,
		),
	)
	return container.NewVBox(sectionTitle("Stress test & maintenance"), form)
}

// refresh recalcule tout ce qui dépend des résultats
func (f *fichePanel) refresh() {
	s := f.s
	done, ok, _, _ := s.counts()
	f.counts.SetText(fmt.Sprintf("%d / %d tests · %d OK", done, len(s.Tests), ok))
	f.progress.set(float32(done) / float32(len(s.Tests)))

	f.machineName.Text = s.Machine.Name()
	f.machineSN.Text = "S/N " + orDash(s.Machine.Serial)
	f.machineName.Refresh()
	f.machineSN.Refresh()

	v := s.verdict()
	f.verdictBg.FillColor, f.verdictBg.StrokeColor = v.Fill, v.Color
	f.verdictCap.Color, f.verdictText.Color = v.Color, v.Color
	f.verdictText.Text, f.verdictSub.Text = v.Label, v.Sub
	f.verdictBg.Refresh()
	f.verdictCap.Refresh()
	f.verdictText.Refresh()
	f.verdictSub.Refresh()

	for _, p := range f.pills {
		p.p.set(s.status(p.id))
	}
	f.stressBefore.SetText(orDash(s.data("stress", "before")))
	f.stressAfter.SetText(orDash(s.data("stress", "after")))

	// Détail des tests confirmés
	f.detailBox.RemoveAll()
	if f.detailed {
		f.detailBox.Add(sectionTitle("Détail des tests réalisés"))
		n := 0
		for _, t := range s.Tests {
			r, ok := s.Results[t.ID]
			if !ok {
				continue
			}
			n++
			p := newPill(r.Status)
			f.detailBox.Add(container.NewBorder(nil, nil,
				text(t.Title, 12, true, colInk), p.root, wrapLabel(r.Detail)))
		}
		if n == 0 {
			f.detailBox.Add(widget.NewLabel("Aucun test confirmé pour l'instant."))
		}
	}
	f.detailBox.Refresh()

	// Inventaire
	if len(s.Inventory) > 0 {
		f.inventoryBox.RemoveAll()
		form := container.New(layout.NewFormLayout())
		for _, c := range s.Inventory {
			form.Add(text(c.Name, 12, true, colInk))
			if c.Err != nil {
				form.Add(wrapLabel("Erreur : " + c.Err.Error()))
				continue
			}
			form.Add(wrapLabel(c.Summary))
			for _, fl := range c.Fields {
				form.Add(text(fl.Label, 11, false, colMuted))
				form.Add(wrapLabel(fl.Value))
			}
		}
		f.inventoryBox.Add(form)
		f.inventoryBox.Refresh()
	}
}

// watchTemperature met à jour « CPU : xx °C » toutes les 3 s
func (f *fichePanel) watchTemperature() {
	for {
		if t, err := probe.CPUTemperature(); err == nil {
			fyne.Do(func() {
				f.cpuTemp.Text = fmt.Sprintf("CPU : %.0f °C", t)
				f.cpuTemp.Refresh()
			})
		}
		time.Sleep(3 * time.Second)
	}
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
