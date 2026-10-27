package gui

import (
	"fmt"
	"image/color"
	"time"
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

type statusStyle struct {
	label, sub string
	soft, ink  color.Color // pastille
	solid      color.Color // bouton
}

func (s status) style() statusStyle {
	switch s {
	case statusRunning:
		return statusStyle{"EN COURS", "Test en cours…", colBlueSoft, colBlueInk, colPanel}
	case statusOK:
		return statusStyle{"OK", "Validé · relancer", colTealSoft, colTeal, colTeal}
	case statusWarn:
		return statusStyle{"MOYEN", "Avec réserve", colAmberSoft, colAmberInk, colAmber}
	case statusFail:
		return statusStyle{"NUL", "Échec · relancer", colRedSoft, colRedInk, colRed}
	case statusNA:
		return statusStyle{"ABSENT", "Non applicable", colGraySoft, colGrayInk, colLineSoft}
	}
	return statusStyle{"À FAIRE", "Lancer", colGraySoft, colGrayInk, colPanel}
}

// result résultat d'un test, inscrit dans la fiche après confirmation
type result struct {
	Status   status
	Detail   string
	Data     map[string]string // valeurs reprises dans la fiche (températures…)
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
	Color    color.Color
	Run      func(t *testCtx) result
}

func testDefs() []*testDef {
	p, s, y := colPeripheral, colStorage, colSystem
	return []*testDef{
		{"stress", "Stress", catPeripherals, "Charge CPU 100 % et suivi de la température", p, runStress},
		{"ports", "Ports", catPeripherals, "Détection des branchements USB et vidéo", p, runPorts},
		{"son", "Son", catPeripherals, "Tonalité gauche / droite sur les haut-parleurs", p, runSound},
		{"reseau", "Réseau", catPeripherals, "Interfaces, DNS et accès internet", p, runNetwork},
		{"clavier", "Clavier", catPeripherals, "Appui sur chaque touche du clavier", p, runKeyboard},
		{"souris", "Souris", catPeripherals, "Clics, défilement et pavé tactile", p, runMouse},
		{"webcam", "Webcam", catPeripherals, "Aperçu de la caméra", p, runWebcam},
		{"sd", "Carte SD", catPeripherals, "Détection d'une carte insérée", p, runSD},
		{"hdd", "S. HDD", catStorage, "État SMART des disques durs", s, runSmart("hdd")},
		{"ssd", "S. SSD", catStorage, "État SMART des SSD SATA", s, runSmart("ssd")},
		{"nvme", "S. NVMe", catStorage, "État SMART des SSD NVMe", s, runSmart("nvme")},
		{"clone", "Clone", catSystem, "Clonage d'une image système", y, runManual},
		{"deploy", "Déploiement", catSystem, "Post-installation et pilotes", y, runManual},
		{"maj", "MAJ", catSystem, "Mise à jour du système", y, runManual},
	}
}

// ---- Session de reconditionnement ----------------------------------------

type ficheForm struct {
	Tech       string
	Aspect     string // Neuf, Occasion, Nul
	CleanIn    bool
	CleanOut   bool
	Paste      string // oui / non
	Shred      string
	Charger    string
	CMOS       string
	DVD        string // oui / non / absent
	USBBad     string
	OtherVideo string
	Other      string
	Notes      string
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

type verdict struct {
	Label, Sub string
	Color      color.Color
	Fill       color.Color
}

// verdict même règle que la maquette : >1 échec = non apte, réserve = grade B
func (s *session) verdict() verdict {
	done, ok, warn, fail := s.counts()
	total := len(s.Tests)
	switch {
	case done < total:
		return verdict{"En cours", fmt.Sprintf("%d test(s) restant(s)", total-done), colGrayInk, colPanel}
	case fail > 1:
		return verdict{"Non apte", fmt.Sprintf("%d tests en échec", fail), colRed, rgb(0xFE, 0xF2, 0xF2)}
	case fail+warn > 0:
		return verdict{"Apte à la vente", fmt.Sprintf("Grade B · %d / %d OK", ok, total), colTeal, rgb(0xF0, 0xFA, 0xF8)}
	}
	return verdict{"Apte à la vente", fmt.Sprintf("Grade A · %d / %d OK", ok, total), colTeal, rgb(0xF0, 0xFA, 0xF8)}
}

// elapsed durée du reconditionnement depuis l'ouverture
func (s *session) elapsed() string {
	d := time.Since(s.Started).Round(time.Minute)
	return fmt.Sprintf("%d h %02d", int(d.Hours()), int(d.Minutes())%60)
}
