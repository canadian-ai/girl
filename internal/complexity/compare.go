package complexity

import "sort"

// Compare applies a ratchet policy: increases to existing functions are
// regressions; new functions are regressions only when they exceed the current
// threshold. Removed functions never fail the comparison.
func Compare(current, baseline *Report) *Comparison {
	comparison := &Comparison{
		BaselineFunctions: len(baseline.Functions),
		Changes:           []Change{},
		NetComplexity:     current.Summary.Total - baseline.Summary.Total,
	}
	previous := make(map[string]FunctionMetric, len(baseline.Functions))
	for _, metric := range baseline.Functions {
		previous[metric.ID] = metric
	}
	seen := make(map[string]bool, len(current.Functions))
	for i := range current.Functions {
		metric := &current.Functions[i]
		seen[metric.ID] = true
		before, exists := previous[metric.ID]
		change := Change{ID: metric.ID, File: metric.File, Symbol: metric.Symbol, After: metric.Complexity}
		if !exists {
			comparison.Added++
			metric.Change = "added"
			change.Status = "added"
			change.Delta = metric.Complexity
			change.Regression = metric.Complexity > current.Threshold
		} else {
			metric.BaselineComplexity = before.Complexity
			metric.Delta = metric.Complexity - before.Complexity
			change.Before = before.Complexity
			change.Delta = metric.Delta
			switch {
			case metric.Delta > 0:
				comparison.Increased++
				metric.Change = "increased"
				change.Status = "increased"
				change.Regression = true
			case metric.Delta < 0:
				comparison.Decreased++
				metric.Change = "decreased"
				change.Status = "decreased"
			default:
				metric.Change = "unchanged"
			}
		}
		if change.Status != "" {
			comparison.Changes = append(comparison.Changes, change)
		}
		if change.Regression {
			comparison.Regressions++
		}
	}
	for _, metric := range baseline.Functions {
		if seen[metric.ID] {
			continue
		}
		comparison.Removed++
		comparison.Changes = append(comparison.Changes, Change{
			ID: metric.ID, File: metric.File, Symbol: metric.Symbol,
			Before: metric.Complexity, Delta: -metric.Complexity, Status: "removed",
		})
	}
	sort.Slice(comparison.Changes, func(i, j int) bool {
		left, right := comparison.Changes[i], comparison.Changes[j]
		if left.Regression != right.Regression {
			return left.Regression
		}
		if left.Delta != right.Delta {
			return left.Delta > right.Delta
		}
		return left.ID < right.ID
	})
	current.Comparison = comparison
	return comparison
}
