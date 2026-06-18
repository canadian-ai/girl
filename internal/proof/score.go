package proof

// HealthScore calculates a 0-100 score from severity counts.
func HealthScore(high, medium, low int) int {
	penalty := high*8 + medium*3 + low
	if penalty > 100 {
		penalty = 100
	}
	score := 100 - penalty
	if score < 0 {
		return 0
	}
	return score
}

// Status returns a health label for a score.
func Status(score int) string {
	switch {
	case score >= 90:
		return "Excellent"
	case score >= 75:
		return "Good"
	case score >= 50:
		return "Needs attention"
	default:
		return "High risk"
	}
}
