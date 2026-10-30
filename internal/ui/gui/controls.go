package gui

import (
	"fmt"
	"strconv"
	"strings"
)

// Sections de la fiche : chaque composant réunit ses caractéristiques et les
// résultats de ses tests. Partagées par la fiche à l'écran et la fiche imprimée.

// ctrlItem ligne de contrôle : renseignée par un test gobox, ou saisie par
// le technicien quand aucun test ne la couvre
type ctrlItem struct {
	Label  string
	TestID string   // test gobox ; "" : saisie du technicien
	Key    string   // donnée du test propre à la ligne (son : "speakers")
	Scale  []status // choix du technicien (OK / Nul ou OK / Moyen / Nul)
	Free   bool     // libellé à préciser (« Autre »)
}

var (
	scaleOKNul    = []status{statusOK, statusFail}
	scaleOKMoyNul = []status{statusOK, statusWarn, statusFail}

	ctrlOtherVideo = ctrlItem{Label: "Autre sortie vidéo", Scale: scaleOKNul, Free: true}
	ctrlOther      = ctrlItem{Label: "Autre", Scale: scaleOKMoyNul, Free: true}
	// lignes saisies par le technicien, sans test gobox
	ctrlManual = []ctrlItem{ctrlOtherVideo, ctrlOther}

	ctrlPeripherals = []ctrlItem{
		{Label: "Port carte SD", TestID: "sd"},
		{Label: "Clavier", TestID: "clavier"},
		{Label: "Pad / souris", TestID: "souris"},
		{Label: "Webcam", TestID: "webcam"},
	}
	ctrlSound = []ctrlItem{
		{Label: "Carte son", TestID: "son", Key: "card"},
		{Label: "Haut-parleurs", TestID: "son", Key: "speakers"},
		{Label: "Sortie audio", TestID: "son", Key: "jack"},
		{Label: "Entrée micro", TestID: "son", Key: "mic_in"},
		{Label: "Micro interne", TestID: "son", Key: "mic_int"},
	}
	ctrlSmart = []ctrlItem{
		{Label: "SMART HDD", TestID: "hdd"},
		{Label: "SMART SSD", TestID: "ssd"},
		{Label: "SMART NVMe", TestID: "nvme"},
	}
	ctrlSystem = []ctrlItem{
		{Label: "Clonage système", TestID: "clone"},
		{Label: "Déploiement post-install", TestID: "deploy"},
		{Label: "Mises à jour", TestID: "maj"},
	}
)

// itemState statut d'une ligne
func (s *session) itemState(it ctrlItem) status {
	if it.TestID == "" {
		return s.Form.Checks[it.Label]
	}
	r, ok := s.Results[it.TestID]
	if !ok || it.Key == "" {
		return s.status(it.TestID)
	}
	if st, ok := parseStatusData(r.Data[it.Key]); ok {
		return st
	}
	return r.Status
}

// parseStatusData statut écrit par statusData (« st:ok »)
func parseStatusData(v string) (status, bool) {
	code, ok := strings.CutPrefix(v, "st:")
	if !ok {
		return statusIdle, false
	}
	for _, s := range []status{statusOK, statusWarn, statusFail, statusNA} {
		if strings.ToLower(s.style().label) == code {
			return s, true
		}
	}
	return statusIdle, false
}

// ficheRow ligne d'une section : libellé, valeur, et résultat de test éventuel
type ficheRow struct {
	Label, Value string
	Tested       bool   // la ligne porte un résultat
	Status       status // résultat
	Pill         string // libellé du résultat quand il diffère du statut
}

// pillLabel libellé affiché pour le résultat
func (r ficheRow) pillLabel() string {
	if r.Pill != "" {
		return r.Pill
	}
	return r.Status.style().label
}

// ficheSection un composant de la machine et ses résultats
type ficheSection struct {
	Name, Icon, Summary string
	Rows                []ficheRow
}

const pillUntested = "NON TESTÉ"

// testRow ligne du résultat d'un test, avec son détail
func (s *session) testRow(label, id string) ficheRow {
	return ficheRow{Label: label, Value: s.Results[id].Detail, Tested: true, Status: s.status(id)}
}

func (s *session) itemRows(items []ctrlItem) []ficheRow {
	var out []ficheRow
	for _, it := range items {
		r := ficheRow{Label: it.Label, Tested: true, Status: s.itemState(it), Pill: s.data(it.TestID, "pill")}
		if it.Key == "" {
			r.Value = s.Results[it.TestID].Detail
		}
		out = append(out, r)
	}
	return out
}

// manualRow ligne « Autre » : seulement si le technicien l'a renseignée
func (s *session) manualRow(it ctrlItem) []ficheRow {
	label, st := s.Form.Texts[it.Label], s.Form.Checks[it.Label]
	if label == "" && st == statusIdle {
		return nil
	}
	return []ficheRow{{Label: orUnknown(label, it.Label), Tested: st != statusIdle, Status: st}}
}

// parsePortCount « 2 / 3 » → 2, 3 ; « 2 / ? » → 2, -1
func parsePortCount(v string) (tested, total int, ok bool) {
	a, b, ok := strings.Cut(v, "/")
	if !ok {
		return 0, 0, false
	}
	tested, err := strconv.Atoi(strings.TrimSpace(a))
	if err != nil {
		return 0, 0, false
	}
	if strings.TrimSpace(b) == "?" {
		return tested, -1, true
	}
	total, err = strconv.Atoi(strings.TrimSpace(b))
	return tested, total, err == nil
}

// portRows une ligne par prise vidéo, avec son résultat. Une prise testée est
// OK ; une prise non branchée suit le verdict du technicien s'il est négatif,
// sinon elle reste « non testée ». Sans liste des prises, seules celles
// branchées sont nommées et une ligne porte le verdict du reste.
func (s *session) portRows() []ficheRow {
	r, ran := s.Results["ports"]
	var out []ficheRow
	unknown := !ran && len(s.Ports.Video) == 0
	listed := false
	for _, typ := range videoTypes {
		tested, total := 0, s.Ports.Video[typ]
		if ran {
			var ok bool
			if tested, total, ok = parsePortCount(r.Data[typ]); !ok {
				continue
			}
		}
		listed = true
		unknown = unknown || total < 0
		n := max(tested, total)
		for i := 1; i <= n; i++ {
			row := ficheRow{Label: typ, Tested: true}
			if n > 1 {
				row.Label = fmt.Sprintf("%s %d", typ, i)
			}
			switch {
			case i <= tested:
				row.Status, row.Value = statusOK, "écran détecté"
			case !ran:
				row.Status = statusIdle
			case r.Status == statusWarn || r.Status == statusFail:
				row.Status = r.Status
			default:
				row.Status, row.Pill = statusIdle, pillUntested
			}
			out = append(out, row)
		}
	}
	if unknown || (ran && !listed) {
		row := ficheRow{Label: "Sorties vidéo", Value: "nombre de prises non lu", Tested: true, Status: s.status("ports")}
		if listed {
			row.Label = "Autres sorties vidéo"
		}
		out = append(out, row)
	}
	return out
}

// usbRows une ligne par type de prise USB : testées / présentes
func (s *session) usbRows() []ficheRow {
	r, ran := s.Results["ports"]
	var out []ficheRow
	for _, kind := range usbKindOrder {
		row := ficheRow{Label: kind, Tested: true}
		if !ran {
			n := s.Ports.USBKinds[kind]
			if n == 0 {
				continue
			}
			row.Value = fmt.Sprintf("%d prise(s)", n)
			out = append(out, row)
			continue
		}
		tested, total, ok := parsePortCount(r.Data[kind])
		if !ok {
			continue
		}
		row.Value = fmt.Sprintf("%d / %d testé(s)", tested, total)
		switch {
		case tested >= total:
			row.Status = statusOK
		case r.Status == statusWarn || r.Status == statusFail:
			row.Status = r.Status
		default:
			row.Status, row.Pill = statusIdle, pillUntested
		}
		out = append(out, row)
	}
	if len(out) > 0 {
		return out
	}
	// nombre de prises inconnu
	row := ficheRow{Label: "Ports USB", Tested: true, Status: s.status("ports")}
	if tested, total, ok := parsePortCount(r.Data["usb"]); ran && ok {
		row.Value = fmt.Sprintf("%d testé(s), nombre de prises non lu", tested)
		if total >= 0 {
			row.Value = fmt.Sprintf("%d / %d testé(s)", tested, total)
		}
	}
	return []ficheRow{row}
}

// stressRow résultat du stress test : températures et erreurs de calcul
func (s *session) stressRow() ficheRow {
	row := s.testRow("Stress test", "stress")
	if row.Value == "" {
		if before, after := s.data("stress", "before"), s.data("stress", "after"); before != "" {
			row.Value = before + " au repos, " + after + " au maximum"
		}
	}
	return row
}

// cleaningText état du nettoyage
func cleaningText(fm *ficheForm) string {
	switch {
	case fm.CleanIn && fm.CleanOut:
		return "intérieur et extérieur"
	case fm.CleanIn:
		return "intérieur"
	case fm.CleanOut:
		return "extérieur"
	}
	return "à faire"
}

// ficheSections la fiche, composant par composant
func (s *session) ficheSections() []ficheSection {
	fm := &s.Form
	info := func(label, v string) ficheRow { return ficheRow{Label: label, Value: v} }
	var smart []ficheRow
	for _, row := range s.itemRows(ctrlSmart) {
		if row.Status != statusNA || row.Pill != "" { // sinon : pas de disque de ce type
			smart = append(smart, row)
		}
	}
	storage := append(smart, info("Shred des disques", fm.Shred))
	if fm.DVD != "absent" {
		storage = append(storage, info("Lecteur DVD fonctionnel", fm.DVD))
	}
	periph := append(append(s.usbRows(), info("USB défectueux", orUnknown(strings.TrimSpace(fm.USBBad), "0"))), s.itemRows(ctrlPeripherals)...)

	prep := append([]ficheRow{info("Aspect", strings.ToLower(fm.Aspect)), info("Nettoyage", cleaningText(fm)), info("Pile setup changée", fm.CMOS)}, s.itemRows(ctrlSystem)...)

	// Du cœur de la machine vers l'extérieur, puis sa remise en état
	extras := []struct {
		name, icon string
		rows       []ficheRow
	}{
		{"Processeur", "microchip", []ficheRow{s.stressRow(), info("Pâte thermique changée", fm.Paste)}},
		{"Mémoire RAM", "memory", nil},
		{"Stockage", "hard-drive", storage},
		{"Carte graphique", "display", nil},
		{"Batterie", "battery-half", []ficheRow{info("Chargeur en bon état", fm.Charger)}},
		{"Affichage", "tv", append(s.portRows(), s.manualRow(ctrlOtherVideo)...)},
		{"Ports et périphériques", "plug", append(periph, s.manualRow(ctrlOther)...)},
		{"Son", "volume-high", s.itemRows(ctrlSound)},
		{"Cartes réseau", "wifi", []ficheRow{s.testRow("Test réseau", "reseau")}},
		{"Préparation", "rocket", prep},
	}
	components := map[string]Component{}
	for _, c := range s.Inventory {
		components[c.Name] = c
	}
	var out []ficheSection
	for _, e := range extras {
		c := components[e.name]
		sec := ficheSection{Name: e.name, Icon: e.icon, Summary: c.Summary}
		if c.Err != nil {
			sec.Rows = append(sec.Rows, info("Erreur", c.Err.Error()))
		}
		for _, f := range c.Fields {
			sec.Rows = append(sec.Rows, info(f.Label, f.Value))
		}
		if len(c.Fields) == 0 && c.Summary != "" {
			sec.Rows = append(sec.Rows, info("État", c.Summary))
		}
		sec.Rows = append(sec.Rows, e.rows...)
		if len(sec.Rows) > 0 {
			out = append(out, sec)
		}
	}
	return out
}

// scaleLabel libellé d'un choix du technicien
func scaleLabel(s status) string {
	switch s {
	case statusOK:
		return "OK"
	case statusWarn:
		return "Moyen"
	case statusFail:
		return "Nul"
	}
	return ""
}

// maintRow question oui / non de la fiche
type maintRow struct {
	Short string // libellé à l'écran
	Value func(*ficheForm) *string
	Opts  []string
}

var maintRows = []maintRow{
	{"Pâte thermique changée", func(f *ficheForm) *string { return &f.Paste }, []string{"oui", "non"}},
	{"Shred des disques", func(f *ficheForm) *string { return &f.Shred }, []string{"oui", "non"}},
	{"Chargeur en bon état", func(f *ficheForm) *string { return &f.Charger }, []string{"oui", "non"}},
	{"Pile setup changée", func(f *ficheForm) *string { return &f.CMOS }, []string{"oui", "non"}},
	{"Lecteur DVD", func(f *ficheForm) *string { return &f.DVD }, []string{"oui", "non", "absent"}},
}
