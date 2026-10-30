package gui

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// La fiche finale, composée directement en PDF comme une fiche produit de
// boutique : le produit et sa note d'abord, l'essentiel, puis le détail
// composant par composant. Colonnes équilibrées et taille du
// texte choisie pour remplir une page A4 sans en déborder. Noir sur blanc pour
// économiser l'encre : seul le mot de chaque résultat est en couleur.

type sheetSpec struct{ Label, Icon, Value string }

// ficheSheet contenu de la fiche finale
type ficheSheet struct {
	Name, Serial, Date, Tech string
	Grade, Verdict, Sub      string
	Kind                     verdictKind
	Specs                    []sheetSpec
	Sections                 []ficheSection
	Notes                    string
}

// sheetHidden lignes utiles à l'atelier, sans intérêt sur une fiche produit
var sheetHidden = map[string][]string{
	"Processeur":      {"Modèle", "Architecture", "Fabricant"},
	"Batterie":        {"Charge", "Temps restant", "Puissance", "Tension", "Fabricant", "Modèle", "Série"},
	"Carte graphique": {"Modèle", "Pilote", "Fabricant"},
	"Affichage":       {"État"},
}

var (
	cpuNoise   = regexp.MustCompile(`\(R\)|\(TM\)|\bCPU\b|\bProcessor\b|@.*$`)
	spaces     = regexp.MustCompile(`\s+`)
	gpuBracket = regexp.MustCompile(`\[([^\]]*[a-zA-Z][^\]]*)\]`)
)

func shortCPU(model string) string {
	return strings.TrimSpace(spaces.ReplaceAllString(cpuNoise.ReplaceAllString(model, ""), " "))
}

// shortGPU « Skylake GT2 [HD Graphics 520] [1916] » → « HD Graphics 520 »
func shortGPU(model string) string {
	if m := gpuBracket.FindStringSubmatch(model); m != nil {
		return m[1]
	}
	return model
}

// productName évite « HP HP EliteBook » : le modèle reprend parfois la marque
func productName(m machineInfo) string {
	if m.Vendor != "" && strings.HasPrefix(strings.ToLower(m.Product), strings.ToLower(m.Vendor)) {
		return m.Product
	}
	return m.Name()
}

// diskSpec « Samsung 870 · SSD · 232.9 Go · 2 partition(s) » → « 232.9 Go SSD »
func diskSpec(value string) string {
	parts := strings.Split(value, " · ")
	if len(parts) < 3 {
		return ""
	}
	return parts[2] + " " + parts[1]
}

var sheetIcons = map[string]string{
	"Processeur": "microchip", "Mémoire": "memory", "Stockage": "hard-drive",
	"Écran": "tv", "Graphique": "display", "Batterie": "battery-half",
}

func sheetSpecs(components map[string]Component) []sheetSpec {
	var out []sheetSpec
	add := func(label, v string) {
		if v != "" {
			out = append(out, sheetSpec{label, sheetIcons[label], v})
		}
	}
	add("Processeur", shortCPU(components["Processeur"].Summary))
	add("Mémoire", components["Mémoire RAM"].Summary)
	var disks []string
	for _, f := range components["Stockage"].Fields {
		if d := diskSpec(f.Value); d != "" {
			disks = append(disks, d)
		}
	}
	add("Stockage", strings.Join(disks, " + "))
	for _, f := range components["Affichage"].Fields {
		if f.Label == internalScreen {
			add("Écran", strings.TrimSpace(strings.TrimPrefix(f.Value, "connecté")))
		}
	}
	add("Graphique", shortGPU(components["Carte graphique"].Summary))
	if b := components["Batterie"]; len(b.Fields) > 0 {
		add("Batterie", b.Summary)
	}
	return out
}

// sheetRow adapte une ligne à la fiche produit : ni adresse MAC ou IP, ni
// partition, et un libellé compréhensible à la place du nom système
func sheetRow(section string, r ficheRow) ficheRow {
	parts := strings.Split(r.Value, " · ")
	switch section {
	case "Stockage":
		if len(parts) >= 3 {
			r.Label, r.Value = "Disque", strings.Join(parts[:3], " · ")
		}
	case "Cartes réseau":
		for i, p := range parts {
			name := map[string]string{"ethernet": "Ethernet", "wifi": "Wi-Fi"}[p]
			if name != "" {
				r.Label, r.Value = name, strings.Join(parts[:i], " · ")
				break
			}
		}
	}
	return r
}

func newFicheSheet(s *session) ficheSheet {
	v := s.verdict()
	sh := ficheSheet{
		Name: productName(s.Machine), Serial: s.Machine.Serial, Date: time.Now().Format("02/01/2006"),
		Tech: strings.TrimSpace(s.Form.Tech), Verdict: v.Label, Sub: v.Sub, Kind: v.Kind, Notes: strings.TrimSpace(s.Form.Notes),
	}
	if rest, ok := strings.CutPrefix(v.Sub, "Grade "); ok {
		sh.Grade, sh.Sub, _ = strings.Cut(rest, " · ")
	}
	components := map[string]Component{}
	for _, c := range s.Inventory {
		components[c.Name] = c
	}
	sh.Specs = sheetSpecs(components)
	for _, sec := range s.ficheSections() {
		kept := ficheSection{Name: sec.Name, Icon: sec.Icon}
		for _, r := range sec.Rows {
			hidden := false
			for _, h := range sheetHidden[sec.Name] {
				hidden = hidden || (r.Label == h && !r.Tested)
			}
			if hidden {
				continue
			}
			r = sheetRow(sec.Name, r)
			kept.Rows = append(kept.Rows, r)
		}
		if len(kept.Rows) > 0 {
			sh.Sections = append(sh.Sections, kept)
		}
	}
	return sh
}

// ---- Composition ----------------------------------------------------------

const (
	pageW, pageH = 210.0, 297.0
	pageMargin   = 14.0
	colGap       = 9.0
)

type rgb struct{ r, g, b int }

// couleurs du thème clair (theme.go), lisibles sur papier
var (
	inkText  = rgb{27, 29, 34}
	inkMuted = rgb{91, 96, 105}
	inkRule  = rgb{154, 160, 168}
	inkOK    = rgb{0x1e, 0x7a, 0x46}
	inkWarn  = rgb{0x8a, 0x53, 0x00}
	inkFail  = rgb{0xb3, 0x26, 0x1e}
)

func statusInk(s status) rgb {
	switch s {
	case statusOK:
		return inkOK
	case statusWarn:
		return inkWarn
	case statusFail:
		return inkFail
	}
	return inkMuted
}

type sheetPDF struct {
	*fpdf.Fpdf
	scale float64
}

func (p *sheetPDF) style(size float64, bold bool, c rgb) {
	st := ""
	if bold {
		st = "B"
	}
	p.SetFont("exo", st, size*p.scale)
	p.SetTextColor(c.r, c.g, c.b)
}

// text écrit une ligne avec son haut en y
func (p *sheetPDF) text(x, y, w float64, align, s string, h float64) {
	p.SetXY(x, y)
	p.CellFormat(w, h, s, "", 0, align, false, 0, "")
}

func (p *sheetPDF) lineH() float64 { return 4.3 * p.scale }

// rowLayout largeurs d'une ligne de section : libellé, valeur, résultat
func (p *sheetPDF) rowHeight(r ficheRow, w float64) float64 {
	lw, sw := w*0.34, 19*p.scale
	vw := w - lw - sw
	if !r.Tested {
		vw += sw
	}
	p.style(9, false, inkText)
	n := max(len(p.SplitText(r.Label, lw-1.5)), len(p.SplitText(r.Value, vw-1)))
	return float64(max(n, 1))*p.lineH() + 0.5
}

func (p *sheetPDF) sectionHeight(sec ficheSection, w float64) float64 {
	h := 6.2*p.scale + 3.5
	for _, r := range sec.Rows {
		h += p.rowHeight(r, w)
	}
	return h
}

func (p *sheetPDF) drawLines(x, y, w float64, s string) {
	for i, l := range p.SplitText(s, w) {
		p.text(x, y+float64(i)*p.lineH(), w, "L", l, p.lineH())
	}
}

func (p *sheetPDF) drawRow(x, y, w float64, r ficheRow) float64 {
	lw, sw := w*0.34, 19*p.scale
	vw := w - lw - sw
	if !r.Tested {
		vw += sw
	}
	h := p.rowHeight(r, w)
	bad := r.Tested && (r.Status == statusWarn || r.Status == statusFail)
	lc := inkMuted
	if bad {
		lc = inkText
	}
	p.style(9, bad, lc)
	p.drawLines(x, y, lw-1.5, r.Label)
	p.style(9, false, inkText)
	p.drawLines(x+lw, y, vw-1, r.Value)
	if r.Tested {
		c := statusInk(r.Status)
		p.style(7.5, true, c)
		label := r.pillLabel()
		tw := p.GetStringWidth(label)
		p.text(x+w-tw, y, tw, "R", label, p.lineH())
		if r.Status == statusOK || bad {
			p.SetFillColor(c.r, c.g, c.b)
			p.Circle(x+w-tw-1.8, y+p.lineH()/2, 0.75*p.scale, "F")
		}
	}
	return h
}

// drawSectionTitle icône, intitulé en capitales et filet pointillé
func (p *sheetPDF) drawSectionTitle(x, y, w float64, icon, title string) float64 {
	h := 5 * p.scale
	tx := x
	if icon != "" {
		size := 3.4 * p.scale
		drawIcon(p.Fpdf, icon, x, y+(h-size)/2, size, inkText)
		tx += size + 1.5
	}
	p.style(8, true, inkText)
	label := strings.ToUpper(title)
	p.text(tx, y, w, "L", label, h)
	p.SetDrawColor(inkRule.r, inkRule.g, inkRule.b)
	p.SetLineWidth(0.2)
	p.SetDashPattern([]float64{0.8, 0.8}, 0)
	lx := tx + p.GetStringWidth(label) + 2
	p.Line(lx, y+h/2, x+w, y+h/2)
	p.SetDashPattern(nil, 0)
	return 6.2 * p.scale
}

func (p *sheetPDF) drawSection(x, y, w float64, sec ficheSection) float64 {
	start := y
	y += p.drawSectionTitle(x, y, w, sec.Icon, sec.Name)
	for _, r := range sec.Rows {
		y += p.drawRow(x, y, w, r)
	}
	return y - start + 3.5
}

// splitColumns coupe la suite des sections en deux colonnes de hauteurs les plus
// proches possible, sans changer leur ordre
func splitColumns(heights []float64) int {
	var total, left float64
	for _, h := range heights {
		total += h
	}
	best, bestMax := 0, total
	for i, h := range heights {
		left += h
		if m := max(left, total-left); m < bestMax {
			best, bestMax = i+1, m
		}
	}
	return best
}

// drawHeader le produit à gauche, sa note à droite ; rend le bas de l'en-tête
func (p *sheetPDF) drawHeader(sh ficheSheet) float64 {
	x, w := pageMargin, pageW-2*pageMargin
	gradeW := 44.0
	y := pageMargin

	p.style(9, false, inkMuted)
	p.text(x, y, w-gradeW, "L", "La Capsule · PC reconditionné", 5)
	y += 6

	p.style(25, true, inkText)
	nameH := 10.5 * p.scale
	for _, l := range p.SplitText(orUnknown(sh.Name, "Machine"), w-gradeW-4) {
		p.text(x, y, w-gradeW-4, "L", l, nameH)
		y += nameH
	}
	y += 1
	meta := []string{}
	if sh.Serial != "" {
		meta = append(meta, "N° de série "+sh.Serial)
	}
	control := "Contrôlé le " + sh.Date
	if sh.Tech != "" {
		control += " par " + sh.Tech
	}
	p.style(9.5, false, inkMuted)
	p.drawLines(x, y, w-gradeW-4, strings.Join(append(meta, control), " · "))
	y += p.lineH() * 1.2

	// La note : une lettre si le verdict la donne
	gx := x + w - gradeW
	c := inkMuted
	switch sh.Kind {
	case verdictOK:
		c = inkOK
	case verdictBad:
		c = inkFail
	}
	gy := pageMargin
	if sh.Grade != "" {
		p.style(8, true, inkMuted)
		p.text(gx, gy, gradeW, "R", "GRADE", 5)
		p.style(38, true, inkText)
		p.text(gx, gy+4, gradeW, "R", sh.Grade, 15)
		gy += 20
	}
	p.style(11, true, c)
	p.text(gx, gy, gradeW, "R", sh.Verdict, 5.5)
	p.style(8.5, false, inkMuted)
	p.text(gx, gy+5.5, gradeW, "R", sh.Sub, 4.5)
	y = max(y, gy+11)

	p.SetDrawColor(inkText.r, inkText.g, inkText.b)
	p.SetLineWidth(0.6)
	p.Line(x, y+1.5, x+w, y+1.5)
	return y + 5
}

// drawSpecs l'essentiel du produit, comme sur une étiquette de rayon
func (p *sheetPDF) drawSpecs(y float64, specs []sheetSpec) float64 {
	if len(specs) == 0 {
		return y
	}
	const cols = 3
	w := (pageW - 2*pageMargin - 2*6) / cols
	for i := 0; i < len(specs); i += cols {
		rowH := 0.0
		for j := i; j < min(i+cols, len(specs)); j++ {
			x := pageMargin + float64(j-i)*(w+6)
			size := 3.2 * p.scale
			drawIcon(p.Fpdf, specs[j].Icon, x, y+(4.5*p.scale-size)/2, size, inkMuted)
			p.style(8, false, inkMuted)
			p.text(x+size+1.2, y, w, "L", specs[j].Label, 4.5*p.scale)
			p.style(12, true, inkText)
			lh := 5.4 * p.scale
			lines := p.SplitText(specs[j].Value, w)
			for k, l := range lines {
				p.text(x, y+4.5*p.scale+float64(k)*lh, w, "L", l, lh)
			}
			rowH = max(rowH, 4.5*p.scale+float64(len(lines))*lh)
		}
		y += rowH + 2.5*p.scale
	}
	return y + 1
}

func (p *sheetPDF) notesHeight(sh ficheSheet, w float64) float64 {
	if sh.Notes == "" {
		return 0
	}
	p.style(9, false, inkText)
	return 6.2*p.scale + float64(len(p.SplitText(sh.Notes, w)))*p.lineH() + 3
}

func addFonts(pdf *fpdf.Fpdf) {
	pdf.AddUTF8FontFromBytes("exo", "", exoRegular)
	pdf.AddUTF8FontFromBytes("exo", "B", exoBold)
}

// measure hauteur totale de la fiche à une échelle donnée
func measureSheet(sh ficheSheet, scale float64) (float64, []float64) {
	pdf := fpdf.New("P", "mm", "A4", "")
	addFonts(pdf)
	pdf.AddPage()
	p := &sheetPDF{pdf, scale}
	w := pageW - 2*pageMargin
	y := p.drawHeader(sh)
	y = p.drawSpecs(y, sh.Specs)
	cw := (w - colGap) / 2
	var heights []float64
	for _, sec := range sh.Sections {
		heights = append(heights, p.sectionHeight(sec, cw))
	}
	k := splitColumns(heights)
	var left, right float64
	for i, h := range heights {
		if i < k {
			left += h
		} else {
			right += h
		}
	}
	return y + max(left, right) + p.notesHeight(sh, w), heights
}

// writeSheet compose la fiche : la plus grande échelle qui tient sur une page
func writeSheet(w io.Writer, sh ficheSheet) error {
	scale := 1.2
	bottom := pageH - pageMargin
	for scale > 0.7 {
		if h, _ := measureSheet(sh, scale); h <= bottom {
			break
		}
		scale -= 0.025
	}
	_, heights := measureSheet(sh, scale)

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Fiche "+sh.Name, true)
	pdf.SetCreator("gobox", true)
	pdf.SetCellMargin(0)
	pdf.SetAutoPageBreak(false, 0)
	addFonts(pdf)
	pdf.AddPage()
	p := &sheetPDF{pdf, scale}
	full := pageW - 2*pageMargin
	y := p.drawHeader(sh)
	y = p.drawSpecs(y, sh.Specs)

	cw := (full - colGap) / 2
	k := splitColumns(heights)
	colY := [2]float64{y, y}
	for i, sec := range sh.Sections {
		c := 0
		if i >= k {
			c = 1
		}
		if colY[c]+heights[i] > bottom+0.01 { // repli : le texte le plus petit ne tient pas
			pdf.AddPage()
			colY = [2]float64{pageMargin, pageMargin}
		}
		colY[c] += p.drawSection(pageMargin+float64(c)*(cw+colGap), colY[c], cw, sec)
	}
	if sh.Notes != "" {
		ny := max(colY[0], colY[1])
		if ny+p.notesHeight(sh, full) > bottom+0.01 {
			pdf.AddPage()
			ny = pageMargin
		}
		ny += p.drawSectionTitle(pageMargin, ny, full, "", "Observations du technicien")
		p.style(9, false, inkText)
		p.drawLines(pageMargin, ny, full, sh.Notes)
	}
	return pdf.Output(w)
}

// fichePDF la fiche finale, en mémoire
func fichePDF(s *session) ([]byte, error) {
	var b bytes.Buffer
	if err := writeSheet(&b, newFicheSheet(s)); err != nil {
		return nil, fmt.Errorf("composition du PDF : %w", err)
	}
	return b.Bytes(), nil
}
