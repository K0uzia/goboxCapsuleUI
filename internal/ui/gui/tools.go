package gui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// toolsPanel colonne de droite : grille des tests, ou le test en cours
type toolsPanel struct {
	s       *session
	win     fyne.Window
	root    *fyne.Container
	menu    fyne.CanvasObject
	buttons map[string]*toolButton
	view    *testView
	stepTag map[string]fyne.CanvasObject // « Étape en cours » par catégorie
	catDone map[string]*tText            // « 3 / 8 faits » par catégorie

	active  *testDef
	cancel  context.CancelFunc
	pending result
}

func newToolsPanel(s *session, win fyne.Window) *toolsPanel {
	p := &toolsPanel{s: s, win: win, buttons: map[string]*toolButton{}, stepTag: map[string]fyne.CanvasObject{}, catDone: map[string]*tText{}}

	menu := container.NewVBox()
	for i, cat := range []struct {
		name string
		cols int
	}{{catPeripherals, 4}, {catStorage, 3}, {catSystem, 3}} {
		grid := container.NewGridWithColumns(cat.cols)
		for _, t := range s.Tests {
			if t.Category == cat.name {
				b := newToolButton(t, func() { p.launch(t) })
				p.buttons[t.ID] = b
				grid.Add(b)
			}
		}
		if i > 0 {
			menu.Add(container.New(&padLayout{v: 10}, hRule(cLine)))
		}
		ink, _ := categoryRoles(cat.name)
		dot := newRect(ink, "", 4)
		dot.MinSz = fyne.NewSize(8, 8)
		tagText := text("ÉTAPE EN COURS", 10, true, cAccent)
		tag := container.NewStack(newRect(cAccentSoft, "", 9), container.New(&padLayout{h: 8, v: 2}, tagText))
		tag.Hide()
		p.stepTag[cat.name] = tag
		done := text("", 12, false, cMuted)
		p.catDone[cat.name] = done
		head := container.NewBorder(nil, nil,
			container.NewHBox(container.NewCenter(dot), text(strings.ToUpper(cat.name), 12, true, ink), container.NewCenter(tag)),
			container.NewCenter(done))
		menu.Add(container.New(&padLayout{v: 4}, head))
		menu.Add(grid)
	}
	menu.Add(container.New(&padLayout{v: 10}, hRule(cLine)))
	menu.Add(guide())
	p.menu = menu
	p.view = newTestView(p)
	p.root = container.NewStack(p.menu, p.view.root)
	p.view.root.Hide()

	s.onChange(p.refreshButtons)
	p.refreshButtons()
	return p
}

// guide déroulé d'un reconditionnement, sous les boutons de tests
func guide() fyne.CanvasObject {
	steps := []string{
		"Remplir l'en-tête de la fiche : technicien, aspect, nettoyage.",
		"Lancer chaque test, suivre ses consignes, puis confirmer le résultat.",
		"Compléter le stress test, la maintenance et les observations.",
		"PRINT : enregistrer la fiche en PDF ou l'imprimer.",
	}
	list := container.NewVBox(caption("Déroulé"))
	for i, s := range steps {
		num := text(fmt.Sprintf("%d", i+1), 13, true, cAccent)
		num.Alignment = fyne.TextAlignCenter
		badge := container.NewStack(newRect(cAccentSoft, "", 11), container.New(&padLayout{h: 7, v: 2}, num))
		list.Add(container.NewHBox(badge, container.NewCenter(text(s, 13, false, cText))))
	}
	list.Add(text("Échap annule un test, Entrée confirme son résultat.", 12, false, cMuted))
	return box(list, cLine, cSurface)
}

// setStep met en avant la catégorie de l'étape choisie ("" : aucune)
func (p *toolsPanel) setStep(cat string) {
	for name, tag := range p.stepTag {
		if name == cat {
			tag.Show()
		} else {
			tag.Hide()
		}
	}
}

func (p *toolsPanel) refreshButtons() {
	for cat, t := range p.catDone {
		done, total := 0, 0
		for _, d := range p.s.Tests {
			if d.Category == cat {
				total++
				if st := p.s.status(d.ID); st != statusIdle && st != statusRunning {
					done++
				}
			}
		}
		t.SetText(fmt.Sprintf("%d / %d faits", done, total))
	}
	for id, b := range p.buttons {
		st := p.s.status(id)
		if p.active != nil && p.active.ID == id {
			st = statusRunning
		}
		b.set(st)
	}
}

// launch démarre un test dans une goroutine
func (p *toolsPanel) launch(t *testDef) {
	if p.cancel != nil {
		p.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.active, p.cancel = t, cancel
	p.view.start(t)
	p.menu.Hide()
	p.view.root.Show()
	p.refreshButtons()

	tc := &testCtx{ctx: ctx, win: p.win, view: p.view}
	go func() {
		start := time.Now()
		r := t.Run(tc)
		if ctx.Err() != nil || r.Status == statusIdle {
			return // annulé
		}
		r.Duration = time.Since(start)
		fyne.Do(func() {
			if p.active == t {
				p.pending = r
				p.view.showResult(r)
			}
		})
	}()
}

// close revient au menu, en enregistrant ou non le résultat
func (p *toolsPanel) close(save bool) {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	id := p.active.ID
	p.active = nil
	p.view.root.Hide()
	p.menu.Show()
	if save {
		p.s.setResult(id, p.pending) // déclenche aussi refreshButtons
	} else {
		p.refreshButtons()
	}
}

// typedKey raccourcis : Échap annule le test, Entrée confirme le résultat.
// Inactifs pendant le test clavier, qui a besoin de toutes les touches.
func (p *toolsPanel) typedKey(e *fyne.KeyEvent) {
	if p.active == nil || p.active.ID == "clavier" {
		return
	}
	switch e.Name {
	case fyne.KeyEscape:
		p.close(false)
	case fyne.KeyReturn, fyne.KeyEnter:
		if p.view.finished.Visible() {
			p.close(true)
		}
	}
}

// ---- Vue d'un test --------------------------------------------------------

type testView struct {
	p    *toolsPanel
	root fyne.CanvasObject

	card, band      *tRect
	category, title *tText
	desc            *tText
	simBadge        fyne.CanvasObject
	bandIcon        *widget.Icon
	bar             *thinBar
	barPct          *tText
	barInf          *widget.ProgressBarInfinite
	panel           *fyne.Container
	askLabel        *widget.Label
	askButtons      *fyne.Container
	logRows         *fyne.Container
	logEmpty        *tText
	logLast         *widget.Label
	logBody         *fyne.Container
	logChevron      *widget.Icon
	logOpen         bool
	logCount        *tText
	logScroll       *container.Scroll
	logLines        []string
	running         *fyne.Container // boutons pendant le test
	resultBox       *fyne.Container
	resultBg        *tRect
	resultPill      *pill
	resultHead      *tText
	resultDetail    *widget.Label
	resultMeta      *tText
	resultAdvice    *fyne.Container
	adviceList      *fyne.Container
	finished        *fyne.Container // boutons après le test
	body            *fyne.Container
}

func newTestView(p *toolsPanel) *testView {
	v := &testView{p: p}
	v.category = caption("")
	v.title = text("", 22, true, cText)
	v.desc = text("", 13, false, cMuted)
	v.band = newRect(cSurface2, "", 10)
	v.simBadge = container.NewStack(newRect(cWarningBg, cWarning, 9), container.New(&padLayout{h: 9, v: 3}, text("DONNÉES SIMULÉES", 10, true, cWarning)))
	v.simBadge.Hide()
	v.bandIcon = widget.NewIcon(nil)
	bigIcon := container.NewCenter(container.NewGridWrap(fyne.NewSize(34, 34), v.bandIcon))
	band := container.NewStack(v.band, container.New(&padLayout{h: 16, v: 12}, container.NewBorder(nil, nil,
		container.NewHBox(bigIcon, container.New(&padLayout{h: 2})), container.NewVBox(v.simBadge), container.NewVBox(v.category, v.title, v.desc))))

	v.bar = newThinBar(cAccent, 6)
	v.barPct = text("0 %", 12, true, cMuted)
	v.barPct.Alignment = fyne.TextAlignTrailing
	v.barInf = widget.NewProgressBarInfinite()
	progress := container.NewStack(
		container.NewBorder(nil, nil, nil, container.New(&widthLayout{w: 44}, v.barPct), container.NewVBox(layout.NewSpacer(), v.bar.root, layout.NewSpacer())),
		v.barInf,
	)

	v.panel = container.NewStack()
	v.askLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	v.askLabel.Wrapping = fyne.TextWrapWord
	v.askButtons = container.NewHBox()

	logBox := v.newLog()

	v.resultPill = newPill(statusIdle)
	v.resultHead = text("", 17, true, cText)
	v.resultDetail = wrapLabel("")
	v.resultMeta = text("", 12, false, cMuted)
	v.adviceList = container.NewVBox()
	v.resultAdvice = container.NewStack(newRect(cSurface2, cLine, 8), container.New(&padLayout{h: 14, v: 10}, container.NewVBox(
		container.NewHBox(iconOf("lightbulb", cAccent, 14), container.NewCenter(text("Pistes d'amélioration", 13, true, cAccent))),
		v.adviceList,
	)))
	v.resultBg = newRect(cSurface2, cLine, 8)
	v.resultBox = container.NewVBox(
		container.NewStack(v.resultBg, container.New(&padLayout{h: 14, v: 10}, container.NewVBox(
			container.NewHBox(container.NewCenter(v.resultPill.root), v.resultHead),
			v.resultDetail, v.resultMeta,
		))),
		v.resultAdvice,
		text("La confirmation inscrit le résultat dans la fiche.", 12, false, cMuted),
	)

	v.running = container.NewHBox(layout.NewSpacer(),
		dangerButton("Annuler le test", theme.CancelIcon(), func() { p.close(false) }))
	v.finished = container.NewHBox(layout.NewSpacer(),
		secondaryButton("Relancer", theme.ViewRefreshIcon(), func() { p.launch(p.active) }),
		primaryButton("Confirmer et revenir au menu", theme.ConfirmIcon(), func() { p.close(true) }),
	)

	v.body = container.NewVBox(
		progress, v.resultBox, v.panel, v.askLabel, v.askButtons, logBox,
		container.NewStack(v.running, v.finished),
	)
	v.card = newRect(cSurface, cLine, 12)
	card := container.NewStack(v.card, container.NewVBox(band, container.New(&padLayout{h: 16, v: 12}, v.body)))
	v.root = container.NewVScroll(card)
	return v
}

func (v *testView) start(t *testDef) {
	v.category.Text, v.category.Role = strings.ToUpper(t.Category), t.Color
	v.title.Text, v.title.Role = strings.ToUpper(t.Title), t.Color
	v.category.Refresh()
	v.title.Refresh()
	v.desc.SetText(t.Desc)
	v.simBadge.Hide()
	v.bandIcon.SetResource(coloredIcon(t.Icon, t.Color))
	v.band.setRoles(t.Soft, "")
	v.card.setRoles(cSurface, t.Color)
	v.clearLog()
	v.setPanel(nil)
	v.setAsk("", nil, nil)
	v.setProgress(0)
	v.resultBox.Hide()
	v.finished.Hide()
	v.running.Show()
	v.body.Refresh()
}

// ---- Journal --------------------------------------------------------------

const logMaxRows = 300

// newLog journal du test, replié par défaut : une barre (flèche, nombre de
// lignes, dernier message, copier) qui déplie la liste des messages
func (v *testView) newLog() fyne.CanvasObject {
	v.logRows = container.NewVBox()
	v.logEmpty = text("En attente du premier message…", 13, false, cMuted)
	v.logCount = text("", 12, false, cMuted)
	v.logLast = widget.NewLabel("")
	v.logLast.Truncation = fyne.TextTruncateEllipsis
	v.logLast.TextStyle.Monospace = true
	v.logLast.Importance = widget.LowImportance
	v.logScroll = container.NewVScroll(container.New(&padLayout{h: 10, v: 8}, container.NewVBox(v.logEmpty, v.logRows)))
	v.logScroll.SetMinSize(fyne.NewSize(0, 160))
	v.logBody = container.NewStack(newRect(cLog, cLine, 8), v.logScroll)

	v.logChevron = widget.NewIcon(nil)
	copyBtn := secondaryButton("Copier", buttonIcon("copy"), func() {
		fyne.CurrentApp().Clipboard().SetContent(strings.Join(v.logLines, "\n"))
	})
	bar := newRect(cLog, cLine, 8)
	toggle := newTapArea(container.NewStack(bar, container.New(&padLayout{h: 10, v: 2}, container.NewBorder(nil, nil,
		container.NewHBox(sized(v.logChevron, 13), iconOf("terminal", cMuted, 13), container.NewCenter(caption("Journal")),
			container.New(&widthLayout{w: 70}, container.NewCenter(v.logCount))),
		nil, v.logLast))), func() { v.setLogOpen(!v.logOpen) })
	toggle.onState = func(hover, focus bool) {
		stroke := cLine
		if hover || focus {
			stroke = cBorder
		}
		bar.setRoles(cLog, stroke)
	}
	v.setLogOpen(false)
	return container.NewVBox(container.NewBorder(nil, nil, nil, copyBtn, toggle), v.logBody)
}

// setLogOpen déplie ou replie le journal
func (v *testView) setLogOpen(open bool) {
	v.logOpen = open
	v.logBody.Hidden = !open
	if open {
		v.logChevron.SetResource(coloredIcon("chevron-down", cMuted))
		v.logScroll.ScrollToBottom()
	} else {
		v.logChevron.SetResource(coloredIcon("chevron-right", cMuted))
	}
	if v.body != nil {
		v.body.Refresh()
	}
}

func (v *testView) clearLog() {
	v.logLines = nil
	v.logRows.RemoveAll()
	v.logEmpty.Show()
	v.logCount.SetText("")
	v.logLast.SetText("En attente du premier message…")
	v.setLogOpen(false)
}

func (v *testView) appendLog(msg string) {
	now := time.Now().Format("15:04:05")
	v.logLines = append(v.logLines, now+"  "+msg)
	if len(v.logRows.Objects) >= logMaxRows {
		v.logRows.Remove(v.logRows.Objects[0])
	}
	dot := newRect(logSeverity(msg), "", 3)
	dot.MinSz = fyne.NewSize(6, 6)
	clock := text(now, 12, false, cMuted)
	clock.Mono = true
	line := wrapLabel(msg)
	line.TextStyle.Monospace = true
	left := container.NewHBox(container.NewCenter(dot), container.New(&widthLayout{w: 62}, container.NewCenter(clock)))
	// décalage du libellé (marge intérieure du Label) pour aligner heure et texte
	v.logRows.Add(container.NewBorder(nil, nil, container.NewVBox(container.New(&padLayout{v: 6}, left)), nil, line))
	v.logEmpty.Hide()
	v.logCount.SetText(fmt.Sprintf("%d ligne(s)", len(v.logLines)))
	v.logLast.SetText(msg)
	v.logScroll.Refresh()
	v.logScroll.ScrollToBottom()
}

// logSeverity couleur de la pastille d'un message, d'après ses mots
func logSeverity(msg string) fyne.ThemeColorName {
	m := strings.ToLower(msg)
	for _, w := range []string{"erreur", "échec", "impossible", "arrêt immédiat"} {
		if strings.Contains(m, w) {
			return cDanger
		}
	}
	for _, w := range []string{"throttling", "seuil", "attention", "pas de", "simulées", "non "} {
		if strings.Contains(m, w) {
			return cWarning
		}
	}
	return cAccent
}

func (v *testView) setProgress(f float64) {
	if f < 0 {
		v.bar.root.Hide()
		v.barPct.Hide()
		v.barInf.Show()
		v.barInf.Start()
		return
	}
	v.barInf.Stop()
	v.barInf.Hide()
	v.bar.root.Show()
	v.barPct.Show()
	v.bar.set(float32(f))
	v.barPct.SetText(fmt.Sprintf("%.0f %%", f*100))
}

func (v *testView) setPanel(o fyne.CanvasObject) {
	v.panel.RemoveAll()
	if o != nil {
		v.panel.Add(o)
	}
	v.panel.Refresh()
}

func (v *testView) setAsk(prompt string, opts []choice, answer func(status)) {
	v.askLabel.SetText(prompt)
	v.askLabel.Hidden = prompt == ""
	v.askButtons.RemoveAll()
	for _, o := range opts {
		v.askButtons.Add(secondaryButton(o.Label, nil, func() { answer(o.Status) }))
	}
	v.askButtons.Refresh()
	v.askLabel.Refresh()
	if v.body != nil {
		v.body.Refresh()
	}
}

func (v *testView) showResult(r result) {
	v.setProgress(1)
	v.setAsk("", nil, nil)
	headlines := map[status]string{statusOK: "Test réussi", statusWarn: "Réussi avec réserve", statusFail: "Test échoué", statusNA: "Non applicable sur cette machine"}
	st := r.Status.style()
	v.resultPill.set(r.Status)
	v.resultBg.setRoles(st.soft, st.ink)
	v.resultHead.Text, v.resultHead.Role = headlines[r.Status], st.ink
	v.resultHead.Refresh()
	v.resultDetail.SetText(r.Detail)
	v.resultDetail.Hidden = r.Detail == ""
	v.adviceList.RemoveAll()
	for _, a := range r.Advice {
		dot := newRect(cAccent, "", 3)
		dot.MinSz = fyne.NewSize(6, 6)
		v.adviceList.Add(container.NewBorder(nil, nil, container.NewVBox(container.New(&padLayout{v: 13}, container.NewCenter(dot))), nil, wrapLabel(a)))
	}
	v.resultAdvice.Hidden = len(r.Advice) == 0
	v.resultMeta.SetText("Durée : " + formatDuration(r.Duration))
	v.resultBox.Show()
	v.running.Hide()
	v.finished.Show()
	v.body.Refresh()
}

// formatDuration « 42 s », « 3 min 05 s »
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%d s", int(d.Seconds()))
	}
	return fmt.Sprintf("%d min %02d s", int(d.Minutes()), int(d.Seconds())%60)
}
