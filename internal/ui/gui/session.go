package gui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
)

// ---- Statut d'un test -----------------------------------------------------

type status int

const (
	statusIdle status = iota
	statusRunning
	statusOK
	statusWarn
	statusFail
	statusNA
)

// statusStyle libellés et rôles de couleur d'un statut (pastille, bouton)
type statusStyle struct {
	label, sub string
	soft, ink  fyne.ThemeColorName
}

func (s status) style() statusStyle {
	switch s {
	case statusRunning:
		return statusStyle{"EN COURS", "Test en cours…", cAccentSoft, cAccent}
	case statusOK:
		return statusStyle{"OK", "Validé · relancer", cSuccessBg, cSuccess}
	case statusWarn:
		return statusStyle{"MOYEN", "Avec réserve · relancer", cWarningBg, cWarning}
	case statusFail:
		return statusStyle{"NUL", "Échec · relancer", cDangerBg, cDanger}
	case statusNA:
		return statusStyle{"ABSENT", "Non applicable", cSurface2, cMuted}
	}
	return statusStyle{"À FAIRE", "Lancer", cSurface2, cMuted}
}

// result résultat d'un test, inscrit dans la fiche après confirmation
type result struct {
	Status   status
	Detail   string
	Data     map[string]string // valeurs reprises dans la fiche (températures…)
	Advice   []string          // pistes d'amélioration affichées avec le résultat
	Duration time.Duration
}

// ---- Définition des tests -------------------------------------------------

const (
	catPeripherals = "Périphériques"
	catStorage     = "Stockage"
	catSystem      = "Système"
)

type testDef struct {
	ID       string
	Title    string
	Category string
	Desc     string // ce que fait le test (affiché pendant l'exécution)
	Color    fyne.ThemeColorName
	Soft     fyne.ThemeColorName
	Icon     string // icône Font Awesome (icons/)
	Run      func(t *testCtx) result
}

// categoryRoles couleur et fond teinté d'une catégorie
func categoryRoles(cat string) (fyne.ThemeColorName, fyne.ThemeColorName) {
	switch cat {
	case catStorage:
		return cStorage, cStorageBg
	case catSystem:
		return cSystem, cSystemBg
	}
	return cPeripheral, cPeripheralBg
}

// testIcons icône de chaque test
var testIcons = map[string]string{
	"stress": "fire", "ports": "plug", "son": "volume-high", "reseau": "network-wired",
	"clavier": "keyboard", "souris": "computer-mouse", "webcam": "camera", "sd": "sd-card",
	"hdd": "hard-drive", "ssd": "memory", "nvme": "microchip",
	"clone": "clone", "deploy": "rocket", "maj": "arrows-rotate",
}

func testDefs() []*testDef {
	defs := []*testDef{
		{ID: "stress", Title: "Stress", Category: catPeripherals, Desc: "Charge CPU 100 % et suivi de la température", Run: runStress},
		{ID: "ports", Title: "Ports", Category: catPeripherals, Desc: "Détection des branchements USB et vidéo", Run: runPorts},
		{ID: "son", Title: "Son", Category: catPeripherals, Desc: "Haut-parleurs, prise casque, micro interne et entrée micro", Run: runSound},
		{ID: "reseau", Title: "Réseau", Category: catPeripherals, Desc: "Interfaces, DNS et accès internet", Run: runNetwork},
		{ID: "clavier", Title: "Clavier", Category: catPeripherals, Desc: "Appui sur chaque touche du clavier", Run: runKeyboard},
		{ID: "souris", Title: "Souris", Category: catPeripherals, Desc: "Clics, défilement et pavé tactile", Run: runMouse},
		{ID: "webcam", Title: "Webcam", Category: catPeripherals, Desc: "Aperçu de la caméra", Run: runWebcam},
		{ID: "sd", Title: "Carte SD", Category: catPeripherals, Desc: "Détection d'une carte insérée", Run: runSD},
		{ID: "hdd", Title: "S. HDD", Category: catStorage, Desc: "État SMART des disques durs", Run: runSmart("hdd")},
		{ID: "ssd", Title: "S. SSD", Category: catStorage, Desc: "État SMART des SSD SATA", Run: runSmart("ssd")},
		{ID: "nvme", Title: "S. NVMe", Category: catStorage, Desc: "État SMART des SSD NVMe", Run: runSmart("nvme")},
		{ID: "clone", Title: "Clone", Category: catSystem, Desc: "Clonage d'une image système", Run: runManual},
		{ID: "deploy", Title: "Déploiement", Category: catSystem, Desc: "Post-installation et pilotes", Run: runManual},
		{ID: "maj", Title: "MAJ", Category: catSystem, Desc: "Mise à jour du système", Run: runManual},
	}
	for _, d := range defs {
		d.Color, d.Soft = categoryRoles(d.Category)
		d.Icon = testIcons[d.ID]
	}
	return defs
}

// ---- Session de reconditionnement ----------------------------------------

type ficheForm struct {
	Tech     string
	Aspect   string // Neuf, Occasion, Nul
	CleanIn  bool
	CleanOut bool
	Paste    string // oui / non
	Shred    string
	Charger  string
	CMOS     string
	DVD      string // oui / non / absent
	USBBad   string
	Checks   map[string]status // contrôles saisis à la main (ctrlItem.Label)
	Texts    map[string]string // précisions des lignes « Autre »
	Notes    string
}

type session struct {
	Ref       string
	Started   time.Time
	Form      ficheForm
	Tests     []*testDef
	Results   map[string]result
	Inventory []Component
	Machine   machineInfo

	listeners []func()
}

func newSession() *session {
	now := time.Now()
	return &session{
		Ref:     fmt.Sprintf("RC-%s", now.Format("2006-0102-1504")),
		Started: now,
		Form: ficheForm{
			Aspect: "Occasion", Paste: "non", Shred: "non", Charger: "oui",
			CMOS: "non", DVD: "absent", USBBad: "0",
			Checks: map[string]status{}, Texts: map[string]string{},
		},
		Tests:   testDefs(),
		Results: map[string]result{},
	}
}

// onChange enregistre une fonction appelée à chaque modification
func (s *session) onChange(f func()) { s.listeners = append(s.listeners, f) }

func (s *session) changed() {
	for _, f := range s.listeners {
		f()
	}
}

func (s *session) setResult(id string, r result) {
	s.Results[id] = r
	s.changed()
}

func (s *session) status(id string) status {
	if r, ok := s.Results[id]; ok {
		return r.Status
	}
	return statusIdle
}

func (s *session) data(id, key string) string {
	if r, ok := s.Results[id]; ok {
		return r.Data[key]
	}
	return ""
}

func (s *session) counts() (done, ok, warn, fail int) {
	for _, t := range s.Tests {
		switch s.status(t.ID) {
		case statusOK:
			ok++
		case statusWarn:
			warn++
		case statusFail:
			fail++
		}
		if st := s.status(t.ID); st != statusIdle && st != statusRunning {
			done++
		}
	}
	return
}

type verdictKind int

const (
	verdictPending verdictKind = iota
	verdictOK
	verdictBad
)

type verdict struct {
	Label, Sub string
	Kind       verdictKind
}

// roles couleur du texte et du fond du verdict
func (v verdict) roles() (ink, fill fyne.ThemeColorName) {
	switch v.Kind {
	case verdictOK:
		return cSuccess, cSuccessBg
	case verdictBad:
		return cDanger, cDangerBg
	}
	return cMuted, cSurface2
}

// verdict même règle que la maquette : >1 échec = non apte, réserve = grade
// B. Les tests sans objet (composant absent) ne comptent pas dans le total.
func (s *session) verdict() verdict {
	done, ok, warn, fail := s.counts()
	total := len(s.Tests)
	applicable := total - (done - ok - warn - fail) // sans les absents
	switch {
	case done < total:
		return verdict{"En cours", fmt.Sprintf("%d test(s) restant(s)", total-done), verdictPending}
	case fail > 1:
		return verdict{"Non apte", fmt.Sprintf("%d tests en échec", fail), verdictBad}
	case fail+warn > 0:
		return verdict{"Apte à la vente", fmt.Sprintf("Grade B · %d / %d OK", ok, applicable), verdictOK}
	}
	return verdict{"Apte à la vente", fmt.Sprintf("Grade A · %d / %d OK", ok, applicable), verdictOK}
}

// markAbsent inscrit « absent » pour les tests dont le composant n'existe
// pas sur cette machine (sans écraser un résultat déjà confirmé)
func (s *session) markAbsent(absent map[string]string) {
	for id, reason := range absent {
		if _, done := s.Results[id]; !done {
			s.Results[id] = result{Status: statusNA, Detail: reason + " (détection automatique)", Data: map[string]string{"auto": "1"}}
		}
	}
}

// elapsed durée du reconditionnement depuis l'ouverture
func (s *session) elapsed() string {
	d := time.Since(s.Started).Round(time.Minute)
	return fmt.Sprintf("%d h %02d", int(d.Hours()), int(d.Minutes())%60)
}
