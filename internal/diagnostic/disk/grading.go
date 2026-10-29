package disk

import (
	"time"

	"gobox/internal/diagnostic/common"
)

// DiskHealthTest résultat du test disque
type DiskHealthTest struct {
	DiskName      string       // ex: "nvme0n1"
	Vendor        string       // ex: "Samsung"
	Model         string       // ex: "980 PRO"
	Type          string       // "SSD" ou "HDD"
	Grade         common.Grade // "A", "B", "C", "F"
	SizeGB        float64      // Taille en GB
	HasPartitions bool         // true si partitions détectées
	PartitionList []string     // Liste des partitions
	Issues        []string     // Problèmes détectés
	Timestamp     time.Time
}

// DiskGradingCriteria critères de notation disque
type DiskGradingCriteria struct {
	// Taille minimale acceptable (en GB)
	MinSizeForA float64 // ex: 200 GB → Grade A
	MinSizeForB float64 // ex: 120 GB → Grade B
	MinSizeForC float64 // ex: 60 GB → Grade C
	// < 60 GB = automatiquement F

	// Bonus SSD (si true, SSD ne peut pas être < B)
	SSDMinGrade common.Grade // ex: GradeB (SSD jamais en C ou F sauf si autre problème)
}

// DefaultDiskGradingCriteria retourne les critères par défaut
func DefaultDiskGradingCriteria() DiskGradingCriteria {
	// TODO: Retourne struct avec :
	// MinSizeForA: 200.0
	// MinSizeForB: 120.0
	// MinSizeForC: 60.0
	// SSDMinGrade: common.GradeB
}

// ComputeGrade calcule le grade du disque
func ComputeGrade(
	criteria DiskGradingCriteria,
	diskType string,
	sizeGB float64,
	hasPartitions bool,
) common.Grade {
	// TODO: Implémente la logique :
	// 1. Si hasPartitions == true → return common.GradeF
	// 2. Si sizeGB < criteria.MinSizeForC → return common.GradeF
	// 3. Si diskType == "SSD" et sizeGB >= criteria.MinSizeForA → return common.GradeA
	// 4. Sinon, utilise gradeFromSize(criteria, sizeGB)
	// 5. Si diskType == "SSD", applique SSDMinGrade (ne jamais descendre sous B)
}

// gradeFromSize calcule le grade selon la taille uniquement
func gradeFromSize(criteria DiskGradingCriteria, sizeGB float64) common.Grade {
	// TODO: Switch comme gradeFromHealth dans Battery
	// >= MinSizeForA → A
	// >= MinSizeForB → B
	// >= MinSizeForC → C
	// Sinon → F
}

// DetectIssues détecte les problèmes du disque
func DetectIssues(
	diskType string,
	sizeGB float64,
	hasPartitions bool,
	partitionList []string,
	criteria DiskGradingCriteria,
) []string {
	// TODO: Retourne []string{} et ajoute des issues si :
	// - hasPartitions == true → "Partitions résiduelles détectées: [...]"
	// - sizeGB < criteria.MinSizeForC → "Disque trop petit (< 60 GB)"
	// - sizeGB < criteria.MinSizeForB → "Capacité limitée (< 120 GB)"
}
