package ui

import (
	"fmt"

	"gobox/internal/probe"
)

func DisplayRamInfo() {
	info, err := probe.GetMemoryInfo()
	if err != nil {
		fmt.Println("Erreur :", err)
		return
	}

	fmt.Println("─── Rapport Mémoire ───────────────────────────────────────────────")

	for _, s := range info.Slots {
		fmt.Printf("Slot: %-30s | Type: %-6s | %6d MB @ %-5d MHz\n",
			s.Slot, s.Type, s.SizeMB, s.Speed)

		if s.ConfiguredSpeed > 0 && s.ConfiguredSpeed != s.Speed {
			fmt.Printf("  ↳ Vitesse configurée : %d MHz\n", s.ConfiguredSpeed)
		}

		fmt.Printf("  Fabricant     : %-15s  |  Forme : %-8s  |  Rank : %d\n",
			s.Manufacturer, s.FormFactor, s.Rank)

		if s.ConfiguredVoltage > 0 {
			fmt.Printf("  Tension       : %.1f V\n", s.ConfiguredVoltage)
		}

		if s.PartNumber != "" {
			fmt.Printf("  Numéro pièce  : %s\n", s.PartNumber)
		}

		if s.SerialNumber != "" {
			fmt.Printf("  Numéro série  : %s\n", s.SerialNumber)
		}

		if s.AssetTag != "" {
			fmt.Printf("  Tag inventaire: %s\n", s.AssetTag)
		}

		fmt.Printf("  Technologie   : %-10s | Mode : %s\n",
			s.Technology, s.OperatingMode)

		fmt.Println("───────────────────────────────────────────────────────────────────")
	}

	fmt.Printf("Mémoire totale détectée : %d MB (%.1f GB)\n",
		info.TotalMB, float64(info.TotalMB)/1024)
}
