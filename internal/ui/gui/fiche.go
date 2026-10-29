package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const notMeasured = "non mesuré"

// fichePanel colonne de gauche : la fiche, dans l'ordre de la fiche papier,
// mise à jour à chaque test confirmé
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
	updaters     []func() // lignes reliées aux résultats des tests
	detailBox    *fyne.Container
	inventoryBox *fyne.Container
	detailed     bool
}

func newFichePanel(s *session) *fichePanel {
	f := &fichePanel{s: s}
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

	// Contrôles, sur deux colonnes si la place le permet
	two := func(a, b fyne.CanvasObject) fyne.CanvasObject {
		return container.New(&respGrid{minCol: 235, cols: []int{2, 1}, gap: 18}, a, b)
	}
	usbBad := widget.NewEntry()
	usbBad.SetText(s.Form.USBBad)
	usbBad.SetPlaceHolder("Nombre")
	usbBad.OnChanged = func(v string) { s.Form.USBBad = v }
	video := f.ctrlSection("Sorties vidéo & ports", ctrlVideo)
	video.Add(formRow("USB défectueux", container.New(&widthLayout{w: 90}, f.pass(usbBad))))
	controls := container.NewVBox(
		two(video, f.ctrlSection("Saisie & audio", ctrlAudio)),
		two(f.maintenance(), f.ctrlSection("Tests gobox", ctrlGobox)),
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

// statusRoles couleurs d'une case OK / Moyen / Nul
func statusRoles(label string) (fyne.ThemeColorName, fyne.ThemeColorName) {
	for _, st := range []status{statusOK, statusWarn, statusFail} {
		if scaleLabel(st) == label {
			sty := st.style()
			return sty.soft, sty.ink
		}
	}
	return cAccentSoft, cAccent
}

// ctrlSection liste de contrôles : pastille du test qui la renseigne, ou
// choix du technicien quand aucun test ne la couvre
func (f *fichePanel) ctrlSection(title string, items []ctrlItem) *fyne.Container {
	list := container.NewVBox(sectionTitle(title))
	for _, it := range items {
		if it.TestID == "" {
			list.Add(f.manualRow(it))
			continue
		}
		p := newPill(statusIdle)
		value := text("", 12, false, cMuted)
		value.Mono = true
		f.updaters = append(f.updaters, func() {
			st, v := f.s.itemState(it)
			p.set(st)
			value.SetText(v)
		})
		list.Add(listRow(text(it.Label, 13, false, cText), container.NewHBox(container.NewCenter(value), container.NewCenter(p.root))))
	}
	return list
}

// manualRow ligne saisie par le technicien : cases de la fiche papier
func (f *fichePanel) manualRow(it ctrlItem) fyne.CanvasObject {
	form := &f.s.Form
	opts := make([]string, len(it.Scale))
	for i, st := range it.Scale {
		opts[i] = scaleLabel(st)
	}
	bar := newChoiceBar(opts, scaleLabel(form.Checks[it.Label]), true, statusRoles, func(v string) {
		st := statusIdle
		for _, x := range it.Scale {
			if scaleLabel(x) == v {
				st = x
			}
		}
		if st == statusIdle {
			delete(form.Checks, it.Label)
		} else {
			form.Checks[it.Label] = st
		}
	})
	var label fyne.CanvasObject = text(it.Label, 13, false, cText)
	if it.Free {
		e := widget.NewEntry()
		e.SetPlaceHolder(it.Label + "…")
		e.SetText(form.Texts[it.Label])
		e.OnChanged = func(v string) { form.Texts[it.Label] = v }
		label = container.New(&padLayout{h: 0, v: 2}, f.pass(e))
	}
	row := container.NewBorder(nil, nil, nil, container.NewCenter(container.New(&widthLayout{w: float32(52 * len(opts))}, bar.root)), container.NewVBox(layout.NewSpacer(), label, layout.NewSpacer()))
	return container.NewVBox(container.New(&padLayout{v: 2}, row), hRule(cLine))
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

// maintenance températures du stress test (case « moins de 90 °C ») et
// questions oui / non de la fiche papier
func (f *fichePanel) maintenance() fyne.CanvasObject {
	list := container.NewVBox(sectionTitle("Stress test & maintenance"))
	for _, m := range []struct{ label, key string }{{"CPU avant stress", "before"}, {"CPU après stress", "after"}} {
		value := text(notMeasured, 13, false, cMuted)
		p := newPill(statusIdle)
		p.root.Hide()
		f.updaters = append(f.updaters, func() {
			v := f.s.data("stress", m.key)
			setMeasure(value, v)
			if _, below, ok := stressTemp(v); ok {
				st := statusFail
				if below {
					st = statusOK
				}
				p.set(st)
				p.text.SetText(tempLabel(v))
				p.root.Show()
			} else {
				p.root.Hide()
			}
		})
		list.Add(listRow(text(m.label, 13, false, cText), container.NewHBox(container.NewCenter(value), container.NewCenter(p.root))))
	}
	form := &f.s.Form
	for _, m := range maintRows {
		v := m.Value(form)
		bar := newChoiceBar(m.Opts, *v, false, nil, func(x string) { *v = x })
		list.Add(listRow(text(m.Short, 13, false, cText), container.New(&widthLayout{w: float32(52 * len(m.Opts))}, bar.root)))
	}
	return list
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

	for _, u := range f.updaters {
		u()
	}

	// Détail des tests confirmés (désactivé par défaut, voir Réglages)
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

	// Inventaire : une carte par composant, icône et texte
	if len(s.Inventory) > 0 {
		f.inventoryBox.RemoveAll()
		cards := []fyne.CanvasObject{inventoryCard(machineComponent(s.Machine))}
		for _, c := range s.Inventory {
			cards = append(cards, inventoryCard(c))
		}
		f.inventoryBox.Add(container.New(&respGrid{minCol: 250, cols: []int{2, 1}, gap: 10}, cards...))
		f.inventoryBox.Refresh()
	}
}

// machineComponent identité de la machine, première carte de l'inventaire
func machineComponent(m machineInfo) Component {
	return Component{Name: "Machine", Icon: "computer", Summary: m.Name(), Fields: []Field{
		{"Marque", orUnknown(m.Vendor, "non lue")},
		{"Modèle", orUnknown(m.Product, "non lu")},
		{"N° de série", orUnknown(m.Serial, "non lu (droits root)")},
	}}
}

// inventoryCard carte d'un composant : icône, nom, résumé, puis ses valeurs
func inventoryCard(c Component) fyne.CanvasObject {
	ico := c.Icon
	if ico == "" {
		ico = "microchip"
	}
	badge := container.NewStack(newRect(cAccentSoft, "", 8), container.New(&padLayout{h: 7, v: 7}, iconOf(ico, cAccent, 18)))
	summary := wrapLabel(c.Summary)
	summary.TextStyle.Bold = true
	head := container.NewBorder(nil, nil, container.NewVBox(badge), nil, container.NewVBox(caption(c.Name), summary))
	rows := container.NewVBox()
	if c.Err != nil {
		l := wrapLabel("Erreur : " + c.Err.Error())
		l.Importance = widget.DangerImportance
		rows.Add(l)
	}
	for _, fl := range c.Fields {
		value := widget.NewLabel(fl.Value)
		value.Wrapping = fyne.TextWrapWord
		value.SizeName = theme.SizeNameCaptionText
		label := text(fl.Label, 11, false, cMuted)
		rows.Add(container.NewBorder(nil, nil, container.New(&widthLayout{w: 110}, container.NewVBox(container.New(&padLayout{v: 7}, label))), nil, value))
	}
	content := container.NewVBox(head)
	if len(rows.Objects) > 0 {
		content.Add(hRule(cLine))
		content.Add(rows)
	}
	return box(content, cLine, cSurface2)
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
			app := fyne.CurrentApp() // nil si l'application a été fermée entre-temps
			if app == nil || app.Driver() == nil {
				return
			}
			if c := app.Driver().CanvasForObject(obj); c != nil && c.Content() != nil {
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
