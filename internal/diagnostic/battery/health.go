package battery

import (
	"fmt"
	"time"

	"gobox/internal/probe"
)

// RunBatteryTest exécute le test batterie et retourne le résultat structuré
// Retourne (résultat, erreur)
func RunBatteryTest() (BatteryHealthTest, error) {
	// 1. Récupérer les données brutes
	info, err := probe.GetBatteryInfo()
	if err != nil {
		return BatteryHealthTest{}, fmt.Errorf("récupération batterie: %w", err)
	}

	// 2. Vérifier que la batterie existe
	if info.Capacity < 0 {
		return BatteryHealthTest{}, fmt.Errorf("batterie non détectée")
	}

	// 3. Charger les critères
	criteria := DefaultBatteryGradingCriteria()

	// 4. Calculer la santé
	healthPercent := CalculateHealthPercent(info.DesignCapacity, info.CurrentCapacity)

	// 5. Obtenir le grade
	grade := ComputeGrade(criteria, healthPercent, info.Cycle)

	// 6. Détecter les problèmes
	issues := DetectIssues(info.Status, healthPercent, info.Cycle, criteria)

	// 7. Construire et retourner le résultat
	return BatteryHealthTest{
		Status:           info.Status,
		Grade:            grade,
		HealthPercentage: healthPercent,
		CycleCount:       info.Cycle,
		Issues:           issues,
		Timestamp:        time.Now(),
	}, nil
}
