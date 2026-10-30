package disk

import (
	"strings"
	"testing"

	"gobox/internal/diagnostic/common"
)

func TestComputeGrade(t *testing.T) {
	c := DefaultDiskGradingCriteria()
	cases := []struct {
		name       string
		diskType   string
		size       float64
		partitions bool
		want       common.Grade
	}{
		{"grand HDD", "HDD", 500, false, common.GradeA},
		{"HDD moyen", "HDD", 150, false, common.GradeB},
		{"petit HDD", "HDD", 80, false, common.GradeC},
		{"seuil exact A", "HDD", 200, false, common.GradeA},
		{"seuil exact C", "HDD", 60, false, common.GradeC},
		{"trop petit", "HDD", 59.9, false, common.GradeF},
		{"grand SSD", "SSD", 256, false, common.GradeA},
		{"petit SSD relevé", "SSD", 80, false, common.GradeB},
		{"SSD trop petit", "SSD", 32, false, common.GradeF},
		{"partitions résiduelles", "SSD", 500, true, common.GradeF},
	}
	for _, tc := range cases {
		if got := ComputeGrade(c, tc.diskType, tc.size, tc.partitions); got != tc.want {
			t.Errorf("%s : %s, attendu %s", tc.name, got, tc.want)
		}
	}
}

func TestDetectIssues(t *testing.T) {
	c := DefaultDiskGradingCriteria()
	if got := DetectIssues("SSD", 500, false, nil, c); len(got) != 0 {
		t.Errorf("disque sain : %v", got)
	}
	got := DetectIssues("HDD", 30, true, []string{"sda1", "sda2"}, c)
	if len(got) != 2 || !strings.Contains(got[0], "sda1, sda2") || !strings.Contains(got[1], "trop petit") {
		t.Errorf("partitions et trop petit : %v", got)
	}
	got = DetectIssues("HDD", 100, false, nil, c)
	if len(got) != 1 || !strings.Contains(got[0], "limitée") {
		t.Errorf("capacité limitée : %v", got)
	}
}
