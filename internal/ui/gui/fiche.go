package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
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

const notMeasured = "non mesuré"

// fichePanel colonne de gauche : la fiche, mise à jour à chaque test confirmé
type fichePanel struct {
	s        *session
	root     fyne.CanvasObject
	scroll   *container.Scroll
	notes    *widget.Entry
	techRing *tRect
	techHint *tText

	counts       *tText
	progress     *thinBar
	machineName  *tText
	machineModel *tText
	machineSN    *tText
	verdictBg    *tRect
	verdictCap   *tText
	verdictText  *tText
	verdictSub   *tText
	pills        []struct {
		p  *pill
		id string
	}
	stressBefore, stressAfter *tText
	detailBox                 *fyne.Container
	inventoryBox              *fyne.Container
	detailed                  bool
}

func newFichePanel(s *session) *fichePanel {
	f := &fichePanel{s: s, detailed: true}
	f.scroll = container.NewVScroll(nil)

	// En-tête : référence, titre, date ; avancement à droite
	tech := widget.NewEntry()
	tech.SetPlaceHolder("Nom du technicien")
	tech.SetText(s.Form.Tech)
	tech.OnChanged = func(v string) {
		wasEmpty := s.Form.Tech == ""
		s.Form.Tech = v
		if wasEmpty != (v == "") {
			f.showTechHint()
			s.changed() // l'étape « Diagnostic initial » dépend du technicien
		}
	}
	f.counts = text("", 12, false, cMuted)
	f.counts.Alignment = fyne.TextAlignTrailing
	f.progress = newThinBar(cAccent, 6)

	ref := text(s.Ref+" · page 1-2", 12, false, cAccent)
	ref.Mono = true
	title := text("Fiche de reconditionnement", 20, true, cText)
	date := container.NewHBox(text("Fait le", 13, false, cMuted), text(s.Started.Format("02/01/2006 à 15:04"), 13, true, cText))
	progress := container.NewVBox(f.counts, container.New(&widthLayout{w: 170}, f.progress.root))
	header := container.NewBorder(nil, nil, nil, progress, container.NewVBox(ref, title, date))
	// Tant que le technicien manque, son champ est entouré et signalé : c'est
	// la première étape
	f.techRing = newRect("", cAccent, 7)
	f.techRing.StrokeWidth = 2
	f.techHint = text("À remplir pour commencer", 12, true, cAccent)
	techRow := container.NewHBox(
		container.NewVBox(layout.NewSpacer(), text("Fait par", 13, false, cMuted), layout.NewSpacer()),
		container.New(&widthLayout{w: 260}, container.NewStack(tech, f.techRing)),
		container.NewCenter(f.techHint),
	)
	f.showTechHint()

	// Synthèse : machine, aspect, nettoyage, verdict
	f.machineName = text("Détection…", 13, true, cText)
	f.machineModel = text("", 13, true, cText)
	f.machineSN = text("", 11, false, cMuted)
	f.machineSN.Mono = true
	machine := box(container.NewVBox(caption("Machine"), f.machineName, f.machineModel, f.machineSN), cLine, cSurface)

	aspect := widget.NewRadioGroup([]string{"Neuf", "Occasion", "Nul"}, func(v string) { s.Form.Aspect = v })
	aspect.SetSelected(s.Form.Aspect)
	aspectBox := box(container.NewVBox(caption("Aspect"), aspect), cLine, cSurface)

	cleanIn := widget.NewCheck("Intérieur", func(b bool) { s.Form.CleanIn = b })
	cleanIn.SetChecked(s.Form.CleanIn)
	cleanOut := widget.NewCheck("Extérieur", func(b bool) { s.Form.CleanOut = b })
	cleanOut.SetChecked(s.Form.CleanOut)
	cleanBox := box(container.NewVBox(caption("Nettoyage"), cleanIn, cleanOut), cLine, cSurface)

	f.verdictCap = caption("Verdict")
	f.verdictText = text("", 15, true, cText)
	f.verdictSub = text("", 12, false, cMuted)
	f.verdictBg = newRect(cSurface2, cLineStrong, 8)
	verdictBox := container.NewStack(f.verdictBg,
		container.New(&padLayout{h: 10, v: 8}, container.NewVBox(f.verdictCap, f.verdictText, f.verdictSub)))

	summary := container.New(&respGrid{minCol: 118, cols: []int{4, 2}, gap: 8}, machine, aspectBox, cleanBox, verdictBox)

	// Contrôles et maintenance, sur deux colonnes si la place le permet
	two := func(a, b fyne.CanvasObject) fyne.CanvasObject {
		return container.New(&respGrid{minCol: 250, cols: []int{2, 1}, gap: 20}, a, b)
	}
	f.stressBefore, f.stressAfter = text(notMeasured, 13, false, cMuted), text(notMeasured, 13, false, cMuted)
	maintA, maintB := f.maintenance()
	controls := container.NewVBox(
		two(f.ctrlSection("Sorties vidéo & ports", ctrlVideo), f.ctrlSection("Saisie & audio", ctrlAudio)),
		two(f.ctrlSection("Système & stockage", ctrlSystem), maintA),
		maintB,
	)

	notes := widget.NewMultiLineEntry()
	notes.SetPlaceHolder("Observations du technicien (reportées sur la fiche)")
	notes.SetMinRowsVisible(3)
	notes.SetText(s.Form.Notes)
	notes.OnChanged = func(v string) { s.Form.Notes = v }
	f.notes = notes

	f.detailBox = container.NewVBox()
	f.inventoryBox = container.NewVBox(text("Détection du matériel…", 13, false, cMuted))

	gap := func() fyne.CanvasObject { return container.New(&padLayout{v: 4}) }
	// En-tête fixe (référence, avancement, technicien), le reste défile
	top := container.New(&padLayout{h: 18, v: 14}, container.NewVBox(header, techRow, gap(), hRule(cLineStrong)))
	paper := container.NewVBox(
		summary, gap(), controls, gap(),
		sectionTitle("Observations du technicien"), f.pass(notes),
		f.detailBox, gap(),
		sectionTitle("Inventaire matériel"), f.inventoryBox,
	)
	f.scroll.Content = container.New(&padLayout{h: 18, v: 12}, paper)
	f.root = container.NewStack(newRect(cSurface, cLine, 12), container.NewBorder(top, nil, nil, nil, f.scroll))

	s.onChange(f.refresh)
	f.refresh()
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
		list.Add(listRow(text(r.Label, 13, false, cText), container.NewCenter(p.root)))
	}
	return list
}

// listRow ligne compacte : libellé à gauche, valeur à droite, filet dessous
func listRow(label, value fyne.CanvasObject) fyne.CanvasObject {
	row := container.NewBorder(nil, nil, nil, value, container.NewVBox(layout.NewSpacer(), label, layout.NewSpacer()))
	return container.NewVBox(container.New(&padLayout{v: 2}, row), hRule(cLine))
}

// formRow libellé à largeur fixe puis champ
func formRow(label string, field fyne.CanvasObject) fyne.CanvasObject {
	l := container.New(&widthLayout{w: 128}, container.NewVBox(layout.NewSpacer(), text(label, 13, false, cMuted), layout.NewSpacer()))
	return container.NewBorder(nil, nil, l, nil, field)
}

// maintenance relevés du stress test (à côté des contrôles système) et
// maintenance (pleine largeur, deux colonnes)
func (f *fichePanel) maintenance() (fyne.CanvasObject, fyne.CanvasObject) {
	s := &f.s.Form
	sel := func(opts []string, v *string) *widget.Select {
		// construit à la main : NewSelect cherche une traduction absente en français
		w := &widget.Select{Options: opts, PlaceHolder: "Choisir", OnChanged: func(x string) { *v = x }}
		w.ExtendBaseWidget(w)
		w.SetSelected(*v)
		return w
	}
	entry := func(v *string, placeholder string) fyne.CanvasObject {
		e := widget.NewEntry()
		e.SetText(*v)
		e.SetPlaceHolder(placeholder)
		e.OnChanged = func(x string) { *v = x }
		return f.pass(e)
	}
	yesNo := []string{"oui", "non"}

	stress := container.NewVBox(sectionTitle("Stress test"),
		listRow(text("Temp. CPU avant stress", 13, false, cText), f.stressBefore),
		listRow(text("Temp. CPU après stress", 13, false, cText), f.stressAfter),
		formRow("Pâte thermique", sel(yesNo, &s.Paste)),
		formRow("Chargeur OK", sel(yesNo, &s.Charger)),
	)
	maint := container.NewVBox(sectionTitle("Maintenance"),
		container.New(&respGrid{minCol: 250, cols: []int{2, 1}, gap: 20},
			container.NewVBox(
				formRow("Shred disques", sel(yesNo, &s.Shred)),
				formRow("Pile setup changée", sel(yesNo, &s.CMOS)),
				formRow("Lecteur DVD", sel([]string{"oui", "non", "absent"}, &s.DVD)),
			),
			container.NewVBox(
				formRow("USB défectueux", entry(&s.USBBad, "Nombre de ports")),
				formRow("Vidéo autre", entry(&s.OtherVideo, "Préciser si besoin")),
				formRow("Autre", entry(&s.Other, "Préciser si besoin")),
			),
		),
	)
	return stress, maint
}

// refresh recalcule tout ce qui dépend des résultats
func (f *fichePanel) refresh() {
	s := f.s
	done, ok, _, _ := s.counts()
	f.counts.SetText(fmt.Sprintf("%d / %d tests · %d OK", done, len(s.Tests), ok))
	f.progress.set(float32(done) / float32(len(s.Tests)))

	vendor, model := s.Machine.Vendor, s.Machine.Product
	if vendor == "" && model == "" {
		vendor = s.Machine.Name()
	}
	f.machineName.SetText(vendor)
	f.machineModel.SetText(model)
	f.machineSN.SetText("S/N " + orUnknown(s.Machine.Serial, "non lu"))

	v := s.verdict()
	ink, fill := v.roles()
	f.verdictBg.setRoles(fill, ink)
	f.verdictCap.Role, f.verdictText.Role = ink, ink
	f.verdictText.Text, f.verdictSub.Text = v.Label, v.Sub
	f.verdictCap.Refresh()
	f.verdictText.Refresh()
	f.verdictSub.Refresh()

	for _, p := range f.pills {
		p.p.set(s.status(p.id))
	}
	setMeasure(f.stressBefore, s.data("stress", "before"))
	setMeasure(f.stressAfter, s.data("stress", "after"))

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
			f.detailBox.Add(container.NewBorder(nil, hRule(cLine), container.New(&widthLayout{w: 110}, text(t.Title, 13, true, cText)),
				container.NewCenter(p.root), wrapLabel(r.Detail)))
		}
		if n == 0 {
			f.detailBox.Add(text("Aucun test confirmé pour l'instant.", 13, false, cMuted))
		}
	}
	f.detailBox.Refresh()

	// Inventaire
	if len(s.Inventory) > 0 {
		f.inventoryBox.RemoveAll()
		for _, c := range s.Inventory {
			f.inventoryBox.Add(text(c.Name, 13, true, cText))
			if c.Err != nil {
				l := wrapLabel("Erreur : " + c.Err.Error())
				l.Importance = widget.DangerImportance
				f.inventoryBox.Add(l)
				continue
			}
			if c.Summary != "" {
				f.inventoryBox.Add(wrapLabel(c.Summary))
			}
			for _, fl := range c.Fields {
				f.inventoryBox.Add(formRow(fl.Label, wrapLabel(fl.Value)))
			}
		}
		f.inventoryBox.Refresh()
	}
}

// setMeasure valeur relevée, ou « non mesuré » en discret
func setMeasure(t *tText, v string) {
	t.Text, t.Role, t.Bold = orUnknown(v, notMeasured), cText, v != ""
	if v == "" {
		t.Role = cMuted
	}
	t.Refresh()
}

// showTechHint signale le champ technicien tant qu'il est vide
func (f *fichePanel) showTechHint() {
	f.techRing.Hidden = f.s.Form.Tech != ""
	f.techHint.Hidden = f.techRing.Hidden
	f.techRing.Refresh()
	f.techHint.Refresh()
}

// setDetailed affiche ou masque le détail des tests confirmés
func (f *fichePanel) setDetailed(b bool) {
	f.detailed = b
	f.refresh()
}

func orUnknown(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// respGrid grille dont le nombre de colonnes suit la largeur : la première
// valeur de cols qui laisse au moins minCol par colonne
type respGrid struct {
	minCol float32
	cols   []int
	gap    float32
	cur    int // colonnes de la dernière mise en page (0 : pas encore)
}

func (g *respGrid) columns(w float32) int {
	for _, c := range g.cols {
		if w >= float32(c)*g.minCol+float32(c-1)*g.gap {
			return c
		}
	}
	return g.cols[len(g.cols)-1]
}

func (g *respGrid) rows(objs []fyne.CanvasObject, cols int) []float32 {
	var hs []float32
	for i, o := range objs {
		if i%cols == 0 {
			hs = append(hs, 0)
		}
		hs[len(hs)-1] = max(hs[len(hs)-1], o.MinSize().Height)
	}
	return hs
}

// MinSize largeur d'une seule colonne (la grille peut rétrécir), hauteur de
// la disposition en cours
func (g *respGrid) MinSize(objs []fyne.CanvasObject) fyne.Size {
	cols := g.cur
	if cols == 0 {
		cols = g.cols[0]
	}
	var w, h float32
	for _, o := range objs {
		w = max(w, o.MinSize().Width)
	}
	hs := g.rows(objs, cols)
	for _, rh := range hs {
		h += rh
	}
	return fyne.NewSize(w, h+float32(len(hs)-1)*g.gap)
}

func (g *respGrid) Layout(objs []fyne.CanvasObject, s fyne.Size) {
	cols := g.columns(s.Width)
	assumed := g.cur
	if assumed == 0 {
		assumed = g.cols[0]
	}
	g.cur = cols
	if cols != assumed && len(objs) > 0 {
		// la hauteur annoncée (MinSize) ne correspond plus : on redemande la
		// mise en page de la fenêtre, après celle-ci
		obj := objs[0]
		go fyne.Do(func() {
			if c := fyne.CurrentApp().Driver().CanvasForObject(obj); c != nil && c.Content() != nil {
				c.Content().Refresh()
			}
		})
	}
	cw := (s.Width - float32(cols-1)*g.gap) / float32(cols)
	hs := g.rows(objs, cols)
	y := float32(0)
	for i, o := range objs {
		r, c := i/cols, i%cols
		if c == 0 && r > 0 {
			y += hs[r-1] + g.gap
		}
		o.Move(fyne.NewPos(float32(c)*(cw+g.gap), y))
		o.Resize(fyne.NewSize(cw, hs[r]))
	}
}

// pass pose sur un champ une couche qui renvoie la molette à la fiche : le
// défilement interne d'un champ Fyne garde la molette même quand il n'a rien
// à faire défiler, ce qui bloquait la fiche sous le pointeur
func (f *fichePanel) pass(field fyne.CanvasObject) fyne.CanvasObject {
	return container.NewStack(field, newScrollPass(func(e *fyne.ScrollEvent) { f.scroll.Scrolled(e) }))
}
