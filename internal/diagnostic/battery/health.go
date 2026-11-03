package battery

import (
	"fmt"
	"time"

	"gobox/internal/diagnostic/common"

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

	return Evaluate(info, criteria), nil
}

// Evaluate note une batterie lue. Sans capacités, la santé est inconnue : ni
// note ni alerte inventée à partir d'un 0.
func Evaluate(info probe.BatteryInfo, criteria BatteryGradingCriteria) BatteryHealthTest {
	known := info.DesignCapacity > 0 && info.CurrentCapacity > 0
	healthPercent := 0.0
	var grade common.Grade
	issues := []string{}
	if known {
		healthPercent = CalculateHealthPercent(info.DesignCapacity, info.CurrentCapacity)
		cycles := info.Cycle
		if !info.CycleKnown {
			cycles = 0
		}
		grade = ComputeGrade(criteria, healthPercent, cycles)
		issues = DetectIssues(info.Status, healthPercent, cycles, criteria)
	}

	return BatteryHealthTest{
		Status:           info.Status,
		Grade:            grade,
		HealthPercentage: healthPercent,
		HealthKnown:      known,
		CycleCount:       info.Cycle,
		CycleKnown:       info.CycleKnown,
		Issues:           issues,
		Timestamp:        time.Now(),
	}
}
