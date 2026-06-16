package grp

import (
	"encoding/json"
	"os"
	"path/filepath"
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
				ID:       "step_001_go.high-complexity_handleRequest",
				Title:    "Simplify branching in handleRequest",
				Action:   "Extract guard clauses and reduce nesting in handleRequest",
				Target:   Target{File: "internal/server/handler.go"},
				Risk:     SeverityMedium,
				Requires: []string{"diag_001"},
			},
		},
		Verification: []Verification{
			{
				Command:    "go test ./...",
				Required:   true,
				Source:     "go",
				Confidence: "medium",
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
			ID:         "diag_001",
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

func TestValidatePlanAbsoluteSubject(t *testing.T) {
	p := validPlan()
	p.Subject = "/absolute/path"
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasFieldMsg(result.Errors, "subject", "absolute") {
		t.Errorf("expected error for absolute subject, got: %v", result.Errors)
	}
}

func TestValidatePlanAbsoluteSubjectWindows(t *testing.T) {
	p := validPlan()
	p.Subject = "C:\\Users\\ola\\project"
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasFieldMsg(result.Errors, "subject", "absolute") {
		t.Errorf("expected error for Windows absolute subject, got: %v", result.Errors)
	}
}

func TestValidatePlanAbsoluteSubjectUNC(t *testing.T) {
	p := validPlan()
	p.Subject = "\\\\server\\share\\repo"
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasFieldMsg(result.Errors, "subject", "absolute") {
		t.Errorf("expected error for UNC absolute subject, got: %v", result.Errors)
	}
}

func TestValidatePlanAbsoluteStepTarget(t *testing.T) {
	p := validPlan()
	p.Steps = []Step{
		{
			ID: "step_001", Title: "title", Action: "action",
			Target: Target{File: "/absolute/path"}, Risk: SeverityLow,
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasFieldMsg(result.Errors, "steps[0].target.file", "absolute") {
		t.Errorf("expected error for absolute step target, got: %v", result.Errors)
	}
}

func TestValidatePlanAbsoluteStepTargetWindows(t *testing.T) {
	p := validPlan()
	p.Steps = []Step{
		{
			ID: "step_001", Title: "title", Action: "action",
			Target: Target{File: "D:\\repo\\src\\App.tsx"}, Risk: SeverityLow,
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasFieldMsg(result.Errors, "steps[0].target.file", "absolute") {
		t.Errorf("expected error for Windows absolute step target, got: %v", result.Errors)
	}
}

func TestValidatePlanAbsoluteStepTargetUNC(t *testing.T) {
	p := validPlan()
	p.Steps = []Step{
		{
			ID: "step_001", Title: "title", Action: "action",
			Target: Target{File: "\\\\nas\\share\\projects\\file.go"}, Risk: SeverityLow,
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasFieldMsg(result.Errors, "steps[0].target.file", "absolute") {
		t.Errorf("expected error for UNC absolute step target, got: %v", result.Errors)
	}
}

func TestValidatePlanMissingSubject(t *testing.T) {
	p := validPlan()
	p.Subject = ""
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasField(result.Errors, "subject") {
		t.Errorf("expected error for missing subject, got: %v", result.Errors)
	}
}

func TestValidatePlanMissingGoal(t *testing.T) {
	p := validPlan()
	p.Goal = ""
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan")
	}
	if !hasField(result.Errors, "goal") {
		t.Errorf("expected error for missing goal, got: %v", result.Errors)
	}
}

func loadPlan(t *testing.T, dir string) *Plan {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "conformance", dir, "plan.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", path, err)
	}
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("failed to unmarshal fixture %s: %v", path, err)
	}
	return &p
}

func TestConformanceValidMinimal(t *testing.T) {
	p := loadPlan(t, "valid-minimal")
	result := ValidatePlan(p)
	if !result.Valid {
		t.Errorf("valid-minimal expected Valid=true, got errors: %v", result.Errors)
	}
}

func TestConformanceValidFull(t *testing.T) {
	p := loadPlan(t, "valid-full")
	result := ValidatePlan(p)
	if !result.Valid {
		t.Errorf("valid-full expected Valid=true, got errors: %v", result.Errors)
	}
}

func TestConformanceInvalidMissingFields(t *testing.T) {
	p := loadPlan(t, "invalid-missing-fields")
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("invalid-missing-fields expected Valid=false")
	}
	if !hasField(result.Errors, "specversion") {
		t.Errorf("expected error for specversion, got: %v", result.Errors)
	}
	if !hasField(result.Errors, "id") {
		t.Errorf("expected error for id, got: %v", result.Errors)
	}
	if !hasField(result.Errors, "type") {
		t.Errorf("expected error for type, got: %v", result.Errors)
	}
}

func TestConformanceInvalidIDs(t *testing.T) {
	p := loadPlan(t, "invalid-ids")
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("invalid-ids expected Valid=false")
	}
	if !hasFieldMsg(result.Errors, "id", "grp_") {
		t.Errorf("expected error for plan id prefix, got: %v", result.Errors)
	}
	if !hasFieldMsg(result.Errors, "diagnostics[0].id", "diag_") {
		t.Errorf("expected error for diagnostic id prefix, got: %v", result.Errors)
	}
	if !hasFieldMsg(result.Errors, "steps[0].id", "step_") {
		t.Errorf("expected error for step id prefix, got: %v", result.Errors)
	}
}

func TestConformanceInvalidRisk(t *testing.T) {
	p := loadPlan(t, "invalid-risk")
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("invalid-risk expected Valid=false")
	}
	if !hasField(result.Errors, "risk") {
		t.Errorf("expected error for risk, got: %v", result.Errors)
	}
}

func TestConformanceInvalidDuplicateDiag(t *testing.T) {
	p := loadPlan(t, "invalid-duplicate-diag")
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("invalid-duplicate-diag expected Valid=false")
	}
	if !hasFieldMsg(result.Errors, "diagnostics[1].id", "duplicate") {
		t.Errorf("expected error for duplicate diagnostic ID, got: %v", result.Errors)
	}
}

func TestConformanceInvalidAbsolutePath(t *testing.T) {
	p := loadPlan(t, "invalid-absolute-path")
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("invalid-absolute-path expected Valid=false")
	}
	if !hasFieldMsg(result.Errors, "subject", "absolute") {
		t.Errorf("expected error for absolute subject, got: %v", result.Errors)
	}
	if !hasFieldMsg(result.Errors, "diagnostics[0].file", "absolute") {
		t.Errorf("expected error for absolute diagnostic file, got: %v", result.Errors)
	}
	if !hasFieldMsg(result.Errors, "steps[0].target.file", "absolute") {
		t.Errorf("expected error for absolute step target file, got: %v", result.Errors)
	}
}

func TestConformanceInvalidStepRequires(t *testing.T) {
	p := loadPlan(t, "invalid-step-requires")
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("invalid-step-requires expected Valid=false")
	}
	if !hasFieldMsg(result.Errors, "steps[0].requires", "unknown diagnostic") {
		t.Errorf("expected error for unknown requires, got: %v", result.Errors)
	}
}

func TestConformanceInvalidUnsupportedSpecVersion(t *testing.T) {
	p := loadPlan(t, "invalid-unsupported-specversion")
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("invalid-unsupported-specversion expected Valid=false")
	}
	if !hasFieldMsg(result.Errors, "specversion", "9.9") {
		t.Errorf("expected error for unsupported specversion, got: %v", result.Errors)
	}
}

func TestValidatePlanEmptyDiagnostics(t *testing.T) {
	p := validPlan()
	p.Diagnostics = []Diagnostic{}
	p.Steps = []Step{}
	result := ValidatePlan(p)
	if !result.Valid {
		t.Errorf("empty diagnostics should be valid, got errors: %v", result.Errors)
	}
}

func TestValidatePlanEmptySteps(t *testing.T) {
	p := validPlan()
	p.Steps = []Step{}
	result := ValidatePlan(p)
	if !result.Valid {
		t.Errorf("empty steps should be valid, got errors: %v", result.Errors)
	}
}

func TestValidatePlanWithReviewability(t *testing.T) {
	p := validPlan()
	p.Reviewability = &Reviewability{
		Status: "pass",
		Budget: ReviewabilityBudget{
			MaxDiffLines:    500,
			MaxTouchedFiles: 10,
			MaxRisk:         SeverityMedium,
		},
		Observed: ReviewabilityObserved{
			AddedLines:   100,
			DeletedLines: 50,
			ChangedLines: 150,
			ChangedFiles: 3,
			LargestDelta: 80,
		},
		Recommendation: "review",
		Reason:         "Within acceptable change budget",
	}
	result := ValidatePlan(p)
	if !result.Valid {
		t.Errorf("expected valid plan with reviewability, got errors: %v", result.Errors)
	}
}

func TestValidatePlanWithInvalidReviewability(t *testing.T) {
	p := validPlan()
	p.Reviewability = &Reviewability{
		Status:         "invalid_status",
		Recommendation: "invalid_rec",
		Budget: ReviewabilityBudget{
			MaxDiffLines:    -1,
			MaxTouchedFiles: -1,
		},
		Observed: ReviewabilityObserved{
			AddedLines:   -1,
			DeletedLines: -1,
			ChangedFiles: -1,
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan with bad reviewability")
	}
	if !hasField(result.Errors, "reviewability.status") {
		t.Errorf("expected error for reviewability.status, got: %v", result.Errors)
	}
	if !hasField(result.Errors, "reviewability.budget.maxDiffLines") {
		t.Errorf("expected error for reviewability.budget.maxDiffLines, got: %v", result.Errors)
	}
	if !hasField(result.Errors, "reviewability.budget.maxTouchedFiles") {
		t.Errorf("expected error for reviewability.budget.maxTouchedFiles, got: %v", result.Errors)
	}
	if !hasField(result.Errors, "reviewability.observed.addedLines") {
		t.Errorf("expected error for reviewability.observed.addedLines, got: %v", result.Errors)
	}
	if !hasField(result.Errors, "reviewability.observed.deletedLines") {
		t.Errorf("expected error for reviewability.observed.deletedLines, got: %v", result.Errors)
	}
	if !hasField(result.Errors, "reviewability.observed.changedFiles") {
		t.Errorf("expected error for reviewability.observed.changedFiles, got: %v", result.Errors)
	}
}

func TestValidatePlanWithDecomposition(t *testing.T) {
	p := validPlan()
	p.Decomposition = &Decomposition{
		Strategy:   "by-boundary",
		ParentPlan: "grp_parent",
		Tasks: []DecompositionTask{
			{
				ID:             "task_001",
				Goal:           "Extract schema types",
				AllowedFiles:   []string{"schema/"},
				MaxDiffLines:   100,
				Parallelizable: false,
				DependsOn:      nil,
				Verification:   []string{"go build ./..."},
			},
			{
				ID:             "task_002",
				Goal:           "Update API handlers",
				AllowedFiles:   []string{"api/"},
				MaxDiffLines:   200,
				Parallelizable: true,
				DependsOn:      []string{"task_001"},
				Verification:   []string{"go test ./..."},
			},
		},
	}
	result := ValidatePlan(p)
	if !result.Valid {
		t.Errorf("expected valid plan with decomposition, got errors: %v", result.Errors)
	}
}

func TestValidatePlanWithDuplicateDecompTaskIDs(t *testing.T) {
	p := validPlan()
	p.Decomposition = &Decomposition{
		Strategy: "by-boundary",
		Tasks: []DecompositionTask{
			{ID: "task_001", Goal: "First task"},
			{ID: "task_001", Goal: "Duplicate ID"},
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan with duplicate task IDs")
	}
	if !hasFieldMsg(result.Errors, "decomposition.tasks[1].id", "duplicate") {
		t.Errorf("expected error for duplicate task ID, got: %v", result.Errors)
	}
}

func TestValidatePlanWithUnknownDecompDep(t *testing.T) {
	p := validPlan()
	p.Decomposition = &Decomposition{
		Strategy: "by-boundary",
		Tasks: []DecompositionTask{
			{ID: "task_001", Goal: "First task", DependsOn: []string{"task_999"}},
		},
	}
	result := ValidatePlan(p)
	if result.Valid {
		t.Fatal("expected invalid plan with unknown dependency")
	}
	if !hasFieldMsg(result.Errors, "decomposition.tasks[0].dependsOn", "unknown task") {
		t.Errorf("expected error for unknown dependency, got: %v", result.Errors)
	}
}

func TestValidatePlanEmptyVerification(t *testing.T) {
	p := validPlan()
	p.Verification = []Verification{}
	result := ValidatePlan(p)
	if !result.Valid {
		t.Errorf("empty verification should be valid, got errors: %v", result.Errors)
	}
}
