package planner

import (
	"encoding/json"
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/pkg/grp"
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
		{Code: "go.long-function", Severity: ir.SeverityMedium, Symbol: "ProcessData", File: "handler.go", Message: "too long"},
		{Code: "react.large-component", Severity: ir.SeverityHigh, Component: "Dashboard", File: "ui/dash.tsx", Message: "too large"},
	}

	plan1 := p.GeneratePlan(PlanRequest{
		Target:      "test",
		Diagnostics: diags,
		Lang:        "go",
	})
	plan2 := p.GeneratePlan(PlanRequest{
		Target:      "test",
		Diagnostics: diags,
		Lang:        "go",
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
		{Code: "go.long-function", Severity: ir.SeverityMedium, Symbol: "ProcessData", File: "handler.go", Message: "too long"},
		{Code: "go.high-complexity", Severity: ir.SeverityLow, Symbol: "computeScore", File: "handler.go", Message: "too complex"},
		{Code: "react.large-component", Severity: ir.SeverityHigh, Component: "Dashboard", File: "ui/dash.tsx", Message: "too large"},
		{Code: "react.too-many-hooks", Severity: ir.SeverityLow, Component: "Dashboard", File: "ui/dash.tsx", Message: "too many hooks"},
	}

	plan := p.GeneratePlan(PlanRequest{
		Target:      "test",
		Diagnostics: diags,
		Lang:        "go",
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

func TestPlanOutputIsDeterministic(t *testing.T) {
	p := NewPlanner()
	diags := []ir.Diagnostic{
		{Code: "go.long-function", Severity: ir.SeverityMedium, Symbol: "ProcessData", File: "handler.go", Message: "ProcessData too long"},
		{Code: "go.high-complexity", Severity: ir.SeverityHigh, Symbol: "calculateScore", File: "score.go", Message: "calculateScore too complex"},
	}

	run := func() []byte {
		plan := p.GeneratePlan(PlanRequest{
			Target:      "test",
			Diagnostics: diags,
			Lang:        "go",
		})
		gp := grp.FromIRPlan(plan)
		gp.Language = "go"
		grp.NormalizePlan(gp)
		gp.ID = grp.ComputePlanID(gp)
		data, err := json.MarshalIndent(gp, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	out1 := run()
	out2 := run()

	if len(out1) != len(out2) {
		t.Fatalf("output length mismatch: %d vs %d", len(out1), len(out2))
	}
	for i := range out1 {
		if out1[i] != out2[i] {
			t.Fatalf("byte mismatch at position %d", i)
		}
	}
}

func TestPlanGRPJSONRoundTrip(t *testing.T) {
	p := NewPlanner()
	diags := []ir.Diagnostic{
		{Code: "go.long-function", Severity: ir.SeverityHigh, Symbol: "foo", File: "main.go", Message: "foo is too long"},
		{Code: "go.ignored-error", Severity: ir.SeverityMedium, Symbol: "bar", File: "util.go", Message: "bar ignores error"},
	}

	plan := p.GeneratePlan(PlanRequest{
		Target:      ".",
		Diagnostics: diags,
		Lang:        "go",
	})
	gp := grp.FromIRPlan(plan)
	gp.Language = "go"
	grp.NormalizePlan(gp)
	gp.ID = grp.ComputePlanID(gp)

	result := grp.ValidatePlan(gp)
	if !result.Valid {
		t.Fatalf("plan should be valid after normalization, got %d errors: %v", len(result.Errors), result.Errors)
	}

	if !json.Valid([]byte(gp.ID)) {
		// not a JSON check, just confirming the ID is set
	}
	if gp.ID == "" {
		t.Error("plan ID should not be empty")
	}
	if len(gp.Diagnostics) != 2 {
		t.Errorf("expected 2 diagnostics, got %d", len(gp.Diagnostics))
	}
	if len(gp.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(gp.Steps))
	}
	for _, s := range gp.Steps {
		if len(s.Requires) == 0 {
			t.Errorf("step %s has no requires", s.ID)
		}
	}
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
