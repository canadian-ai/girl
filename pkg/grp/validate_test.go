package grp

import (
	"strings"
	"testing"
)

func validPlan() *Plan {
	return &Plan{
		SpecVersion: "0.1",
		ID:          "grp_test",
		Type:        "dev.refactor.plan",
		Source:      "github.com/canadian-ai/girl",
		Subject:     ".",
		Language:    "go",
		Goal:        "Refactor long functions into smaller focused units",
		Risk:        SeverityMedium,
		Diagnostics: []Diagnostic{
			{
				ID:         "diag_001",
				Code:       "go.high-complexity",
				Severity:   SeverityHigh,
				Confidence: ConfidenceHigh,
				Message:    "Function handleRequest has cyclomatic complexity 22",
				File:       "internal/server/handler.go",
			},
		},
		Steps: []Step{
			{
				ID:     "step_001_go.high-complexity_handleRequest",
				Title:  "Simplify branching in handleRequest",
				Action: "Extract guard clauses and reduce nesting in handleRequest",
				Target: Target{File: "internal/server/handler.go"},
				Risk:   SeverityMedium,
				Requires: []string{"diag_001"},
			},
		},
		Verification: []Verification{
			{
				Command:    "go test ./...",
				Required:   true,
				Source:     "go",
				Confidence: "high",
			},
		},
	}
}

func TestValidatePlanValid(t *testing.T) {
	p := validPlan()
	result := ValidatePlan(p)
	if !result.Valid {
		t.Errorf("expected valid plan, got %d errors: %v", len(result.Errors), result.Errors)
	}
}

func TestValidatePlanNil(t *testing.T) {
	result := ValidatePlan(nil)
	if result.Valid {
		t.Errorf("expected invalid for nil plan")
	}
	if len(result.Errors) == 0 {
		t.Errorf("expected at least 1 error for nil plan")
	}
}

func TestValidatePlanMissingSpecVersion(t *testing.T) {
	p := validPlan()
	p.SpecVersion = ""
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasField(result.Errors, "specversion") {
		t.Errorf("expected error on specversion, got: %v", result.Errors)
	}
}

func TestValidatePlanInvalidID(t *testing.T) {
	p := validPlan()
	p.ID = "bad_id"
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasFieldMsg(result.Errors, "id", "must start with") {
		t.Errorf("expected error for ID prefix, got: %v", result.Errors)
	}
}

func TestValidatePlanInvalidRisk(t *testing.T) {
	p := validPlan()
	p.Risk = "critical"
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasField(result.Errors, "risk") {
		t.Errorf("expected error on risk, got: %v", result.Errors)
	}
}

func TestValidatePlanDiagnosticMissingFields(t *testing.T) {
	p := validPlan()
	p.Diagnostics = []Diagnostic{
		{
			ID: "diag_001",
			Severity:   SeverityHigh,
			Confidence: ConfidenceHigh,
			Message:    "test",
			File:       "test.go",
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasField(result.Errors, "diagnostics[0].code") {
		t.Errorf("expected error for missing code, got: %v", result.Errors)
	}
}

func TestValidatePlanDiagnosticDuplicateID(t *testing.T) {
	p := validPlan()
	p.Diagnostics = []Diagnostic{
		{
			ID: "diag_001", Code: "go.high-complexity", Severity: SeverityHigh,
			Confidence: ConfidenceHigh, Message: "msg1", File: "a.go",
		},
		{
			ID: "diag_001", Code: "go.long-function", Severity: SeverityMedium,
			Confidence: ConfidenceMedium, Message: "msg2", File: "b.go",
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasFieldMsg(result.Errors, "diagnostics[1].id", "duplicate") {
		t.Errorf("expected error for duplicate diagnostic ID, got: %v", result.Errors)
	}
}

func TestValidatePlanDiagnosticAbsolutePath(t *testing.T) {
	p := validPlan()
	p.Diagnostics = []Diagnostic{
		{
			ID: "diag_001", Code: "go.high-complexity", Severity: SeverityHigh,
			Confidence: ConfidenceHigh, Message: "test",
			File: "/absolute/path",
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasFieldMsg(result.Errors, "diagnostics[0].file", "absolute") {
		t.Errorf("expected error for absolute path, got: %v", result.Errors)
	}
}

func TestValidatePlanStepInvalidID(t *testing.T) {
	p := validPlan()
	p.Steps = []Step{
		{
			ID: "invalid_id", Title: "title", Action: "action",
			Target: Target{File: "test.go"}, Risk: SeverityLow,
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasFieldMsg(result.Errors, "steps[0].id", "step_") {
		t.Errorf("expected error for step ID prefix, got: %v", result.Errors)
	}
}

func TestValidatePlanStepRequiresMissingDiag(t *testing.T) {
	p := validPlan()
	p.Diagnostics = nil
	p.Steps = []Step{
		{
			ID: "step_001", Title: "title", Action: "action",
			Target: Target{File: "test.go"}, Risk: SeverityLow,
			Requires: []string{"diag_999"},
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasField(result.Errors, "steps[0].requires") {
		t.Errorf("expected error for missing diag reference, got: %v", result.Errors)
	}
}

func TestValidatePlanStepDuplicateID(t *testing.T) {
	p := validPlan()
	p.Steps = []Step{
		{
			ID: "step_001", Title: "title1", Action: "action1",
			Target: Target{File: "a.go"}, Risk: SeverityLow,
		},
		{
			ID: "step_001", Title: "title2", Action: "action2",
			Target: Target{File: "b.go"}, Risk: SeverityMedium,
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasFieldMsg(result.Errors, "steps[1].id", "duplicate") {
		t.Errorf("expected error for duplicate step ID, got: %v", result.Errors)
	}
}

func TestValidatePlanVerificationMissingCommand(t *testing.T) {
	p := validPlan()
	p.Verification = []Verification{
		{
			Required:   true,
			Source:     "go",
			Confidence: "high",
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasField(result.Errors, "verification[0].command") {
		t.Errorf("expected error for missing command, got: %v", result.Errors)
	}
}

func TestValidatePlanExtensionsAllowed(t *testing.T) {
	p := validPlan()
	p.Extensions = map[string]interface{}{
		"custom-key": "custom-value",
		"nested":     map[string]interface{}{"key": "val"},
	}
	result := ValidatePlan(p)
	if !result.Valid {
		t.Errorf("expected valid plan with extensions, got errors: %v", result.Errors)
	}
}

func TestValidatePlanWithExamples(t *testing.T) {
	p := &Plan{
		SpecVersion: "0.1",
		ID:          "grp_example",
		Type:        "dev.refactor.plan",
		Source:      "github.com/canadian-ai/girl",
		Subject:     "pkg/server",
		Language:    "go",
		Goal:        "Reduce cyclomatic complexity in hot paths",
		Risk:        SeverityHigh,
	}
	result := ValidatePlan(p)
	if !result.Valid {
		t.Errorf("expected valid plan with required fields, got errors: %v", result.Errors)
	}
}

func hasField(errs []ValidationError, field string) bool {
	for _, e := range errs {
		if e.Field == field {
			return true
		}
	}
	return false
}

func hasFieldMsg(errs []ValidationError, field, msgSub string) bool {
	for _, e := range errs {
		if e.Field == field && strings.Contains(e.Message, msgSub) {
			return true
		}
	}
	return false
}
