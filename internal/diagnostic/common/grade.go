package common

type Grade string

const (
	GradeA Grade = "A"
	GradeB Grade = "B"
	GradeC Grade = "C"
	GradeF Grade = "F"
)

// Receiver method sur Grade
func (g Grade) ToScore() int {
	switch g {
	case GradeA:
		return 4
	case GradeB:
		return 3
	case GradeC:
		return 2
	case GradeF:
		return 1
	default:
		return 0
	}
}

func WorseGrade(g1, g2 Grade) Grade {
	if g1.ToScore() < g2.ToScore() {
		return g1
	}
	return g2
}
