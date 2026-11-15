//go:build !(demo && js)

package gui

import (
	"encoding/binary"
	"math"
	"os"
)

// writeTone fichier WAV de test : left (1 s, 440 Hz à gauche), right (1 s,
// 660 Hz à droite) ou both (les deux à la suite)
func writeTone(channel string) (string, error) {
	const rate = 44100
	f, err := os.CreateTemp("", "gobox-son-*.wav")
	if err != nil {
		return "", err
	}
	defer f.Close()

	var parts []bool // true : gauche
	switch channel {
	case "left":
		parts = []bool{true}
	case "right":
		parts = []bool{false}
	default:
		parts = []bool{true, false}
	}
	samples := make([]int16, 0, rate*2*len(parts))
	for _, left := range parts {
		freq := 440.0
		if !left {
			freq = 660.0
		}
		for i := 0; i < rate; i++ {
			v := int16(math.Sin(2*math.Pi*freq*float64(i)/rate) * 12000)
			if left {
				samples = append(samples, v, 0)
			} else {
				samples = append(samples, 0, v)
			}
		}
	}
	size := uint32(len(samples) * 2)
	header := []any{
		[4]byte{'R', 'I', 'F', 'F'}, 36 + size, [4]byte{'W', 'A', 'V', 'E'},
		[4]byte{'f', 'm', 't', ' '}, uint32(16), uint16(1), uint16(2), uint32(rate), uint32(rate * 4), uint16(4), uint16(16),
		[4]byte{'d', 'a', 't', 'a'}, size,
	}
	for _, h := range header {
		if err := binary.Write(f, binary.LittleEndian, h); err != nil {
			return "", err
		}
	}
	if err := binary.Write(f, binary.LittleEndian, samples); err != nil {
		return "", err
	}
	// lisible par l'utilisateur de la session si gobox tourne en root
	_ = f.Chmod(0o644)
	return f.Name(), nil
}
