package gui

import (
	"math"
	"regexp"
	"strconv"

	"github.com/go-pdf/fpdf"
)

// Icônes de la fiche : les tracés SVG intégrés au binaire sont redessinés en
// vectoriel dans le PDF (nets à l'impression, quelques octets chacun).

var (
	svgViewBox = regexp.MustCompile(`viewBox="0 0 ([\d.]+) ([\d.]+)"`)
	svgPath    = regexp.MustCompile(` d="([^"]+)"`)
)

// pathScanner lit les nombres d'un tracé SVG, collés ou non (« 1-64 », « .5.5 »)
type pathScanner struct {
	d string
	i int
}

func (s *pathScanner) skip() {
	for s.i < len(s.d) && (s.d[s.i] == ' ' || s.d[s.i] == ',' || s.d[s.i] == '\n' || s.d[s.i] == '\t') {
		s.i++
	}
}

// more vrai si un nombre suit (sinon, une nouvelle commande)
func (s *pathScanner) more() bool {
	s.skip()
	if s.i >= len(s.d) {
		return false
	}
	c := s.d[s.i]
	return c == '-' || c == '+' || c == '.' || (c >= '0' && c <= '9')
}

func (s *pathScanner) num() float64 {
	s.skip()
	start, dot := s.i, false
	if s.i < len(s.d) && (s.d[s.i] == '-' || s.d[s.i] == '+') {
		s.i++
	}
	for s.i < len(s.d) {
		c := s.d[s.i]
		if c == '.' && !dot {
			dot = true
		} else if c < '0' || c > '9' {
			break
		}
		s.i++
	}
	v, _ := strconv.ParseFloat(s.d[start:s.i], 64)
	return v
}

// flag indicateur d'arc : un seul chiffre, parfois collé au nombre suivant
func (s *pathScanner) flag() bool {
	s.skip()
	on := s.i < len(s.d) && s.d[s.i] == '1'
	s.i++
	return on
}

// pathPen sortie d'un tracé : le PDF, ou un compteur dans les tests
type pathPen interface {
	MoveTo(x, y float64)
	LineTo(x, y float64)
	CurveBezierCubicTo(cx0, cy0, cx1, cy1, x, y float64)
	ClosePath()
}

// drawSVGPath trace un chemin SVG ; (ox, oy) et k placent le repère de l'icône
func drawSVGPath(pen pathPen, d string, ox, oy, k float64) {
	s := &pathScanner{d: d}
	var x, y, startX, startY, ctrlX, ctrlY float64
	var cmd, prev byte
	mx := func(v float64) float64 { return ox + v*k }
	my := func(v float64) float64 { return oy + v*k }
	curve := func(x1, y1, x2, y2, ex, ey float64) {
		pen.CurveBezierCubicTo(mx(x1), my(y1), mx(x2), my(y2), mx(ex), my(ey))
		ctrlX, ctrlY, x, y = x2, y2, ex, ey
	}
	for {
		s.skip()
		if s.i >= len(d) {
			return
		}
		if !s.more() {
			cmd = d[s.i]
			s.i++
		}
		rel := cmd >= 'a'
		var bx, by float64
		if rel {
			bx, by = x, y
		}
		switch cmd {
		case 'M', 'm':
			x, y = bx+s.num(), by+s.num()
			startX, startY = x, y
			pen.MoveTo(mx(x), my(y))
			cmd-- // M devient L, m devient l : des coordonnées de plus sont des lignes
		case 'L', 'l':
			x, y = bx+s.num(), by+s.num()
			pen.LineTo(mx(x), my(y))
		case 'H', 'h':
			x = bx + s.num()
			pen.LineTo(mx(x), my(y))
		case 'V', 'v':
			y = by + s.num()
			pen.LineTo(mx(x), my(y))
		case 'C', 'c':
			x1, y1, x2, y2 := bx+s.num(), by+s.num(), bx+s.num(), by+s.num()
			ex, ey := bx+s.num(), by+s.num()
			curve(x1, y1, x2, y2, ex, ey)
		case 'S', 's':
			x1, y1 := x, y
			if prev == 'C' || prev == 'c' || prev == 'S' || prev == 's' {
				x1, y1 = 2*x-ctrlX, 2*y-ctrlY
			}
			x2, y2, ex, ey := bx+s.num(), by+s.num(), bx+s.num(), by+s.num()
			curve(x1, y1, x2, y2, ex, ey)
		case 'A', 'a':
			rx, ry, rot := s.num(), s.num(), s.num()
			large, sweep := s.flag(), s.flag()
			ex, ey := bx+s.num(), by+s.num()
			arcToCurves(x, y, rx, ry, rot, large, sweep, ex, ey, curve)
			x, y = ex, ey
		case 'Z', 'z':
			pen.ClosePath()
			x, y = startX, startY
		default:
			return // commande inconnue : l'icône s'arrête plutôt que de boucler
		}
		prev = cmd
	}
}

// arcToCurves convertit un arc SVG en courbes de Bézier (SVG 1.1, annexe F.6)
func arcToCurves(x1, y1, rx, ry, rotDeg float64, large, sweep bool, x2, y2 float64, curve func(a, b, c, d, e, f float64)) {
	if rx == 0 || ry == 0 || (x1 == x2 && y1 == y2) {
		curve(x1, y1, x2, y2, x2, y2)
		return
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	phi := rotDeg * math.Pi / 180
	cos, sin := math.Cos(phi), math.Sin(phi)
	dx, dy := (x1-x2)/2, (y1-y2)/2
	xp, yp := cos*dx+sin*dy, -sin*dx+cos*dy
	if l := xp*xp/(rx*rx) + yp*yp/(ry*ry); l > 1 {
		rx, ry = rx*math.Sqrt(l), ry*math.Sqrt(l)
	}
	num := rx*rx*ry*ry - rx*rx*yp*yp - ry*ry*xp*xp
	den := rx*rx*yp*yp + ry*ry*xp*xp
	co := math.Sqrt(math.Max(0, num/den))
	if large == sweep {
		co = -co
	}
	cxp, cyp := co*rx*yp/ry, -co*ry*xp/rx
	cx, cy := cos*cxp-sin*cyp+(x1+x2)/2, sin*cxp+cos*cyp+(y1+y2)/2
	start := math.Atan2((yp-cyp)/ry, (xp-cxp)/rx)
	delta := math.Atan2((-yp-cyp)/ry, (-xp-cxp)/rx) - start
	if sweep && delta < 0 {
		delta += 2 * math.Pi
	} else if !sweep && delta > 0 {
		delta -= 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(delta) / (math.Pi / 2)))
	step := delta / float64(n)
	t := 4.0 / 3 * math.Tan(step/4)
	point := func(a float64) (float64, float64) {
		return cx + rx*cos*math.Cos(a) - ry*sin*math.Sin(a), cy + rx*sin*math.Cos(a) + ry*cos*math.Sin(a)
	}
	deriv := func(a float64) (float64, float64) {
		return -rx*cos*math.Sin(a) - ry*sin*math.Cos(a), -rx*sin*math.Sin(a) + ry*cos*math.Cos(a)
	}
	for i := 0; i < n; i++ {
		a0, a1 := start+float64(i)*step, start+float64(i+1)*step
		px0, py0 := point(a0)
		px1, py1 := point(a1)
		d0x, d0y := deriv(a0)
		d1x, d1y := deriv(a1)
		curve(px0+t*d0x, py0+t*d0y, px1-t*d1x, py1-t*d1y, px1, py1)
	}
}

// drawIcon dessine une icône dans un carré de côté size, centrée
func drawIcon(pdf *fpdf.Fpdf, name string, x, y, size float64, c rgb) {
	b, err := iconFiles.ReadFile("icons/" + name + ".svg")
	if err != nil {
		return
	}
	box, path := svgViewBox.FindSubmatch(b), svgPath.FindSubmatch(b)
	if box == nil || path == nil {
		return
	}
	w, _ := strconv.ParseFloat(string(box[1]), 64)
	h, _ := strconv.ParseFloat(string(box[2]), 64)
	k := size / math.Max(w, h)
	pdf.SetFillColor(c.r, c.g, c.b)
	drawSVGPath(pdf, string(path[1]), x+(size-w*k)/2, y+(size-h*k)/2, k)
	pdf.DrawPath("F")
}
