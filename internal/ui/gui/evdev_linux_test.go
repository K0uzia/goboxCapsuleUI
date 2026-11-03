//go:build linux

package gui

import (
	"encoding/binary"
	"testing"
)

// Volume + enfoncé, relâché, puis un évènement de synchronisation : un seul appui
func TestDecodeKeyEvents(t *testing.T) {
	ev := func(typ, code uint16, value int32) []byte {
		b := make([]byte, 24)
		binary.LittleEndian.PutUint16(b[16:], typ)
		binary.LittleEndian.PutUint16(b[18:], code)
		binary.LittleEndian.PutUint32(b[20:], uint32(value))
		return b
	}
	var buf []byte
	buf = append(buf, ev(4, 4, 0xe9)...) // EV_MSC
	buf = append(buf, ev(1, 115, 1)...)  // KEY_VOLUMEUP enfoncée
	buf = append(buf, ev(0, 0, 0)...)    // EV_SYN
	buf = append(buf, ev(1, 115, 0)...)  // relâchée
	var got []int
	decodeKeyEvents(buf, 24, func(c int) { got = append(got, c) })
	if len(got) != 1 || got[0] != 123 {
		t.Fatalf("codes %v, attendu [123] (Vol+ en X11)", got)
	}
}
