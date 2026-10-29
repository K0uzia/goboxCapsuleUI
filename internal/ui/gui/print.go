package gui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// writeFiche génère la fiche A4 (HTML), lisible par le navigateur de
// l'utilisateur même quand gobox tourne en root
func writeFiche(s *session) (string, error) {
	path := filepath.Join(os.TempDir(), "gobox-"+s.Ref+".html")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := ficheTemplate.Execute(f, newPrintData(s)); err != nil {
		return "", err
	}
	_ = f.Chmod(0o644)
	return path, nil
}

// openInBrowser ouvre un fichier dans le navigateur de la session : essaie
// xdg-open, gio puis Firefox, et rend l'erreur de chacun si aucun n'y arrive
func openInBrowser(path string) error {
	var errs []error
	for _, args := range [][]string{{"xdg-open", path}, {"gio", "open", path}, {"firefox", "--new-tab", path}} {
		if firstAvailable(args[0]) == "" {
			continue
		}
		var out bytes.Buffer
		cmd := userCommand(context.Background(), args[0], args[1:]...)
		cmd.Stdout, cmd.Stderr = &out, &out
		if err := cmd.Start(); err != nil {
			errs = append(errs, fmt.Errorf("%s : %w", args[0], err))
			continue
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			if err == nil {
				return nil
			}
			errs = append(errs, fmt.Errorf("%s : %v %s", args[0], err, strings.TrimSpace(out.String())))
		case <-time.After(4 * time.Second):
			return nil // le navigateur tourne : il a été lancé par cette commande
		}
	}
	if len(errs) == 0 {
		return errors.New("aucun outil pour ouvrir un navigateur (xdg-open, gio, firefox)")
	}
	return errors.Join(errs...)
}

// ---- Données de la fiche imprimée ------------------------------------------

// printBox case de la fiche papier
type printBox struct {
	Label string
	On    bool
}

// printCheck ligne de contrôle : cases, et précision éventuelle
type printCheck struct {
	Label  string
	Boxes  []printBox
	Absent bool   // composant absent : « absent » à la place des cases
	Note   string // « 1 / 3 testé(s) », mesure…
}

type printSection struct {
	Name   string
	Icon   template.HTML
	Fields []Field
	Err    string
}

type printData struct {
	Ref, Date, Tech, Duration string
	Machine, Serial           string
	CleanIn, CleanOut         bool
	Aspect                    []printBox
	Verdict, VerdictSub       string
	VerdictClass              string // "" (en cours), ok, bad
	Video, Audio              []printCheck
	Temps, Maintenance        []printCheck
	USBBad                    string
	Gobox                     []printRow
	Notes                     string
	Inventory                 []printSection
	Generated                 string
}

type printRow struct {
	Label, Status, Class, Detail string
}

func newPrintData(s *session) printData {
	fm := s.Form
	checks := func(items []ctrlItem) []printCheck {
		var out []printCheck
		for _, it := range items {
			st, v := s.itemState(it)
			c := printCheck{Label: it.Label}
			if it.Free && fm.Texts[it.Label] != "" {
				c.Label += " : " + fm.Texts[it.Label]
			}
			for _, x := range it.Scale {
				c.Boxes = append(c.Boxes, printBox{scaleLabel(x), st == x})
			}
			switch {
			case st == statusNA:
				c.Absent = true
			case v != "":
				c.Note = v + " testé(s)"
			}
			out = append(out, c)
		}
		return out
	}
	yesNo := func(label, v string, opts []string) printCheck {
		c := printCheck{Label: label}
		for _, o := range opts {
			c.Boxes = append(c.Boxes, printBox{o, v == o})
		}
		return c
	}
	temp := func(label, key string) printCheck {
		v := s.data("stress", key)
		_, below, ok := stressTemp(v)
		c := printCheck{Label: label, Note: orUnknown(v, notMeasured), Boxes: []printBox{
			{fmt.Sprintf("moins de %.0f °C", stressFailTemp), ok && below},
			{fmt.Sprintf("plus de %.0f °C", stressFailTemp), ok && !below},
		}}
		return c
	}
	var maint []printCheck
	for _, m := range maintRows {
		maint = append(maint, yesNo(m.Label, *m.Value(&fm), m.Opts))
	}
	var gobox []printRow
	for _, it := range ctrlGobox {
		st := s.status(it.TestID)
		gobox = append(gobox, printRow{it.Label, st.style().label, statusClass(st), s.Results[it.TestID].Detail})
	}
	var inv []printSection
	for _, c := range append([]Component{machineComponent(s.Machine)}, s.Inventory...) {
		sec := printSection{Name: c.Name, Icon: inlineIcon(c.Icon), Fields: c.Fields}
		if len(sec.Fields) == 0 && c.Summary != "" {
			sec.Fields = []Field{{"État", c.Summary}}
		}
		if c.Err != nil {
			sec.Err = c.Err.Error()
		}
		inv = append(inv, sec)
	}
	v := s.verdict()
	now := time.Now()
	return printData{
		Ref: s.Ref, Date: now.Format("02/01/2006"), Tech: orUnknown(fm.Tech, "non renseigné"), Duration: s.elapsed(),
		Machine: s.Machine.Name(), Serial: orUnknown(s.Machine.Serial, "non lu"),
		CleanIn: fm.CleanIn, CleanOut: fm.CleanOut,
		Aspect:  []printBox{{"neuf", fm.Aspect == "Neuf"}, {"occasion", fm.Aspect == "Occasion"}, {"nul", fm.Aspect == "Nul"}},
		Verdict: v.Label, VerdictSub: v.Sub, VerdictClass: verdictClass(v),
		Video: checks(ctrlVideo), Audio: checks(ctrlAudio),
		Temps:       []printCheck{temp("Température processeur avant stress test", "before"), temp("Température processeur après stress test", "after")},
		Maintenance: maint, USBBad: orUnknown(strings.TrimSpace(fm.USBBad), "0"),
		Gobox:     gobox,
		Notes:     strings.TrimSpace(fm.Notes),
		Inventory: inv,
		Generated: now.Format("02/01/2006 à 15:04"),
	}
}

// inlineIcon icône SVG insérée dans la page (aucun fichier externe)
func inlineIcon(name string) template.HTML {
	if name == "" {
		return ""
	}
	b, err := iconFiles.ReadFile("icons/" + name + ".svg")
	if err != nil {
		return ""
	}
	return template.HTML(b) // fichier intégré au binaire, pas une saisie
}

func verdictClass(v verdict) string {
	switch v.Kind {
	case verdictOK:
		return "ok"
	case verdictBad:
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

// La fiche imprimée reprend l'ordre de la fiche papier de l'atelier :
// en-tête, contrôles cochés, températures et maintenance, puis l'inventaire
// du matériel par sections.
var ficheTemplate = template.Must(template.New("fiche").Parse(`<!doctype html>
<html lang="fr"><head><meta charset="utf-8"><title>Fiche {{.Ref}}</title>
<style>
@page { size: A4; margin: 12mm 12mm 12mm }
* { box-sizing: border-box }
body { margin: 0; background: #fff; font-family: Inter, "Segoe UI", system-ui, sans-serif; color: #1B1D22; font-size: 10.5px; line-height: 1.35 }
.page { max-width: 186mm; margin: auto; display: flex; flex-direction: column; gap: 9px }
.head { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; padding-bottom: 7px; border-bottom: 2px solid #1B1D22 }
h1 { margin: 0; font-size: 18px }
.muted { color: #5B6069 }
.mono { font-family: "JetBrains Mono", ui-monospace, monospace }
.fill { display: inline-block; min-width: 160px; border-bottom: 1px solid #1B1D22; font-weight: 600; font-size: 12px; padding: 0 4px }
.line { display: flex; flex-wrap: wrap; gap: 4px 22px; align-items: center }
.boxes { flex-shrink: 0; white-space: nowrap; text-align: right }
.absent { flex-shrink: 0; font-style: italic; color: #5B6069; padding-right: 2px }
.box { display: inline-flex; align-items: center; gap: 4px; margin-left: 9px; white-space: nowrap }
.box i { width: 11px; height: 11px; border: 1.2px solid #1B1D22; border-radius: 2px; display: inline-flex; align-items: center; justify-content: center; font-style: normal; font-size: 9px; font-weight: 700; line-height: 1 }
.box.on i { background: #1B1D22; color: #fff }
.box.on { font-weight: 600 }
.panel { border: 1px solid #C9CDD4; border-radius: 6px; padding: 7px 10px }
.cols { display: grid; grid-template-columns: 1fr 1fr; gap: 0 22px }
.row { display: flex; justify-content: space-between; align-items: center; gap: 8px; padding: 2.5px 0; border-bottom: 1px solid #ECEEF1 }
.row:last-child { border-bottom: 0 }
.row .note { color: #5B6069; font-size: 9.5px; margin-left: 6px }
.sec { display: flex; align-items: center; gap: 8px; margin: 2px 0 3px; font-size: 9px; font-weight: 700; letter-spacing: .12em; text-transform: uppercase; color: #3D424B }
.sec::after { content: ""; flex: 1; border-top: 1px dashed #9AA0A8 }
.sec svg { width: 12px; height: 12px; fill: #3D424B }
.inv { display: grid; grid-template-columns: 1fr 1fr; gap: 6px 22px }
.kv { display: grid; grid-template-columns: 104px 1fr; gap: 1px 8px; break-inside: avoid }
.kv span:nth-child(odd) { color: #5B6069 }
.pill { font-size: 8.5px; font-weight: 700; letter-spacing: .06em; padding: 1px 7px; border-radius: 9px; background: #EEF0F3; color: #4B5059 }
.pill.ok { background: #E6F4F1; color: #0F766E } .pill.warn { background: #FDF3E1; color: #92400E } .pill.fail { background: #FEE2E2; color: #991B1B }
.verdict { display: flex; justify-content: space-between; align-items: center; border: 1.5px solid #4B5059; background: #F3F4F6; border-radius: 6px; padding: 7px 12px }
.verdict b { font-size: 14px }
.verdict.ok { border-color: #0F766E; background: #F0FAF8; color: #0F766E }
.verdict.bad { border-color: #B91C1C; background: #FEF2F2; color: #B91C1C }
.foot { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; padding-top: 8px; border-top: 1px solid #C9CDD4; font-size: 9px; color: #5B6069 }
.avoid { break-inside: avoid }
</style></head><body><div class="page">

<div class="head">
  <div><div class="muted">gobox · La Capsule · <span class="mono">{{.Ref}}</span></div><h1>Fiche de reconditionnement</h1></div>
  <div class="line"><span>Fait le <b>{{.Date}}</b></span><span>Reconditionnement fait par : <span class="fill">{{.Tech}}</span></span></div>
</div>

<div class="line">
  <span>Nettoyage intérieur<span class="box {{if .CleanIn}}on{{end}}"><i>{{if .CleanIn}}✓{{end}}</i></span></span>
  <span>Nettoyage extérieur<span class="box {{if .CleanOut}}on{{end}}"><i>{{if .CleanOut}}✓{{end}}</i></span></span>
  <span>Aspect{{range .Aspect}}<span class="box {{if .On}}on{{end}}">{{.Label}} <i>{{if .On}}✓{{end}}</i></span>{{end}}</span>
</div>

{{define "checks"}}{{range .}}<div class="row"><span>{{.Label}}{{if .Note}}<span class="note">{{.Note}}</span>{{end}}</span>
{{if .Absent}}<span class="absent">absent</span>{{else}}<span class="boxes">{{range .Boxes}}<span class="box {{if .On}}on{{end}}">{{.Label}} <i>{{if .On}}✓{{end}}</i></span>{{end}}</span>{{end}}</div>{{end}}{{end}}

<div class="panel cols">
  <div>{{template "checks" .Video}}<div class="row"><span>Ports USB défectueux (nombre)</span><b>{{.USBBad}}</b></div></div>
  <div>{{template "checks" .Audio}}</div>
</div>

<div class="panel cols">
  <div>{{template "checks" .Temps}}{{template "checks" (slice .Maintenance 0 2)}}</div>
  <div>{{template "checks" (slice .Maintenance 2)}}</div>
</div>

<div class="panel avoid"><div class="sec">Tests gobox</div><div class="cols">
{{range .Gobox}}<div class="row"><span>{{.Label}}{{if .Detail}}<span class="note">{{.Detail}}</span>{{end}}</span><span class="pill {{.Class}}">{{.Status}}</span></div>{{end}}
</div></div>

<div class="verdict {{.VerdictClass}}"><span>VERDICT · <b>{{.Verdict}}</b></span><span>{{.VerdictSub}}</span></div>

{{if .Notes}}<div class="panel avoid"><div class="sec">Observations du technicien</div><div style="white-space:pre-wrap">{{.Notes}}</div></div>{{end}}

<div class="inv">
{{range .Inventory}}<div class="avoid"><div class="sec">{{.Icon}}{{.Name}}</div><div class="kv">
{{if .Err}}<span>Erreur</span><span>{{.Err}}</span>{{end}}{{range .Fields}}<span>{{.Label}}</span><span>{{.Value}}</span>{{end}}
</div></div>{{end}}
</div>

<div class="foot">
  <div>Fiche générée le {{.Generated}} · {{.Ref}} · {{.Machine}} · S/N {{.Serial}} · durée {{.Duration}}<br>
  Les lignes reliées à un test ont été mesurées par gobox ; les autres sont saisies par le technicien.</div>
</div>
</div></body></html>`))
