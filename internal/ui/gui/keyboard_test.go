package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Plans sans touche ni code en double, sans chevauchement
func TestKeyboardPlan(t *testing.T) {
	for _, laptop := range []bool{false, true} {
		plan := keyboardPlan(laptop)
		seen := map[int]bool{}
		for i, a := range plan {
			for _, c := range append([]int{a.Code}, a.Alt...) {
				if seen[c] {
					t.Fatalf("portable %v : code %d en double", laptop, c)
				}
				seen[c] = true
			}
			if a.X+a.W > keyboardCols || a.Y+a.H > keyboardRows {
				t.Fatalf("touche %s hors du plan", a.Azerty)
			}
			for _, b := range plan[i+1:] {
				if a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H {
					t.Fatalf("portable %v : %s chevauche %s", laptop, a.Azerty, b.Azerty)
				}
			}
		}
	}
}

// Portable : pavé grisé qui apparaît au premier appui, Impr testée par Fn + Inser,
// touches de son qui apparaissent ensemble
func TestKeyboardLaptopModel(t *testing.T) {
	m := newKeyboardModel(keyboardPlan(true))
	if m.hasSection(secPad) || m.hasSection(secMedia) || m.kind() != "sans pavé numérique" {
		t.Fatalf("portable : %s", m.kind())
	}
	_, laptopTotal := m.counts()
	m.press(84) // 5 du pavé
	if _, total := m.counts(); !m.hasSection(secPad) || total != laptopTotal+17 {
		t.Fatalf("pavé non apparu : %d touches", total)
	}
	m.press(107) // Impr = Fn + Inser
	if !m.pressed[118] || !m.present[118] {
		t.Fatal("Impr doit valider la touche Inser")
	}
	m.press(122) // Volume −
	if !m.present[121] || !m.present[123] || m.present[232] || !strings.Contains(m.kind(), "multimédia") {
		t.Fatal("les trois touches de son apparaissent ensemble, pas la luminosité")
	}
	if !m.toggle(119) || m.present[119] {
		t.Fatal("Suppr doit pouvoir être retirée")
	}
	if m.toggle(84) {
		t.Fatal("une touche appuyée ne se retire pas")
	}
	if m.known(152) {
		t.Fatal("touche du constructeur hors du plan")
	}
}

// Bureau : Impr, Défil et Pause sont des touches à part, tout est attendu sauf le
// multimédia
func TestKeyboardDesktopModel(t *testing.T) {
	m := newKeyboardModel(keyboardPlan(false))
	if !m.present[107] || !m.hasSection(secPad) || m.hasSection(secMedia) {
		t.Fatal("clavier de bureau incomplet")
	}
	m.press(107)
	if m.pressed[118] {
		t.Fatal("sur un clavier de bureau, Impr n'est pas Inser")
	}
}

func TestHasKeyBits(t *testing.T) {
	// clavier : Q (16) et Espace (57) dans le mot le plus bas
	if !hasKeyBits("1000000000007 ff9f207ac14057ff febeffdfffefffff fffffffffffffffe", 16, 57) {
		t.Fatal("clavier non reconnu")
	}
	if hasKeyBits("1f0000 0 0 0 0", 16, 57) {
		t.Fatal("souris prise pour un clavier")
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
