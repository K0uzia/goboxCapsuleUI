package gui

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"gobox/internal/probe"
)

const (
	stressDuration = 3 * time.Minute // LACAPSULE : 190 s
	stressMaxTemp  = 90.0            // seuil de la fiche
	stressSample   = 2 * time.Second
	stressBars     = 24
)

func runStress(t *testCtx) result {
	before, err := probe.CPUTemperature()
	hasSensor := err == nil
	if hasSensor {
		t.log("Température au repos : %.0f °C", before)
	} else {
		t.log("Pas de capteur de température : %v", err)
	}

	// Panneau : 3 valeurs + histogramme de température
	var tempTxt, loadTxt, freqTxt *canvas.Text
	var chart *barChart
	t.panel(func() fyne.CanvasObject {
		stat := func(label string) (*canvas.Text, fyne.CanvasObject) {
			v := text("—", 22, true, colInk)
			return v, box(container.NewVBox(caption(label), v), colLine, colPanel)
		}
		var a, b, c fyne.CanvasObject
		tempTxt, a = stat("Temp. CPU")
		loadTxt, b = stat("Charge")
		freqTxt, c = stat("Fréquence")
		chart = newBarChart(stressBars, stressMaxTemp)
		return container.NewVBox(container.NewGridWithColumns(3, a, b, c), chart.root)
	})

	workers := runtime.NumCPU()
	t.log("Lancement de la charge sur %d threads pendant %s", workers, stressDuration)
	ctx, stop := context.WithCancel(t.ctx)
	defer stop()
	for i := 0; i < workers; i++ {
		go burn(ctx)
	}

	start := time.Now()
	maxTemp := before
	prevTimes, _ := probe.ReadCPUTimes()
	nextLog := 30 * time.Second
	for time.Since(start) < stressDuration {
		if !t.sleep(stressSample) {
			return result{}
		}
		elapsed := time.Since(start)
		temp, _ := probe.CPUTemperature()
		freq, _ := probe.CPUFrequencyGHz()
		curTimes, _ := probe.ReadCPUTimes()
		load := probe.CPULoad(prevTimes, curTimes)
		prevTimes = curTimes
		if temp > maxTemp {
			maxTemp = temp
		}

		bar := int(float64(stressBars) * elapsed.Seconds() / stressDuration.Seconds())
		t.ui(func() {
			if hasSensor {
				tempTxt.Text = fmt.Sprintf("%.0f °C", temp)
			}
			loadTxt.Text = fmt.Sprintf("%.0f %%", load)
			freqTxt.Text = fmt.Sprintf("%.2f GHz", freq)
			tempTxt.Refresh()
			loadTxt.Refresh()
			freqTxt.Refresh()
			if hasSensor {
				chart.set(min(bar, stressBars-1), temp)
			}
		})
		t.progress(elapsed.Seconds() / stressDuration.Seconds())

		if elapsed >= nextLog {
			t.log("%s · %.0f °C · charge %.0f %% · %.2f GHz", elapsed.Round(time.Second), temp, load, freq)
			nextLog += 30 * time.Second
		}
	}
	stop()
	t.log("Arrêt de la charge")

	if !hasSensor {
		return result{Status: statusWarn, Detail: "Charge OK, température non mesurée"}
	}
	r := result{
		Detail: fmt.Sprintf("%.0f °C → %.0f °C après %s", before, maxTemp, stressDuration),
		Data: map[string]string{
			"before": fmt.Sprintf("%.0f °C", before),
			"after":  fmt.Sprintf("%.0f °C", maxTemp),
		},
		Status: statusOK,
	}
	if maxTemp >= stressMaxTemp {
		r.Status = statusFail
		t.log("Température maximale %.0f °C ≥ %.0f °C : pâte thermique / ventilation à vérifier", maxTemp, stressMaxTemp)
	}
	return r
}

// burn occupe un cœur à 100 % jusqu'à l'annulation
func burn(ctx context.Context) {
	x := 1.0
	for {
		for i := 0; i < 1_000_000; i++ {
			x = x*1.0000001 + 0.0000001
		}
		if ctx.Err() != nil {
			_ = x
			return
		}
	}
}

// ---- Histogramme ----------------------------------------------------------

type barChart struct {
	root *fyne.Container
	bars []*canvas.Rectangle
	vals []float64
	max  float64
}

func newBarChart(n int, max float64) *barChart {
	c := &barChart{bars: make([]*canvas.Rectangle, n), vals: make([]float64, n), max: max}
	objs := make([]fyne.CanvasObject, n)
	for i := range c.bars {
		r := canvas.NewRectangle(colLineSoft)
		r.CornerRadius = 2
		c.bars[i] = r
		objs[i] = r
	}
	c.root = container.New(&barsLayout{chart: c}, objs...)
	return c
}

func (c *barChart) set(i int, v float64) {
	c.vals[i] = v
	c.bars[i].FillColor = colTeal
	if v > 75 {
		c.bars[i].FillColor = colAmber
	}
	if v >= c.max {
		c.bars[i].FillColor = colRed
	}
	c.root.Refresh()
}

type barsLayout struct{ chart *barChart }

func (l *barsLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(200, 120) }

func (l *barsLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	n := float32(len(objs))
	gap := float32(3)
	w := (size.Width - gap*(n-1)) / n
	for i, o := range objs {
		frac := float32(l.chart.vals[i] / l.chart.max)
		if frac <= 0 {
			frac = 0.02
		}
		if frac > 1 {
			frac = 1
		}
		h := size.Height * frac
		o.Resize(fyne.NewSize(w, h))
		o.Move(fyne.NewPos(float32(i)*(w+gap), size.Height-h))
	}
}
