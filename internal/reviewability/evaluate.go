package reviewability

import (
	"fmt"
	"strings"

	"github.com/canadian-ai/girl/internal/diffstats"
	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/structural"
)

type EvalResult struct {
	Result      ir.ReviewabilityResult
	Diagnostics []ir.Diagnostic
	Structural  *structural.Classification
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

	// Structural analysis (second pass)
	class := structural.Classify(diff)
	r.Structural = class

	// Structural diagnostics
	structDiags := structuralDiagnosticsFor(class)
	diagnostics = append(diagnostics, structDiags...)

	r.Diagnostics = diagnostics
	return r
}

func structuralDiagnosticsFor(class *structural.Classification) []ir.Diagnostic {
	if class == nil {
		return nil
	}
	var diags []ir.Diagnostic

	// agent.high-overhead: WARN when structural_overhead_ratio > 0.5
	if class.Ratios.StructuralOverhead > 0.5 {
		diags = append(diags, ir.Diagnostic{
			Code:       "agent.high-overhead",
			Severity:   ir.SeverityMedium,
			Confidence: "high",
			Message:    fmt.Sprintf("Structural overhead ratio is %.2f (threshold: 0.50)", class.Ratios.StructuralOverhead),
			File:       ".",
		})
	}

	// agent.low-cohesion: WARN when cohesion_variance > 0.6
	if class.Cohesion.Variance > 0.6 {
		clusters := ""
		if len(class.Cohesion.SuggestedClusters) > 0 {
			var desc []string
			for _, cl := range class.Cohesion.SuggestedClusters {
				desc = append(desc, fmt.Sprintf("[%s...%s]", cl[0], cl[len(cl)-1]))
			}
			clusters = fmt.Sprintf(" (suggested clusters: %s)", strings.Join(desc, ", "))
		}
		diags = append(diags, ir.Diagnostic{
			Code:       "agent.low-cohesion",
			Severity:   ir.SeverityMedium,
			Confidence: "high",
			Message:    fmt.Sprintf("Cohesion variance is %.2f (threshold: 0.60)%s", class.Cohesion.Variance, clusters),
			File:       ".",
		})
	}

	// agent.test-to-code-imbalance: WARN when test_to_logic_ratio > 3.0 AND ephemeral > reusable
	if class.Ratios.TestToLogic > 3.0 && class.Added.EphemeralSupport > class.Added.ReusableSupport {
		diags = append(diags, ir.Diagnostic{
			Code:       "agent.test-to-code-imbalance",
			Severity:   ir.SeverityMedium,
			Confidence: "medium",
			Message:    fmt.Sprintf("Test-to-logic ratio is %.1f (threshold: 3.0) with no reusable scaffold", class.Ratios.TestToLogic),
			File:       ".",
		})
	}

	// agent.ceremonial-noise: HIGH when high-overhead AND low-cohesion
	hasHighOverhead := false
	hasLowCohesion := false
	for _, d := range diags {
		switch d.Code {
		case "agent.high-overhead":
			hasHighOverhead = true
		case "agent.low-cohesion":
			hasLowCohesion = true
		}
	}
	if hasHighOverhead && hasLowCohesion {
		diags = append(diags, ir.Diagnostic{
			Code:       "agent.ceremonial-noise",
			Severity:   ir.SeverityHigh,
			Confidence: "high",
			Message:    "Diff has high structural overhead and low cohesion — high ceremonial noise",
			File:       ".",
		})
	}

	// agent.productive-scaffold: INFO when productive_scaffold_ratio > 0.5 AND reusable_support >= 20
	if class.Ratios.ProductiveScaffold > 0.5 && class.Added.ReusableSupport >= 20 {
		diags = append(diags, ir.Diagnostic{
			Code:       "agent.productive-scaffold",
			Severity:   ir.SeverityLow,
			Confidence: "high",
			Message:    fmt.Sprintf("Productive scaffold ratio is %.2f with %d reusable lines", class.Ratios.ProductiveScaffold, class.Added.ReusableSupport),
			File:       ".",
		})
	}

	return diags
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
