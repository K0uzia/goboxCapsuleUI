package gui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// Sections du clavier : un portable n'a souvent ni pavé numérique ni bloc de
// navigation complet. Les touches absentes sont grisées et ne comptent pas ; un
// appui sur l'une d'elles prouve qu'elle existe. Le système ne dit pas quelles
// touches sont gravées sur le clavier : le plan part du type de machine, puis
// s'ajuste à chaque appui.
const (
	secFn     = "fn"     // Échap, F1 à F12
	secMain   = "main"   // bloc alphanumérique
	secNav    = "nav"    // Impr, Inser, Début, Fin, pages, Suppr
	secArrows = "arrows" // flèches
	secPad    = "pad"    // pavé numérique
	secMedia  = "media"  // son, lecture, luminosité
)

// keyDef touche physique identifiée par son code X11 (indépendant de la
// disposition), avec ses libellés AZERTY / QWERTY et sa place sur le plan du
// clavier, en largeurs de touche. Une fonction obtenue avec Fn (Impr sur Inser,
// volume sur F2…) envoie son propre code : elle a sa propre touche sur le plan.
type keyDef struct {
	Code       int
	Azerty     string
	Qwerty     string
	X, Y, W, H float32
	Sec        string
	Group      string // un appui sur une touche absente fait apparaître tout son groupe
	Off        bool   // absente tant qu'on n'appuie pas dessus
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

// Touches multimédia (codes X11) : seulement si on appuie dessus
var mediaKeys = []struct {
	code  int
	label string
	group string
}{
	{121, "Muet", "audio"}, {122, "Vol−", "audio"}, {123, "Vol+", "audio"},
	{172, "Lect", ""}, {232, "Lum−", "lum"}, {233, "Lum+", "lum"},
}

// keyboardPlan plan ISO 105 touches et touches multimédia. Le système ne dit pas
// quelles touches sont gravées : pavé numérique et multimédia partent absents et
// apparaissent au premier appui ; sur un portable, les touches de navigation
// rares aussi. Codes X11 = codes evdev + 8.
func keyboardPlan(laptop bool) []keyDef {
	var ks []keyDef
	add := func(k ...keyDef) { ks = append(ks, k...) }
	add(row(secFn, 0, 0, same(9, "Échap", 1))...)
	add(row(secFn, 0, 2, same(67, "F1", 1), same(68, "F2", 1), same(69, "F3", 1), same(70, "F4", 1))...)
	add(row(secFn, 0, 6.5, same(71, "F5", 1), same(72, "F6", 1), same(73, "F7", 1), same(74, "F8", 1))...)
	add(row(secFn, 0, 11, same(75, "F9", 1), same(76, "F10", 1), same(95, "F11", 1), same(96, "F12", 1))...)
	add(row(secMain, 1.25, 0, key(49, "²", "`", 1), key(10, "&", "1", 1), key(11, "é", "2", 1), key(12, "\"", "3", 1), key(13, "'", "4", 1), key(14, "(", "5", 1), key(15, "-", "6", 1), key(16, "è", "7", 1), key(17, "_", "8", 1), key(18, "ç", "9", 1), key(19, "à", "0", 1), key(20, ")", "-", 1), key(21, "=", "=", 1), same(22, "⌫", 2))...)
	add(row(secMain, 2.25, 0, same(23, "⇥", 1.5), key(24, "A", "Q", 1), key(25, "Z", "W", 1), same(26, "E", 1), same(27, "R", 1), same(28, "T", 1), same(29, "Y", 1), same(30, "U", 1), same(31, "I", 1), same(32, "O", 1), same(33, "P", 1), key(34, "^", "[", 1), key(35, "$", "]", 1))...)
	add(keyDef{Code: 36, Azerty: "⏎", Qwerty: "⏎", X: 13.5, Y: 2.25, W: 1.5, H: 2, Sec: secMain})
	add(row(secMain, 3.25, 0, same(66, "⇪", 1.5), key(38, "Q", "A", 1), same(39, "S", 1), same(40, "D", 1), same(41, "F", 1), same(42, "G", 1), same(43, "H", 1), same(44, "J", 1), same(45, "K", 1), same(46, "L", 1), key(47, "M", ";", 1), key(48, "ù", "'", 1), key(51, "*", "\\", 1))...)
	add(row(secMain, 4.25, 0, same(50, "⇧", 1.25), same(94, "<", 1), key(52, "W", "Z", 1), same(53, "X", 1), same(54, "C", 1), same(55, "V", 1), same(56, "B", 1), same(57, "N", 1), key(58, ",", "M", 1), key(59, ";", ",", 1), key(60, ":", ".", 1), key(61, "!", "/", 1), same(62, "⇧", 2.75))...)
	add(row(secMain, 5.25, 0, same(37, "Ctrl", 1.5), same(133, "⊞", 1.25), same(64, "Alt", 1.25), key(65, "Espace", "Space", 6), same(108, "AltGr", 1.25), same(135, "Menu", 1.25), same(105, "Ctrl", 2.5))...)
	add(row(secNav, 0, 15.5, same(107, "Impr", 1), same(78, "Défil", 1), same(127, "Pause", 1))...)
	add(row(secNav, 1.25, 15.5, same(118, "Inser", 1), same(110, "Début", 1), same(112, "Pg↑", 1))...)
	add(row(secNav, 2.25, 15.5, same(119, "Suppr", 1), same(115, "Fin", 1), same(117, "Pg↓", 1))...)
	add(row(secArrows, 4.25, 16.5, same(111, "↑", 1))...)
	add(row(secArrows, 5.25, 15.5, same(113, "←", 1), same(116, "↓", 1), same(114, "→", 1))...)
	add(row(secPad, 1.25, 19, same(77, "Num", 1), same(106, "/", 1), same(63, "*", 1), same(82, "-", 1))...)
	add(row(secPad, 2.25, 19, same(79, "7", 1), same(80, "8", 1), same(81, "9", 1))...)
	add(keyDef{Code: 86, Azerty: "+", Qwerty: "+", X: 22, Y: 2.25, W: 1, H: 2, Sec: secPad})
	add(row(secPad, 3.25, 19, same(83, "4", 1), same(84, "5", 1), same(85, "6", 1))...)
	add(row(secPad, 4.25, 19, same(87, "1", 1), same(88, "2", 1), same(89, "3", 1))...)
	add(keyDef{Code: 104, Azerty: "⏎", Qwerty: "⏎", X: 22, Y: 4.25, W: 1, H: 2, Sec: secPad})
	add(row(secPad, 5.25, 19, same(90, "0", 2), same(91, ".", 1))...)
	// son et lecture au-dessus du pavé ; luminosité (portable) sous la navigation
	x := float32(19)
	for _, m := range mediaKeys {
		k := keyDef{Code: m.code, Azerty: m.label, Qwerty: m.label, X: x, Y: 0, W: 1, H: 1, Sec: secMedia, Group: m.group, Off: true}
		if m.group == "lum" {
			if !laptop {
				continue // la luminosité ne se règle pas depuis un clavier de bureau
			}
			k.X, k.Y = 16+float32(m.code-232), 3.25
		} else {
			x++
		}
		add(k)
	}
	for i := range ks {
		switch {
		case ks[i].Sec == secPad:
			ks[i].Group, ks[i].Off = "pad", true
		case laptop && ks[i].Sec == secNav && ks[i].Code != 119:
			ks[i].Off = true // seule Suppr est presque toujours sur un portable
		case laptop && ks[i].Code == 135:
			ks[i].Off = true
		}
	}
	return ks
}

const (
	keyboardCols = 23   // largeur du plan, en touches
	keyboardRows = 6.25 // hauteur du plan, en touches
)

const sysInput = "/sys/class/input"

// isLaptop vrai sur un portable : type de châssis du BIOS, sinon un capot
// (interrupteur « Lid Switch ») ou une batterie
func isLaptop(dmiRoot, powerRoot string) bool {
	switch readTrim(filepath.Join(dmiRoot, "chassis_type")) {
	case "8", "9", "10", "11", "14", "30", "31", "32":
		return true
	case "3", "4", "5", "6", "7", "13", "15", "16", "17", "23", "24", "35", "36":
		return false
	}
	return hasLid(sysInput) || hasBattery(powerRoot)
}

// hasLid vrai si le noyau voit un capot : seul un portable en a un
func hasLid(root string) bool {
	if lids, _ := filepath.Glob("/proc/acpi/button/lid/*"); len(lids) > 0 {
		return true
	}
	names, _ := filepath.Glob(filepath.Join(root, "input*", "name"))
	for _, n := range names {
		if strings.EqualFold(readTrim(n), "Lid Switch") {
			return true
		}
	}
	return false
}

// inputDevice clavier vu par le noyau
type inputDevice struct{ Name, Phys string }

// inputKeyboards périphériques qui ont des touches de lettres (Q et Espace)
func inputKeyboards(root string) []inputDevice {
	var out []inputDevice
	dirs, _ := filepath.Glob(filepath.Join(root, "input*"))
	for _, d := range dirs {
		if hasKeyBits(readTrim(filepath.Join(d, "capabilities", "key")), 16, 57) {
			out = append(out, inputDevice{readTrim(filepath.Join(d, "name")), readTrim(filepath.Join(d, "phys"))})
		}
	}
	return out
}

// hasKeyBits vrai si le masque de touches du noyau (mots hexadécimaux de 64 bits,
// le dernier pour les codes les plus bas) contient tous les codes evdev donnés
func hasKeyBits(mask string, codes ...int) bool {
	words := strings.Fields(mask)
	for _, c := range codes {
		i := len(words) - 1 - c/64
		if i < 0 {
			return false
		}
		w, err := strconv.ParseUint(words[i], 16, 64)
		if err != nil || w&(1<<(c%64)) == 0 {
			return false
		}
	}
	return len(words) > 0
}

// keyboardModel état des touches : présentes sur ce clavier, appuyées
type keyboardModel struct {
	plan    []keyDef
	present map[int]bool
	pressed map[int]bool
	group   map[int]string
	sec     map[int]string
}

func newKeyboardModel(plan []keyDef) *keyboardModel {
	m := &keyboardModel{plan: plan, present: map[int]bool{}, pressed: map[int]bool{}, group: map[int]string{}, sec: map[int]string{}}
	for _, k := range plan {
		m.present[k.Code] = !k.Off
		m.sec[k.Code] = k.Sec
		m.group[k.Code] = k.Group
	}
	return m
}

// known vrai si le code correspond à une touche du plan
func (m *keyboardModel) known(code int) bool { _, ok := m.sec[code]; return ok }

// press enregistre un appui ; rend les touches dont l'état a changé. Une touche
// grisée appuyée existe donc, et fait apparaître son groupe (pavé, son).
func (m *keyboardModel) press(k int) []int {
	if !m.known(k) || m.pressed[k] {
		return nil
	}
	changed := []int{k}
	if g := m.group[k]; !m.present[k] && g != "" {
		for c, cg := range m.group {
			if cg == g && c != k && !m.present[c] && !m.pressed[c] {
				m.present[c] = true
				changed = append(changed, c)
			}
		}
	}
	m.present[k] = true
	m.pressed[k] = true
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
	var parts []string
	switch {
	case m.hasSection(secPad) && m.hasSection(secFn):
		parts = append(parts, "clavier complet")
	case m.hasSection(secFn):
		parts = append(parts, "sans pavé numérique")
	case m.hasSection(secPad):
		parts = append(parts, "sans touches F")
	default:
		parts = append(parts, "compact")
	}
	if m.hasSection(secMedia) {
		parts = append(parts, "touches multimédia")
	}
	return strings.Join(parts, ", ")
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
	laptop := isLaptop("/sys/class/dmi/id", sysPower)
	azerty, layoutKnown := keyboardLayoutAzerty()
	if simulated {
		laptop, azerty, layoutKnown = true, true, true
	}
	t.log("Machine : %s", map[bool]string{true: "portable", false: "poste fixe"}[laptop])
	t.log("Disposition : %s", map[bool]string{true: "AZERTY", false: "QWERTY"}[azerty])
	for _, k := range inputKeyboards(sysInput) {
		t.log("Clavier vu par le système : %s", k.Name)
	}

	m := newKeyboardModel(keyboardPlan(laptop))
	unknown := map[int]bool{} // codes hors du plan (touches spéciales du constructeur)
	caps := map[int]*keyCapW{}
	var counter, kindText, padText *tText
	var trap *keyTrap

	repaint := func(codes ...int) {
		for _, code := range codes {
			if c := caps[code]; c != nil {
				c.paint(m.present[code], m.pressed[code], azerty)
			}
		}
		done, total := m.counts()
		counter.SetText(fmt.Sprintf("%d / %d touches", done, total))
		kindText.SetText(fmt.Sprintf("%s · %s · %s", map[bool]string{true: "Portable", false: "Poste fixe"}[laptop],
			map[bool]string{true: "AZERTY", false: "QWERTY"}[azerty], m.kind()))
		if m.hasSection(secPad) {
			padText.SetText("Pavé numérique détecté")
			padText.Role = cSuccess
		} else {
			padText.SetText("Pavé numérique : appuyez sur une de ses touches s'il y en a un")
			padText.Role = cMuted
		}
		padText.Refresh()
	}
	all := func() []int {
		codes := make([]int, 0, len(m.plan))
		for _, k := range m.plan {
			codes = append(codes, k.Code)
		}
		return codes
	}
	// press touche reçue (fenêtre, capture X11 ou noyau) ; name : lettre produite,
	// qui corrige la disposition si elle n'a pas pu être lue
	sources := map[string]bool{} // voies par lesquelles une touche est déjà arrivée
	pressFrom := func(source string, code int, name string, press func(int, string)) {
		if !sources[source] {
			sources[source] = true
			t.log("Première touche reçue par %s (code %d)", source, code)
		}
		press(code, name)
	}
	press := func(code int, name string) {
		if !layoutKnown && code == 24 && name != "" {
			layoutKnown, azerty = true, strings.EqualFold(name, "A")
			repaint(all()...)
		}
		if !m.known(code) {
			if !unknown[code] {
				unknown[code] = true
				t.log("Touche spéciale du constructeur détectée (code %d)", code)
			}
			return
		}
		repaint(m.press(code)...)
	}

	t.panel(func() fyne.CanvasObject {
		counter = text("", 13, true, cText)
		kindText = text("", 12, false, cMuted)
		padText = text("", 12, true, cMuted)
		objs := make([]fyne.CanvasObject, len(m.plan))
		for i, d := range m.plan {
			c := newKeyCap(d, func() {
				if m.toggle(d.Code) {
					repaint(d.Code)
				}
			})
			caps[d.Code] = c
			objs[i] = c
		}
		plan := container.New(&keyboardLayout{keys: m.plan}, objs...)
		repaint(all()...)
		trap = newKeyTrap(t.win.Canvas(), func(ev *fyne.KeyEvent) { pressFrom("la fenêtre", ev.Physical.ScanCode, string(ev.Name), press) })
		head := container.NewVBox(container.NewHBox(counter, container.NewCenter(kindText)), padText)
		hint := mutedNote("Appuyez sur chaque touche, Fn compris : une fonction obtenue avec Fn (Impr, volume, luminosité) s'allume à part. " +
			"Pavé numérique et touches multimédia apparaissent au premier appui. Touche grisée : absente, elle ne compte pas ; un clic la retire ou la remet.")
		return container.NewVBox(head, plan, hint, trap)
	})

	// Lecture dans le noyau : touches de son d'un clavier USB, que le bureau garde
	if !simulated && manageSuperKey {
		stopEvdev, names := readEvdevKeys(sysInput, func(code int) { fyne.Do(func() { pressFrom("le noyau", code, "", press) }) })
		defer stopEvdev()
		if len(names) > 0 {
			t.log("Touches lues directement sur : %s", strings.Join(names, ", "))
		} else {
			t.log("Lecture directe des claviers impossible (droits sur /dev/input ?)")
		}
	}

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
		release, ok := grabKeyboard(func(code int) { fyne.Do(func() { pressFrom("la capture X11", code, "", press) }) })
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
		for _, k := range m.plan {
			if m.present[k.Code] && !m.pressed[k.Code] {
				missing = append(missing, map[bool]string{true: k.Azerty, false: k.Qwerty}[azerty])
			}
		}
	})
	var layout string
	fyne.DoAndWait(func() { layout = map[bool]string{true: "AZERTY", false: "QWERTY"}[azerty] })
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

// keyboardLayoutAzerty disposition active : la lettre que produit la touche à
// droite de Tab, lue au serveur X ; known faux si elle n'a pas pu être lue (la
// première frappe sur cette touche tranche alors)
func keyboardLayoutAzerty() (azerty, known bool) {
	switch strings.ToLower(keySymbol(24)) {
	case "a":
		return true, true
	case "q":
		return false, true
	}
	return detectAzerty(), false
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
