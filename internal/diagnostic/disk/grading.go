package disk

import (
	"fmt"
	"strings"
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
	return DiskGradingCriteria{
		MinSizeForA: 200.0,
		MinSizeForB: 120.0,
		MinSizeForC: 60.0,
		SSDMinGrade: common.GradeB,
	}
}

// ComputeGrade calcule le grade du disque : des partitions résiduelles ou une
// taille sous le minimum donnent F, sinon la taille décide, et un SSD ne descend
// pas sous SSDMinGrade
func ComputeGrade(
	criteria DiskGradingCriteria,
	diskType string,
	sizeGB float64,
	hasPartitions bool,
) common.Grade {
	if hasPartitions || sizeGB < criteria.MinSizeForC {
		return common.GradeF
	}
	grade := gradeFromSize(criteria, sizeGB)
	if diskType == "SSD" && grade.ToScore() < criteria.SSDMinGrade.ToScore() {
		return criteria.SSDMinGrade
	}
	return grade
}

// gradeFromSize calcule le grade selon la taille uniquement
func gradeFromSize(criteria DiskGradingCriteria, sizeGB float64) common.Grade {
	switch {
	case sizeGB >= criteria.MinSizeForA:
		return common.GradeA
	case sizeGB >= criteria.MinSizeForB:
		return common.GradeB
	case sizeGB >= criteria.MinSizeForC:
		return common.GradeC
	default:
		return common.GradeF
	}
}

// DetectIssues détecte les problèmes du disque
func DetectIssues(
	diskType string,
	sizeGB float64,
	hasPartitions bool,
	partitionList []string,
	criteria DiskGradingCriteria,
) []string {
	issues := []string{}
	if hasPartitions {
		issues = append(issues, fmt.Sprintf("Partitions résiduelles détectées: [%s]", strings.Join(partitionList, ", ")))
	}
	switch {
	case sizeGB < criteria.MinSizeForC:
		issues = append(issues, fmt.Sprintf("Disque trop petit (< %.0f GB)", criteria.MinSizeForC))
	case sizeGB < criteria.MinSizeForB:
		issues = append(issues, fmt.Sprintf("Capacité limitée (< %.0f GB)", criteria.MinSizeForB))
	}
	return issues
}
