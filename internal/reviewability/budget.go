package reviewability

import "github.com/canadian-ai/girl/internal/ir"

func DefaultBudget() ir.ReviewabilityBudget {
	return ir.ReviewabilityBudget{
		MaxDiffLines:    1500,
		MaxTouchedFiles: 12,
		MaxRisk:         ir.SeverityMedium,
	}
}

func OverrideBudget(base ir.ReviewabilityBudget, maxDiffLines, maxTouchedFiles int, maxRisk string) ir.ReviewabilityBudget {
	if maxDiffLines > 0 {
		base.MaxDiffLines = maxDiffLines
	}
	if maxTouchedFiles > 0 {
		base.MaxTouchedFiles = maxTouchedFiles
	}
	if maxRisk != "" {
		base.MaxRisk = ir.Severity(maxRisk)
	}
	return base
}
