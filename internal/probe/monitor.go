package probe

import (
	"fmt"
	"log"
	"math"
	"os/exec"
	"regexp"
	"strconv"
)

func DetectDisplay() {
	// 1. Exécuter la commande xrandr
	out, err := exec.Command("xrandr").Output()
	if err != nil {
		log.Fatalf("Erreur d'exécution de xrandr: %v", err)
	}

	// 2. Convertir la sortie en string
	output := string(out)

	// 3. Regex pour extraire les dimensions "XXXmm x YYYmm"
	re := regexp.MustCompile(`([0-9]+)mm x ([0-9]+)mm`)

	// 4. Itérer sur chaque correspondance
	matches := re.FindAllStringSubmatch(output, -1)
	for _, m := range matches {
		w, _ := strconv.ParseFloat(m[1], 64)
		h, _ := strconv.ParseFloat(m[2], 64)

		// 5. Calcul de la diagonale en pouces
		diag := math.Sqrt(math.Pow(w, 2)+math.Pow(h, 2)) / 25.4
		fmt.Printf("Taille en pouces: %.1f\n", diag)
	}
}
