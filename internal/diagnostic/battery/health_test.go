package battery

import (
	"testing"

	"gobox/internal/probe"
)

// Capacités absentes : santé inconnue, aucune alerte « trop basse »
func TestEvaluateUnknownHealth(t *testing.T) {
	r := Evaluate(probe.BatteryInfo{Status: "Discharging", Capacity: 50}, DefaultBatteryGradingCriteria())
	if r.HealthKnown || len(r.Issues) > 0 || r.Grade != "" {
		t.Fatalf("santé inventée : %+v", r)
	}
}

func TestEvaluateHealth(t *testing.T) {
	r := Evaluate(probe.BatteryInfo{Status: "Full", DesignCapacity: 50e6, CurrentCapacity: 45e6, Cycle: 120, CycleKnown: true}, DefaultBatteryGradingCriteria())
	if !r.HealthKnown || r.HealthPercentage != 90 || len(r.Issues) > 0 {
		t.Fatalf("évaluation fausse : %+v", r)
	}
}
