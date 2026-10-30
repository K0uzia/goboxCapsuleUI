package gui

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
)

// choice bouton de réponse proposé au technicien
type choice struct {
	Label  string
	Status status
}

var verdictChoices = []choice{{"OK", statusOK}, {"Moyen", statusWarn}, {"Nul", statusFail}}

// testCtx ce qu'un test peut faire : journaliser, afficher, demander
type testCtx struct {
	ctx  context.Context
	win  fyne.Window
	view *testView
	// le technicien a répondu : le verdict est le sien, inutile de lui montrer le
	// résultat (lu et écrit sur le fil de l'interface)
	answered bool
}

// log ajoute une ligne au journal du test
func (t *testCtx) log(format string, a ...any) {
	if t.cancelled() {
		return
	}
	msg := fmt.Sprintf(format, a...)
	fyne.Do(func() { t.view.appendLog(msg) })
}

// progress fixe l'avancement (0–1), ou une barre indéterminée si < 0
func (t *testCtx) progress(f float64) {
	if t.cancelled() {
		return
	}
	fyne.Do(func() { t.view.setProgress(f) })
}

// panel construit (sur le thread UI) et affiche un contenu propre au test
func (t *testCtx) panel(build func() fyne.CanvasObject) {
	fyne.DoAndWait(func() { t.view.setPanel(build()) })
}

// ui exécute f sur le thread de l'interface
func (t *testCtx) ui(f func()) { fyne.Do(f) }

// markSimulated signale à l'écran que les mesures sont simulées
func (t *testCtx) markSimulated() { fyne.Do(func() { t.view.simBadge.Show() }) }

// lockCancel retire l'annulation : le test doit aller à son terme
func (t *testCtx) lockCancel() {
	fyne.DoAndWait(func() {
		t.view.p.locked = true
		t.view.running.Hide()
	})
}

// setPanelEmpty retire le contenu propre au test
func (t *testCtx) setPanelEmpty() { fyne.Do(func() { t.view.setPanel(nil) }) }

// ask affiche une question avec des boutons, sans bloquer
func (t *testCtx) ask(prompt string, opts ...choice) <-chan status {
	ch := make(chan status, 1)
	fyne.Do(func() {
		t.view.setAsk(prompt, opts, func(s status) {
			t.answered = true
			select {
			case ch <- s:
			default:
			}
		})
	})
	return ch
}

// choose pose une question et attend la réponse (statusIdle si annulé)
func (t *testCtx) choose(prompt string, opts ...choice) status {
	select {
	case s := <-t.ask(prompt, opts...):
		t.clearAsk()
		return s
	case <-t.ctx.Done():
		return statusIdle
	}
}

func (t *testCtx) clearAsk() {
	fyne.Do(func() { t.view.setAsk("", nil, nil) })
}

// sleep attend d, retourne false si le test a été annulé
func (t *testCtx) sleep(d time.Duration) bool {
	select {
	case <-time.After(d):
		return true
	case <-t.ctx.Done():
		return false
	}
}

func (t *testCtx) cancelled() bool { return t.ctx.Err() != nil }
