package gui

import (
	"encoding/binary"
	"os"
	"path/filepath"
)

// VBT : table du microprogramme Intel qui décrit les sorties câblées sur la carte.
// C'est la seule source du nombre réel de prises vidéo quand le BIOS ne les liste pas.
// Une sortie est souvent déclarée à la fois DP et HDMI : elle compte alors en « sortie
// vidéo », et son type se lit au branchement.

const (
	vbtBlockGeneral = 2
	vbtAnalog       = 1 << 0
	vbtDP           = 1 << 2
	vbtTMDS         = 1 << 4
	vbtNotHDMI      = 1 << 11
	vbtInternal     = 1 << 12
)

// vbtKind type d'une sortie, "" si elle est interne ou inutilisée
func vbtKind(deviceType uint16) string {
	dp, tmds, hdmi := deviceType&vbtDP != 0, deviceType&vbtTMDS != 0, deviceType&vbtNotHDMI == 0
	switch {
	case deviceType == 0 || deviceType&vbtInternal != 0:
		return ""
	case deviceType&vbtAnalog != 0 && !dp && !tmds:
		return "VGA"
	case dp && tmds && hdmi:
		return videoAny
	case dp:
		return "DisplayPort"
	case tmds && hdmi:
		return "HDMI"
	case tmds:
		return "DVI"
	}
	return ""
}

// parseVBT compte les sorties externes de la table ; vide si elle est illisible
func parseVBT(d []byte) map[string]int {
	if len(d) < 32 || string(d[:4]) != "$VBT" {
		return nil
	}
	off := int(binary.LittleEndian.Uint32(d[28:]))
	if off+22 > len(d) || string(d[off:off+15]) != "BIOS_DATA_BLOCK" {
		return nil
	}
	b := d[off:]
	for i := int(binary.LittleEndian.Uint16(b[18:])); i+3 <= len(b); {
		id, n := b[i], int(binary.LittleEndian.Uint16(b[i+1:]))
		if id != vbtBlockGeneral {
			i += 3 + n
			continue
		}
		if i+3+n > len(b) || n < 5 {
			return nil
		}
		g := b[i+3 : i+3+n]
		size := int(g[4])
		if size < 4 {
			return nil
		}
		out := map[string]int{}
		for j := 5; j+size <= len(g); j += size {
			if kind := vbtKind(binary.LittleEndian.Uint16(g[j+2:])); kind != "" {
				out[kind]++
			}
		}
		return out
	}
	return nil
}

// readVBTPorts prises d'après la table Intel (droits root) ; vide si indisponible
func readVBTPorts() map[string]int {
	files, _ := filepath.Glob("/sys/kernel/debug/dri/*/i915_vbt")
	for _, f := range files {
		if data, err := os.ReadFile(f); err == nil {
			if out := parseVBT(data); len(out) > 0 {
				return out
			}
		}
	}
	return nil
}
