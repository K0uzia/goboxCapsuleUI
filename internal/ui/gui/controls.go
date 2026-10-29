package gui

import (
	"fmt"
	"strconv"
	"strings"
)

// Contrôles de la fiche, dans l'ordre de la fiche papier de l'atelier.
// Partagés par la fiche à l'écran et la fiche imprimée.

// ctrlItem ligne de contrôle : renseignée par un test gobox, ou saisie par
// le technicien quand aucun test ne la couvre
type ctrlItem struct {
	Label  string
	TestID string   // test gobox ; "" : saisie du technicien
	Key    string   // donnée du test propre à la ligne (ports : "HDMI")
	Scale  []status // cases de la fiche papier (OK / Nul ou OK / Moyen / Nul)
	Free   bool     // libellé à préciser (« Autre »)
}

var (
	scaleOKNul    = []status{statusOK, statusFail}
	scaleOKMoyNul = []status{statusOK, statusWarn, statusFail}

	ctrlVideo = []ctrlItem{
		{Label: "HDMI", TestID: "ports", Key: "HDMI", Scale: scaleOKNul},
		{Label: "DisplayPort", TestID: "ports", Key: "DisplayPort", Scale: scaleOKNul},
		{Label: "VGA", TestID: "ports", Key: "VGA", Scale: scaleOKNul},
		{Label: "DVI", TestID: "ports", Key: "DVI", Scale: scaleOKNul},
		{Label: "Autre sortie vidéo", Scale: scaleOKNul, Free: true},
		{Label: "Ports USB", TestID: "ports", Key: "usb", Scale: scaleOKNul},
		{Label: "Webcam", TestID: "webcam", Scale: scaleOKMoyNul},
		{Label: "Port carte SD", TestID: "sd", Scale: scaleOKNul},
	}
	ctrlAudio = []ctrlItem{
		{Label: "Clavier", TestID: "clavier", Scale: scaleOKMoyNul},
		{Label: "Pad / souris", TestID: "souris", Scale: scaleOKMoyNul},
		{Label: "Carte son", TestID: "son", Key: "card", Scale: scaleOKNul},
		{Label: "Haut-parleurs", TestID: "son", Key: "speakers", Scale: scaleOKMoyNul},
		{Label: "Sortie audio", TestID: "son", Key: "jack", Scale: scaleOKNul},
		{Label: "Entrée micro", TestID: "son", Key: "mic_in", Scale: scaleOKNul},
		{Label: "Micro interne", TestID: "son", Key: "mic_int", Scale: scaleOKNul},
		{Label: "Autre", Scale: scaleOKMoyNul, Free: true},
	}
	// tests gobox absents de la fiche papier
	ctrlGobox = []ctrlItem{
		{Label: "Stress test CPU", TestID: "stress"},
		{Label: "Réseau", TestID: "reseau"},
		{Label: "SMART disque HDD", TestID: "hdd"},
		{Label: "SMART disque SSD", TestID: "ssd"},
		{Label: "SMART disque NVMe", TestID: "nvme"},
		{Label: "Clonage système", TestID: "clone"},
		{Label: "Déploiement post-install", TestID: "deploy"},
		{Label: "Mises à jour", TestID: "maj"},
	}
)

// itemState statut d'une ligne et valeur à afficher à côté (« 1 / 3 »)
func (s *session) itemState(it ctrlItem) (status, string) {
	if it.TestID == "" {
		return s.Form.Checks[it.Label], ""
	}
	r, ok := s.Results[it.TestID]
	if !ok || it.Key == "" {
		return s.status(it.TestID), ""
	}
	v, has := r.Data[it.Key]
	if st, ok := parseStatusData(v); ok {
		return st, ""
	}
	if !has {
		if it.TestID == "ports" && it.Key != "usb" {
			return statusNA, "aucun" // pas de connecteur de ce type
		}
		return r.Status, ""
	}
	tested, total, ok := parseRatio(v)
	switch {
	case !ok:
		return r.Status, v
	case total == 0:
		return statusNA, "aucun"
	case tested > 0:
		return statusOK, v
	}
	return r.Status, v
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

// parseRatio « 2 / 9 » → 2, 9
func parseRatio(v string) (int, int, bool) {
	a, b, ok := strings.Cut(v, "/")
	if !ok {
		return 0, 0, false
	}
	x, err1 := strconv.Atoi(strings.TrimSpace(a))
	y, err2 := strconv.Atoi(strings.TrimSpace(b))
	return x, y, err1 == nil && err2 == nil
}

// stressTemp température relevée (« 41 °C ») et sa case de la fiche papier :
// moins de 90 °C, ou 90 °C et plus
func stressTemp(v string) (temp float64, below90, ok bool) {
	f := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "°C"))
	t, err := strconv.ParseFloat(strings.ReplaceAll(f, ",", "."), 64)
	if err != nil {
		return 0, false, false
	}
	return t, t < stressFailTemp, true
}

// tempLabel case cochée pour une température, en court (« < 90 °C »)
func tempLabel(v string) string {
	if _, below, ok := stressTemp(v); ok {
		if below {
			return fmt.Sprintf("< %.0f °C", stressFailTemp)
		}
		return fmt.Sprintf("≥ %.0f °C", stressFailTemp)
	}
	return ""
}

// scaleLabel libellé d'une case de la fiche papier
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

// maintRow question oui / non de la fiche papier
type maintRow struct {
	Label string // libellé de la fiche papier (impression)
	Short string // libellé court, à l'écran
	Value func(*ficheForm) *string
	Opts  []string
}

var maintRows = []maintRow{
	{"Changement de pâte thermique", "Pâte thermique changée", func(f *ficheForm) *string { return &f.Paste }, []string{"oui", "non"}},
	{"Shred des disques durs", "Shred des disques", func(f *ficheForm) *string { return &f.Shred }, []string{"oui", "non"}},
	{"Chargeur en bon état", "Chargeur en bon état", func(f *ficheForm) *string { return &f.Charger }, []string{"oui", "non"}},
	{"Pile setup changée", "Pile setup changée", func(f *ficheForm) *string { return &f.CMOS }, []string{"oui", "non"}},
	{"Lecteur DVD fonctionnel", "Lecteur DVD", func(f *ficheForm) *string { return &f.DVD }, []string{"oui", "non", "absent"}},
}
