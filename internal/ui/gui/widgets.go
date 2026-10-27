package gui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// ---- Zone cliquable -------------------------------------------------------

// tapArea rend n'importe quel contenu cliquable (curseur main)
type tapArea struct {
	widget.BaseWidget
	content  fyne.CanvasObject
	onTapped func()
}

func newTapArea(content fyne.CanvasObject, onTapped func()) *tapArea {
	t := &tapArea{content: content, onTapped: onTapped}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tapArea) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(t.content) }
func (t *tapArea) Cursor() desktop.Cursor              { return desktop.PointerCursor }

func (t *tapArea) Tapped(*fyne.PointEvent) {
	if t.onTapped != nil {
		t.onTapped()
	}
}

// ---- Bouton d'outil coloré ------------------------------------------------

// toolButton bouton de test : bordure couleur de catégorie, fond selon l'état
type toolButton struct {
	*tapArea
	bg    *canvas.Rectangle
	title *canvas.Text
	sub   *canvas.Text
	test  *testDef
}

func newToolButton(t *testDef, onTap func()) *toolButton {
	bg := canvas.NewRectangle(color.White)
	bg.CornerRadius = 6
	bg.StrokeWidth = 1.5
	bg.SetMinSize(fyne.NewSize(0, 56))

	title := canvas.NewText(strings.ToUpper(t.Title), t.Color)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 13
	title.Alignment = fyne.TextAlignCenter

	sub := canvas.NewText("Lancer", t.Color)
	sub.TextSize = 10
	sub.Alignment = fyne.TextAlignCenter

	content := container.NewStack(bg, container.NewVBox(layout.NewSpacer(), title, sub, layout.NewSpacer()))
	b := &toolButton{tapArea: newTapArea(content, onTap), bg: bg, title: title, sub: sub, test: t}
	b.set(statusIdle)
	return b
}

func (b *toolButton) set(s status) {
	st := s.style()
	border := b.test.Color
	switch s {
	case statusWarn, statusFail:
		border = st.solid
	case statusNA:
		border = colLine
	}
	fill, fg := color.Color(color.White), color.Color(b.test.Color)
	switch s {
	case statusRunning:
		fill, fg = colPanel, colInk
	case statusOK:
		fill, fg = b.test.Color, color.White
	case statusWarn, statusFail:
		fill, fg = st.solid, color.White
	case statusNA:
		fill, fg = colLineSoft, colGrayInk
	}
	b.bg.FillColor, b.bg.StrokeColor = fill, border
	b.title.Color, b.sub.Color = fg, fg
	b.sub.Text = st.sub
	b.bg.Refresh()
	b.title.Refresh()
	b.sub.Refresh()
}

// ---- Pastille de statut ---------------------------------------------------

type pill struct {
	root *fyne.Container
	bg   *canvas.Rectangle
	text *canvas.Text
}

func newPill(s status) *pill {
	bg := canvas.NewRectangle(colGraySoft)
	bg.CornerRadius = 9
	text := canvas.NewText("", colGrayInk)
	text.TextSize = 10
	text.TextStyle = fyne.TextStyle{Bold: true}
	p := &pill{root: container.NewStack(bg, container.NewPadded(container.NewCenter(text))), bg: bg, text: text}
	p.set(s)
	return p
}

func (p *pill) set(s status) {
	st := s.style()
	p.bg.FillColor, p.text.Color, p.text.Text = st.soft, st.ink, st.label
	p.bg.Refresh()
	p.text.Refresh()
}

// ---- Textes ---------------------------------------------------------------

// caption petit titre en capitales gris (« MACHINE », « VERDICT »…)
func caption(s string) *canvas.Text {
	t := canvas.NewText(strings.ToUpper(s), colMuted)
	t.TextSize = 10
	t.TextStyle = fyne.TextStyle{Bold: true}
	return t
}

func text(s string, size float32, bold bool, c color.Color) *canvas.Text {
	t := canvas.NewText(s, c)
	t.TextSize = size
	t.TextStyle = fyne.TextStyle{Bold: bold}
	return t
}

// sectionTitle titre de section souligné
func sectionTitle(s string) fyne.CanvasObject {
	line := canvas.NewRectangle(colInk)
	line.SetMinSize(fyne.NewSize(0, 1))
	return container.NewVBox(caption(s), line)
}

// box cadre à bordure fine
func box(content fyne.CanvasObject, border, fill color.Color) fyne.CanvasObject {
	bg := canvas.NewRectangle(fill)
	bg.StrokeColor = border
	bg.StrokeWidth = 1
	bg.CornerRadius = 5
	return container.NewStack(bg, container.NewPadded(content))
}

// wrapLabel label qui passe à la ligne
func wrapLabel(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Wrapping = fyne.TextWrapWord
	return l
}

// solidButton bouton plein (PRINT, Confirmer…)
func solidButton(label string, fill color.Color, onTap func()) fyne.CanvasObject {
	bg := canvas.NewRectangle(fill)
	bg.CornerRadius = 4
	bg.SetMinSize(fyne.NewSize(120, 36))
	t := text(label, 13, true, color.White)
	return newTapArea(container.NewStack(bg, container.NewCenter(t)), onTap)
}

// outlineButton bouton à bordure (Relancer, Annuler…)
func outlineButton(label string, onTap func()) fyne.CanvasObject {
	bg := canvas.NewRectangle(color.White)
	bg.StrokeColor = colInk
	bg.StrokeWidth = 1.5
	bg.CornerRadius = 4
	bg.SetMinSize(fyne.NewSize(120, 36))
	t := text(label, 13, true, colInk)
	return newTapArea(container.NewStack(bg, container.NewCenter(t)), onTap)
}

// ---- Barre d'avancement fine ----------------------------------------------

type thinBar struct {
	root   *fyne.Container
	layout *fracLayout
}

func newThinBar() *thinBar {
	track := canvas.NewRectangle(colLineSoft)
	track.CornerRadius = 4
	track.SetMinSize(fyne.NewSize(60, 8))
	fill := canvas.NewRectangle(colTeal)
	fill.CornerRadius = 4
	l := &fracLayout{}
	return &thinBar{root: container.NewStack(track, container.New(l, fill)), layout: l}
}

func (b *thinBar) set(f float32) {
	b.layout.frac = f
	b.root.Refresh()
}
