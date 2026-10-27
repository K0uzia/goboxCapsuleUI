package gui

import (
	"context"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// printFiche génère la fiche A4 (HTML) et l'ouvre dans le navigateur,
// d'où le technicien l'imprime ou l'enregistre en PDF.
func printFiche(s *session) (string, error) {
	path := filepath.Join(os.TempDir(), "gobox-"+s.Ref+".html")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := ficheTemplate.Execute(f, newPrintData(s)); err != nil {
		return "", err
	}
	_ = f.Chmod(0o644) // lisible par le navigateur de l'utilisateur
	return path, userCommand(context.Background(), "xdg-open", path).Start()
}

type printRow struct {
	Label, Status, Class string
}

type printData struct {
	Ref, Date, Tech, Duration string
	Machine, Serial           string
	Aspect                    string
	CleanIn, CleanOut         bool
	Verdict, VerdictSub       string
	VerdictClass              string // "" (en cours), ok, bad
	Video, Audio, System      []printRow
	Maintenance               []printRow
	Notes                     string
	Inventory                 []Component
	Generated                 string
}

func newPrintData(s *session) printData {
	rows := func(ctrls []ctrlRow) []printRow {
		var out []printRow
		for _, c := range ctrls {
			st := s.status(c.TestID)
			out = append(out, printRow{c.Label, st.style().label, statusClass(st)})
		}
		return out
	}
	v := s.verdict()
	fm := s.Form
	now := time.Now()
	return printData{
		Ref: s.Ref, Date: now.Format("02/01/2006"), Tech: orDash(fm.Tech), Duration: s.elapsed(),
		Machine: s.Machine.Name(), Serial: orDash(s.Machine.Serial),
		Aspect: fm.Aspect, CleanIn: fm.CleanIn, CleanOut: fm.CleanOut,
		Verdict: v.Label, VerdictSub: v.Sub, VerdictClass: verdictClass(v),
		Video: rows(ctrlVideo), Audio: rows(ctrlAudio), System: rows(ctrlSystem),
		Maintenance: []printRow{
			{"Temp. CPU avant stress", orDash(s.data("stress", "before")), ""},
			{"Temp. CPU après stress", orDash(s.data("stress", "after")), ""},
			{"Pâte thermique changée", fm.Paste, ""},
			{"Shred des disques durs", fm.Shred, ""},
			{"Chargeur en bon état", fm.Charger, ""},
			{"Pile setup changée", fm.CMOS, ""},
			{"Lecteur DVD", fm.DVD, ""},
			{"Ports USB défectueux", fm.USBBad, ""},
			{"Sortie vidéo autre", orDash(fm.OtherVideo), ""},
			{"Autre", orDash(fm.Other), ""},
		},
		Notes:     strings.TrimSpace(fm.Notes),
		Inventory: s.Inventory,
		Generated: now.Format("02/01/2006 à 15:04"),
	}
}

func verdictClass(v verdict) string {
	switch v.Color {
	case colTeal:
		return "ok"
	case colRed:
		return "bad"
	}
	return ""
}

func statusClass(s status) string {
	switch s {
	case statusOK:
		return "ok"
	case statusWarn:
		return "warn"
	case statusFail:
		return "fail"
	}
	return "na"
}

var ficheTemplate = template.Must(template.New("fiche").Parse(`<!doctype html>
<html lang="fr"><head><meta charset="utf-8"><title>Fiche {{.Ref}}</title>
<style>
@page { size: A4; margin: 0 }
body { margin: 0; background: #fff; font-family: Inter, "Segoe UI", system-ui, sans-serif; color: #1B1D22; font-size: 10.5px; line-height: 1.35 }
.page { width: 794px; min-height: 1123px; box-sizing: border-box; padding: 36px 40px 30px; display: flex; flex-direction: column; gap: 12px; margin: auto }
.head { display: flex; justify-content: space-between; padding-bottom: 10px; border-bottom: 2px solid #1B1D22 }
.kicker { font-size: 9px; font-weight: 600; letter-spacing: .14em; color: #0F766E }
h1 { margin: 0; font-size: 21px }
.muted { color: #5B6069 }
.meta { display: grid; grid-template-columns: auto auto; gap: 2px 12px; text-align: right; font-size: 10px }
.mono { font-family: "JetBrains Mono", ui-monospace, monospace }
.cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px }
.card { border: 1px solid #D5D8DE; border-radius: 5px; padding: 7px 9px; display: flex; flex-direction: column; gap: 1px }
.card .cap { font-size: 8.5px; font-weight: 600; letter-spacing: .1em; color: #5B6069 }
.card b { font-size: 11.5px }
.card.verdict { border: 1.5px solid #4B5059; background: #F3F4F6 }
.card.verdict.ok { border-color: #0F766E; background: #F0FAF8 }
.card.verdict.ok .cap, .card.verdict.ok b { color: #0F766E }
.card.verdict.bad { border-color: #B91C1C; background: #FEF2F2 }
.card.verdict.bad .cap, .card.verdict.bad b { color: #B91C1C }
.cols { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0 16px }
h2 { margin: 0 0 3px; font-size: 9px; font-weight: 600; letter-spacing: .12em; color: #5B6069; padding-bottom: 4px; border-bottom: 1px solid #1B1D22; text-transform: uppercase }
.row { display: flex; justify-content: space-between; align-items: center; padding: 3px 0; border-bottom: 1px solid #ECEEF1 }
.pill { font-size: 8.5px; font-weight: 600; letter-spacing: .06em; padding: 1px 7px; border-radius: 9px; background: #EEF0F3; color: #4B5059 }
.pill.ok { background: #E6F4F1; color: #0F766E } .pill.warn { background: #FDF3E1; color: #92400E } .pill.fail { background: #FEE2E2; color: #991B1B }
.inv { display: grid; grid-template-columns: 1fr 1fr; gap: 0 20px }
.kv { display: grid; grid-template-columns: 110px 1fr; gap: 3px 8px; align-content: start }
.kv .sec { grid-column: 1 / -1; font-size: 9px; font-weight: 600; letter-spacing: .08em; color: #5B6069; padding-top: 6px; text-transform: uppercase }
.foot { margin-top: auto; display: flex; justify-content: space-between; align-items: flex-end; padding-top: 10px; border-top: 1px solid #D5D8DE; font-size: 9px; color: #5B6069 }
.sign { width: 190px; display: flex; flex-direction: column; gap: 22px; font-weight: 600; letter-spacing: .1em }
.sign div { border-bottom: 1px solid #1B1D22 }
@media print { .page { margin: 0 } }
</style></head><body><div class="page">

<div class="head">
  <div><div class="kicker">GOBOX · ATELIER DE RECONDITIONNEMENT</div><h1>Fiche de reconditionnement</h1>
  <div class="muted">Générée automatiquement · La Capsule</div></div>
  <div class="meta"><span class="muted">Référence</span><span class="mono">{{.Ref}}</span>
  <span class="muted">Fait le</span><b>{{.Date}}</b><span class="muted">Fait par</span><b>{{.Tech}}</b>
  <span class="muted">Durée</span><b>{{.Duration}}</b></div>
</div>

<div class="cards">
  <div class="card"><span class="cap">MACHINE</span><b>{{.Machine}}</b><span class="mono">S/N {{.Serial}}</span></div>
  <div class="card"><span class="cap">ASPECT</span><b>{{.Aspect}}</b></div>
  <div class="card"><span class="cap">NETTOYAGE</span><b>Intérieur {{if .CleanIn}}☑{{else}}☐{{end}} · Extérieur {{if .CleanOut}}☑{{else}}☐{{end}}</b></div>
  <div class="card verdict {{.VerdictClass}}"><span class="cap">VERDICT</span><b>{{.Verdict}}</b><span>{{.VerdictSub}}</span></div>
</div>

<div class="cols">
  <div><h2>Sorties vidéo &amp; ports</h2>{{range .Video}}<div class="row"><span>{{.Label}}</span><span class="pill {{.Class}}">{{.Status}}</span></div>{{end}}</div>
  <div><h2>Saisie &amp; audio</h2>{{range .Audio}}<div class="row"><span>{{.Label}}</span><span class="pill {{.Class}}">{{.Status}}</span></div>{{end}}</div>
  <div><h2>Système &amp; stockage</h2>{{range .System}}<div class="row"><span>{{.Label}}</span><span class="pill {{.Class}}">{{.Status}}</span></div>{{end}}</div>
</div>

<div><h2>Stress test &amp; maintenance</h2><div class="cols">{{range .Maintenance}}<div class="row"><span>{{.Label}}</span><b>{{.Status}}</b></div>{{end}}</div></div>

{{if .Notes}}<div><h2>Observations du technicien</h2><p style="margin:0">{{.Notes}}</p></div>{{end}}

<div><h2>Inventaire matériel détecté</h2><div class="inv"><div class="kv">
{{range $i, $c := .Inventory}}{{if eq $i 3}}</div><div class="kv">{{end}}
  <span class="sec">{{$c.Name}}</span>
  {{if $c.Err}}<span class="muted">Erreur</span><span>{{$c.Err}}</span>{{else}}
  {{range $c.Fields}}<span class="muted">{{.Label}}</span><span>{{.Value}}</span>{{else}}<span class="muted">État</span><span>{{$c.Summary}}</span>{{end}}{{end}}
{{end}}</div></div></div>

<div class="foot">
  <div>Fiche générée le {{.Generated}} · {{.Ref}} · {{.Machine}} · S/N {{.Serial}}<br>
  Les tests marqués OK ont été exécutés par gobox ; les contrôles visuels et la maintenance sont saisis par le technicien.</div>
  <div class="sign">SIGNATURE TECHNICIEN<div></div></div>
</div>
</div></body></html>`))
