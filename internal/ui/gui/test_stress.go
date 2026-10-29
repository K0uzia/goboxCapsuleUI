package gui

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// Stress test CPU, sur le modèle fonctionnel d'OCCT : réglages, suivi en
// direct (températures, fréquences, charge, consommation, erreurs de calcul),
// arrêt automatique sur seuil, verdict expliqué critère par critère.
const (
	stressWarnTemp = 85.0 // réserve : pâte thermique ou ventilation à vérifier
	stressFailTemp = 90.0 // échec, et arrêt immédiat de la charge
	stressInterval = 2 * time.Second
	// throttling : fréquence sous 85 % du maximum observé en charge, sur
	// 3 relevés de suite, après 20 s de chauffe
	stressThrottleRatio   = 0.85
	stressThrottleSamples = 3
	stressWarmup          = 20 * time.Second
)

var stressDurations = []struct {
	label string
	d     time.Duration
}{{"3 min", 3 * time.Minute}, {"10 min", 10 * time.Minute}, {"30 min", 30 * time.Minute}, {"1 h", time.Hour}}

// stat minimum, moyenne et maximum d'une mesure
type stat struct {
	min, max, sum float64
	n             int
}

func (s *stat) add(v float64) {
	if s.n == 0 || v < s.min {
		s.min = v
	}
	if s.n == 0 || v > s.max {
		s.max = v
	}
	s.sum += v
	s.n++
}

func (s *stat) avg() float64 {
	if s.n == 0 {
		return 0
	}
	return s.sum / float64(s.n)
}

func runStress(t *testCtx) result {
	if simulated {
		t.markSimulated()
		t.log("Données simulées : aucune charge réelle, mesures d'une machine fictive")
	}

	// 1. Réglages
	duration, variable, ok := stressSettings(t)
	if !ok {
		return result{}
	}

	// 2. Suivi en direct
	sensors := newStressSensors()
	first := sensors.sample()
	before := first.Temp
	if first.HasTemp {
		t.log("Température au repos : %.0f °C", before)
	} else {
		t.log("Pas de capteur de température : critère de température non vérifiable")
	}
	mon := newStressMonitor(len(first.CoreLoad), first.HasPower)
	t.panel(func() fyne.CanvasObject { return mon.root })

	threads := runtime.NumCPU()
	mode := "constante"
	if variable {
		mode = "variable"
	}
	t.log("Charge %s sur %d threads pendant %s", mode, threads, formatClock(duration))
	ctx, stop := context.WithCancel(t.ctx)
	defer stop()
	var errs atomic.Int64
	if sensors.burns() {
		want := burnChunk(burnSeed)
		for i := 0; i < threads; i++ {
			go burn(ctx, variable, want, &errs)
		}
	}

	start := time.Now()
	var temp, freq, load, power stat
	maxFreq, slowRun, throttled := 0.0, 0, false
	stopReason := ""
	for time.Since(start) < duration {
		if !t.sleep(stressInterval) {
			return result{}
		}
		elapsed := time.Since(start)
		s := sensors.sample()
		if s.HasTemp {
			temp.add(s.Temp)
		}
		freq.add(s.Freq)
		load.add(s.Load)
		if s.HasPower {
			power.add(s.Power)
		}
		nErr := errs.Load()

		// throttling : la fréquence chute sous charge
		if s.Load > 90 && elapsed >= stressWarmup {
			if s.Freq < stressThrottleRatio*maxFreq {
				slowRun++
			} else {
				slowRun = 0
			}
			if slowRun >= stressThrottleSamples && !throttled {
				throttled = true
				t.log("Throttling : %s contre %s au maximum", ghz(s.Freq), ghz(maxFreq))
			}
		}
		if s.Load > 90 {
			maxFreq = max(maxFreq, s.Freq)
		}

		t.ui(func() { mon.update(s, &temp, &freq, &load, &power, nErr, elapsed, duration) })
		t.progress(elapsed.Seconds() / duration.Seconds())
		if int(elapsed/stressInterval)%15 == 0 {
			t.log("%s · %.0f °C · %s · charge %.0f %%", formatClock(elapsed), s.Temp, ghz(s.Freq), s.Load)
		}

		if s.HasTemp && s.Temp >= stressFailTemp {
			stopReason = fmt.Sprintf("arrêt à %.0f °C", s.Temp)
			t.log("Seuil de %.0f °C atteint : arrêt immédiat de la charge", stressFailTemp)
			break
		}
		if nErr > 0 {
			stopReason = "arrêt sur erreur de calcul"
			t.log("Erreur de calcul détectée : arrêt de la charge")
			break
		}
	}
	stop()
	t.log("Arrêt de la charge")

	// 3. Verdict
	v := stressVerdict{
		hasTemp: first.HasTemp, before: before, maxTemp: temp.max,
		errors: errs.Load(), throttled: throttled, ran: time.Since(start), planned: duration, stopReason: stopReason,
		avgFreq: freq.avg(), maxFreq: maxFreq,
	}
	r := v.result()
	t.panel(func() fyne.CanvasObject { return v.summary(mon.tempChart) })
	return r
}

// stressSettings écran de réglages ; ok faux si le test est annulé
func stressSettings(t *testCtx) (time.Duration, bool, bool) {
	start := make(chan struct{})
	durLabels := make([]string, len(stressDurations))
	for i, d := range stressDurations {
		durLabels[i] = d.label
	}
	dur := newSegmented(durLabels, durLabels[0], 80, nil)
	load := newSegmented([]string{"Constante", "Variable"}, "Constante", 110, nil)

	t.panel(func() fyne.CanvasObject {
		launch := primaryButton("Lancer le stress test", theme.MediaPlayIcon(), func() { close(start) })
		field := func(ico, label string, seg *segmented) fyne.CanvasObject {
			head := container.NewHBox(iconOf(ico, cMuted, 13), container.NewCenter(caption(label)))
			return container.NewVBox(head, seg.root)
		}
		return box(container.NewVBox(
			field("clock", "Durée", dur),
			field("gauge-high", "Charge", load),
			mutedNote("Constante : 100 % en continu. Variable : alterne pleine charge et demi-charge, utile pour révéler une alimentation instable."),
			mutedNote(fmt.Sprintf("Réserve à %.0f °C ou en cas de throttling ; échec et arrêt à %.0f °C ou à la première erreur de calcul.", stressWarnTemp, stressFailTemp)),
			container.NewHBox(launch),
		), cLine, cSurface2)
	})
	select {
	case <-start:
	case <-t.ctx.Done():
		return 0, false, false
	}
	d := stressDurations[0].d
	for _, x := range stressDurations {
		if x.label == dur.Selected {
			d = x.d
		}
	}
	return d, load.Selected == "Variable", true
}

// ---- Écran de suivi -------------------------------------------------------

type stressMonitor struct {
	root                         fyne.CanvasObject
	temp, freq, load, power, err *tText
	tempSub, freqSub, loadSub    *tText
	powerSub                     *tText
	clock                        *tText
	tempChart, freqChart         *lineChart
	cores                        []*coreTile
}

type coreTile struct {
	bar  *thinBar
	freq *tText
}

func newStressMonitor(nCores int, hasPower bool) *stressMonitor {
	m := &stressMonitor{}
	tile := func(ico, label string) (*tText, *tText, fyne.CanvasObject) {
		v := text("…", 22, true, cText)
		v.Mono = true
		sub := text("", 11, false, cMuted)
		head := container.NewHBox(iconOf(ico, cMuted, 13), container.NewCenter(caption(label)))
		return v, sub, box(container.NewVBox(head, v, sub), cLine, cSurface2)
	}
	var a, b, c, d, e fyne.CanvasObject
	m.temp, m.tempSub, a = tile("temperature-half", "Température")
	m.freq, m.freqSub, b = tile("gauge-high", "Fréquence")
	m.load, m.loadSub, c = tile("microchip", "Charge")
	m.power, m.powerSub, d = tile("bolt", "Consommation")
	var errSub *tText
	m.err, errSub, e = tile("bug", "Erreurs")
	errSub.Text = "calculs vérifiés"
	tiles := []fyne.CanvasObject{a, b, c, e}
	if hasPower {
		tiles = []fyne.CanvasObject{a, b, c, d, e}
	}

	m.tempChart = newLineChart(30, 100, "°C", cAccent)
	m.tempChart.Thresholds = []threshold{{stressWarnTemp, cWarning}, {stressFailTemp, cDanger}}
	m.freqChart = newLineChart(0, 4, "GHz", cInfo)
	m.freqChart.Height = 90

	coreObjs := make([]fyne.CanvasObject, nCores)
	for i := range coreObjs {
		ct := &coreTile{bar: newThinBar(cAccent, 4), freq: text("…", 11, false, cMuted)}
		ct.freq.Mono = true
		m.cores = append(m.cores, ct)
		coreObjs[i] = box(container.NewVBox(
			container.NewBorder(nil, nil, text(fmt.Sprintf("Cœur %d", i), 11, true, cText), ct.freq), ct.bar.root,
		), cLine, cSurface2)
	}

	m.clock = text("", 13, true, cText)
	m.clock.Mono = true
	m.root = container.NewVBox(
		container.NewBorder(nil, nil, caption("Suivi en direct"), m.clock),
		container.New(&respGrid{minCol: 110, cols: []int{len(tiles), 3, 2}, gap: 8}, tiles...),
		caption("Température"), m.tempChart,
		caption("Fréquence moyenne"), m.freqChart,
		caption("Par cœur"), container.New(&respGrid{minCol: 120, cols: []int{4, 2}, gap: 6}, coreObjs...),
	)
	return m
}

func (m *stressMonitor) update(s stressSample, temp, freq, load, power *stat, nErr int64, elapsed, total time.Duration) {
	if s.HasTemp {
		m.temp.SetText(fmt.Sprintf("%.0f °C", s.Temp))
		m.temp.Role = cText
		if s.Temp >= stressWarnTemp {
			m.temp.Role = cWarning
		}
		m.tempSub.SetText(fmt.Sprintf("max %.0f °C", temp.max))
		m.tempChart.add(s.Temp)
	} else {
		m.temp.SetText("non lue")
	}
	m.freq.SetText(ghz(s.Freq))
	m.freqSub.SetText("moy " + ghz(freq.avg()))
	m.freqChart.Max = math.Ceil(max(m.freqChart.Max, s.Freq+0.3))
	m.freqChart.add(s.Freq)
	m.load.SetText(fmt.Sprintf("%.0f %%", s.Load))
	m.loadSub.SetText(fmt.Sprintf("moy %.0f %%", load.avg()))
	if s.HasPower {
		m.power.SetText(fmt.Sprintf("%.0f W", s.Power))
		m.powerSub.SetText(fmt.Sprintf("moy %.0f W", power.avg()))
	}
	m.err.SetText(fmt.Sprintf("%d", nErr))
	m.err.Role = cSuccess
	if nErr > 0 {
		m.err.Role = cDanger
	}
	m.err.Refresh()
	for i, c := range m.cores {
		if i < len(s.CoreLoad) {
			c.bar.set(float32(s.CoreLoad[i] / 100))
		}
		if i < len(s.CoreFreq) {
			c.freq.SetText(ghz(s.CoreFreq[i]))
		}
	}
	m.clock.SetText(formatClock(elapsed) + " / " + formatClock(total))
}

// formatClock « 02:15 », « 1:02:15 »
func formatClock(d time.Duration) string {
	d = d.Round(time.Second)
	h, m, s := int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// ---- Verdict --------------------------------------------------------------

type stressVerdict struct {
	hasTemp          bool
	before, maxTemp  float64
	errors           int64
	throttled        bool
	ran, planned     time.Duration
	stopReason       string
	avgFreq, maxFreq float64
}

func (v stressVerdict) tempStatus() status {
	switch {
	case !v.hasTemp:
		return statusNA
	case v.maxTemp >= stressFailTemp:
		return statusFail
	case v.maxTemp >= stressWarnTemp:
		return statusWarn
	}
	return statusOK
}

func (v stressVerdict) result() result {
	st := statusOK
	switch {
	case v.errors > 0 || v.tempStatus() == statusFail:
		st = statusFail
	case v.throttled || v.tempStatus() == statusWarn || !v.hasTemp:
		st = statusWarn
	}
	detail := fmt.Sprintf("%d erreur(s) de calcul", v.errors)
	if v.hasTemp {
		detail = fmt.Sprintf("%.0f °C au repos, %.0f °C au maximum en %s, ", v.before, v.maxTemp, formatClock(v.ran)) + detail
	} else {
		detail = "Température non mesurée, " + detail
	}
	if v.throttled {
		detail += ", throttling"
	}
	if v.stopReason != "" {
		detail += " (" + v.stopReason + ")"
	}
	r := result{Status: st, Detail: detail}
	r.Advice = v.advice()
	if v.hasTemp {
		r.Data = map[string]string{
			"before": fmt.Sprintf("%.0f °C", v.before),
			"after":  fmt.Sprintf("%.0f °C", v.maxTemp),
		}
	}
	return r
}

// summary critères du verdict, puis la courbe complète de température
func (v stressVerdict) summary(chart *lineChart) fyne.CanvasObject {
	row := func(label, value string, s status) fyne.CanvasObject {
		p := newPill(s)
		return listRow(container.NewHBox(text(label, 13, false, cText), text(value, 13, true, cText)), container.NewCenter(p.root))
	}
	errSt := statusOK
	if v.errors > 0 {
		errSt = statusFail
	}
	temp := "non mesurée"
	if v.hasTemp {
		temp = fmt.Sprintf("%.0f °C (repos %.0f °C, seuils %.0f / %.0f °C)", v.maxTemp, v.before, stressWarnTemp, stressFailTemp)
	}
	thr, thrSt := "stable, "+ghz(v.avgFreq)+" en moyenne", statusOK
	if v.throttled {
		thr, thrSt = "baisse sous charge (max "+ghz(v.maxFreq)+")", statusWarn
	}
	dur, durSt := formatClock(v.ran)+" sur "+formatClock(v.planned), statusOK
	if v.stopReason != "" {
		dur, durSt = dur+", "+v.stopReason, statusFail
	}
	return container.NewVBox(
		caption("Critères"),
		row("Erreurs de calcul : ", fmt.Sprintf("%d", v.errors), errSt),
		row("Température max : ", temp, v.tempStatus()),
		row("Fréquence : ", thr, thrSt),
		row("Durée : ", dur, durSt),
		caption("Température pendant le test"), chart,
	)
}

// ---- Charge ---------------------------------------------------------------

const burnSeed = 0x9E3779B97F4A7C15

// burnChunk calcul déterministe : le même résultat est attendu à chaque
// passage ; une différence signale une erreur matérielle (instabilité)
func burnChunk(seed uint64) uint64 {
	x := seed
	f := 1.0
	for i := 0; i < 2_000_000; i++ {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		f = f*1.0000001 + float64(x&0xff)*1e-9
	}
	return x ^ math.Float64bits(f)
}

// burn occupe un cœur jusqu'à l'annulation et compte les résultats faux. En
// charge variable : 8 s à pleine charge, puis 4 s à demi-charge.
func burn(ctx context.Context, variable bool, want uint64, errs *atomic.Int64) {
	start := time.Now()
	for ctx.Err() == nil {
		if variable && time.Since(start)%(12*time.Second) >= 8*time.Second {
			t0 := time.Now()
			if burnChunk(burnSeed) != want {
				errs.Add(1)
			}
			time.Sleep(time.Since(t0)) // demi-charge
			continue
		}
		if burnChunk(burnSeed) != want {
			errs.Add(1)
		}
	}
}

// ghz fréquence à la française : « 3,38 GHz »
func ghz(v float64) string {
	return strings.Replace(fmt.Sprintf("%.2f GHz", v), ".", ",", 1)
}

// advice pistes d'amélioration selon ce qui a limité le résultat
func (v stressVerdict) advice() []string {
	var out []string
	if v.errors > 0 {
		out = append(out,
			"Erreur de calcul : instabilité matérielle. Retirer tout overclock, lancer le test mémoire, puis relancer le stress test.",
			"Si l'erreur revient avec une mémoire saine, le processeur ou la carte mère est en cause : machine à écarter de la vente.")
	}
	if v.hasTemp && v.maxTemp >= stressWarnTemp {
		out = append(out,
			"Changer la pâte thermique du processeur (sèche ou mal répartie au-delà de quelques années).",
			"Dépoussiérer le radiateur et le ventilateur à l'air sec, sans faire tourner le ventilateur à vide.",
			"Vérifier que le ventilateur tourne et accélère sous charge ; le remplacer s'il est bruyant ou bloqué.")
	}
	if v.throttled {
		out = append(out, "Baisse de fréquence sous charge : sur un portable, brancher le chargeur d'origine et passer le profil d'alimentation en performance.")
		if !v.hasTemp || v.maxTemp < stressWarnTemp {
			out = append(out, "Le throttling sans surchauffe vient souvent d'une limite de puissance : vérifier le chargeur (puissance en watts) et la batterie.")
		}
	}
	if !v.hasTemp {
		out = append(out, "Capteur de température illisible : mesurer avec OCCT sur la même machine pour valider le refroidissement.")
	}
	if len(out) > 0 {
		out = append(out, "Après l'intervention, relancer le stress test pour valider la machine.")
	}
	return out
}
