package recipes

import (
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

func TestStepForDiagnostic_GoLongFunction(t *testing.T) {
	diag := ir.Diagnostic{
		Code:     "go.long-function",
		Severity: ir.SeverityMedium,
		Symbol:   "ProcessData",
		File:     "handler.go",
	}
	step := StepForDiagnostic(diag)
	if step.Recipe != "go.extract-function" {
		t.Errorf("expected recipe go.extract-function, got %s", step.Recipe)
	}
	if step.Risk != ir.SeverityMedium {
		t.Errorf("expected risk medium, got %s", step.Risk)
	}
	if len(step.Verify) == 0 || step.Verify[0] != "go build ./..." {
		t.Errorf("expected verify to include go build, got %v", step.Verify)
	}
}

func TestStepForDiagnostic_ReactLargeComponent(t *testing.T) {
	diag := ir.Diagnostic{
		Code:      "react.large-component",
		Severity:  ir.SeverityHigh,
		Component: "Dashboard",
		File:      "dashboard.tsx",
	}
	step := StepForDiagnostic(diag)
	if step.Recipe != "react.split-large-component" {
		t.Errorf("expected recipe react.split-large-component, got %s", step.Recipe)
	}
	if step.Risk != ir.SeverityHigh {
		t.Errorf("expected risk high, got %s", step.Risk)
	}
	if len(step.Verify) == 0 || step.Verify[0] != "typecheck" {
		t.Errorf("expected verify to include typecheck, got %v", step.Verify)
	}
}

func TestStepForDiagnostic_UnknownCode(t *testing.T) {
	diag := ir.Diagnostic{
		Code: "unknown.code",
	}
	step := StepForDiagnostic(diag)
	if step.Recipe != "" || step.Action != "" {
		t.Errorf("expected zero-value GrpStep, got %+v", step)
	}
}

func TestStepForDiagnostic_AgentDiagnostics(t *testing.T) {
	cases := []struct {
		code       string
		wantRecipe string
	}{
		{"agent.diff-too-large", "agent.decompose-large-diff"},
		{"agent.too-many-files-touched", "agent.decompose-large-diff"},
		{"agent.mixed-boundaries", "agent.split-mixed-boundary"},
		{"agent.unreviewable-plan", "agent.extract-reviewable-task"},
		{"agent.parallelization-opportunity", "agent.generate-parallel-task-packs"},
	}
	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			diag := ir.Diagnostic{Code: c.code, File: "test.go", Symbol: "Test"}
			step := StepForDiagnostic(diag)
			if step.Recipe != c.wantRecipe {
				t.Errorf("StepForDiagnostic(%q).Recipe = %q, want %q", c.code, step.Recipe, c.wantRecipe)
			}
			if step.Action == "" {
				t.Errorf("StepForDiagnostic(%q).Action should not be empty", c.code)
			}
			if step.File != "test.go" {
				t.Errorf("StepForDiagnostic(%q).File = %q, want %q", c.code, step.File, "test.go")
			}
		})
	}
}

func TestRegistered_NotEmpty(t *testing.T) {
	registered := Registered()
	if len(registered) < 19 {
		t.Errorf("expected at least 14 registered recipes, got %d", len(registered))
	}
}

func TestDiagnosticRecipe_ActionUsesSymbol(t *testing.T) {
	diag := ir.Diagnostic{
		Code:      "go.long-function",
		Severity:  ir.SeverityLow,
		Symbol:    "calculateTotal",
		Component: "OrderComponent",
		File:      "order.go",
	}
	step := StepForDiagnostic(diag)
	if step.Action == "" {
		t.Fatal("expected non-empty action")
	}
	if step.Action != "Extract smaller functions from calculateTotal" {
		t.Errorf("expected action to use Symbol 'calculateTotal', got %q", step.Action)
	}
}
