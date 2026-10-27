package gui

import (
	"fmt"
	"image/color"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
)

// keyDef touche identifiée par son code physique X11 (indépendant de la
// disposition), avec ses libellés AZERTY / QWERTY et sa largeur relative.
type keyDef struct {
	Code   int
	Azerty string
	Qwerty string
	Flex   float32
}

func k(code int, az, qw string, flex float32) keyDef { return keyDef{code, az, qw, flex} }

// Codes X11 = codes evdev + 8
var keyboardRows = [][]keyDef{
	{k(49, "²", "`", 1), k(10, "&", "1", 1), k(11, "é", "2", 1), k(12, "\"", "3", 1), k(13, "'", "4", 1), k(14, "(", "5", 1), k(15, "-", "6", 1), k(16, "è", "7", 1), k(17, "_", "8", 1), k(18, "ç", "9", 1), k(19, "à", "0", 1), k(20, ")", "-", 1), k(21, "=", "=", 1), k(22, "⌫", "⌫", 2)},
	{k(23, "⇥", "⇥", 1.5), k(24, "A", "Q", 1), k(25, "Z", "W", 1), k(26, "E", "E", 1), k(27, "R", "R", 1), k(28, "T", "T", 1), k(29, "Y", "Y", 1), k(30, "U", "U", 1), k(31, "I", "I", 1), k(32, "O", "O", 1), k(33, "P", "P", 1), k(34, "^", "[", 1), k(35, "$", "]", 1), k(36, "⏎", "⏎", 1.5)},
	{k(66, "⇪", "⇪", 1.8), k(38, "Q", "A", 1), k(39, "S", "S", 1), k(40, "D", "D", 1), k(41, "F", "F", 1), k(42, "G", "G", 1), k(43, "H", "H", 1), k(44, "J", "J", 1), k(45, "K", "K", 1), k(46, "L", "L", 1), k(47, "M", ";", 1), k(48, "ù", "'", 1), k(51, "*", "\\", 1.2)},
	{k(50, "⇧", "⇧", 1.3), k(94, "<", "<", 1), k(52, "W", "Z", 1), k(53, "X", "X", 1), k(54, "C", "C", 1), k(55, "V", "V", 1), k(56, "B", "B", 1), k(57, "N", "N", 1), k(58, ",", "M", 1), k(59, ";", ",", 1), k(60, ":", ".", 1), k(61, "!", "/", 1), k(62, "⇧", "⇧", 2.2)},
	{k(37, "Ctrl", "Ctrl", 1.4), k(133, "⊞", "⊞", 1.2), k(64, "Alt", "Alt", 1.2), k(65, "Espace", "Space", 6), k(108, "AltGr", "AltGr", 1.2), k(105, "Ctrl", "Ctrl", 1.4)},
}

type keyCap struct {
	def   keyDef
	bg    *canvas.Rectangle
	label *canvas.Text
}

func runKeyboard(t *testCtx) result {
	t.progress(-1)
	azerty := detectAzerty()
	t.log("Disposition détectée : %s", map[bool]string{true: "AZERTY (fr)", false: "QWERTY"}[azerty])
	t.log("Appuyez sur chaque touche du clavier à tester.")

	pressed := map[int]bool{}
	caps := map[int]*keyCap{}
	total := 0
	var counter *canvas.Text

	relabel := func() {
		for _, c := range caps {
			c.label.Text = c.def.Qwerty
			if azerty {
				c.label.Text = c.def.Azerty
			}
			c.label.Refresh()
		}
	}

	t.panel(func() fyne.CanvasObject {
		rows := container.NewVBox()
		for _, row := range keyboardRows {
			objs := make([]fyne.CanvasObject, len(row))
			flex := make([]float32, len(row))
			for i, d := range row {
				bg := canvas.NewRectangle(color.White)
				bg.StrokeColor, bg.StrokeWidth, bg.CornerRadius = colLine, 1, 4
				bg.SetMinSize(fyne.NewSize(0, 30))
				lbl := text("", 11, true, colInk)
				lbl.Alignment = fyne.TextAlignCenter
				caps[d.Code] = &keyCap{def: d, bg: bg, label: lbl}
				objs[i] = container.NewStack(bg, container.NewCenter(lbl))
				flex[i] = d.Flex
				total++
			}
			rows.Add(container.New(&flexRowLayout{flex: flex, gap: 4}, objs...))
		}
		relabel()
		counter = text(fmt.Sprintf("0 / %d touches", total), 13, true, colInk)
		toggle := outlineButton("AZERTY / QWERTY", func() { azerty = !azerty; relabel() })
		return container.NewVBox(container.NewBorder(nil, nil, counter, toggle), rows)
	})

	// Capture des touches sur toute la fenêtre
	t.ui(func() {
		t.win.Canvas().Unfocus()
		if dc, ok := t.win.Canvas().(desktop.Canvas); ok {
			dc.SetOnKeyDown(func(ev *fyne.KeyEvent) {
				c, ok := caps[ev.Physical.ScanCode]
				if !ok || pressed[c.def.Code] {
					return
				}
				pressed[c.def.Code] = true
				c.bg.FillColor, c.bg.StrokeColor, c.label.Color = colTeal, colTeal, color.White
				c.bg.Refresh()
				c.label.Refresh()
				counter.Text = fmt.Sprintf("%d / %d touches", len(pressed), total)
				counter.Refresh()
			})
		}
	})
	defer t.ui(func() {
		if dc, ok := t.win.Canvas().(desktop.Canvas); ok {
			dc.SetOnKeyDown(nil)
		}
	})

	if t.choose("Quand toutes les touches ont été testées :", choice{"Terminer le test clavier", statusOK}) == statusIdle {
		return result{}
	}

	var n int
	fyne.DoAndWait(func() { n = len(pressed) })
	layout := map[bool]string{true: "AZERTY", false: "QWERTY"}[azerty]
	switch {
	case n >= total:
		return result{Status: statusOK, Detail: fmt.Sprintf("%d / %d touches · %s", n, total, layout)}
	case float64(n) >= 0.8*float64(total):
		return result{Status: statusWarn, Detail: fmt.Sprintf("%d touche(s) non détectée(s) sur %d", total-n, total)}
	}
	return result{Status: statusFail, Detail: fmt.Sprintf("%d / %d touches détectées", n, total)}
}

// detectAzerty lit la disposition système (Debian / Mint)
func detectAzerty() bool {
	b, err := os.ReadFile("/etc/default/keyboard")
	if err != nil {
		return true
	}
	return strings.Contains(string(b), `XKBLAYOUT="fr`) || strings.Contains(string(b), `XKBLAYOUT="be`)
}

// flexRowLayout répartit la largeur selon des poids (touches larges)
type flexRowLayout struct {
	flex []float32
	gap  float32
}

func (l *flexRowLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	h := float32(0)
	for _, o := range objs {
		h = max(h, o.MinSize().Height)
	}
	return fyne.NewSize(float32(len(objs))*24, h)
}

func (l *flexRowLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	var sum float32
	for _, f := range l.flex {
		sum += f
	}
	unit := (size.Width - l.gap*float32(len(objs)-1)) / sum
	x := float32(0)
	for i, o := range objs {
		w := unit * l.flex[i]
		o.Resize(fyne.NewSize(w, size.Height))
		o.Move(fyne.NewPos(x, 0))
		x += w + l.gap
	}
}
