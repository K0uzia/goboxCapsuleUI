package gui

import (
	"strings"
	"testing"
	"time"
)

// Règles du verdict validées le 2026-10-28 : erreur ou ≥ 90 °C = NUL ;
// throttling ou 85 à 89 °C = MOYEN ; sans capteur = MOYEN.
func TestStressVerdict(t *testing.T) {
	base := stressVerdict{hasTemp: true, before: 40, ran: 3 * time.Minute, planned: 3 * time.Minute}
	cases := []struct {
		name string
		edit func(*stressVerdict)
		want status
	}{
		{"80 °C sans incident", func(v *stressVerdict) { v.maxTemp = 80 }, statusOK},
		{"84,9 °C", func(v *stressVerdict) { v.maxTemp = 84.9 }, statusOK},
		{"85 °C", func(v *stressVerdict) { v.maxTemp = 85 }, statusWarn},
		{"89 °C", func(v *stressVerdict) { v.maxTemp = 89 }, statusWarn},
		{"90 °C", func(v *stressVerdict) { v.maxTemp = 90 }, statusFail},
		{"throttling à 75 °C", func(v *stressVerdict) { v.maxTemp = 75; v.throttled = true }, statusWarn},
		{"une erreur de calcul", func(v *stressVerdict) { v.maxTemp = 70; v.errors = 1 }, statusFail},
		{"sans capteur", func(v *stressVerdict) { v.hasTemp = false }, statusWarn},
	}
	for _, c := range cases {
		v := base
		c.edit(&v)
		if got := v.result().Status; got != c.want {
			t.Errorf("%s : %s, attendu %s", c.name, got.style().label, c.want.style().label)
		}
	}
}

// Le calcul de vérification donne toujours le même résultat : c'est ce qui
// permet de compter une différence comme une erreur matérielle.
func TestBurnChunkDeterministic(t *testing.T) {
	if burnChunk(burnSeed) != burnChunk(burnSeed) {
		t.Fatal("burnChunk doit être déterministe")
	}
	if burnChunk(burnSeed) == burnChunk(burnSeed+1) {
		t.Fatal("burnChunk doit dépendre de sa graine")
	}
}

func TestFormatClock(t *testing.T) {
	for d, want := range map[time.Duration]string{
		135 * time.Second:                          "02:15",
		time.Hour + 2*time.Minute + 15*time.Second: "1:02:15",
	} {
		if got := formatClock(d); got != want {
			t.Errorf("formatClock(%s) = %s, attendu %s", d, got, want)
		}
	}
}

// Une réserve s'accompagne de pistes ; un test sans incident n'en a pas.
func TestStressAdvice(t *testing.T) {
	ok := stressVerdict{hasTemp: true, before: 40, maxTemp: 78}
	if a := ok.advice(); len(a) != 0 {
		t.Fatalf("aucune piste attendue sans incident, reçu %v", a)
	}
	hot := stressVerdict{hasTemp: true, before: 40, maxTemp: 87}
	found := false
	for _, a := range hot.advice() {
		if strings.Contains(a, "pâte thermique") {
			found = true
		}
	}
	if !found {
		t.Fatal("une réserve en température doit proposer de changer la pâte thermique")
	}
}
