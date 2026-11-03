package gui

import (
	"os"
	"path/filepath"
	"testing"
)

// Plan sans touche en double ni touche qui en chevauche une autre
func TestKeyboardPlan(t *testing.T) {
	seen := map[int]bool{}
	for i, a := range keyboardKeys {
		if seen[a.Code] {
			t.Fatalf("code %d en double", a.Code)
		}
		seen[a.Code] = true
		if a.X+a.W > keyboardCols || a.Y+a.H > keyboardRows {
			t.Fatalf("touche %s hors du plan", a.Azerty)
		}
		for _, b := range keyboardKeys[i+1:] {
			if a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H {
				t.Fatalf("%s chevauche %s", a.Azerty, b.Azerty)
			}
		}
	}
}

// Portable : pavé numérique grisé, un appui sur une de ses touches fait
// apparaître tout le pavé ; une touche grisée ne compte pas
func TestKeyboardLaptopModel(t *testing.T) {
	m := newKeyboardModel(true)
	if m.hasSection(secPad) || m.kind() != "sans pavé numérique" {
		t.Fatalf("portable : %s", m.kind())
	}
	_, laptopTotal := m.counts()
	m.press(84) // 5 du pavé
	if !m.hasSection(secPad) || m.kind() != "clavier complet" {
		t.Fatal("un appui sur le pavé doit le faire apparaître")
	}
	if _, total := m.counts(); total != laptopTotal+17 {
		t.Fatalf("total %d, attendu %d (pavé de 17 touches)", total, laptopTotal+17)
	}
	m.press(110) // Début, grisée sur un portable
	if !m.present[110] {
		t.Fatal("une touche grisée appuyée existe")
	}
	if !m.toggle(119) || m.present[119] {
		t.Fatal("Suppr doit pouvoir être retirée")
	}
	if m.toggle(84) {
		t.Fatal("une touche appuyée ne se retire pas")
	}
	full := newKeyboardModel(false)
	if _, total := full.counts(); total != len(keyboardKeys) {
		t.Fatalf("poste fixe : %d touches attendues, %d", len(keyboardKeys), total)
	}
}

func TestIsLaptop(t *testing.T) {
	dmi, power := t.TempDir(), t.TempDir()
	write := func(v string) { _ = os.WriteFile(filepath.Join(dmi, "chassis_type"), []byte(v+"\n"), 0o644) }
	write("10")
	if !isLaptop(dmi, power) {
		t.Fatal("châssis 10 (notebook) : portable")
	}
	write("3")
	if isLaptop(dmi, power) {
		t.Fatal("châssis 3 (desktop) : poste fixe")
	}
}
