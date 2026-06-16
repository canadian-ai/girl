package reviewability

import (
	"strings"
	"testing"

	"github.com/canadian-ai/girl/internal/diffstats"
	"github.com/canadian-ai/girl/internal/ir"
)

func TestEvaluatePass(t *testing.T) {
	diff := &diffstats.DiffStats{
		TotalAdded:   50,
		TotalDeleted: 10,
		TotalChanged: 60,
		TotalFiles:   3,
		LargestDelta: 40,
	}
	budget := DefaultBudget()
	r := Evaluate(diff, budget)
	if r == nil {
		t.Fatal("expected result")
	}
	if r.Result.Status != "pass" {
		t.Errorf("expected pass, got %s", r.Result.Status)
	}
	if len(r.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(r.Diagnostics))
	}
}

func TestEvaluateFailDiffTooLarge(t *testing.T) {
	diff := &diffstats.DiffStats{
		TotalAdded:   2000,
		TotalDeleted: 500,
		TotalChanged: 2500,
		TotalFiles:   5,
		LargestDelta: 1000,
	}
	budget := DefaultBudget()
	r := Evaluate(diff, budget)
	if r.Result.Status != "fail" {
		t.Errorf("expected fail, got %s", r.Result.Status)
	}
	if r.Result.Recommendation != "decompose" {
		t.Errorf("expected decompose, got %s", r.Result.Recommendation)
	}
	hasDiffTooLarge := false
	for _, d := range r.Diagnostics {
		if d.Code == "agent.diff-too-large" {
			hasDiffTooLarge = true
			break
		}
	}
	if !hasDiffTooLarge {
		t.Error("expected agent.diff-too-large diagnostic")
	}
}

func TestEvaluateFailTooManyFiles(t *testing.T) {
	diff := &diffstats.DiffStats{
		TotalAdded:   2000,
		TotalDeleted: 500,
		TotalChanged: 2500,
		TotalFiles:   20,
		LargestDelta: 30,
	}
	budget := DefaultBudget()
	r := Evaluate(diff, budget)
	if r.Result.Status != "fail" {
		t.Errorf("expected fail, got %s", r.Result.Status)
	}
	hasTooManyFiles := false
	for _, d := range r.Diagnostics {
		if d.Code == "agent.too-many-files-touched" {
			hasTooManyFiles = true
			break
		}
	}
	if !hasTooManyFiles {
		t.Error("expected agent.too-many-files-touched diagnostic")
	}
}

func TestEvaluateMixedBoundaries(t *testing.T) {
	diff := &diffstats.DiffStats{
		TotalAdded:   100,
		TotalDeleted: 20,
		TotalChanged: 120,
		TotalFiles:   5,
		LargestDelta: 30,
		Categories:   []string{"go", "typescript", "config"},
	}
	budget := DefaultBudget()
	r := Evaluate(diff, budget)
	hasMixed := false
	for _, d := range r.Diagnostics {
		if d.Code == "agent.mixed-boundaries" {
			hasMixed = true
			break
		}
	}
	if !hasMixed {
		t.Error("expected agent.mixed-boundaries diagnostic for multi-category diff")
	}
}

func TestEvaluateWarn(t *testing.T) {
	diff := &diffstats.DiffStats{
		TotalAdded:   1400,
		TotalDeleted: 200,
		TotalChanged: 1600,
		TotalFiles:   3,
		LargestDelta: 800,
	}
	budget := DefaultBudget()
	r := Evaluate(diff, budget)
	if r.Result.Status != "warn" {
		t.Errorf("expected warn, got %s", r.Result.Status)
	}
}

func TestEvaluateNilDiff(t *testing.T) {
	r := Evaluate(nil, DefaultBudget())
	if r.Result.Status != "pass" {
		t.Errorf("expected pass for nil diff, got %s", r.Result.Status)
	}
}

func TestDefaultBudget(t *testing.T) {
	b := DefaultBudget()
	if b.MaxDiffLines != 1500 {
		t.Errorf("expected 1500, got %d", b.MaxDiffLines)
	}
	if b.MaxTouchedFiles != 12 {
		t.Errorf("expected 12, got %d", b.MaxTouchedFiles)
	}
	if b.MaxRisk != ir.SeverityMedium {
		t.Errorf("expected medium, got %s", b.MaxRisk)
	}
}

func TestEvaluateFailBoth(t *testing.T) {
	diff := &diffstats.DiffStats{
		TotalAdded:   2000,
		TotalDeleted: 500,
		TotalChanged: 2500,
		TotalFiles:   20,
		LargestDelta: 1000,
		Categories:   []string{"go", "typescript", "config"},
	}
	budget := DefaultBudget()
	r := Evaluate(diff, budget)
	if r.Result.Status != "fail" {
		t.Errorf("expected fail, got %s", r.Result.Status)
	}
	if r.Result.Recommendation != "decompose" {
		t.Errorf("expected decompose, got %s", r.Result.Recommendation)
	}
	hasDiffTooLarge := false
	hasTooManyFiles := false
	hasUnreviewable := false
	hasParallelization := false
	for _, d := range r.Diagnostics {
		switch d.Code {
		case "agent.diff-too-large":
			hasDiffTooLarge = true
		case "agent.too-many-files-touched":
			hasTooManyFiles = true
		case "agent.unreviewable-plan":
			hasUnreviewable = true
		case "agent.parallelization-opportunity":
			hasParallelization = true
		}
	}
	if !hasDiffTooLarge {
		t.Error("expected agent.diff-too-large diagnostic")
	}
	if !hasTooManyFiles {
		t.Error("expected agent.too-many-files-touched diagnostic")
	}
	if !hasUnreviewable {
		t.Error("expected agent.unreviewable-plan diagnostic")
	}
	if !hasParallelization {
		t.Error("expected agent.parallelization-opportunity diagnostic for multi-file diff")
	}
}

func TestEvaluateExactlyAtBudget(t *testing.T) {
	diff := &diffstats.DiffStats{
		TotalAdded:   1000,
		TotalDeleted: 500,
		TotalChanged: 1500,
		TotalFiles:   12,
		LargestDelta: 600,
	}
	budget := DefaultBudget()
	r := Evaluate(diff, budget)
	if r.Result.Status != "pass" {
		t.Errorf("expected pass when exactly at budget, got %s", r.Result.Status)
	}
	if len(r.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics exactly at budget, got %d", len(r.Diagnostics))
	}
}

func TestEvaluateJustOverDiffLines(t *testing.T) {
	diff := &diffstats.DiffStats{
		TotalAdded:   1001,
		TotalDeleted: 500,
		TotalChanged: 1501,
		TotalFiles:   3,
		LargestDelta: 700,
	}
	budget := DefaultBudget()
	r := Evaluate(diff, budget)
	if r.Result.Status != "warn" {
		t.Errorf("expected warn for just over diff lines, got %s", r.Result.Status)
	}
	if r.Result.Recommendation != "review" {
		t.Errorf("expected review recommendation for warn, got %s", r.Result.Recommendation)
	}
}

func TestEvaluateGeneratedFiles(t *testing.T) {
	diff := &diffstats.DiffStats{
		TotalAdded:   100,
		TotalDeleted: 20,
		TotalChanged: 120,
		TotalFiles:   2,
		LargestDelta: 60,
		HasGenerated: true,
		HasLockfile:  true,
	}
	budget := DefaultBudget()
	r := Evaluate(diff, budget)
	if r.Result.Status != "warn" {
		t.Errorf("expected warn for small diff with generated files, got %s", r.Result.Status)
	}
	if !strings.Contains(r.Result.Reason, "generated") && !strings.Contains(r.Result.Reason, "lockfile") {
		t.Errorf("expected reason to mention generated/lockfile, got %q", r.Result.Reason)
	}
}

func TestEvaluateLargeDiffWithGeneratedFiles(t *testing.T) {
	diff := &diffstats.DiffStats{
		TotalAdded:   2000,
		TotalDeleted: 500,
		TotalChanged: 2500,
		TotalFiles:   10,
		LargestDelta: 1000,
		HasGenerated: true,
		HasLockfile:  true,
	}
	budget := DefaultBudget()
	r := Evaluate(diff, budget)
	if r.Result.Status != "fail" {
		t.Errorf("expected fail, got %s", r.Result.Status)
	}
	if !strings.Contains(r.Result.Reason, "generated") && !strings.Contains(r.Result.Reason, "lockfile") {
		t.Errorf("expected reason to mention generated/lockfile, got %q", r.Result.Reason)
	}
}

func TestEvaluateNoMixedBoundariesForOneCategory(t *testing.T) {
	diff := &diffstats.DiffStats{
		TotalAdded:   2000,
		TotalDeleted: 500,
		TotalChanged: 2500,
		TotalFiles:   5,
		LargestDelta: 1000,
		Categories:   []string{"go"},
	}
	budget := DefaultBudget()
	r := Evaluate(diff, budget)
	for _, d := range r.Diagnostics {
		if d.Code == "agent.mixed-boundaries" {
			t.Error("did not expect agent.mixed-boundaries for single-category diff")
		}
	}
}

func TestOverrideBudget(t *testing.T) {
	base := DefaultBudget()
	b := OverrideBudget(base, 2000, 0, "")
	if b.MaxDiffLines != 2000 {
		t.Errorf("expected 2000, got %d", b.MaxDiffLines)
	}
	if b.MaxTouchedFiles != 12 {
		t.Errorf("expected 12 (default), got %d", b.MaxTouchedFiles)
	}
	if b.MaxRisk != ir.SeverityMedium {
		t.Errorf("expected medium (default), got %s", b.MaxRisk)
	}
}
