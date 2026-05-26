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

func TestRegistered_NotEmpty(t *testing.T) {
	registered := Registered()
	if len(registered) < 14 {
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
