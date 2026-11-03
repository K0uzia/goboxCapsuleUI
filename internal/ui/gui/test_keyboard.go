package gui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// Sections du clavier : un portable n'a souvent ni pavé numérique ni bloc de
// navigation complet. Les touches d'une section absente sont grisées et ne
// comptent pas ; un appui sur l'une d'elles prouve qu'elle existe.
const (
	secFn     = "fn"     // Échap, F1 à F12
	secMain   = "main"   // bloc alphanumérique
	secNav    = "nav"    // Impr, Inser, Début, Fin, pages, Suppr
	secArrows = "arrows" // flèches
	secPad    = "pad"    // pavé numérique
)

// keyDef touche identifiée par son code physique X11 (indépendant de la
// disposition), avec ses libellés AZERTY / QWERTY et sa place sur le plan du
// clavier, en largeurs de touche
type keyDef struct {
	Code       int
	Azerty     string
	Qwerty     string
	X, Y, W, H float32
	Sec        string
	LaptopOff  bool // souvent absente d'un portable : grisée au départ
}

// kd touche d'une rangée : code, libellés, largeur
type kd struct {
	code   int
	az, qw string
	w      float32
}

func key(code int, az, qw string, w float32) kd { return kd{code, az, qw, w} }
func same(code int, l string, w float32) kd     { return kd{code, l, l, w} }

// row place des touches côte à côte à partir de x
func row(sec string, y, x float32, keys ...kd) []keyDef {
	out := make([]keyDef, 0, len(keys))
	for _, k := range keys {
		out = append(out, keyDef{Code: k.code, Azerty: k.az, Qwerty: k.qw, X: x, Y: y, W: k.w, H: 1, Sec: sec})
		x += k.w
	}
	return out
}

// Plan ISO 105 touches. Codes X11 = codes evdev + 8.
var keyboardKeys = func() []keyDef {
	var ks []keyDef
	add := func(k ...keyDef) { ks = append(ks, k...) }
	// Échap et touches de fonction
	add(row(secFn, 0, 0, same(9, "Échap", 1))...)
	add(row(secFn, 0, 2, same(67, "F1", 1), same(68, "F2", 1), same(69, "F3", 1), same(70, "F4", 1))...)
	add(row(secFn, 0, 6.5, same(71, "F5", 1), same(72, "F6", 1), same(73, "F7", 1), same(74, "F8", 1))...)
	add(row(secFn, 0, 11, same(75, "F9", 1), same(76, "F10", 1), same(95, "F11", 1), same(96, "F12", 1))...)
	// Bloc principal
	add(row(secMain, 1.25, 0, key(49, "²", "`", 1), key(10, "&", "1", 1), key(11, "é", "2", 1), key(12, "\"", "3", 1), key(13, "'", "4", 1), key(14, "(", "5", 1), key(15, "-", "6", 1), key(16, "è", "7", 1), key(17, "_", "8", 1), key(18, "ç", "9", 1), key(19, "à", "0", 1), key(20, ")", "-", 1), key(21, "=", "=", 1), same(22, "⌫", 2))...)
	add(row(secMain, 2.25, 0, same(23, "⇥", 1.5), key(24, "A", "Q", 1), key(25, "Z", "W", 1), same(26, "E", 1), same(27, "R", 1), same(28, "T", 1), same(29, "Y", 1), same(30, "U", 1), same(31, "I", 1), same(32, "O", 1), same(33, "P", 1), key(34, "^", "[", 1), key(35, "$", "]", 1))...)
	add(keyDef{Code: 36, Azerty: "⏎", Qwerty: "⏎", X: 13.5, Y: 2.25, W: 1.5, H: 2, Sec: secMain})
	add(row(secMain, 3.25, 0, same(66, "⇪", 1.5), key(38, "Q", "A", 1), same(39, "S", 1), same(40, "D", 1), same(41, "F", 1), same(42, "G", 1), same(43, "H", 1), same(44, "J", 1), same(45, "K", 1), same(46, "L", 1), key(47, "M", ";", 1), key(48, "ù", "'", 1), key(51, "*", "\\", 1))...)
	add(row(secMain, 4.25, 0, same(50, "⇧", 1.25), same(94, "<", 1), key(52, "W", "Z", 1), same(53, "X", 1), same(54, "C", 1), same(55, "V", 1), same(56, "B", 1), same(57, "N", 1), key(58, ",", "M", 1), key(59, ";", ",", 1), key(60, ":", ".", 1), key(61, "!", "/", 1), same(62, "⇧", 2.75))...)
	add(row(secMain, 5.25, 0, same(37, "Ctrl", 1.5), same(133, "⊞", 1.25), same(64, "Alt", 1.25), key(65, "Espace", "Space", 6), same(108, "AltGr", 1.25), same(135, "Menu", 1.25), same(105, "Ctrl", 2.5))...)
	// Navigation et flèches
	add(row(secNav, 0, 15.5, same(107, "Impr", 1), same(78, "Défil", 1), same(127, "Pause", 1))...)
	add(row(secNav, 1.25, 15.5, same(118, "Inser", 1), same(110, "Début", 1), same(112, "Pg↑", 1))...)
	add(row(secNav, 2.25, 15.5, same(119, "Suppr", 1), same(115, "Fin", 1), same(117, "Pg↓", 1))...)
	add(row(secArrows, 4.25, 16.5, same(111, "↑", 1))...)
	add(row(secArrows, 5.25, 15.5, same(113, "←", 1), same(116, "↓", 1), same(114, "→", 1))...)
	// Pavé numérique
	add(row(secPad, 1.25, 19, same(77, "Num", 1), same(106, "/", 1), same(63, "*", 1), same(82, "-", 1))...)
	add(row(secPad, 2.25, 19, same(79, "7", 1), same(80, "8", 1), same(81, "9", 1))...)
	add(keyDef{Code: 86, Azerty: "+", Qwerty: "+", X: 22, Y: 2.25, W: 1, H: 2, Sec: secPad})
	add(row(secPad, 3.25, 19, same(83, "4", 1), same(84, "5", 1), same(85, "6", 1))...)
	add(row(secPad, 4.25, 19, same(87, "1", 1), same(88, "2", 1), same(89, "3", 1))...)
	add(keyDef{Code: 104, Azerty: "⏎", Qwerty: "⏎", X: 22, Y: 4.25, W: 1, H: 2, Sec: secPad})
	add(row(secPad, 5.25, 19, same(90, "0", 2), same(91, ".", 1))...)

	// Sur un portable, seules Suppr et les flèches sont presque toujours là
	for i := range ks {
		switch ks[i].Code {
		case 135, 107, 78, 127, 118, 110, 112, 115, 117:
			ks[i].LaptopOff = true
		}
		if ks[i].Sec == secPad {
			ks[i].LaptopOff = true
		}
	}
	return ks
}()

const (
	keyboardCols = 23   // largeur du plan, en touches
	keyboardRows = 6.25 // hauteur du plan, en touches
)

// isLaptop vrai sur un portable (type de châssis du BIOS, sinon une batterie)
func isLaptop(dmiRoot, powerRoot string) bool {
	switch readTrim(filepath.Join(dmiRoot, "chassis_type")) {
	case "8", "9", "10", "11", "14", "30", "31", "32":
		return true
	case "3", "4", "5", "6", "7", "13", "15", "16", "17", "23", "24", "35", "36":
		return false
	}
	return hasBattery(powerRoot)
}

// keyboardModel état des touches : présentes sur ce clavier, appuyées
type keyboardModel struct {
	present map[int]bool
	pressed map[int]bool
	sec     map[int]string
}

func newKeyboardModel(laptop bool) *keyboardModel {
	m := &keyboardModel{present: map[int]bool{}, pressed: map[int]bool{}, sec: map[int]string{}}
	for _, k := range keyboardKeys {
		m.present[k.Code] = !(laptop && k.LaptopOff)
		m.sec[k.Code] = k.Sec
	}
	return m
}

// press enregistre un appui ; rend les touches dont l'état a changé. Une touche
// grisée appuyée existe donc ; une touche du pavé numérique fait apparaître tout
// le pavé.
func (m *keyboardModel) press(code int) []int {
	sec, ok := m.sec[code]
	if !ok || m.pressed[code] {
		return nil
	}
	changed := []int{code}
	if !m.present[code] && sec == secPad {
		changed = append(changed, m.setSection(secPad, true)...)
	}
	m.present[code] = true
	m.pressed[code] = true
	return changed
}

// toggle retire ou remet une touche non encore appuyée
func (m *keyboardModel) toggle(code int) bool {
	if m.pressed[code] {
		return false
	}
	m.present[code] = !m.present[code]
	return true
}

// setSection présente ou retire toute une section (les touches appuyées restent)
func (m *keyboardModel) setSection(sec string, on bool) []int {
	var changed []int
	for code, s := range m.sec {
		if s == sec && !m.pressed[code] && m.present[code] != on {
			m.present[code] = on
			changed = append(changed, code)
		}
	}
	return changed
}

func (m *keyboardModel) hasSection(sec string) bool {
	for code, s := range m.sec {
		if s == sec && m.present[code] {
			return true
		}
	}
	return false
}

// counts touches appuyées et touches présentes
func (m *keyboardModel) counts() (done, total int) {
	for code, p := range m.present {
		if p {
			total++
			if m.pressed[code] {
				done++
			}
		}
	}
	return done, total
}

// kind description du clavier pour la fiche
func (m *keyboardModel) kind() string {
	switch {
	case m.hasSection(secPad) && m.hasSection(secFn):
		return "clavier complet"
	case m.hasSection(secFn):
		return "sans pavé numérique"
	case m.hasSection(secPad):
		return "sans touches F"
	}
	return "compact"
}

// keyCapW touche du plan : cliquer la retire ou la remet (clavier sans elle)
type keyCapW struct {
	widget.BaseWidget
	def   keyDef
	bg    *tRect
	label *tText
	onTap func()
}

func newKeyCap(d keyDef, onTap func()) *keyCapW {
	c := &keyCapW{def: d, bg: newRect(cSurface2, cLineStrong, 5), onTap: onTap}
	c.label = text("", 11, true, cText)
	c.label.Alignment = fyne.TextAlignCenter
	c.ExtendBaseWidget(c)
	return c
}

func (c *keyCapW) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(c.bg, container.NewCenter(c.label)))
}
func (c *keyCapW) Tapped(*fyne.PointEvent) { c.onTap() }
func (c *keyCapW) Cursor() desktop.Cursor  { return desktop.PointerCursor }

// paint couleurs selon l'état : appuyée, à tester, absente de ce clavier
func (c *keyCapW) paint(present, pressed, azerty bool) {
	c.label.Text = c.def.Qwerty
	if azerty {
		c.label.Text = c.def.Azerty
	}
	// libellés longs (Inser, Pause) en plus petit sur une touche simple
	c.label.TextSize = 11
	if utf8.RuneCountInString(c.label.Text) > 3 && c.def.W < 1.5 {
		c.label.TextSize = 8.5
	}
	switch {
	case pressed:
		c.bg.setRoles(cSuccess, cSuccess)
		c.label.Role = cAccentText
	case present:
		c.bg.setRoles(cSurface2, cLineStrong)
		c.label.Role = cText
	default:
		c.bg.setRoles("", cLine)
		c.label.Role = cLine
	}
	c.label.Refresh()
}

// keyboardLayout place chaque touche sur le plan, mis à l'échelle de la largeur
type keyboardLayout struct{ keys []keyDef }

const keyGap = 3

func (l *keyboardLayout) unit(w float32) float32 { return min(w/keyboardCols, 44) }

func (l *keyboardLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(keyboardCols*26, keyboardRows*28)
}

func (l *keyboardLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	u := l.unit(size.Width)
	uy := min(u, size.Height/keyboardRows)
	x0 := (size.Width - u*keyboardCols) / 2
	for i, o := range objs {
		k := l.keys[i]
		o.Move(fyne.NewPos(x0+k.X*u, k.Y*uy))
		o.Resize(fyne.NewSize(k.W*u-keyGap, k.H*uy-keyGap))
	}
}

func runKeyboard(t *testCtx) result {
	t.progress(-1)
	azerty := detectAzerty()
	laptop := isLaptop("/sys/class/dmi/id", sysPower)
	if simulated {
		laptop = true
	}
	m := newKeyboardModel(laptop)
	t.log("Disposition détectée : %s", map[bool]string{true: "AZERTY (fr)", false: "QWERTY"}[azerty])
	if laptop {
		t.log("Portable : pavé numérique et touches de navigation rares grisés, un appui les fait apparaître")
	} else {
		t.log("Poste fixe : clavier complet attendu")
	}

	caps := map[int]*keyCapW{}
	var counter, kindText *tText
	var padCheck *widget.Check
	var trap *keyTrap

	repaint := func(codes ...int) {
		for _, code := range codes {
			if c := caps[code]; c != nil {
				c.paint(m.present[code], m.pressed[code], azerty)
			}
		}
		done, total := m.counts()
		counter.SetText(fmt.Sprintf("%d / %d touches", done, total))
		kindText.SetText(strings.ToUpper(m.kind()[:1]) + m.kind()[1:])
		if padCheck != nil && padCheck.Checked != m.hasSection(secPad) {
			padCheck.SetChecked(m.hasSection(secPad))
		}
	}
	all := func() []int {
		codes := make([]int, 0, len(keyboardKeys))
		for _, k := range keyboardKeys {
			codes = append(codes, k.Code)
		}
		return codes
	}
	press := func(code int) { repaint(m.press(code)...) }

	t.panel(func() fyne.CanvasObject {
		objs := make([]fyne.CanvasObject, len(keyboardKeys))
		for i, d := range keyboardKeys {
			c := newKeyCap(d, func() {
				if m.toggle(d.Code) {
					repaint(d.Code)
				}
			})
			caps[d.Code] = c
			objs[i] = c
		}
		plan := container.New(&keyboardLayout{keys: keyboardKeys}, objs...)
		counter = text("", 13, true, cText)
		kindText = text("", 12, false, cMuted)
		padCheck = widget.NewCheck("Pavé numérique", func(on bool) {
			if on != m.hasSection(secPad) {
				repaint(m.setSection(secPad, on)...)
			}
		})
		repaint(all()...)
		toggle := secondaryButton("AZERTY / QWERTY", nil, func() { azerty = !azerty; repaint(all()...) })
		trap = newKeyTrap(t.win.Canvas(), func(ev *fyne.KeyEvent) { press(ev.Physical.ScanCode) })
		head := container.NewBorder(nil, nil, container.NewHBox(counter, container.NewCenter(kindText)), container.NewHBox(padCheck, toggle))
		hint := mutedNote("Touche grisée : absente de ce clavier, elle ne compte pas. Cliquer une touche pour la retirer ou la remettre. Touches F qui règlent le son ou la luminosité : les tester en maintenant Fn.")
		return container.NewVBox(head, plan, hint, trap)
	})

	// Le piège garde le focus pour que Tab ne le déplace pas sur un bouton, et
	// reçoit les touches : Fyne n'appelle SetOnKeyDown du canvas que si aucun
	// widget n'a le focus
	t.ui(func() { t.win.Canvas().Focus(trap) })
	defer t.ui(func() {
		trap.active = false
		if t.win.Canvas().Focused() == trap {
			t.win.Canvas().Unfocus()
		}
	})

	// La touche Windows ouvre le menu du bureau avant d'arriver à gobox :
	// raccourci du bureau suspendu le temps du test
	restoreSuper := suspendSuperKey(t)
	defer restoreSuper()

	// Le clavier entier est capturé : les raccourcis du bureau (menu de Cinnamon)
	// ne sortent plus du test
	if !simulated && manageSuperKey {
		release, ok := grabKeyboard(func(code int) { fyne.Do(func() { press(code) }) })
		defer release()
		if ok {
			t.log("Clavier capturé pendant le test : terminez à la souris")
		} else {
			t.log("Capture du clavier impossible : un raccourci du bureau peut sortir du test")
		}
	}

	if t.choose("Quand toutes les touches ont été testées :", choice{"Terminer le test clavier", statusOK}) == statusIdle {
		return result{}
	}

	var n, total int
	var kind string
	var missing []string
	fyne.DoAndWait(func() {
		n, total = m.counts()
		kind = m.kind()
		for _, k := range keyboardKeys {
			if m.present[k.Code] && !m.pressed[k.Code] {
				missing = append(missing, map[bool]string{true: k.Azerty, false: k.Qwerty}[azerty])
			}
		}
	})
	layout := map[bool]string{true: "AZERTY", false: "QWERTY"}[azerty]
	data := map[string]string{"kind": kind, "layout": layout}
	if len(missing) > 0 {
		t.log("Touches non détectées : %s", strings.Join(missing, " "))
	}
	switch {
	case n >= total:
		return result{Status: statusOK, Detail: fmt.Sprintf("%d / %d touches · %s · %s", n, total, layout, kind), Data: data}
	case float64(n) >= 0.8*float64(total):
		return result{Status: statusWarn, Detail: fmt.Sprintf("%d / %d touches · non détectées : %s", n, total, strings.Join(missing, " ")), Data: data}
	}
	return result{Status: statusFail, Detail: fmt.Sprintf("%d / %d touches détectées · %s", n, total, kind), Data: data}
}

// detectAzerty lit la disposition système : Debian / Mint
// (/etc/default/keyboard), Rocky / Fedora (xorg.conf.d, vconsole.conf)
func detectAzerty() bool {
	for _, f := range []string{"/etc/default/keyboard", "/etc/X11/xorg.conf.d/00-keyboard.conf", "/etc/vconsole.conf"} {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		c := strings.ToLower(string(b))
		for _, line := range strings.Split(c, "\n") {
			if strings.Contains(line, "xkblayout") || strings.Contains(line, "keymap") {
				return strings.Contains(line, `"fr`) || strings.Contains(line, `"be`)
			}
		}
	}
	return true
}

// keyTrap élément invisible qui garde le focus pendant le test clavier : il
// reçoit chaque touche enfoncée, accepte Tab (sinon Fyne passe au bouton
// suivant) et reprend le focus si un clic le lui retire
type keyTrap struct {
	widget.BaseWidget
	canvas fyne.Canvas
	active bool
	onKey  func(*fyne.KeyEvent)
}

var _ desktop.Keyable = (*keyTrap)(nil)

func newKeyTrap(c fyne.Canvas, onKey func(*fyne.KeyEvent)) *keyTrap {
	k := &keyTrap{canvas: c, active: true, onKey: onKey}
	k.ExtendBaseWidget(k)
	return k
}

func (k *keyTrap) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewWithoutLayout())
}
func (k *keyTrap) AcceptsTab() bool          { return true }
func (k *keyTrap) FocusGained()              {}
func (k *keyTrap) TypedRune(rune)            {}
func (k *keyTrap) TypedKey(*fyne.KeyEvent)   {}
func (k *keyTrap) KeyDown(ev *fyne.KeyEvent) { k.onKey(ev) }
func (k *keyTrap) KeyUp(*fyne.KeyEvent)      {}
func (k *keyTrap) FocusLost() {
	if k.active {
		go fyne.Do(func() {
			if k.active {
				k.canvas.Focus(k)
			}
		})
	}
}

// superKeySettings réglages du bureau qui donnent la touche Windows au menu
var superKeySettings = [][2]string{
	{"org.gnome.mutter", "overlay-key"},    // GNOME (Rocky)
	{"org.cinnamon.muffin", "overlay-key"}, // Cinnamon (Mint)
}

// manageSuperKey faux dans les tests automatiques : ne touche ni au bureau ni
// au clavier
var manageSuperKey = true

var (
	superMu      sync.Mutex
	superRestore []func()
)

// suspendSuperKey retire la touche Windows au bureau ; la fonction rendue la
// lui redonne (aussi appelée à la fermeture de gobox, voir restoreSuperKey)
func suspendSuperKey(t *testCtx) func() {
	if simulated || !manageSuperKey {
		return func() {}
	}
	var undo []func()
	for _, kv := range superKeySettings {
		out, err := userCommand(t.ctx, "gsettings", "get", kv[0], kv[1]).Output()
		if err != nil {
			continue // réglage absent sur ce bureau
		}
		old := strings.Trim(strings.TrimSpace(string(out)), "'")
		if old == "" {
			continue
		}
		if err := userCommand(context.Background(), "gsettings", "set", kv[0], kv[1], "").Run(); err != nil {
			t.log("Touche Windows : réglage %s non modifiable (%v)", kv[0], err)
			continue
		}
		t.log("Touche Windows retirée au menu du bureau pendant le test")
		undo = append(undo, func() { _ = userCommand(context.Background(), "gsettings", "set", kv[0], kv[1], old).Run() })
	}
	superMu.Lock()
	superRestore = append(superRestore, undo...)
	superMu.Unlock()
	return restoreSuperKey
}

// restoreSuperKey redonne la touche Windows au bureau
func restoreSuperKey() {
	superMu.Lock()
	undo := superRestore
	superRestore = nil
	superMu.Unlock()
	for _, f := range undo {
		f()
	}
}
