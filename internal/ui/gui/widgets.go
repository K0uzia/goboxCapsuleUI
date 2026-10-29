package gui

import (
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// col couleur d'un rôle dans le thème courant ("" = transparent)
func col(name fyne.ThemeColorName) color.Color {
	if name == "" {
		return color.Transparent
	}
	return theme.Color(name)
}

// ---- Rectangle et texte suivant le thème ----------------------------------

// tRect rectangle dont les couleurs sont des rôles du thème
type tRect struct {
	widget.BaseWidget
	Fill, Stroke fyne.ThemeColorName
	StrokeWidth  float32
	CornerRadius float32
	MinSz        fyne.Size
	rect         *canvas.Rectangle
}

func newRect(fill, stroke fyne.ThemeColorName, radius float32) *tRect {
	r := &tRect{Fill: fill, Stroke: stroke, CornerRadius: radius, rect: canvas.NewRectangle(color.Transparent)}
	if stroke != "" {
		r.StrokeWidth = 1
	}
	r.ExtendBaseWidget(r)
	return r
}

// setRoles change les rôles puis rafraîchit
func (r *tRect) setRoles(fill, stroke fyne.ThemeColorName) {
	r.Fill, r.Stroke = fill, stroke
	r.Refresh()
}

// sync applique les rôles et la forme au rectangle dessiné
func (r *tRect) sync() {
	r.rect.FillColor, r.rect.StrokeColor = col(r.Fill), col(r.Stroke)
	r.rect.StrokeWidth, r.rect.CornerRadius = r.StrokeWidth, r.CornerRadius
}

func (r *tRect) CreateRenderer() fyne.WidgetRenderer {
	r.sync()
	return &rectRenderer{r: r}
}

type rectRenderer struct{ r *tRect }

func (rr *rectRenderer) Layout(s fyne.Size)           { rr.r.rect.Resize(s) }
func (rr *rectRenderer) MinSize() fyne.Size           { return rr.r.MinSz }
func (rr *rectRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{rr.r.rect} }
func (rr *rectRenderer) Destroy()                     {}
func (rr *rectRenderer) Refresh() {
	rr.r.sync()
	rr.r.rect.Refresh()
}

// tText texte d'une ligne dont la couleur est un rôle du thème
type tText struct {
	widget.BaseWidget
	Text      string
	Role      fyne.ThemeColorName
	TextSize  float32
	Bold      bool
	Mono      bool
	Alignment fyne.TextAlign
	txt       *canvas.Text
}

func text(s string, size float32, bold bool, role fyne.ThemeColorName) *tText {
	t := &tText{Text: s, Role: role, TextSize: size, Bold: bold, txt: canvas.NewText(s, color.Transparent)}
	t.ExtendBaseWidget(t)
	t.sync()
	return t
}

func (t *tText) SetText(s string) {
	t.Text = s
	t.Refresh()
}

func (t *tText) sync() {
	t.txt.Text, t.txt.Color, t.txt.TextSize = t.Text, col(t.Role), t.TextSize
	t.txt.TextStyle = fyne.TextStyle{Bold: t.Bold, Monospace: t.Mono}
	t.txt.Alignment = t.Alignment
}

func (t *tText) CreateRenderer() fyne.WidgetRenderer {
	t.sync()
	return &textRenderer{t: t}
}

type textRenderer struct{ t *tText }

func (tr *textRenderer) Layout(s fyne.Size)           { tr.t.txt.Resize(s) }
func (tr *textRenderer) MinSize() fyne.Size           { tr.t.sync(); return tr.t.txt.MinSize() }
func (tr *textRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{tr.t.txt} }
func (tr *textRenderer) Destroy()                     {}
func (tr *textRenderer) Refresh() {
	tr.t.sync()
	tr.t.txt.Refresh()
}

// ---- Zone cliquable -------------------------------------------------------

// tapArea rend un contenu cliquable : curseur main, survol, focus clavier
// (Tab), activation par Entrée ou Espace
type tapArea struct {
	widget.BaseWidget
	content  fyne.CanvasObject
	onTapped func()
	onState  func(hover, focus bool)
	hover    bool
	focus    bool
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

func (t *tapArea) state() {
	if t.onState != nil {
		t.onState(t.hover, t.focus)
	}
}

func (t *tapArea) MouseIn(*desktop.MouseEvent)    { t.hover = true; t.state() }
func (t *tapArea) MouseMoved(*desktop.MouseEvent) {}
func (t *tapArea) MouseOut()                      { t.hover = false; t.state() }
func (t *tapArea) FocusGained()                   { t.focus = true; t.state() }
func (t *tapArea) FocusLost()                     { t.focus = false; t.state() }
func (t *tapArea) TypedRune(r rune) {
	if r == ' ' {
		t.Tapped(nil)
	}
}
func (t *tapArea) TypedKey(e *fyne.KeyEvent) {
	if e.Name == fyne.KeyReturn || e.Name == fyne.KeyEnter {
		t.Tapped(nil)
	}
}

// ---- Bouton de test -------------------------------------------------------

// toolButton bouton de test : contour à la couleur de la catégorie, fond et
// texte selon l'état, contour animé pendant le test
type toolButton struct {
	*tapArea
	bg     *tRect
	title  *tText
	sub    *tText
	icon   *widget.Icon
	head   *fyne.Container
	test   *testDef
	status status
	anim   *fyne.Animation
}

func newToolButton(t *testDef, onTap func()) *toolButton {
	bg := newRect(cSurface, t.Color, 8)
	bg.MinSz = fyne.NewSize(0, 58)

	title := text(strings.ToUpper(t.Title), 14, true, t.Color)
	title.Alignment = fyne.TextAlignCenter
	sub := text("Lancer", 11, false, cMuted)
	sub.Alignment = fyne.TextAlignCenter

	icon := widget.NewIcon(nil)
	head := container.NewHBox(sized(icon, 16), title)
	content := container.NewStack(bg, container.NewVBox(layout.NewSpacer(), container.NewCenter(head), sub, layout.NewSpacer()))
	b := &toolButton{tapArea: newTapArea(content, onTap), bg: bg, title: title, sub: sub, icon: icon, head: head, test: t}
	b.onState = func(bool, bool) { b.paint() }
	b.set(statusIdle)
	return b
}

func (b *toolButton) set(s status) {
	b.status = s
	if s == statusRunning && b.anim == nil {
		b.anim = canvas.NewColorRGBAAnimation(col(cAccent), col(cAccentSoft), 700*time.Millisecond, func(c color.Color) {
			b.bg.rect.StrokeColor = c
			b.bg.rect.Refresh()
		})
		b.anim.AutoReverse = true
		b.anim.RepeatCount = fyne.AnimationRepeatForever
		b.anim.Start()
	} else if s != statusRunning && b.anim != nil {
		b.anim.Stop()
		b.anim = nil
	}
	b.paint()
}

func (b *toolButton) paint() {
	st := b.status.style()
	fill, stroke, fg := cSurface, b.test.Color, b.test.Color
	switch b.status {
	case statusRunning:
		fill, stroke, fg = cAccentSoft, cAccent, cAccent
	case statusOK, statusWarn, statusFail:
		fill, stroke, fg = st.soft, st.ink, st.ink
	case statusNA:
		fill, stroke, fg = cSurface2, cLine, cMuted
	}
	if b.hover && b.status != statusRunning {
		if b.status == statusIdle {
			fill = b.test.Soft
		}
	}
	b.bg.StrokeWidth = 1
	if b.focus {
		stroke, b.bg.StrokeWidth = cAccent, 2.5
	}
	b.bg.setRoles(fill, stroke)
	b.title.Role, b.sub.Role = fg, cMuted
	if b.status != statusIdle {
		b.sub.Role = fg
	}
	b.sub.Text = st.sub
	b.title.Refresh()
	b.sub.Refresh()
	res := statusIcon(b.status)
	if res == nil {
		res = coloredIcon(b.test.Icon, fg)
	}
	b.icon.SetResource(res)
	b.head.Refresh()
}

// ---- Pastille de statut ---------------------------------------------------

type pill struct {
	root *fyne.Container
	bg   *tRect
	text *tText
}

func newPill(s status) *pill {
	bg := newRect(cSurface2, "", 10)
	t := text("", 10, true, cMuted)
	t.Alignment = fyne.TextAlignCenter
	p := &pill{root: container.NewStack(bg, container.New(&padLayout{h: 9, v: 3}, t)), bg: bg, text: t}
	p.set(s)
	return p
}

func (p *pill) set(s status) {
	st := s.style()
	p.bg.setRoles(st.soft, "")
	p.text.Role, p.text.Text = st.ink, st.label
	p.text.Refresh()
}

// ---- Textes ---------------------------------------------------------------

// caption petit intitulé en capitales (« MACHINE », « VERDICT »…)
func caption(s string) *tText {
	return text(strings.ToUpper(s), 11, true, cMuted)
}

// sectionTitle intitulé de section suivi d'un filet
func sectionTitle(s string) fyne.CanvasObject {
	line := newRect(cLineStrong, "", 0)
	line.MinSz = fyne.NewSize(0, 1)
	return container.NewVBox(caption(s), line)
}

// box cadre : fond et contour en rôles du thème
func box(content fyne.CanvasObject, stroke, fill fyne.ThemeColorName) fyne.CanvasObject {
	return container.NewStack(newRect(fill, stroke, 8), container.New(&padLayout{h: 10, v: 8}, content))
}

// wrapLabel texte qui passe à la ligne
func wrapLabel(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Wrapping = fyne.TextWrapWord
	return l
}

// hRule filet horizontal
func hRule(role fyne.ThemeColorName) fyne.CanvasObject {
	r := newRect(role, "", 0)
	r.MinSz = fyne.NewSize(0, 1)
	return r
}

// ---- Boutons --------------------------------------------------------------

// primaryButton action principale (fond accent)
func primaryButton(label string, icon fyne.Resource, onTap func()) *widget.Button {
	b := widget.NewButtonWithIcon(label, icon, onTap)
	b.Importance = widget.HighImportance
	return b
}

// secondaryButton action secondaire
func secondaryButton(label string, icon fyne.Resource, onTap func()) *widget.Button {
	return widget.NewButtonWithIcon(label, icon, onTap)
}

// dangerButton action qui abandonne un travail en cours
func dangerButton(label string, icon fyne.Resource, onTap func()) *widget.Button {
	b := widget.NewButtonWithIcon(label, icon, onTap)
	b.Importance = widget.DangerImportance
	return b
}

// iconButton bouton d'icône seule ; sa fonction est dite par le titre de la
// boîte qu'il ouvre
func iconButton(icon fyne.Resource, onTap func()) *widget.Button {
	b := widget.NewButtonWithIcon("", icon, onTap)
	b.Importance = widget.LowImportance
	return b
}

// ---- Barre d'avancement fine ----------------------------------------------

type thinBar struct {
	root   *fyne.Container
	fill   *tRect
	layout *fracLayout
}

func newThinBar(fill fyne.ThemeColorName, height float32) *thinBar {
	track := newRect(cSurface2, "", height/2)
	track.MinSz = fyne.NewSize(60, height)
	f := newRect(fill, "", height/2)
	l := &fracLayout{}
	return &thinBar{root: container.NewStack(track, container.New(l, f)), fill: f, layout: l}
}

func (b *thinBar) set(f float32) {
	b.layout.frac = f
	b.root.Refresh()
}

// ---- Mises en page --------------------------------------------------------

// padLayout marge intérieure horizontale h et verticale v autour des enfants
type padLayout struct{ h, v float32 }

func (l *padLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var m fyne.Size
	for _, o := range objs {
		if o.Visible() {
			m = m.Max(o.MinSize())
		}
	}
	return m.Add(fyne.NewSize(2*l.h, 2*l.v))
}

func (l *padLayout) Layout(objs []fyne.CanvasObject, s fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(l.h, l.v))
		o.Resize(s.Subtract(fyne.NewSize(2*l.h, 2*l.v)))
	}
}

// widthLayout largeur minimale imposée (champ court)
type widthLayout struct{ w float32 }

func (l *widthLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var m fyne.Size
	for _, o := range objs {
		m = m.Max(o.MinSize())
	}
	return fyne.NewSize(max(m.Width, l.w), m.Height)
}

func (l *widthLayout) Layout(objs []fyne.CanvasObject, s fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(s)
	}
}

// ---- Renvoi de la molette -------------------------------------------------

// scrollPass couche transparente qui ne reçoit que la molette et la renvoie ;
// clics, survol, curseur et saisie atteignent le widget dessous
type scrollPass struct {
	widget.BaseWidget
	onScroll func(*fyne.ScrollEvent)
}

func newScrollPass(onScroll func(*fyne.ScrollEvent)) *scrollPass {
	p := &scrollPass{onScroll: onScroll}
	p.ExtendBaseWidget(p)
	return p
}

func (p *scrollPass) Scrolled(e *fyne.ScrollEvent) { p.onScroll(e) }

func (p *scrollPass) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewWithoutLayout())
}

// statusIcon icône d'un statut terminé, à la couleur du statut (nil sinon)
func statusIcon(s status) fyne.Resource {
	switch s {
	case statusOK:
		return theme.NewColoredResource(theme.ConfirmIcon(), cSuccess)
	case statusWarn:
		return theme.NewColoredResource(theme.WarningIcon(), cWarning)
	case statusFail:
		return theme.NewColoredResource(theme.CancelIcon(), cDanger)
	case statusNA:
		return theme.NewColoredResource(theme.ContentRemoveIcon(), cMuted)
	}
	return nil
}

// ---- Choix segmenté -------------------------------------------------------

// segmented choix exclusif présenté en boutons, qui passent sur plusieurs
// lignes quand la place manque (au lieu de boutons radio coupés)
type segmented struct {
	root     fyne.CanvasObject
	Selected string
	items    map[string]*segItem
	onChange func(string)
}

type segItem struct {
	tap   *tapArea
	bg    *tRect
	label *tText
}

func newSegmented(opts []string, selected string, minCol float32, onChange func(string)) *segmented {
	s := &segmented{Selected: selected, items: map[string]*segItem{}, onChange: onChange}
	objs := make([]fyne.CanvasObject, len(opts))
	for i, o := range opts {
		it := &segItem{bg: newRect(cSurface, cLine, 7), label: text(o, 13, false, cText)}
		it.bg.MinSz = fyne.NewSize(0, 36)
		it.label.Alignment = fyne.TextAlignCenter
		it.tap = newTapArea(container.NewStack(it.bg, container.NewCenter(it.label)), func() { s.set(o) })
		it.tap.onState = func(bool, bool) { s.paint() }
		s.items[o] = it
		objs[i] = it.tap
	}
	s.root = container.New(&respGrid{minCol: minCol, cols: []int{len(opts), 2, 1}, gap: 6}, objs...)
	s.paint()
	return s
}

func (s *segmented) set(v string) {
	s.Selected = v
	s.paint()
	if s.onChange != nil {
		s.onChange(v)
	}
}

func (s *segmented) paint() {
	for v, it := range s.items {
		fill, stroke, fg, bold := cSurface, cLine, cText, false
		if v == s.Selected {
			fill, stroke, fg, bold = cAccentSoft, cAccent, cAccent, true
		} else if it.tap.hover {
			fill, stroke = cSurface2, cBorder
		}
		it.bg.StrokeWidth = 1
		if it.tap.focus {
			stroke, it.bg.StrokeWidth = cAccent, 2
		}
		it.bg.setRoles(fill, stroke)
		it.label.Role, it.label.Bold = fg, bold
		it.label.Refresh()
	}
}

// mutedNote texte d'aide discret qui passe à la ligne
func mutedNote(s string) *widget.Label {
	l := wrapLabel(s)
	l.Importance = widget.LowImportance
	l.SizeName = theme.SizeNameCaptionText
	return l
}
