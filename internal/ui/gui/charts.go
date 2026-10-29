package gui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// chartMaxSegments au-delà, les points sont regroupés (maximum par groupe)
// pour garder un tracé fluide sur un test d'une heure
const chartMaxSegments = 240

// threshold ligne de seuil horizontale
type threshold struct {
	Value float64
	Role  fyne.ThemeColorName
}

// lineChart courbe d'une mesure dans le temps, aux couleurs du thème
type lineChart struct {
	widget.BaseWidget
	values     []float64
	Min, Max   float64 // échelle verticale ; Max s'élargit si une valeur dépasse
	Role       fyne.ThemeColorName
	Thresholds []threshold
	Unit       string
	Height     float32
}

func newLineChart(min, max float64, unit string, role fyne.ThemeColorName) *lineChart {
	c := &lineChart{Min: min, Max: max, Unit: unit, Role: role, Height: 120}
	c.ExtendBaseWidget(c)
	return c
}

func (c *lineChart) add(v float64) {
	c.values = append(c.values, v)
	c.Refresh()
}

func (c *lineChart) CreateRenderer() fyne.WidgetRenderer {
	r := &chartRenderer{c: c, bg: canvas.NewRectangle(color.Transparent)}
	r.top = canvas.NewText("", color.Transparent)
	r.bottom = canvas.NewText("", color.Transparent)
	r.top.TextSize, r.bottom.TextSize = 10, 10
	r.rebuild()
	return r
}

type chartRenderer struct {
	c           *lineChart
	bg          *canvas.Rectangle
	top, bottom *canvas.Text
	limits      []*canvas.Line
	segs        []*canvas.Line
	objs        []fyne.CanvasObject
	size        fyne.Size
}

func (r *chartRenderer) MinSize() fyne.Size           { return fyne.NewSize(160, r.c.Height) }
func (r *chartRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *chartRenderer) Destroy()                     {}

func (r *chartRenderer) Layout(s fyne.Size) {
	r.size = s
	r.rebuild()
}

func (r *chartRenderer) Refresh() {
	r.rebuild()
	canvas.Refresh(r.c)
}

// points valeurs à tracer, regroupées si elles sont trop nombreuses
func (r *chartRenderer) points() []float64 {
	v := r.c.values
	if len(v) <= chartMaxSegments+1 {
		return v
	}
	out := make([]float64, 0, chartMaxSegments+1)
	step := float64(len(v)) / float64(chartMaxSegments+1)
	for i := 0; i <= chartMaxSegments; i++ {
		a, b := int(float64(i)*step), int(float64(i+1)*step)
		m := v[a]
		for _, x := range v[a:min(b, len(v))] {
			m = max(m, x)
		}
		out = append(out, m)
	}
	return out
}

func (r *chartRenderer) rebuild() {
	c, s := r.c, r.size
	hi := c.Max
	for _, v := range c.values {
		hi = max(hi, v)
	}
	y := func(v float64) float32 {
		f := (v - c.Min) / (hi - c.Min)
		return s.Height - 4 - float32(f)*(s.Height-8)
	}

	r.bg.FillColor, r.bg.StrokeColor = col(cLog), col(cLine)
	r.bg.StrokeWidth, r.bg.CornerRadius = 1, 6
	r.bg.Resize(s)

	r.top.Text, r.bottom.Text = fmt.Sprintf("%.0f %s", hi, c.Unit), fmt.Sprintf("%.0f %s", c.Min, c.Unit)
	r.top.Color, r.bottom.Color = col(cMuted), col(cMuted)
	r.top.Move(fyne.NewPos(6, 2))
	r.bottom.Move(fyne.NewPos(6, s.Height-14))

	r.limits = r.limits[:0]
	for _, t := range c.Thresholds {
		l := canvas.NewLine(col(t.Role))
		l.StrokeWidth = 1
		l.Position1, l.Position2 = fyne.NewPos(0, y(t.Value)), fyne.NewPos(s.Width, y(t.Value))
		r.limits = append(r.limits, l)
	}

	pts := r.points()
	r.segs = r.segs[:0]
	if n := len(pts); n > 1 {
		dx := s.Width / float32(n-1)
		for i := 1; i < n; i++ {
			l := canvas.NewLine(col(c.Role))
			l.StrokeWidth = 2
			l.Position1 = fyne.NewPos(float32(i-1)*dx, y(pts[i-1]))
			l.Position2 = fyne.NewPos(float32(i)*dx, y(pts[i]))
			r.segs = append(r.segs, l)
		}
	}

	r.objs = []fyne.CanvasObject{r.bg}
	for _, l := range r.limits {
		r.objs = append(r.objs, l)
	}
	for _, l := range r.segs {
		r.objs = append(r.objs, l)
	}
	r.objs = append(r.objs, r.top, r.bottom)
}
