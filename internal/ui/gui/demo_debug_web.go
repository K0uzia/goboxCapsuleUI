//go:build demo && js && demodebug

package gui

// Recette automatique de la démo (étiquette demodebug, jamais dans le site
// publié) : la page de test trouve un texte affiché et sa place pour le
// toucher, la toile WebGL n'ayant pas de DOM à interroger.

import (
	"fmt"
	"strings"
	"syscall/js"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func init() {
	debugHook = func(w fyne.Window, tools *toolsPanel) {
		js.Global().Set("goboxFind", js.FuncOf(func(_ js.Value, args []js.Value) any {
			want := args[0].String()
			exact := len(args) < 2 || args[1].Bool()
			var found []any
			// lecture directe : un seul fil en WebAssembly, et attendre la boucle
			// de Fyne bloquerait la page qui appelle
			func() {
				roots := []fyne.CanvasObject{w.Canvas().Content()}
				roots = append(roots, w.Canvas().Overlays().List()...)
				cs := w.Canvas().Size()
				for _, r := range roots {
					visibleWalk(r, rect{0, 0, cs.Width, cs.Height}, func(o fyne.CanvasObject, clip rect) {
						if !matches(o, want, exact) {
							return
						}
						pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(o)
						sz := o.Size()
						v := rect{pos.X, pos.Y, sz.Width, sz.Height}.inter(clip)
						found = append(found, map[string]any{"x": pos.X, "y": pos.Y, "w": sz.Width, "h": sz.Height,
							"vx": v.x, "vy": v.y, "vw": v.w, "vh": v.h})
					})
				}
			}()
			return found
		}))
		js.Global().Set("goboxState", js.FuncOf(func(js.Value, []js.Value) any {
			st := map[string]any{"active": "", "result": false, "popup": openPopup != nil, "canvasW": 0.0, "canvasH": 0.0}
			func() {
				if tools.active != nil {
					st["active"] = tools.active.ID
				}
				st["result"] = tools.view.resultBox.Visible() && tools.view.root.Visible()
				st["canvasW"] = float64(w.Canvas().Size().Width)
				st["canvasH"] = float64(w.Canvas().Size().Height)
				st["overlays"] = len(w.Canvas().Overlays().List())
				st["relayouts"] = relayouts
				st["focused"] = fmt.Sprintf("%T", w.Canvas().Focused())
				done := 0
				for _, t := range tools.s.Tests {
					if s := tools.s.status(t.ID); s != statusIdle && s != statusRunning {
						done++
					}
				}
				st["done"] = done
			}()
			return st
		}))
	}
}

// visibleWalk objets affichés (eux et tous leurs parents visibles), avec la
// zone où ils se voient (coupée par les zones qui défilent)
type rect struct{ x, y, w, h float32 }

func (a rect) inter(b rect) rect {
	x, y := max(a.x, b.x), max(a.y, b.y)
	r := rect{x, y, min(a.x+a.w, b.x+b.w) - x, min(a.y+a.h, b.y+b.h) - y}
	if r.w < 0 || r.h < 0 {
		return rect{}
	}
	return r
}

func visibleWalk(o fyne.CanvasObject, clip rect, f func(fyne.CanvasObject, rect)) {
	if !o.Visible() {
		return
	}
	f(o, clip)
	if sc, ok := o.(*container.Scroll); ok {
		p := fyne.CurrentApp().Driver().AbsolutePositionForObject(sc)
		clip = clip.inter(rect{p.X, p.Y, sc.Size().Width, sc.Size().Height})
	}
	switch c := o.(type) {
	case *fyne.Container:
		for _, child := range c.Objects {
			visibleWalk(child, clip, f)
		}
	case fyne.Widget:
		for _, child := range test.WidgetRenderer(c).Objects() {
			visibleWalk(child, clip, f)
		}
	}
}

func matches(o fyne.CanvasObject, want string, exact bool) bool {
	var got string
	switch t := o.(type) {
	case *widget.Button:
		got = t.Text
		if got == "" && t.Icon != nil {
			got = "icon:" + t.Icon.Name()
		}
	case *tText:
		got = t.Text
	case *widget.Label:
		got = t.Text
	case *widget.Check:
		got = t.Text
	case *canvas.Text:
		got = t.Text
	case *widget.Entry:
		got = "entry:" + t.PlaceHolder
	default:
		return false
	}
	if exact {
		return got == want
	}
	return strings.Contains(got, want)
}
