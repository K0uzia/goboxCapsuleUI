package ui

import (
	"fmt"

	diagBattery "gobox/internal/diagnostic/battery"
	"gobox/internal/probe"
)

func DisplayBatteryReport() error {
	// 1. Récupérer les données brutes
	info, err := probe.GetBatteryInfo()
	if err != nil || info.Capacity < 0 {
		fmt.Println("❌ Batterie        : Pas de batterie détectée")
		return nil
	}

	// 2. Exécuter le test de santé
	result, err := diagBattery.RunBatteryTest()
	if err != nil {
		return fmt.Errorf("test batterie échoué: %w", err)
	}

	// 3. Afficher le rapport unifié
	fmt.Println("\n╭─────────────────────────────────────────╮")
	fmt.Println("│         RAPPORT BATTERIE                │")
	fmt.Println("╰─────────────────────────────────────────╯")

	// État actuel
	fmt.Printf("\n📊 État actuel\n")
	fmt.Printf("   Charge           : %d%% (%s)\n", info.Capacity, info.Status)
	fmt.Printf("   Note santé       : [%s] (%.1f%%)\n", result.Grade, result.HealthPercentage)
	fmt.Printf("   Cycles           : %d\n", result.CycleCount)

	// Capacités
	fmt.Printf("\n🔋 Capacités\n")
	fmt.Printf("   Capacité actuelle : %.0f mWh (%.0f mAh)\n",
		info.CurrentCapacity, info.EnergyAH)
	fmt.Printf("   Capacité neuve    : %.0f mWh\n", info.DesignCapacity)
	fmt.Printf("   Dégradation       : %.1f%%\n",
		100-result.HealthPercentage)

	// Détails techniques
	fmt.Printf("\n⚙️  Détails techniques\n")
	if info.Manufacturer != nil {
		fmt.Printf("   Fabricant        : %s\n", *info.Manufacturer)
	}
	if info.Model != nil {
		fmt.Printf("   Modèle           : %s\n", *info.Model)
	}
	if info.Serial != nil {
		fmt.Printf("   Numéro série     : %s\n", *info.Serial)
	}
	if info.Technology != nil {
		fmt.Printf("   Technologie      : %s\n", *info.Technology)
	}
	fmt.Printf("   Tension actuelle : %.2f V\n", info.VoltageNow/1_000_000)

	// Problèmes détectés
	if len(result.Issues) > 0 {
		fmt.Printf("\n⚠️  Problèmes détectés\n")
		for _, issue := range result.Issues {
			fmt.Printf("   • %s\n", issue)
		}
	} else {
		fmt.Printf("\n✅ Aucun problème détecté\n")
	}

	fmt.Println()
	return nil
}
