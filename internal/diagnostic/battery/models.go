package battery

import (
	"time"

	"gobox/internal/diagnostic/common"
)

type BatteryHealthTest struct {
	Status           string
	Grade            common.Grade
	HealthPercentage float64
	CycleCount       int
	Issues           []string
	Timestamp        time.Time
}

type BatteryGradingCriteria struct {
	MinHealthForA float64
	MinHealthForB float64
	MinHealthForC float64
	MaxCyclesForA int
	MaxCyclesForB int
	MaxCyclesForC int
}
