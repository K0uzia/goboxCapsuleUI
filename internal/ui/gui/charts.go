package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// threshold ligne de seuil horizontale, étiquetée à droite
type threshold struct {
	Value float64
	Role  fyne.ThemeColorName
}

// lineChart courbe d'une mesure dans le temps, aux couleurs du thème. La
// courbe est dessinée pixel par pixel (trait continu et aire teintée), les
// étiquettes (échelle, seuils, temps, dernière valeur) sont du texte.
type lineChart struct {
	widget.BaseWidget
	values     []float64
	Min, Max   float64 // échelle verticale ; s'élargit si une valeur sort
	Role       fyne.ThemeColorName
	Thresholds []threshold
	Unit       string
	Format     func(float64) string // dernière valeur (par défaut « 72 °C »)
	Step       time.Duration        // écart entre deux relevés (axe du temps)
	Height     float32
}

func newLineChart(min, max float64, unit string, role fyne.ThemeColorName) *lineChart {
	c := &lineChart{Min: min, Max: max, Unit: unit, Role: role, Height: 130, Step: stressInterval}
	c.ExtendBaseWidget(c)
	return c
}

func (c *lineChart) add(v float64) {
	c.values = append(c.values, v)
	c.Refresh()
}

// scale bornes réellement tracées : l'échelle demandée, élargie aux valeurs
func (c *lineChart) scale() (lo, hi float64) {
	lo, hi = c.Min, c.Max
	for _, v := range c.values {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	if hi-lo < 1e-9 {
		hi = lo + 1
	}
	return lo, hi
}

func (c *lineChart) label(v float64) string {
	if c.Format != nil {
		return c.Format(v)
	}
	return fmt.Sprintf("%.0f %s", v, c.Unit)
}

// chartPad marges intérieures du tracé (place des étiquettes)
const (
	chartPadTop    = 8
	chartPadBottom = 18 // axe du temps
	chartPadRight  = 58 // étiquettes des seuils et dernière valeur
)

func (c *lineChart) CreateRenderer() fyne.WidgetRenderer {
	r := &chartRenderer{c: c, bg: canvas.NewRectangle(color.Transparent)}
	r.plot = canvas.NewRaster(r.draw)
	r.plot.ScaleMode = canvas.ImageScaleSmooth
	newLabel := func(size float32, bold bool) *canvas.Text {
		t := canvas.NewText("", color.Transparent)
		t.TextSize, t.TextStyle = size, fyne.TextStyle{Bold: bold}
		return t
	}
	r.top, r.bottom = newLabel(10, false), newLabel(10, false)
	r.start, r.end = newLabel(10, false), newLabel(10, false)
	r.end.Alignment = fyne.TextAlignTrailing
	r.last = newLabel(12, true)
	r.empty = newLabel(12, false)
	r.empty.Alignment = fyne.TextAlignCenter
	r.empty.Text = "En attente des premières mesures…"
	r.refreshObjects()
	return r
}

type chartRenderer struct {
	c                *lineChart
	bg               *canvas.Rectangle
	plot             *canvas.Raster
	top, bottom      *canvas.Text
	start, end, last *canvas.Text
	empty            *canvas.Text
	limits           []*canvas.Text
	objs             []fyne.CanvasObject
	size             fyne.Size
}

func (r *chartRenderer) MinSize() fyne.Size           { return fyne.NewSize(200, r.c.Height) }
func (r *chartRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *chartRenderer) Destroy()                     {}

func (r *chartRenderer) Layout(s fyne.Size) {
	r.size = s
	r.place()
}

func (r *chartRenderer) Refresh() {
	r.refreshObjects()
	r.place()
	r.plot.Refresh()
	canvas.Refresh(r.c)
}

// plotArea zone du tracé, en unités Fyne
func (r *chartRenderer) plotArea() (x, y, w, h float32) {
	return 0, chartPadTop, max(r.size.Width-chartPadRight, 1), max(r.size.Height-chartPadTop-chartPadBottom, 1)
}

// yOf ordonnée d'une valeur dans la zone du tracé
func (r *chartRenderer) yOf(v float64) float32 {
	lo, hi := r.c.scale()
	_, y, _, h := r.plotArea()
	return y + h - float32((v-lo)/(hi-lo))*h
}

func (r *chartRenderer) refreshObjects() {
	c := r.c
	r.bg.FillColor, r.bg.StrokeColor = col(cLog), col(cLine)
	r.bg.StrokeWidth, r.bg.CornerRadius = 1, 6
	lo, hi := c.scale()
	muted := col(cMuted)
	r.top.Text, r.bottom.Text = c.label(hi), c.label(lo)
	for _, t := range []*canvas.Text{r.top, r.bottom, r.start, r.end, r.empty} {
		t.Color = muted
	}
	r.start.Text = "0:00"
	r.end.Text = ""
	if n := len(c.values); n > 1 {
		r.end.Text = formatClock(time.Duration(n-1) * c.Step)
	}
	r.last.Text, r.last.Color = "", col(c.Role)
	if n := len(c.values); n > 0 {
		r.last.Text = c.label(c.values[n-1])
	}
	r.empty.Hidden = len(c.values) > 1

	r.limits = r.limits[:0]
	for _, t := range c.Thresholds {
		l := canvas.NewText(c.label(t.Value), col(t.Role))
		l.TextSize = 10
		r.limits = append(r.limits, l)
	}
	r.objs = []fyne.CanvasObject{r.bg, r.plot, r.top, r.bottom, r.start, r.end, r.empty}
	for _, l := range r.limits {
		r.objs = append(r.objs, l)
	}
	r.objs = append(r.objs, r.last)
}

func (r *chartRenderer) place() {
	s := r.size
	r.bg.Resize(s)
	r.plot.Move(fyne.NewPos(0, 0))
	r.plot.Resize(s)
	px, py, pw, ph := r.plotArea()
	r.top.Move(fyne.NewPos(px+6, py))
	r.bottom.Move(fyne.NewPos(px+6, py+ph-14))
	r.start.Move(fyne.NewPos(px+6, s.Height-chartPadBottom+3))
	r.end.Resize(fyne.NewSize(pw-6, 12))
	r.end.Move(fyne.NewPos(px, s.Height-chartPadBottom+3))
	r.empty.Resize(fyne.NewSize(pw, 16))
	r.empty.Move(fyne.NewPos(px, py+ph/2-8))

	// étiquettes à droite : seuils, puis la dernière valeur à hauteur de la
	// courbe, sans chevauchement avec un seuil
	right := px + pw + 6
	var taken []float32 // ordonnées déjà occupées par une étiquette
	free := func(y float32) float32 {
		for moved := true; moved; {
			moved = false
			for _, t := range taken {
				if math.Abs(float64(y-t)) < 12 {
					y, moved = t+12, true
				}
			}
		}
		taken = append(taken, y)
		return y
	}
	if n := len(r.c.values); n > 0 {
		y := min(max(r.yOf(r.c.values[n-1])-8, 0), s.Height-16)
		taken = append(taken, y)
		r.last.Move(fyne.NewPos(right, y))
	}
	// du plus haut au plus bas : les étiquettes descendent en cas de conflit
	for i := len(r.c.Thresholds) - 1; i >= 0; i-- {
		r.limits[i].Move(fyne.NewPos(right, free(r.yOf(r.c.Thresholds[i].Value)-7)))
	}
}

// columns valeur tracée pour chaque colonne de pixels : interpolée quand il
// y a peu de relevés, maximum du groupe quand il y en a plus que de pixels
func columns(v []float64, w int) []float64 {
	n := len(v)
	out := make([]float64, w)
	for x := 0; x < w; x++ {
		if n > w {
			a, b := x*n/w, (x+1)*n/w
			m := v[a]
			for _, y := range v[a:max(b, a+1)] {
				m = math.Max(m, y)
			}
			out[x] = m
			continue
		}
		f := float64(x) / float64(max(w-1, 1)) * float64(n-1)
		i := int(f)
		if i >= n-1 {
			out[x] = v[n-1]
			continue
		}
		out[x] = v[i] + (v[i+1]-v[i])*(f-float64(i))
	}
	return out
}

// draw image du tracé (grille, seuils, aire, courbe) à la taille en pixels
func (r *chartRenderer) draw(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	if r.size.Width <= 0 || r.size.Height <= 0 {
		return img
	}
	k := float64(w) / float64(r.size.Width) // pixels par unité Fyne
	px, py, pw, ph := r.plotArea()
	x0, y0 := int(float64(px)*k), int(float64(py)*k)
	x1, y1 := int(float64(px+pw)*k), int(float64(py+ph)*k)
	lo, hi := r.c.scale()
	yOf := func(v float64) float64 { return float64(y1) - (v-lo)/(hi-lo)*float64(y1-y0) }

	line, grid := nrgba(col(r.c.Role)), nrgba(col(cLine))
	blend := func(x, y int, c color.NRGBA, a float64) {
		if x < 0 || y < 0 || x >= w || y >= h || a <= 0 {
			return
		}
		a = math.Min(a, 1) * float64(c.A) / 255
		d := img.NRGBAAt(x, y)
		da := float64(d.A) / 255
		oa := a + da*(1-a)
		mix := func(s, d uint8) uint8 {
			return uint8((float64(s)*a + float64(d)*da*(1-a)) / oa)
		}
		img.SetNRGBA(x, y, color.NRGBA{mix(c.R, d.R), mix(c.G, d.G), mix(c.B, d.B), uint8(oa * 255)})
	}

	// grille légère : quarts de l'échelle
	for i := 1; i < 4; i++ {
		y := y0 + (y1-y0)*i/4
		for x := x0; x < x1; x++ {
			blend(x, y, grid, 0.6)
		}
	}
	// seuils en pointillés
	dash := int(4 * k)
	for _, t := range r.c.Thresholds {
		y, c := int(math.Round(yOf(t.Value))), nrgba(col(t.Role))
		for x := x0; x < x1; x++ {
			if (x/max(dash, 1))%2 == 0 {
				blend(x, y, c, 0.9)
			}
		}
	}

	if len(r.c.values) < 2 {
		return img
	}
	ys := columns(r.c.values, x1-x0)
	thick := 1.1 * k // demi-épaisseur du trait
	for i, v := range ys {
		x := x0 + i
		y := yOf(v)
		// aire sous la courbe, plus dense près du trait
		for yy := int(y); yy < y1; yy++ {
			blend(x, yy, line, 0.22*(1-float64(yy-int(y))/float64(max(y1-int(y), 1)))+0.04)
		}
		// trait : segment vertical entre ce point et le précédent, épaissi,
		// bords lissés
		prev := y
		if i > 0 {
			prev = yOf(ys[i-1])
		}
		top, bot := math.Min(prev, y)-thick, math.Max(prev, y)+thick
		for yy := int(math.Floor(top)); yy <= int(math.Ceil(bot)); yy++ {
			cov := math.Min(float64(yy)+1, bot) - math.Max(float64(yy), top)
			blend(x, yy, line, cov)
			blend(x-1, yy, line, cov*0.35)
		}
	}
	return img
}

// nrgba couleur du thème en NRGBA (non prémultipliée)
func nrgba(c color.Color) color.NRGBA {
	return color.NRGBAModel.Convert(c).(color.NRGBA)
}
