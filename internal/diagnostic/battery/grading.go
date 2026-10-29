package battery

import (
	"fmt"

	"gobox/internal/diagnostic/common"
)

const (
	StatusCharging    = "Charging"
	StatusDischarging = "Discharging"
	StatusFull        = "Full"
	StatusNotCharging = "Not charging"
)

func DefaultBatteryGradingCriteria() BatteryGradingCriteria {
	return BatteryGradingCriteria{
		MinHealthForA: 85.0, // 85% de santé = A
		MinHealthForB: 70.0, // 70% de santé = B
		MinHealthForC: 50.0, // 50% de santé = C
		MaxCyclesForA: 300,  // <= 300 cycles = A
		MaxCyclesForB: 500,  // <= 500 cycles = B
		MaxCyclesForC: 800,  // <= 800 cycles = C
	}
}

func CalculateHealthPercent(designCapacity, currentCapacity float64) float64 {
	if designCapacity <= 0 {
		return 0.0
	}
	healthPercentage := (currentCapacity / designCapacity) * 100
	return healthPercentage
}

func ComputeGrade(
	criteria BatteryGradingCriteria,
	healthPercent float64,
	cycleCount int,
) common.Grade {
	healthGrade := gradeFromHealth(criteria, healthPercent)
	cycleGrade := gradeFromCycle(criteria, cycleCount)
	return common.WorseGrade(healthGrade, cycleGrade)
}

func gradeFromHealth(criteria BatteryGradingCriteria, healthPercent float64) common.Grade {
	switch {
	case healthPercent >= criteria.MinHealthForA:
		return common.GradeA
	case healthPercent >= criteria.MinHealthForB:
		return common.GradeB
	case healthPercent >= criteria.MinHealthForC:
		return common.GradeC
	default:
		return common.GradeF
	}
}

func gradeFromCycle(criteria BatteryGradingCriteria, cycleCount int) common.Grade {
	switch {
	case cycleCount <= criteria.MaxCyclesForA:
		return common.GradeA
	case cycleCount <= criteria.MaxCyclesForB:
		return common.GradeB
	case cycleCount <= criteria.MaxCyclesForC:
		return common.GradeC
	default:
		return common.GradeF
	}
}

func DetectIssues(
	status string,
	healthPercentage float64,
	cycleCount int,
	criteria BatteryGradingCriteria,
) []string {
	issues := []string{}

	validStatuses := map[string]bool{
		StatusCharging:    true,
		StatusDischarging: true,
		StatusFull:        true,
		StatusNotCharging: true,
	}

	if !validStatuses[status] {
		issues = append(issues, fmt.Sprintf("Statut invalide: %s", status))
		return issues
	}

	if status == "Discharging" && healthPercentage < 20 {
		issues = append(issues, "Batterie en décharge avec une santé très basse. (<20%)")
	}
	if healthPercentage < criteria.MinHealthForC {
		issues = append(issues, "La santé de la batterie est trop basse. (<50%)")
	}
	if healthPercentage > 100 {
		issues = append(issues, "Anormal santé de la batterie (>100%), vérifier les capteurs.")
	}

	if cycleCount > criteria.MaxCyclesForC {
		issues = append(
			issues,
			"Haut cycle de charge de la batterie (>800), considérer un remplacement.",
		)
	}

	return issues
}
