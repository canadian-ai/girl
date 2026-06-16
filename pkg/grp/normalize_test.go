package grp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePlanSortsDiagnostics(t *testing.T) {
	p := &Plan{
		SpecVersion: "0.1",
		ID:          "grp_test",
		Type:        "dev.refactor.plan",
		Source:      "github.com/canadian-ai/girl",
		Subject:     ".",
		Language:    "go",
		Goal:        "test",
		Risk:        SeverityHigh,
		Diagnostics: []Diagnostic{
			{ID: "diag_old_1", Code: "go.low", Severity: SeverityLow, Message: "low", File: "a.go"},
			{ID: "diag_old_2", Code: "go.high", Severity: SeverityHigh, Message: "high", File: "b.go"},
			{ID: "diag_old_3", Code: "go.medium", Severity: SeverityMedium, Message: "medium", File: "c.go"},
		},
	}

	NormalizePlan(p)

	if len(p.Diagnostics) != 3 {
		t.Fatalf("expected 3 diagnostics, got %d", len(p.Diagnostics))
	}

	if p.Diagnostics[0].ID != "diag_001" {
		t.Errorf("first diag ID = %q, want diag_001", p.Diagnostics[0].ID)
	}
	if p.Diagnostics[0].Code != "go.high" {
		t.Errorf("first diag should be high severity, got %s", p.Diagnostics[0].Code)
	}
	if p.Diagnostics[1].ID != "diag_002" || p.Diagnostics[1].Code != "go.medium" {
		t.Errorf("second diag should be medium, got %s", p.Diagnostics[1].Code)
	}
	if p.Diagnostics[2].Code != "go.low" {
		t.Errorf("third diag should be low, got %s", p.Diagnostics[2].Code)
	}
}

func TestNormalizePlanAssignsSequentialDiagIDs(t *testing.T) {
	p := &Plan{
		SpecVersion: "0.1",
		ID:          "grp_test",
		Type:        "dev.refactor.plan",
		Source:      "github.com/canadian-ai/girl",
		Subject:     ".",
		Language:    "go",
		Goal:        "test",
		Risk:        SeverityLow,
		Diagnostics: []Diagnostic{
			{ID: "a", Code: "go.one", Severity: SeverityLow, Message: "one", File: "a.go"},
			{ID: "b", Code: "go.two", Severity: SeverityMedium, Message: "two", File: "a.go"},
			{ID: "c", Code: "go.three", Severity: SeverityHigh, Message: "three", File: "a.go"},
		},
	}

	NormalizePlan(p)

	for i, d := range p.Diagnostics {
		expectedID := strings.TrimLeft(strings.Replace(d.Code, "go.", "", 1), " ")
		if expectedID == "one" && d.ID != "diag_003" {
			t.Errorf("diag[%d] ID = %q, want diag_003", i, d.ID)
		}
		if expectedID == "two" && d.ID != "diag_002" {
			t.Errorf("diag[%d] ID = %q, want diag_002", i, d.ID)
		}
		if expectedID == "three" && d.ID != "diag_001" {
			t.Errorf("diag[%d] ID = %q, want diag_001", i, d.ID)
		}
	}
}

func TestNormalizePlanRequiresRemapping(t *testing.T) {
	p := &Plan{
		SpecVersion: "0.1",
		ID:          "grp_test",
		Type:        "dev.refactor.plan",
		Source:      "github.com/canadian-ai/girl",
		Subject:     ".",
		Language:    "go",
		Goal:        "test",
		Risk:        SeverityLow,
		Diagnostics: []Diagnostic{
			{
				ID: "old_diag_1", Code: "go.one", Severity: SeverityHigh,
				Message: "test", File: "a.go", Symbol: &Symbol{Name: "Foo"},
			},
		},
		Steps: []Step{
			{
				ID: "step_old", Title: "Fix Foo", Action: "Fix Foo",
				Target: Target{File: "a.go"}, Risk: SeverityMedium,
				Requires: []string{"old_diag_1"},
			},
		},
	}

	NormalizePlan(p)

	if len(p.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(p.Steps))
	}
	if len(p.Steps[0].Requires) != 1 {
		t.Fatalf("expected 1 requires, got %d", len(p.Steps[0].Requires))
	}
	if p.Steps[0].Requires[0] != "diag_001" {
		t.Errorf("requires = %q, want diag_001", p.Steps[0].Requires[0])
	}
	if !strings.HasPrefix(p.Steps[0].ID, "step_001_go.one_foo") {
		t.Errorf("step ID = %q, should start with step_001_go.one_foo", p.Steps[0].ID)
	}
}

func TestNormalizePlanDeterministicSort(t *testing.T) {
	a := &Plan{
		SpecVersion: "0.1", ID: "grp_a", Type: "dev.refactor.plan",
		Source: "s", Subject: ".", Language: "go", Goal: "test",
		Risk: SeverityLow,
		Diagnostics: []Diagnostic{
			{ID: "x", Code: "go.b", Severity: SeverityLow, Message: "b", File: "b.go"},
			{ID: "y", Code: "go.a", Severity: SeverityHigh, Message: "a", File: "a.go"},
		},
	}
	b := &Plan{
		SpecVersion: "0.1", ID: "grp_b", Type: "dev.refactor.plan",
		Source: "s", Subject: ".", Language: "go", Goal: "test",
		Risk: SeverityLow,
		Diagnostics: []Diagnostic{
			{ID: "y", Code: "go.a", Severity: SeverityHigh, Message: "a", File: "a.go"},
			{ID: "x", Code: "go.b", Severity: SeverityLow, Message: "b", File: "b.go"},
		},
	}

	NormalizePlan(a)
	NormalizePlan(b)

	for i := range a.Diagnostics {
		if a.Diagnostics[i].ID != b.Diagnostics[i].ID {
			t.Errorf("diag[%d] mismatch: %s vs %s", i, a.Diagnostics[i].ID, b.Diagnostics[i].ID)
		}
	}
}

func TestComputePlanIDDeterministic(t *testing.T) {
	p1 := &Plan{
		SpecVersion: "0.1", ID: "grp_ignore", Type: "dev.refactor.plan",
		Source: "github.com/canadian-ai/girl", Subject: ".", Language: "go",
		Goal: "Refactor long functions", Risk: SeverityMedium,
		Diagnostics: []Diagnostic{
			{Code: "go.long-function", File: "main.go", Line: 10, Symbol: &Symbol{Name: "foo"}},
		},
		Steps: []Step{
			{Recipe: "go.extract-function", Action: "Refactor foo", Target: Target{File: "main.go"}},
		},
	}
	p2 := &Plan{
		SpecVersion: "0.1", ID: "grp_ignore_too", Type: "dev.refactor.plan",
		Source: "github.com/canadian-ai/girl", Subject: ".", Language: "go",
		Goal: "Refactor long functions", Risk: SeverityMedium,
		Diagnostics: []Diagnostic{
			{Code: "go.long-function", File: "main.go", Line: 10, Symbol: &Symbol{Name: "foo"}},
		},
		Steps: []Step{
			{Recipe: "go.extract-function", Action: "Refactor foo", Target: Target{File: "main.go"}},
		},
	}

	id1 := ComputePlanID(p1)
	id2 := ComputePlanID(p2)

	if id1 != id2 {
		t.Errorf("plan IDs should match: %s vs %s", id1, id2)
	}
	if !strings.HasPrefix(id1, "grp_") {
		t.Errorf("plan ID %q should start with grp_", id1)
	}
}

func TestComputePlanIDChangesOnInput(t *testing.T) {
	p1 := &Plan{
		SpecVersion: "0.1", ID: "grp_1", Type: "dev.refactor.plan",
		Source: "github.com/canadian-ai/girl", Subject: ".", Language: "go",
		Goal: "Refactor", Risk: SeverityLow,
	}
	p2 := &Plan{
		SpecVersion: "0.1", ID: "grp_2", Type: "dev.refactor.plan",
		Source: "github.com/canadian-ai/girl", Subject: ".", Language: "go",
		Goal: "Different goal", Risk: SeverityLow,
	}

	if ComputePlanID(p1) == ComputePlanID(p2) {
		t.Errorf("different goals should produce different IDs")
	}
}

func TestNormalizePlanSortsDecompositionTasks(t *testing.T) {
	p := &Plan{
		SpecVersion: "0.1", ID: "grp_test", Type: "dev.refactor.plan",
		Source: "s", Subject: ".", Language: "go", Goal: "test",
		Risk: SeverityLow,
		Decomposition: &Decomposition{
			Strategy: "by-boundary",
			Tasks: []DecompositionTask{
				{
					ID:             "task_z",
					Goal:           "Last",
					AllowedFiles:   []string{"zzz/", "aaa/"},
					ForbiddenFiles: []string{"data/", "config/"},
					DependsOn:      []string{"task_a", "task_m"},
					Verification:   []string{"ztest", "atest"},
				},
				{
					ID:             "task_a",
					Goal:           "First",
					AllowedFiles:   []string{"aaa/", "bbb/"},
					ForbiddenFiles: []string{"old/", "new/"},
					DependsOn:      []string{},
					Verification:   []string{"btest", "atest"},
				},
			},
		},
	}

	NormalizePlan(p)

	if p.Decomposition == nil {
		t.Fatal("Decomposition should not be nil")
	}
	if len(p.Decomposition.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(p.Decomposition.Tasks))
	}
	if p.Decomposition.Tasks[0].ID != "task_a" {
		t.Errorf("first task ID = %q, want %q", p.Decomposition.Tasks[0].ID, "task_a")
	}
	if p.Decomposition.Tasks[1].ID != "task_z" {
		t.Errorf("second task ID = %q, want %q", p.Decomposition.Tasks[1].ID, "task_z")
	}
	assertSorted := func(name string, got []string) {
		t.Helper()
		for i := 1; i < len(got); i++ {
			if got[i-1] > got[i] {
				t.Errorf("%s not sorted: %v", name, got)
				break
			}
		}
	}
	assertSorted("task_a.AllowedFiles", p.Decomposition.Tasks[0].AllowedFiles)
	assertSorted("task_a.ForbiddenFiles", p.Decomposition.Tasks[0].ForbiddenFiles)
	assertSorted("task_z.AllowedFiles", p.Decomposition.Tasks[1].AllowedFiles)
	assertSorted("task_z.ForbiddenFiles", p.Decomposition.Tasks[1].ForbiddenFiles)
	assertSorted("task_z.DependsOn", p.Decomposition.Tasks[1].DependsOn)
	assertSorted("task_z.Verification", p.Decomposition.Tasks[1].Verification)
}

func TestNormalizePlanNil(t *testing.T) {
	NormalizePlan(nil)
}

func TestNormalizePlanStepRequiresFromFilePathWithoutSymbol(t *testing.T) {
	p := &Plan{
		SpecVersion: "0.1", ID: "grp_test", Type: "dev.refactor.plan",
		Source: "s", Subject: ".", Language: "go", Goal: "test",
		Risk: SeverityLow,
		Diagnostics: []Diagnostic{
			{
				ID: "old", Code: "go.long-function", Severity: SeverityHigh,
				Message: "msg", File: "internal/handler.go",
			},
		},
		Steps: []Step{
			{
				ID: "step_old", Title: "Refactor", Action: "Refactor handler",
				Target: Target{File: "internal/handler.go"}, Risk: SeverityLow,
				Requires: []string{"old"},
			},
		},
	}

	NormalizePlan(p)

	if len(p.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(p.Steps))
	}
	if !strings.Contains(p.Steps[0].ID, "handler") {
		t.Errorf("step ID %q should contain handler", p.Steps[0].ID)
	}
}

func loadPlanFixture(t *testing.T, dir string) *Plan {
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

func TestConformanceNormalizeValidFull(t *testing.T) {
	p := loadPlanFixture(t, "valid-full")
	NormalizePlan(p)

	if len(p.Diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(p.Diagnostics))
	}
	if p.Diagnostics[0].ID != "diag_001" {
		t.Errorf("first diag ID = %q, want diag_001", p.Diagnostics[0].ID)
	}
	if p.Diagnostics[0].Code != "go.high-complexity" {
		t.Errorf("first diag should be highest severity, got %s", p.Diagnostics[0].Code)
	}
	if p.Diagnostics[1].ID != "diag_002" {
		t.Errorf("second diag ID = %q, want diag_002", p.Diagnostics[1].ID)
	}

	if len(p.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(p.Steps))
	}
	if p.Steps[0].Requires[0] != "diag_001" {
		t.Errorf("first step requires = %q, want diag_001", p.Steps[0].Requires[0])
	}
	if p.Steps[1].Requires[0] != "diag_002" {
		t.Errorf("second step requires = %q, want diag_002", p.Steps[1].Requires[0])
	}
}

func TestConformanceComputePlanIDDeterministic(t *testing.T) {
	p1 := loadPlanFixture(t, "valid-full")
	p2 := loadPlanFixture(t, "valid-full")
	id1 := ComputePlanID(p1)
	id2 := ComputePlanID(p2)
	if id1 != id2 {
		t.Errorf("ComputePlanID should be deterministic: %s vs %s", id1, id2)
	}
	if !strings.HasPrefix(id1, "grp_") {
		t.Errorf("plan ID %q should start with grp_", id1)
	}
}

func TestConformanceComputePlanIDDifferentPlans(t *testing.T) {
	id1 := ComputePlanID(loadPlanFixture(t, "valid-minimal"))
	id2 := ComputePlanID(loadPlanFixture(t, "valid-full"))
	if id1 == id2 {
		t.Errorf("different plans should produce different IDs, both got %s", id1)
	}
}
