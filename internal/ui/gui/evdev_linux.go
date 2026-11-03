//go:build linux

package gui

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// Lecture directe des touches dans le noyau (/dev/input, droits root) : chaque
// clavier y arrive, y compris l'interface « Consumer Control » d'un clavier USB
// qui porte muet et volume, que le bureau intercepte avant la fenêtre.

// evdevKeyboardKeys codes evdev qui font d'un périphérique un clavier à lire :
// lettres et espace, ou touches de son et de luminosité
var evdevKeyboardKeys = [][]int{{16, 57}, {113}, {114}, {115}, {164}, {224}, {225}}

// readEvdevKeys lit les claviers de root (/sys/class/input) ; onKey reçoit le
// code X11 (evdev + 8) de chaque touche enfoncée. Rend l'arrêt et les noms des
// claviers lus (vide sans droits).
func readEvdevKeys(root string, onKey func(code int)) (stop func(), names []string) {
	var files []*os.File
	dirs, _ := filepath.Glob(filepath.Join(root, "event*"))
	for _, d := range dirs {
		caps := readTrim(filepath.Join(d, "device", "capabilities", "key"))
		keyboard := false
		for _, codes := range evdevKeyboardKeys {
			keyboard = keyboard || hasKeyBits(caps, codes...)
		}
		if !keyboard {
			continue
		}
		f, err := os.Open(filepath.Join("/dev/input", filepath.Base(d)))
		if err != nil {
			continue
		}
		files = append(files, f)
		names = append(names, readTrim(filepath.Join(d, "device", "name")))
	}
	// struct input_event : horodatage (2 mots machine), type, code, valeur
	size := 16 + 8
	if strconv.IntSize == 32 {
		size = 8 + 8
	}
	var wg sync.WaitGroup
	for _, f := range files {
		wg.Add(1)
		go func(f *os.File) {
			defer wg.Done()
			buf := make([]byte, size*32)
			for {
				n, err := f.Read(buf)
				if err != nil {
					return
				}
				decodeKeyEvents(buf[:n], size, onKey)
			}
		}(f)
	}
	return func() {
		for _, f := range files {
			_ = f.Close()
		}
		wg.Wait()
	}, names
}

// decodeKeyEvents touches enfoncées d'une suite d'évènements input_event de size
// octets : fin de chaque évènement = type, code, valeur
func decodeKeyEvents(buf []byte, size int, onKey func(code int)) {
	for off := 0; off+size <= len(buf); off += size {
		ev := buf[off+size-8 : off+size]
		typ, code := binary.LittleEndian.Uint16(ev[0:]), binary.LittleEndian.Uint16(ev[2:])
		if typ == 1 && int32(binary.LittleEndian.Uint32(ev[4:])) == 1 { // EV_KEY, enfoncée
			onKey(int(code) + 8)
		}
	}
}
