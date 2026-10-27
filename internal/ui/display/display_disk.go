package ui

import (
	"fmt"
	"log"
	"strings"

	"gobox/internal/probe"
)

func DiskInfo() {
	disks, err := probe.ListDisks()
	if err != nil {
		fmt.Printf("Erreur listage disques : %v\n", err)
		return
	}

	if len(disks) == 0 {
		fmt.Println("Aucun disque physique détecté.")
		return
	}

	fmt.Print("\n" + strings.Repeat("=", 70) + "\n")
	fmt.Printf("  Disques détectés : %d\n", len(disks))
	fmt.Print(strings.Repeat("=", 70) + "\n\n")

	for i, diskName := range disks {
		info, err := probe.GetDiskInfo(diskName)
		if err != nil {
			log.Printf("Erreur récupération infos pour %s : %v\n", diskName, err)
			continue
		}

		// Calcul de la taille en unités lisibles
		sizeGB := float64(info.SizeBytes) / (1024 * 1024 * 1024)
		sizeTB := sizeGB / 1024

		// Sélection de l'unité appropriée
		var sizeStr string
		if sizeTB >= 1 {
			sizeStr = fmt.Sprintf("%.2f TB", sizeTB)
		} else {
			sizeStr = fmt.Sprintf("%.2f GB", sizeGB)
		}

		fmt.Printf("Disque #%d : %s\n", i+1, info.Name)
		fmt.Println(strings.Repeat("-", 70))

		if info.Vendor != "" {
			fmt.Printf("  Fabricant       : %s\n", info.Vendor)
		}
		if info.Model != "" {
			fmt.Printf("  Modèle          : %s\n", info.Model)
		}

		fmt.Printf("  Type            : %s\n", info.Type)
		fmt.Printf("  Taille          : %s (%d octets)\n", sizeStr, info.SizeBytes)

		if len(info.Partitions) > 0 {
			fmt.Printf("  Partitions (%d) :\n", len(info.Partitions))
			for _, partition := range info.Partitions {
				fmt.Printf("    - %s\n", partition)
			}
		} else {
			fmt.Printf("  Partitions      : Aucune\n")
		}

		fmt.Println(strings.Repeat("-", 70))

		// Espace entre les disques
		if i < len(disks)-1 {
			fmt.Println()
		}
	}

	fmt.Println()
}
