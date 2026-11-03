//go:build linux

package gui

/*
#cgo LDFLAGS: -lX11
#include <X11/Xlib.h>
#include <X11/XKBlib.h>
#include <stdlib.h>

// Symbole produit par une touche dans la disposition active ("a", "q"…), NULL
// sans serveur X
static const char *key_symbol(int code) {
	Display *d = XOpenDisplay(NULL);
	if (d == NULL) return NULL;
	KeySym s = XkbKeycodeToKeysym(d, (KeyCode)code, 0, 0);
	const char *name = XKeysymToString(s);
	XCloseDisplay(d);
	return name;
}

static int grab_open(Display **out) {
	Display *d = XOpenDisplay(NULL);
	if (d == NULL) return -1;
	*out = d;
	return 0;
}

static int grab_try(Display *d) {
	return XGrabKeyboard(d, DefaultRootWindow(d), False, GrabModeAsync, GrabModeAsync, CurrentTime);
}

// Code de la prochaine touche enfoncée, 0 si aucune en attente
static int grab_next(Display *d) {
	while (XPending(d) > 0) {
		XEvent e;
		XNextEvent(d, &e);
		if (e.type == KeyPress) return (int)e.xkey.keycode;
	}
	return 0;
}

static void grab_close(Display *d) {
	XUngrabKeyboard(d, CurrentTime);
	XSync(d, False);
	XCloseDisplay(d);
}
*/
import "C"

import "time"

// keySymbol symbole de la touche code dans la disposition active du serveur X,
// "" s'il n'est pas lisible
func keySymbol(code int) string {
	name := C.key_symbol(C.int(code))
	if name == nil {
		return ""
	}
	return C.GoString(name)
}

// grabKeyboard capture tout le clavier auprès du serveur X : la touche Windows,
// Alt+Tab et les raccourcis du bureau (dont le menu de Cinnamon, que gsettings
// ne suspend pas) arrivent au test au lieu d'ouvrir un menu. Le serveur libère
// la capture si gobox s'arrête. Sans X11 ou si la capture est refusée : ok faux,
// le test continue par la fenêtre.
func grabKeyboard(onKey func(code int)) (release func(), ok bool) {
	var d *C.Display
	if C.grab_open(&d) != 0 {
		return func() {}, false
	}
	granted := false
	for i := 0; i < 20 && !granted; i++ { // un menu peut tenir le clavier un instant
		granted = C.grab_try(d) == C.GrabSuccess
		if !granted {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if !granted {
		C.XCloseDisplay(d)
		return func() {}, false
	}
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				C.grab_close(d)
				return
			default:
			}
			if code := int(C.grab_next(d)); code != 0 {
				onKey(code)
				continue
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	return func() { close(stop); <-done }, true
}
