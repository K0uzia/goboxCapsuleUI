package gui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
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

	active  *testDef
	cancel  context.CancelFunc
	pending result
}

func newToolsPanel(s *session, win fyne.Window) *toolsPanel {
	p := &toolsPanel{s: s, win: win, buttons: map[string]*toolButton{}}

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
			menu.Add(hardRule())
		}
		menu.Add(caption(cat.name))
		menu.Add(grid)
	}
	p.menu = menu
	p.view = newTestView(p)
	p.root = container.NewStack(p.menu, p.view.root)
	p.view.root.Hide()

	s.onChange(p.refreshButtons)
	return p
}

func hardRule() fyne.CanvasObject {
	r := canvas.NewRectangle(colInk)
	r.SetMinSize(fyne.NewSize(0, 1.5))
	return container.NewPadded(r)
}

func (p *toolsPanel) refreshButtons() {
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

// ---- Vue d'un test --------------------------------------------------------

type testView struct {
	p    *toolsPanel
	root *fyne.Container

	category, title *canvas.Text
	desc            *widget.Label
	bar             *widget.ProgressBar
	barInf          *widget.ProgressBarInfinite
	panel           *fyne.Container
	askLabel        *widget.Label
	askButtons      *fyne.Container
	logText         *widget.Label
	logScroll       *container.Scroll
	logLines        []string
	running         *fyne.Container // boutons pendant le test
	resultBox       *fyne.Container
	resultPill      *pill
	resultHead      *canvas.Text
	resultDetail    *widget.Label
	resultMeta      *widget.Label
	finished        *fyne.Container // boutons après le test
}

func newTestView(p *toolsPanel) *testView {
	v := &testView{p: p}
	v.category = caption("")
	v.title = text("", 22, true, colInk)
	v.desc = widget.NewLabel("")
	v.bar = widget.NewProgressBar()
	v.bar.TextFormatter = func() string { return fmt.Sprintf("%.0f %%", v.bar.Value*100) }
	v.barInf = widget.NewProgressBarInfinite()
	v.panel = container.NewStack()
	v.askLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	v.askButtons = container.NewHBox()

	v.logText = widget.NewLabel("")
	v.logText.TextStyle = fyne.TextStyle{Monospace: true}
	v.logText.Wrapping = fyne.TextWrapWord
	v.logScroll = container.NewVScroll(v.logText)
	v.logScroll.SetMinSize(fyne.NewSize(0, 150))
	logBg := canvas.NewRectangle(colPanel)
	logBg.CornerRadius = 4

	v.resultPill = newPill(statusIdle)
	v.resultHead = text("", 18, true, colInk)
	v.resultDetail = wrapLabel("")
	v.resultMeta = widget.NewLabel("")
	v.resultBox = container.NewVBox(
		container.NewHBox(v.resultPill.root, v.resultHead),
		v.resultDetail, v.resultMeta,
		wrapLabel("La confirmation inscrit le résultat dans la fiche."),
	)

	v.running = container.NewHBox(layout.NewSpacer(), outlineButton("Annuler le test", func() { p.close(false) }))
	v.finished = container.NewHBox(layout.NewSpacer(),
		outlineButton("Relancer", func() { p.launch(p.active) }),
		solidButton("Confirmer et revenir au menu", colInk, func() { p.close(true) }),
	)

	v.root = container.NewBorder(
		container.NewVBox(v.category, v.title, v.desc, container.NewStack(v.bar, v.barInf)),
		container.NewVBox(container.NewStack(logBg, v.logScroll), container.NewStack(v.running, v.finished)),
		nil, nil,
		container.NewVScroll(container.NewVBox(v.resultBox, v.panel, v.askLabel, v.askButtons)),
	)
	return v
}

func (v *testView) start(t *testDef) {
	v.category.Text = strings.ToUpper(t.Category)
	v.category.Color = t.Color
	v.title.Text = strings.ToUpper(t.Title)
	v.category.Refresh()
	v.title.Refresh()
	v.desc.SetText(t.Desc)
	v.logLines = nil
	v.logText.SetText("")
	v.setPanel(nil)
	v.setAsk("", nil, nil)
	v.setProgress(0)
	v.resultBox.Hide()
	v.finished.Hide()
	v.running.Show()
}

func (v *testView) appendLog(msg string) {
	line := fmt.Sprintf("%s  %s", time.Now().Format("15:04:05"), msg)
	v.logLines = append(v.logLines, line)
	v.logText.SetText(strings.Join(v.logLines, "\n"))
	v.logScroll.ScrollToBottom()
}

func (v *testView) setProgress(f float64) {
	if f < 0 {
		v.bar.Hide()
		v.barInf.Show()
		v.barInf.Start()
		return
	}
	v.barInf.Stop()
	v.barInf.Hide()
	v.bar.Show()
	v.bar.SetValue(f)
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
	v.askButtons.RemoveAll()
	for _, o := range opts {
		v.askButtons.Add(widget.NewButton(o.Label, func() { answer(o.Status) }))
	}
	v.askButtons.Refresh()
}

func (v *testView) showResult(r result) {
	v.barInf.Stop()
	v.barInf.Hide()
	v.bar.Show()
	v.bar.SetValue(1)
	v.setAsk("", nil, nil)
	headlines := map[status]string{statusOK: "Test réussi", statusWarn: "Réussi avec réserve", statusFail: "Test échoué", statusNA: "Non applicable sur cette machine"}
	v.resultPill.set(r.Status)
	v.resultHead.Text = headlines[r.Status]
	v.resultHead.Refresh()
	v.resultDetail.SetText(r.Detail)
	v.resultMeta.SetText("Durée : " + r.Duration.Round(time.Second).String())
	v.resultBox.Show()
	v.running.Hide()
	v.finished.Show()
}
