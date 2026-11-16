package planner

import (
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/pythonanalysis"
	"github.com/canadian-ai/girl/internal/recipes"
	"github.com/canadian-ai/girl/pkg/grp"
)

// Every code the Python binding can emit must map to a GRP recipe (#124).
func TestEveryPythonDiagnosticCodeHasARecipe(t *testing.T) {
	for _, code := range pythonanalysis.EmittedCodes() {
		step := recipes.StepForDiagnostic(ir.Diagnostic{Code: code, Severity: ir.SeverityMedium, File: "m.py", Line: 3, Symbol: "f"})
		if step.Recipe == "" || step.Action == "" || len(step.Verify) == 0 {
			t.Errorf("%s has no complete recipe: %+v", code, step)
		}
	}
	syntax := recipes.StepForDiagnostic(ir.SyntaxErrorDiagnostic("py", "bad.py", ir.SyntaxError{Line: 4, Message: "x"}))
	if syntax.Recipe != "py.repair-syntax" || syntax.Risk != ir.SeverityHigh {
		t.Fatalf("syntax errors must require repair, not refactor: %+v", syntax)
	}
}

func pyDiag(code string, sev ir.Severity, line int) ir.Diagnostic {
	return ir.Diagnostic{Code: code, Severity: sev, Confidence: "high", Message: code, File: "pkg/m.py", Line: line, Symbol: "f"}
}

func planFor(t *testing.T, diags []ir.Diagnostic) (*ir.GrpPlan, *grp.Plan) {
	t.Helper()
	irPlan := NewPlanner().GeneratePlan(PlanRequest{Target: "pkg", Lang: "python", Diagnostics: diags})
	gp := grp.FromIRPlan(irPlan)
	gp.ID = "grp_test"
	grp.NormalizePlan(gp)
	return irPlan, gp
}

func TestPythonPlanReadiness(t *testing.T) {
	cases := []struct {
		name          string
		diags         []ir.Diagnostic
		wantSteps     int
		wantUnplanned int
		wantReady     bool
	}{
		{"clean no-op", nil, 0, 0, true},
		{"mapped", []ir.Diagnostic{pyDiag("py.high-complexity", ir.SeverityHigh, 3), pyDiag("py.long-function", ir.SeverityMedium, 3)}, 2, 0, true},
		{"malformed", []ir.Diagnostic{ir.SyntaxErrorDiagnostic("py", "pkg/bad.py", ir.SyntaxError{Line: 2, Message: "invalid syntax"})}, 1, 0, true},
		{"unmapped", []ir.Diagnostic{pyDiag("py.future-rule", ir.SeverityHigh, 9)}, 0, 1, false},
		{"mixed", []ir.Diagnostic{pyDiag("py.deep-nesting", ir.SeverityMedium, 3), pyDiag("py.future-rule", ir.SeverityMedium, 9)}, 1, 1, false},
		{"unmapped low is advisory", []ir.Diagnostic{pyDiag("py.style-nit", ir.SeverityLow, 1)}, 0, 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			irPlan, gp := planFor(t, tc.diags)
			if len(irPlan.Steps) != tc.wantSteps || len(irPlan.Unplanned) != tc.wantUnplanned {
				t.Fatalf("steps=%d unplanned=%+v", len(irPlan.Steps), irPlan.Unplanned)
			}
			if v := grp.ValidatePlan(gp); !v.Valid {
				t.Fatalf("plan must remain structurally valid: %+v", v.Errors)
			}
			r := grp.CheckExecutionReadiness(gp)
			if r.Ready != tc.wantReady {
				t.Fatalf("ready=%v want %v: %+v", r.Ready, tc.wantReady, r)
			}
			for _, u := range gp.Unplanned {
				if u.DiagnosticID == "" || u.Reason == "" {
					t.Fatalf("unplanned finding lacks provenance: %+v", u)
				}
			}
		})
	}
}
