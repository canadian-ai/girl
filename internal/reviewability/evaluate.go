package reviewability

import (
	"fmt"

	"github.com/canadian-ai/girl/internal/diffstats"
	"github.com/canadian-ai/girl/internal/ir"
)

type EvalResult struct {
	Result      ir.ReviewabilityResult
	Diagnostics []ir.Diagnostic
}

func Evaluate(diff *diffstats.DiffStats, budget ir.ReviewabilityBudget) *EvalResult {
	r := &EvalResult{
		Result: ir.ReviewabilityResult{
			Status: "unknown",
		},
	}

	if diff == nil {
		r.Result.Status = "pass"
		r.Result.Recommendation = "review"
		return r
	}

	r.Result.Budget = &ir.ReviewabilityBudget{
		MaxDiffLines:    budget.MaxDiffLines,
		MaxTouchedFiles: budget.MaxTouchedFiles,
		MaxRisk:         budget.MaxRisk,
	}
	r.Result.Observed = &ir.ReviewabilityObserved{
		AddedLines:   diff.TotalAdded,
		DeletedLines: diff.TotalDeleted,
		ChangedLines: diff.TotalChanged,
		ChangedFiles: diff.TotalFiles,
		LargestDelta: diff.LargestDelta,
	}

	var reasons []string
	var diagnostics []ir.Diagnostic

	if diff.TotalChanged > budget.MaxDiffLines {
		reasons = append(reasons, fmt.Sprintf("Diff is %d lines (budget: %d)", diff.TotalChanged, budget.MaxDiffLines))
		diagnostics = append(diagnostics, ir.Diagnostic{
			Code:       "agent.diff-too-large",
			Severity:   ir.SeverityHigh,
			Confidence: "high",
			Message:    fmt.Sprintf("Diff is %d lines, exceeds reviewability budget of %d", diff.TotalChanged, budget.MaxDiffLines),
			File:       ".",
		})
	}

	if diff.TotalFiles > budget.MaxTouchedFiles {
		reasons = append(reasons, fmt.Sprintf("Diff touches %d files (budget: %d)", diff.TotalFiles, budget.MaxTouchedFiles))
		diagnostics = append(diagnostics, ir.Diagnostic{
			Code:       "agent.too-many-files-touched",
			Severity:   ir.SeverityHigh,
			Confidence: "high",
			Message:    fmt.Sprintf("Diff touches %d files, exceeds reviewability budget of %d", diff.TotalFiles, budget.MaxTouchedFiles),
			File:       ".",
		})
	}

	if len(diff.Categories) > 2 {
		diagnostics = append(diagnostics, ir.Diagnostic{
			Code:       "agent.mixed-boundaries",
			Severity:   ir.SeverityMedium,
			Confidence: "medium",
			Message:    fmt.Sprintf("Diff spans %d categories (%v), suggests mixed concerns", len(diff.Categories), diff.Categories),
			File:       ".",
		})
	}

	if diff.HasGenerated || diff.HasLockfile {
		reasons = append(reasons, "Diff includes generated or lockfile changes")
	}

	if len(reasons) == 0 && len(diagnostics) == 0 {
		r.Result.Status = "pass"
		r.Result.Recommendation = "review"
		r.Result.Reason = "Diff is within reviewability budget"
	} else if len(diagnostics) <= 1 && diff.TotalChanged <= budget.MaxDiffLines+500 {
		r.Result.Status = "warn"
		r.Result.Recommendation = "review"
		r.Result.Reason = fmt.Sprintf("Diff approaches budget limits: %s", joinReasons(reasons))
	} else {
		r.Result.Status = "fail"
		r.Result.Recommendation = "decompose"
		r.Result.Reason = fmt.Sprintf("Diff exceeds reviewability budget: %s", joinReasons(reasons))

		diagnostics = append(diagnostics, ir.Diagnostic{
			Code:       "agent.unreviewable-plan",
			Severity:   ir.SeverityHigh,
			Confidence: "high",
			Message:    r.Result.Reason,
			File:       ".",
		})

		if diff.TotalFiles > 1 {
			diagnostics = append(diagnostics, ir.Diagnostic{
				Code:       "agent.parallelization-opportunity",
				Severity:   ir.SeverityLow,
				Confidence: "medium",
				Message:    fmt.Sprintf("Diff touches %d files which could be parallelized across tasks", diff.TotalFiles),
				File:       ".",
			})
		}
	}

	r.Diagnostics = diagnostics
	return r
}

func joinReasons(reasons []string) string {
	result := ""
	for i, r := range reasons {
		if i > 0 {
			result += "; "
		}
		result += r
	}
	return result
}
