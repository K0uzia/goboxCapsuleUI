package gui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// Parcours complet d'un test : clic sur le bouton, résultat, confirmation,
// puis vérification que la fiche a enregistré le résultat.
func TestToolFlow(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(true))
	w := a.NewWindow("test")

	s := newSession()
	s.Tests = []*testDef{{ID: "fake", Title: "Faux", Category: catPeripherals, Color: cPeripheral, Soft: cPeripheralBg,
		Run: func(tc *testCtx) result {
			tc.log("étape 1")
			return result{Status: statusWarn, Detail: "réserve"}
		}}}
	fiche := newFichePanel(s)
	tools := newToolsPanel(s, w)
	_ = fiche

	test.Tap(tools.buttons["fake"])
	if tools.active == nil || tools.menu.Visible() {
		t.Fatal("le test devrait être affiché à la place du menu")
	}

	deadline := time.Now().Add(2 * time.Second)
	for !tools.view.finished.Visible() {
		if time.Now().After(deadline) {
			t.Fatal("le résultat n'est jamais affiché")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if s.status("fake") != statusIdle {
		t.Fatal("le résultat ne doit pas être enregistré avant confirmation")
	}

	tools.close(true)
	if got := s.status("fake"); got != statusWarn {
		t.Fatalf("statut enregistré = %v, attendu MOYEN", got)
	}
	if !tools.menu.Visible() {
		t.Fatal("le menu devrait être réaffiché")
	}
	if v := s.verdict(); v.Label != "Apte à la vente" {
		t.Fatalf("verdict = %q", v.Label)
	}
}

func TestWAVGeneration(t *testing.T) {
	path, err := writeTone("both")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
}

func TestFichePDF(t *testing.T) {
	s := newSession()
	s.Machine = machineInfo{Vendor: "HP", Product: "HP EliteBook 850 G3", Serial: "7XK2Q9A"}
	s.Inventory = demoInventory()
	s.Results["stress"] = result{Status: statusWarn, Detail: "41 °C au repos, 87 °C au maximum"}
	s.Results["sd"] = result{Status: statusFail, Detail: "Carte non détectée"}
	s.Results["ports"] = result{Status: statusOK, Data: map[string]string{"usb": "3 / 4", "HDMI": "1 / 1", "DisplayPort": "0 / 1"}}
	s.Form.Tech = "Alex"
	data, err := fichePDF(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("ce n'est pas un PDF")
	}
	if n := bytes.Count(data, []byte("/Type /Page\n")); n != 1 {
		t.Fatalf("%d page(s), attendu 1", n)
	}
	if out := os.Getenv("GOBOX_FICHE_OUT"); out != "" {
		if err := os.WriteFile(out, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// Une fiche chargée (4 barrettes, longues observations) reste sur une page : le
// texte se réduit. Au-delà du plus petit texte lisible, elle continue sur une
// deuxième page sans rien perdre.
func TestFichePDFFitsOnePage(t *testing.T) {
	load := func(rows, notes int) []byte {
		t.Helper()
		s := newSession()
		s.Inventory = demoInventory()
		s.Form.Notes = strings.Repeat("Observation longue du technicien sur l'état de la machine. ", notes)
		for i := range s.Inventory {
			for j := 0; j < rows; j++ {
				s.Inventory[i].Fields = append(s.Inventory[i].Fields, Field{"Ligne", "Valeur de test assez longue pour passer sur deux lignes dans la colonne"})
			}
		}
		data, err := fichePDF(s)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	if n := bytes.Count(load(2, 6), []byte("/Type /Page\n")); n != 1 {
		t.Errorf("fiche chargée : %d page(s), attendu 1", n)
	}
	if n := bytes.Count(load(12, 60), []byte("/Type /Page\n")); n < 2 {
		t.Errorf("fiche démesurée : %d page(s), attendu au moins 2", n)
	}
}

// Chaque ligne de la fiche suit la partie du test qui la renseigne.
func TestItemState(t *testing.T) {
	s := newSession()
	s.Results["son"] = result{Status: statusFail, Data: map[string]string{
		"speakers": statusData(statusOK), "jack": statusData(statusNA), "mic_int": statusData(statusFail)}}
	cases := []struct {
		key  string
		want status
	}{{"speakers", statusOK}, {"jack", statusNA}, {"mic_int", statusFail}, {"card", statusFail}}
	for _, c := range cases {
		if got := s.itemState(ctrlItem{TestID: "son", Key: c.key}); got != c.want {
			t.Errorf("son/%s : %s, attendu %s", c.key, got.style().label, c.want.style().label)
		}
	}
}

// findRow ligne d'une section de la fiche
func findRow(t *testing.T, s *session, section, label string) ficheRow {
	t.Helper()
	for _, sec := range s.ficheSections() {
		for _, r := range sec.Rows {
			if sec.Name == section && r.Label == label {
				return r
			}
		}
	}
	t.Fatalf("ligne %q absente de la section %q", label, section)
	return ficheRow{}
}

// Le résultat d'un test est écrit avec le composant qu'il concerne, en face de
// chaque prise vidéo.
func TestFicheSectionsMergeResults(t *testing.T) {
	s := newSession()
	s.Results["stress"] = result{Status: statusOK, Detail: "42 °C au repos, 63 °C au maximum"}
	s.Results["ports"] = result{Status: statusOK, Data: map[string]string{"usb": "2 / 4", "HDMI": "1 / 2"}}
	s.Form.Paste = "oui"

	if r := findRow(t, s, "Processeur", "Stress test"); r.Status != statusOK || r.Value == "" {
		t.Errorf("stress : %+v", r)
	}
	if r := findRow(t, s, "Processeur", "Pâte thermique changée"); r.Value != "oui" {
		t.Errorf("pâte : %+v", r)
	}
	if r := findRow(t, s, "Affichage", "HDMI 1"); r.Status != statusOK {
		t.Errorf("HDMI 1 : %+v", r)
	}
	if r := findRow(t, s, "Affichage", "HDMI 2"); r.pillLabel() != pillUntested {
		t.Errorf("HDMI 2 non branché, verdict OK : %+v", r)
	}
	if r := findRow(t, s, "Ports et saisie", "Ports USB"); r.Value != "2 / 4 testé(s)" {
		t.Errorf("USB : %+v", r)
	}
	// verdict négatif : la prise non branchée est celle qui ne marche pas
	s.Results["ports"] = result{Status: statusFail, Data: map[string]string{"HDMI": "1 / 2"}}
	if r := findRow(t, s, "Affichage", "HDMI 2"); r.Status != statusFail {
		t.Errorf("HDMI 2, verdict Nul : %+v", r)
	}
	// nombre de prises inconnu : seules les branchées sont nommées
	s.Results["ports"] = result{Status: statusOK, Data: map[string]string{"DisplayPort": "1 / ?"}}
	if r := findRow(t, s, "Affichage", "DisplayPort"); r.Status != statusOK {
		t.Errorf("DisplayPort : %+v", r)
	}
	findRow(t, s, "Affichage", "Autres sorties vidéo")
}

// La fiche produit nomme le produit, sans adresse réseau ni nom système, et
// reprend en tête ce qui ne va pas.
func TestFicheSheet(t *testing.T) {
	s := newSession()
	s.Machine = machineInfo{Vendor: "HP", Product: "HP EliteBook 850 G3", Serial: "SN1"}
	s.Inventory = []Component{
		{Name: "Processeur", Summary: "Intel(R) Core(TM) i5-6200U CPU @ 2.30GHz"},
		{Name: "Stockage", Fields: []Field{{"sda", "Samsung 870 · SSD · 232.9 Go · 2 partition(s)"}}},
		{Name: "Cartes réseau", Fields: []Field{{"enp0s31f6", "Intel I219-V · ethernet · connecté · ac:e2:d3:96:29:dd · 1000 Mbps · 192.168.1.5"}}},
	}
	s.Results["sd"] = result{Status: statusFail, Detail: "Carte non détectée"}
	sh := newFicheSheet(s)
	if sh.Name != "HP EliteBook 850 G3" {
		t.Errorf("nom : %q", sh.Name)
	}
	if sh.Specs[0].Value != "Intel Core i5-6200U" || sh.Specs[1].Value != "232.9 Go SSD" {
		t.Errorf("essentiel : %+v", sh.Specs)
	}
	if len(sh.Problems) != 1 || !strings.Contains(sh.Problems[0].Label, "Port carte SD") {
		t.Errorf("à signaler : %+v", sh.Problems)
	}
	all := fmt.Sprintf("%+v", sh)
	for _, absent := range []string{s.Ref, "ac:e2", "192.168", "enp0s31f6", "partition", "Tests gobox"} {
		if strings.Contains(all, absent) {
			t.Errorf("la fiche contient %q", absent)
		}
	}
}

// Le verdict donne la note, et les colonnes s'équilibrent.
func TestFicheGradeAndColumns(t *testing.T) {
	s := newSession()
	for _, d := range s.Tests {
		s.Results[d.ID] = result{Status: statusOK}
	}
	s.Results["sd"] = result{Status: statusWarn}
	if sh := newFicheSheet(s); sh.Grade != "B" || sh.Kind != verdictOK {
		t.Errorf("note : %q, verdict %v", sh.Grade, sh.Kind)
	}
	if k := splitColumns([]float64{10, 10, 10, 30}); k != 3 {
		t.Errorf("coupure : %d, attendu 3", k)
	}
}

// Le navigateur par défaut vient des préférences, sans bureau dans
// l'environnement (pkexec, sudo) ; une entrée sans fichier .desktop est ignorée.
func TestFindDefaultBrowser(t *testing.T) {
	home, data := t.TempDir(), t.TempDir()
	apps := filepath.Join(data, "applications")
	if err := os.MkdirAll(apps, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	entry := func(cmd string) string { return "[Desktop Entry]\nExec=" + cmd + " %u\n" }
	write(filepath.Join(apps, "brave-browser.desktop"), entry("brave"))
	write(filepath.Join(apps, "firefox.desktop"), entry("firefox"))
	write(filepath.Join(apps, "mimeapps.list"), "[Default Applications]\nx-scheme-handler/http=absent.desktop\n")
	write(filepath.Join(apps, "gnome-mimeapps.list"), "[Default Applications]\nx-scheme-handler/http=firefox.desktop\n")

	got := findDefaultBrowser(home, []string{data}, nil)
	if len(got) != 1 || got[0] != "firefox" {
		t.Fatalf("sans bureau : %v, attendu firefox", got)
	}
	write(filepath.Join(home, ".config", "mimeapps.list"), "[Default Applications]\nx-scheme-handler/https=brave-browser.desktop;firefox.desktop\n")
	got = findDefaultBrowser(home, []string{data}, []string{"GNOME"})
	if len(got) != 1 || got[0] != "brave" {
		t.Fatalf("choix de l'utilisateur : %v, attendu brave", got)
	}
}

// Le navigateur par défaut est lancé depuis son .desktop, sans ses codes.
func TestDesktopExec(t *testing.T) {
	d := "[Desktop Entry]\nName=Brave\nExec=/usr/bin/flatpak run com.brave.Browser @@u %U @@\n[Desktop Action new]\nExec=/usr/bin/autre\n"
	got := strings.Join(desktopExec(d), " ")
	if got != "/usr/bin/flatpak run com.brave.Browser" {
		t.Fatalf("commande : %q", got)
	}
	if desktopExec("[Desktop Entry]\nExec=\"/opt/mon navigateur\" %u\n") != nil {
		t.Fatal("guillemets : repli sur xdg-open attendu")
	}
	if desktopExec("[Desktop Action new]\nExec=/usr/bin/autre\n") != nil {
		t.Fatal("seule l'entrée principale compte")
	}
}

// Le nombre de prises vient du BIOS : deux machines, deux résultats.
func TestParseDMIConnectors(t *testing.T) {
	block := func(extName, extType, port string) string {
		return "Handle 0x0008, DMI type 8, 9 bytes\nPort Connector Information\n\tInternal Reference Designator: J1\n\tInternal Connector Type: None\n\tExternal Reference Designator: " + extName + "\n\tExternal Connector Type: " + extType + "\n\tPort Type: " + port + "\n"
	}
	out := strings.Join([]string{
		block("HDMI", "Other", "Video Port"), block("DisplayPort 1", "Other", "Video Port"), block("Mini DP", "Other", "Other"),
		block("VGA", "DB-15 female", "Video Port"), block("DVI-D", "Other", "Video Port"),
		block("USB1", "Access Bus (USB)", "USB"), block("USB2", "Access Bus (USB)", "USB"),
		block("Serial", "DB-9 male", "Serial Port 16550A Compatible"), block("Interne HDMI", "None", "Video Port"),
	}, "\n")
	video, usb := parseDMIConnectors(out)
	want := map[string]int{"HDMI": 1, "DisplayPort": 2, "VGA": 1, "DVI": 1}
	for typ, n := range want {
		if video[typ] != n {
			t.Errorf("%s : %d, attendu %d", typ, video[typ], n)
		}
	}
	if usb != 2 {
		t.Errorf("USB : %d, attendu 2", usb)
	}
	if v, u := parseDMIConnectors(""); len(v) != 0 || u != 0 {
		t.Errorf("sortie vide : %v %d", v, u)
	}
}

// Les ports que le contrôleur déclare sans prise ne comptent pas : seul
// « hotplug » est une prise, une moitié USB 3 non renseignée rejoint sa prise.
func TestUSBPortsCount(t *testing.T) {
	p := usbPorts{group: map[string]string{"usb1-port1": "usb1-port1", "usb2-port1": "usb1-port1", "usb1-port3": "usb1-port3"},
		possible: map[string]string{"usb2-port6": "usb1-port3", "usb1-port8": "usb1-port8"}}
	p.finish(0)
	if p.total != 2 {
		t.Fatalf("total %d, attendu 2", p.total)
	}
	if p.portOf("1-8") != "" {
		t.Fatal("un port non renseigné n'est pas une prise tant qu'on n'y branche rien")
	}
	if p.portOf("2-6") != "usb1-port3" {
		t.Fatal("la moitié USB 3 doit rejoindre sa prise")
	}
	if p.adopt("1-8") != "usb1-port8" || p.total != 3 {
		t.Fatalf("branchement sur un port non renseigné : total %d", p.total)
	}
	// aucun port renseigné : BIOS, sinon nombre inconnu
	none := usbPorts{group: map[string]string{}, possible: map[string]string{"usb1-port1": "usb1-port1"}}
	none.finish(4)
	if none.total != 4 {
		t.Fatalf("repli BIOS : %d", none.total)
	}
	none.finish(0)
	if none.total != -1 {
		t.Fatalf("inconnu : %d", none.total)
	}
}

// Pendant le test clavier, le focus reste sur le piège à touches : Tab ne
// doit pas le déplacer sur un bouton, et un clic ailleurs le lui rend.
func TestKeyboardKeepsFocus(t *testing.T) {
	manageSuperKey = false
	a := test.NewApp()
	defer a.Quit()
	a.Settings().SetTheme(newTheme(true))
	w := a.NewWindow("test")
	s := newSession()
	content, tools := mainContent(s, w)
	w.SetContent(content)
	w.Resize(fyne.NewSize(1280, 900))

	test.Tap(tools.buttons["clavier"])
	trapOf := func() *keyTrap {
		k, _ := w.Canvas().Focused().(*keyTrap)
		return k
	}
	deadline := time.Now().Add(2 * time.Second)
	for trapOf() == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	trap := trapOf()
	if trap == nil {
		t.Fatalf("le piège à touches n'a pas le focus : %T", w.Canvas().Focused())
	}
	if !trap.AcceptsTab() {
		t.Fatal("le piège doit accepter Tab")
	}
	// Fyne remet les touches au widget qui a le focus (window.go, processKeyPressed)
	trap.KeyDown(&fyne.KeyEvent{Physical: fyne.HardwareKey{ScanCode: 38}})
	trap.KeyDown(&fyne.KeyEvent{Physical: fyne.HardwareKey{ScanCode: 38}})
	trap.KeyDown(&fyne.KeyEvent{Physical: fyne.HardwareKey{ScanCode: 23}}) // Tab
	w.Canvas().FocusNext()                                                 // ce que ferait Tab sans le piège
	time.Sleep(100 * time.Millisecond)
	if trapOf() == nil {
		t.Fatalf("le focus est parti sur %T", w.Canvas().Focused())
	}
	time.Sleep(100 * time.Millisecond) // la question de fin arrive après la prise de focus
	answerAsk(tools.view, 0)
	deadline = time.Now().Add(2 * time.Second)
	for !tools.view.finished.Visible() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	tools.close(true)
	if got := s.Results["clavier"].Detail; !strings.HasPrefix(got, "2 / ") {
		t.Fatalf("touches comptées : %q, attendu 2 touches", got)
	}
	if w.Canvas().Focused() != nil {
		t.Fatalf("le piège doit rendre le focus à la fin du test : %T", w.Canvas().Focused())
	}
}
