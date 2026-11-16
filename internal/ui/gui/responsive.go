package gui

// Mise en page qui suit la largeur de la fenêtre : seulement dans la démo en
// ligne (téléphone, tablette, ordinateur). L'application de l'atelier
// (responsive faux) garde sa mise en page d'ordinateur, de 1366×768 à
// 2560×1440 : rien ici ne s'y applique.

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"gobox/internal/probe"
)

var (
	// responsive mise en page selon la largeur (démo en ligne)
	responsive bool
	// touchUI écran tactile : chaque cible fait au moins touchTarget de haut
	touchUI bool
)

const (
	// wideMin largeur à partir de laquelle la fiche et les tests sont côte à
	// côte ; en dessous, deux onglets
	wideMin float32 = 1000
	// phoneMax en dessous : téléphone (marges réduites)
	phoneMax float32 = 600
	// touchTarget hauteur minimale d'une cible au doigt
	touchTarget float32 = 44
)

// touchMin hauteur d'une cible : agrandie sur écran tactile
func touchMin(h float32) float32 {
	if touchUI && h < touchTarget {
		return touchTarget
	}
	return h
}

// labelPad décalage vertical qui aligne un repère (puce, heure, libellé) sur
// la première ligne d'un widget.Label, dont la marge intérieure grandit sur
// écran tactile
func labelPad(v float32) float32 {
	if touchUI {
		return v + 7
	}
	return v
}

// rtext texte qui passe à la ligne dans la démo adaptée à la largeur, sur une
// ligne dans l'application de l'atelier
func rtext(s string, size float32, bold bool, role fyne.ThemeColorName) *tText {
	t := text(s, size, bold, role)
	t.Wrap = responsive
	return t
}

// ---- Texte sur plusieurs lignes (tText.Wrap) --------------------------------

func (t *tText) style() fyne.TextStyle { return fyne.TextStyle{Bold: t.Bold, Monospace: t.Mono} }

// wrapCache découpage et mot le plus long, recalculés seulement quand le
// texte, sa taille ou la largeur changent (une mise en page par image)
type wrapCache struct {
	key   wrapKey
	w     float32
	lines []string
	wkey  wrapKey
	word  float32
	lh    float32
}

type wrapKey struct {
	text       string
	size       float32
	bold, mono bool
}

func (t *tText) cacheKey() wrapKey { return wrapKey{t.Text, t.TextSize, t.Bold, t.Mono} }

// wrapLines découpe le texte entre les mots pour tenir dans w
func (t *tText) wrapLines(w float32) []string {
	key := t.cacheKey()
	if t.wrapC.key == key && t.wrapC.w == w {
		return t.wrapC.lines
	}
	out := t.splitLines(w)
	t.wrapC.key, t.wrapC.w, t.wrapC.lines = key, w, out
	return out
}

func (t *tText) splitLines(w float32) []string {
	var out []string
	for _, para := range strings.Split(t.Text, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			try := line + " " + word
			if w > 0 && fyne.MeasureText(try, t.TextSize, t.style()).Width > w {
				out = append(out, line)
				line = word
				continue
			}
			line = try
		}
		out = append(out, line)
	}
	return out
}

func (t *tText) lineHeight() float32 {
	t.wordWidth() // met à jour la hauteur de ligne avec le mot le plus long
	return t.wrapC.lh
}

// wordWidth largeur du mot le plus long (le texte ne se coupe pas dans un mot)
func (t *tText) wordWidth() float32 {
	key := t.cacheKey()
	if t.wrapC.wkey != key {
		var ww float32
		for _, word := range strings.Fields(t.Text) {
			ww = max(ww, fyne.MeasureText(word, t.TextSize, t.style()).Width)
		}
		t.wrapC.wkey, t.wrapC.word = key, ww
		t.wrapC.lh = fyne.MeasureText("Ag", t.TextSize, t.style()).Height
	}
	return t.wrapC.word
}

// wrappedMinSize le mot le plus long en largeur ; en hauteur, les lignes de la
// dernière largeur connue (une seule avant la première mise en page)
func (t *tText) wrappedMinSize() fyne.Size {
	w := t.wordWidth()
	n := 1
	if t.wrapW > 0 {
		n = len(t.wrapLines(t.wrapW))
	} else {
		n = strings.Count(t.Text, "\n") + 1
	}
	return fyne.NewSize(w, float32(n)*t.lineHeight())
}

func (t *tText) layoutWrapped(s fyne.Size) {
	before := 0
	if t.wrapW > 0 {
		before = len(t.wrapLines(t.wrapW))
	} else {
		before = strings.Count(t.Text, "\n") + 1
	}
	t.wrapW = s.Width
	rows := t.wrapLines(s.Width)
	if len(t.lines) == len(rows) && t.lastLayout == s && t.lastRole == t.Role {
		same := true
		for i, r := range rows {
			if t.lines[i].Text != r {
				same = false
				break
			}
		}
		if same {
			return
		}
	}
	t.lastLayout, t.lastRole = s, t.Role
	lines := t.lines
	for len(lines) < len(rows) {
		lines = append(lines, canvas.NewText("", col(t.Role)))
	}
	lines = lines[:len(rows)]
	t.lines = lines
	lh := t.lineHeight()
	for i, r := range rows {
		l := lines[i]
		l.Text, l.Color, l.TextSize, l.TextStyle, l.Alignment = r, col(t.Role), t.TextSize, t.style(), t.Alignment
		l.Move(fyne.NewPos(0, float32(i)*lh))
		l.Resize(fyne.NewSize(s.Width, lh))
		l.Refresh()
	}
	if len(rows) != before {
		relayoutLater(t)
	}
}

// relayouts nombre de mises en page redemandées (recette : une boucle se voit)
var (
	relayouts      int
	relayoutQueued bool
)

// relayoutLater la hauteur annoncée d'un objet a changé avec sa largeur : la
// fenêtre refait sa mise en page, après celle en cours
func relayoutLater(obj fyne.CanvasObject) {
	relayouts++
	// plusieurs demandes dans la même mise en page : une seule reprise
	if relayoutQueued {
		return
	}
	relayoutQueued = true
	go fyne.Do(func() {
		relayoutQueued = false
		app := fyne.CurrentApp()
		if app == nil || app.Driver() == nil {
			return
		}
		if c := app.Driver().CanvasForObject(obj); c != nil && c.Content() != nil {
			c.Content().Refresh()
		}
	})
}

// ---- Rangée qui passe à la ligne ---------------------------------------------

// flowLayout objets à leur taille minimale, de gauche à droite, sur plusieurs
// lignes quand la largeur manque. spread : tout tient sur une ligne, le
// dernier objet est calé à droite (comme une bordure droite). grow : le
// premier objet prend la largeur restante de sa ligne.
type flowLayout struct {
	gap, vgap float32
	spread    bool
	grow      bool
	right     bool // chaque ligne calée à droite
	stretch   bool // sur plusieurs lignes, un objet seul sur sa ligne en prend toute la largeur
	lastW     float32
}

func visibleObjs(objs []fyne.CanvasObject) []fyne.CanvasObject {
	var out []fyne.CanvasObject
	for _, o := range objs {
		if o.Visible() {
			out = append(out, o)
		}
	}
	return out
}

// lines répartit les objets en lignes pour une largeur w (0 : une seule ligne)
func (l *flowLayout) lines(objs []fyne.CanvasObject, w float32) [][]fyne.CanvasObject {
	var out [][]fyne.CanvasObject
	var cur []fyne.CanvasObject
	var x float32
	for _, o := range visibleObjs(objs) {
		ow := o.MinSize().Width
		if len(cur) > 0 && w > 0 && x+l.gap+ow > w {
			out = append(out, cur)
			cur, x = nil, 0
		}
		if len(cur) > 0 {
			x += l.gap
		}
		cur = append(cur, o)
		x += ow
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}

func (l *flowLayout) height(objs []fyne.CanvasObject, w float32) float32 {
	var h float32
	ls := l.lines(objs, w)
	for i, line := range ls {
		var lh float32
		for _, o := range line {
			lh = max(lh, o.MinSize().Height)
		}
		h += lh
		if i > 0 {
			h += l.vgap
		}
	}
	return h
}

func (l *flowLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var w float32
	for _, o := range visibleObjs(objs) {
		w = max(w, o.MinSize().Width)
	}
	return fyne.NewSize(w, l.height(objs, l.lastW))
}

func (l *flowLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	if l.height(objs, size.Width) != l.height(objs, l.lastW) && len(objs) > 0 {
		relayoutLater(objs[0])
	}
	l.lastW = size.Width
	ls := l.lines(objs, size.Width)
	var y float32
	for li, line := range ls {
		var lh, used float32
		for i, o := range line {
			lh = max(lh, o.MinSize().Height)
			used += o.MinSize().Width
			if i > 0 {
				used += l.gap
			}
		}
		x := float32(0)
		if l.right {
			x = max(0, size.Width-used)
		}
		for i, o := range line {
			w := o.MinSize().Width
			if l.grow && i == 0 && li == 0 {
				w += max(0, size.Width-used)
			}
			if l.spread && len(ls) == 1 && i == len(line)-1 && i > 0 {
				x = size.Width - w
			}
			if l.stretch && len(ls) > 1 && len(line) == 1 {
				w = size.Width
			}
			h := o.MinSize().Height
			o.Move(fyne.NewPos(x, y+(lh-h)/2))
			o.Resize(fyne.NewSize(min(w, size.Width), h))
			x += w + l.gap
		}
		y += lh + l.vgap
	}
}

// ---- Largeur maximale ----------------------------------------------------------

// maxWidthLayout objet centré, au plus max de large
type maxWidthLayout struct{ max float32 }

func (l *maxWidthLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var m fyne.Size
	for _, o := range objs {
		m = m.Max(o.MinSize())
	}
	return m
}

func (l *maxWidthLayout) Layout(objs []fyne.CanvasObject, s fyne.Size) {
	w := min(s.Width, max(l.max, l.MinSize(objs).Width))
	for _, o := range objs {
		o.Move(fyne.NewPos((s.Width-w)/2, 0))
		o.Resize(fyne.NewSize(w, s.Height))
	}
}

// boxWidth largeur d'une boîte de dialogue : want, sans dépasser la fenêtre
// (marge de 16 de chaque côté) dans la démo adaptée à la largeur
func boxWidth(w fyne.Window, want float32) float32 {
	if !responsive || w == nil {
		return want
	}
	if cw := w.Canvas().Size().Width; cw > 0 {
		return min(want, cw-64)
	}
	return want
}

// ---- Fenêtre de la démo ---------------------------------------------------------

// respContent en-tête et corps qui suivent la largeur : fiche et tests côte à
// côte sur grand écran, deux onglets (Tests, Fiche) sur tablette et téléphone
func respContent(s *session, w fyne.Window, fiche *fichePanel, tools *toolsPanel, onPrint func()) fyne.CanvasObject {
	body := &respBody{fiche: fiche, tools: tools, tab: "Tests"}
	tabs := newSegmented([]string{"Tests", "Fiche"}, body.tab, 100, func(v string) {
		body.tab = v
		body.apply()
	})
	body.tabs = tabs
	// libellés des onglets : avancement des tests, champs de la fiche à remplir
	label := func() {
		done, _, _, _ := s.counts()
		tabs.items["Tests"].label.SetText(fmt.Sprintf("Tests  %d / %d", done, len(s.Tests)))
		f := "Fiche"
		if n := len(s.missingInfo()); n > 0 {
			f = fmt.Sprintf("Fiche  (%d à remplir)", n)
		}
		tabs.items["Fiche"].label.SetText(f)
	}
	s.onChange(label)
	fiche.onRequired = label
	label()

	body.pad = &padLayout{h: 16, v: 12}
	body.root = container.New(body.pad, container.New(body, fiche.root, tools.root, tabs.root))
	head := respHeaderBar(s, w, fiche, tools, onPrint)
	return container.NewBorder(head, nil, nil, nil, body.root)
}

// respBody corps de la fenêtre : deux colonnes ou deux onglets
type respBody struct {
	fiche   *fichePanel
	tools   *toolsPanel
	tabs    *segmented
	tab     string
	pad     *padLayout
	root    *fyne.Container
	wide    bool
	compact bool
	ready   bool
}

const tabGap = 10

func (b *respBody) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(280, 300) }

func (b *respBody) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	wide := size.Width >= wideMin-32
	// téléphone (ou écran bas) : marges réduites, en-tête de la fiche qui défile
	compact := size.Width < phoneMax || size.Height < 520
	if !b.ready || wide != b.wide || compact != b.compact {
		b.ready, b.wide, b.compact = true, wide, compact
		go fyne.Do(b.apply)
	}
	fiche, tools, tabs := objs[0], objs[1], objs[2]
	if b.wide {
		lw := (size.Width - 24) * 0.46
		fiche.Resize(fyne.NewSize(lw, size.Height))
		fiche.Move(fyne.NewPos(0, 0))
		tools.Resize(fyne.NewSize(size.Width-lw-24, size.Height))
		tools.Move(fyne.NewPos(lw+24, 0))
		return
	}
	th := tabs.MinSize().Height
	tw := min(size.Width, 560)
	tabs.Resize(fyne.NewSize(tw, th))
	tabs.Move(fyne.NewPos((size.Width-tw)/2, 0))
	for _, o := range []fyne.CanvasObject{fiche, tools} {
		o.Resize(fyne.NewSize(size.Width, size.Height-th-tabGap))
		o.Move(fyne.NewPos(0, th+tabGap))
	}
}

// apply montre ce qui va avec la largeur : colonnes ou onglet choisi, marges
func (b *respBody) apply() {
	show := func(o fyne.CanvasObject, on bool) {
		if on && !o.Visible() {
			o.Show()
		} else if !on && o.Visible() {
			o.Hide()
		}
	}
	show(b.tabs.root, !b.wide)
	show(b.fiche.root, b.wide || b.tab == "Fiche")
	show(b.tools.root, b.wide || b.tab == "Tests")
	b.fiche.setCompact(b.compact)
	if b.compact {
		b.pad.h, b.pad.v = 8, 8
	} else {
		b.pad.h, b.pad.v = 16, 12
	}
	b.root.Refresh()
}

// ---- En-tête ----------------------------------------------------------------------

// respHeaderBar en-tête sur une ligne quand il y a la place, sinon marque et
// actions en haut, étapes dessous ; les indicateurs passent dans À propos
// quand la ligne du haut est trop étroite
func respHeaderBar(s *session, w fyne.Window, fiche *fichePanel, tools *toolsPanel, onPrint func()) fyne.CanvasObject {
	brand := container.NewCenter(text("gobox", 20, true, cAccent))
	chrono := text(s.elapsed(), 13, true, cText)
	cpu := text("non lue", 13, true, cText)
	headerCPU = cpu
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
	chips := container.NewHBox(chip("clock", "Durée", chrono), chip("temperature-half", "CPU", cpu))
	headerChips = chips
	actions := container.NewHBox(
		primaryButton("PRINT", coloredIcon("print", cAccentText), onPrint),
		iconButton(buttonIcon("circle-info"), func() { showInfo(s, w) }),
		iconButton(buttonIcon("gear"), func() { showSettings(w, fiche) }),
	)
	pad := &padLayout{h: 16, v: 8}
	bar := container.New(&respHeaderLayout{gap: 12, pad: pad}, brand, stepBar(s, tools.setStep), chips, actions)
	return container.NewVBox(
		container.NewStack(newRect(cSurface, "", 0), container.New(pad, bar)),
		hRule(cLine),
	)
}

// headerCPU température de l'en-tête, reprise dans À propos quand l'en-tête
// est trop étroit pour l'afficher
var (
	headerCPU   *tText
	headerChips fyne.CanvasObject
)

// respHeaderLayout marque, étapes, indicateurs, actions
type respHeaderLayout struct {
	gap     float32
	pad     *padLayout
	twoRows bool
	lastW   float32
}

const stepWidth = 460

func (l *respHeaderLayout) oneRow(objs []fyne.CanvasObject, w float32) bool {
	return objs[0].MinSize().Width+objs[2].MinSize().Width+objs[3].MinSize().Width+3*l.gap+380 <= w
}

func (l *respHeaderLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	brand, step, actions := objs[0].MinSize(), objs[1].MinSize(), objs[3].MinSize()
	top := max(brand.Height, actions.Height, objs[2].MinSize().Height)
	w := brand.Width + actions.Width + l.gap
	if l.twoRows {
		return fyne.NewSize(w, top+8+step.Height)
	}
	return fyne.NewSize(w, max(top, step.Height))
}

func (l *respHeaderLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	brand, step, chips, actions := objs[0], objs[1], objs[2], objs[3]
	two := !l.oneRow(objs, size.Width)
	if two != l.twoRows {
		l.twoRows = two
		relayoutLater(brand)
	}
	if l.pad != nil {
		if size.Width < phoneMax-32 {
			l.pad.h = 8
		} else {
			l.pad.h = 16
		}
	}
	bw, aw := brand.MinSize().Width, actions.MinSize().Width
	top := max(brand.MinSize().Height, actions.MinSize().Height, chips.MinSize().Height)
	if two {
		top = max(brand.MinSize().Height, actions.MinSize().Height)
	}
	brand.Resize(fyne.NewSize(bw, top))
	brand.Move(fyne.NewPos(0, 0))
	actions.Resize(fyne.NewSize(aw, top))
	actions.Move(fyne.NewPos(size.Width-aw, 0))
	cw := chips.MinSize().Width
	fits := bw+cw+aw+2*l.gap <= size.Width
	if fits != chips.Visible() {
		if fits {
			chips.Show()
		} else {
			chips.Hide()
		}
	}
	ch := chips.MinSize().Height
	chips.Resize(fyne.NewSize(cw, ch))
	chips.Move(fyne.NewPos(size.Width-aw-l.gap-cw, (top-ch)/2))

	sh := step.MinSize().Height
	if two {
		sw := min(size.Width, 560)
		step.Resize(fyne.NewSize(sw, sh))
		step.Move(fyne.NewPos((size.Width-sw)/2, top+8))
		return
	}
	from, to := bw+l.gap, size.Width-aw-cw-2*l.gap
	sw := min(stepWidth, to-from)
	x := max(from, min((size.Width-sw)/2, to-sw))
	step.Resize(fyne.NewSize(sw, sh))
	step.Move(fyne.NewPos(x, (size.Height-sh)/2))
}

// colsUpTo colonnes possibles d'une grille de n boutons, de n à 2
func colsUpTo(n int) []int {
	var out []int
	for c := n; c >= 2; c-- {
		out = append(out, c)
	}
	if len(out) == 0 {
		out = []int{1}
	}
	return out
}

// rcaption intitulé en capitales qui passe à la ligne dans la démo
func rcaption(s string) *tText {
	t := caption(s)
	t.Wrap = responsive
	return t
}

// resultHeadRow pastille du résultat et titre ; dans la démo, le titre passe
// à la ligne à côté de la pastille
func resultHeadRow(pill, head fyne.CanvasObject) fyne.CanvasObject {
	if responsive {
		return container.NewBorder(nil, nil, container.NewCenter(pill), nil, head)
	}
	return container.NewHBox(container.NewCenter(pill), head)
}

// touchKeyboardHelp test clavier sur écran tactile, sans clavier : toucher une
// touche du plan l'appuie, ou toutes s'appuient l'une après l'autre
func touchKeyboardHelp(m *keyboardModel, press func(int, string), all func() []int) fyne.CanvasObject {
	var btn *widget.Button
	btn = secondaryButton("Appuyer sur toutes les touches", buttonIcon("keyboard"), func() {
		btn.Disable()
		// une rangée après l'autre, à la cadence d'une frappe rapide ; sans
		// attendre chaque image (une page lente ralentirait tout)
		var todo []int
		for _, code := range all() {
			if m.present[code] && !m.pressed[code] {
				todo = append(todo, code)
			}
		}
		go func() {
			for i := 0; i < len(todo); i += 3 {
				batch := todo[i:min(i+3, len(todo))]
				fyne.Do(func() {
					for _, code := range batch {
						press(code, "")
					}
				})
				time.Sleep(40 * time.Millisecond)
			}
		}()
	})
	return container.NewVBox(
		mutedNote("Sans clavier : touchez chaque touche du plan, ou laissez la démo les appuyer toutes. Pavé numérique et touches multimédia apparaissent au premier appui."),
		container.NewHBox(btn),
	)
}

// critLine critère du verdict du stress test : libellé puis valeur, qui passe
// sous le libellé quand la largeur manque (démo)
func critLine(label, value string) fyne.CanvasObject {
	return container.New(&pairLayout{gap: 4}, text(label, 13, false, cText), rtext(value, 13, true, cText))
}

// pairLayout libellé puis texte sur la même ligne tant que le texte y tient
// en entier, sinon le texte passe dessous sur toute la largeur
type pairLayout struct {
	gap     float32
	stacked bool
}

func naturalWidth(o fyne.CanvasObject) float32 {
	if t, ok := o.(*tText); ok {
		return fyne.MeasureText(t.Text, t.TextSize, t.style()).Width
	}
	return o.MinSize().Width
}

func (l *pairLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	a, b := objs[0].MinSize(), objs[1].MinSize()
	if l.stacked {
		return fyne.NewSize(max(a.Width, b.Width), a.Height+b.Height)
	}
	return fyne.NewSize(max(a.Width, b.Width), max(a.Height, b.Height))
}

func (l *pairLayout) Layout(objs []fyne.CanvasObject, s fyne.Size) {
	a, b := objs[0], objs[1]
	aw := a.MinSize().Width
	stacked := aw+l.gap+naturalWidth(b) > s.Width
	if stacked != l.stacked {
		l.stacked = stacked
		relayoutLater(a)
	}
	a.Move(fyne.NewPos(0, 0))
	a.Resize(fyne.NewSize(aw, a.MinSize().Height))
	if stacked {
		b.Move(fyne.NewPos(0, a.MinSize().Height))
		b.Resize(fyne.NewSize(s.Width, b.MinSize().Height))
		return
	}
	b.Move(fyne.NewPos(aw+l.gap, 0))
	b.Resize(fyne.NewSize(s.Width-aw-l.gap, b.MinSize().Height))
}

// freqRow critère de fréquence du stress test, sans pastille
func freqRow(thr string) fyne.CanvasObject {
	if responsive {
		return listRow(critLine("Fréquence : ", thr), container.NewHBox())
	}
	return listRow(container.NewHBox(text("Fréquence : ", 13, false, cText), text(thr, 13, true, cText)), container.NewHBox())
}

// capHeight objet à hauteur fixe h, centré verticalement
type capHeight struct{ h float32 }

func (l *capHeight) MinSize(objs []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(0, l.h)
}

func (l *capHeight) Layout(objs []fyne.CanvasObject, s fyne.Size) {
	for _, o := range objs {
		o.Resize(fyne.NewSize(s.Width, l.h))
		o.Move(fyne.NewPos(0, (s.Height-l.h)/2))
	}
}
