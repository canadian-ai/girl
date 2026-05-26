package planner

import (
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

func TestPlanner_GeneratePlanNoDiags(t *testing.T) {
	p := NewPlanner()
	plan := p.GeneratePlan(PlanRequest{
		Target: "test.go",
		Lang:   "go",
	})
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if len(plan.Steps) != 0 {
		t.Errorf("expected 0 steps for no diagnostics, got %d", len(plan.Steps))
	}
	if plan.Goal == "" {
		t.Error("expected non-empty goal")
	}
}

func TestPlanner_StepIDsDeterministic(t *testing.T) {
	p := NewPlanner()
	diags := []ir.Diagnostic{
		{Code: "go.long-function", Severity: ir.SeverityMedium, Symbol: "ProcessData", File: "handler.go"},
		{Code: "react.large-component", Severity: ir.SeverityHigh, Component: "Dashboard", File: "ui/dash.tsx"},
	}

	plan1 := p.GeneratePlan(PlanRequest{
		Target:       "test",
		Diagnostics:  diags,
		Lang:         "go",
	})
	plan2 := p.GeneratePlan(PlanRequest{
		Target:       "test",
		Diagnostics:  diags,
		Lang:         "go",
	})

	if len(plan1.Steps) != len(plan2.Steps) {
		t.Fatalf("step count mismatch: %d vs %d", len(plan1.Steps), len(plan2.Steps))
	}
	for i := range plan1.Steps {
		if plan1.Steps[i].ID != plan2.Steps[i].ID {
			t.Errorf("step ID mismatch at %d: %s vs %s", i, plan1.Steps[i].ID, plan2.Steps[i].ID)
		}
	}
}

func TestPlanner_StepIDsUnique(t *testing.T) {
	p := NewPlanner()
	diags := []ir.Diagnostic{
		{Code: "go.long-function", Severity: ir.SeverityMedium, Symbol: "ProcessData", File: "handler.go"},
		{Code: "go.high-complexity", Severity: ir.SeverityLow, Symbol: "computeScore", File: "handler.go"},
		{Code: "react.large-component", Severity: ir.SeverityHigh, Component: "Dashboard", File: "ui/dash.tsx"},
		{Code: "react.too-many-hooks", Severity: ir.SeverityLow, Component: "Dashboard", File: "ui/dash.tsx"},
	}

	plan := p.GeneratePlan(PlanRequest{
		Target:       "test",
		Diagnostics:  diags,
		Lang:         "go",
	})

	seen := map[string]bool{}
	for _, s := range plan.Steps {
		if seen[s.ID] {
			t.Errorf("duplicate step ID: %s", s.ID)
		}
		seen[s.ID] = true
	}
}

func TestPlanner_RemoveExtractTarget(t *testing.T) {
	// compile check: extractTarget was removed from planner.go
}

func TestSlugTarget(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Split Dashboard (ui/dash.tsx) into smaller focused components", "split-dashboard-uidashtsx-into-smaller-f"},
		{"Extract smaller functions from ProcessData", "extract-smaller-functions-from-processda"},
		{"  hello  world  ", "-hello-world"},
		{"", "target"},
		{"abc123", "abc123"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tc := range tests {
		result := slugTarget(tc.input)
		if result != tc.expected {
			t.Errorf("slugTarget(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}
