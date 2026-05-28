package planner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/canadian-ai/girl/internal/analyzer"
	"github.com/canadian-ai/girl/internal/goanalysis"
	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/pkg/grp"
)

type goldenScenario struct {
	name  string
	input string
	lang  string
}

func analyzeGolden(path, lang string) (*ir.AnalyzerResult, error) {
	if lang == "go" {
		return goanalysis.AnalyzePath(path, nil)
	}
	return analyzer.NewAnalyzer(nil).Analyze(path)
}

func TestGoldenGRPPlans(t *testing.T) {
	repoRoot := "../../"

	scenarios := []goldenScenario{
		{name: "minimal-core", input: "testdata/golden/minimal-core/input", lang: "go"},
		{name: "grp-go", input: "testdata/golden/grp-go/input", lang: "go"},
		{name: "grp-react", input: "testdata/golden/grp-react/input", lang: "ts"},
		{name: "go-high-complexity", input: "testdata/golden/go-high-complexity/input", lang: "go"},
		{name: "react-too-many-hooks", input: "testdata/golden/react-too-many-hooks/input", lang: "ts"},
		{name: "go-multi-diag", input: "testdata/golden/go-multi-diag/input", lang: "go"},
		{name: "react-multi-diag", input: "testdata/golden/react-multi-diag/input", lang: "ts"},
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatalf("repo abs: %v", err)
	}
	if err := os.Chdir(repoAbs); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			expectedPath := filepath.Join("testdata/golden", s.name, "expected.plan.json")

			result, err := analyzeGolden(s.input, s.lang)
			if err != nil {
				t.Fatalf("analyze failed: %v", err)
			}

			p := NewPlanner()
			plan := p.GeneratePlan(PlanRequest{
				Target:      s.input,
				Goal:        "Improve code quality",
				Diagnostics: result.Diagnostics,
				Files:       result.Files,
				Lang:        s.lang,
			})

			grpPlan := grp.FromIRPlan(plan)
			grpPlan.Language = s.lang
			grp.NormalizePlan(grpPlan)
			grpPlan.ID = grp.ComputePlanID(grpPlan)
			gotJSON, err := json.MarshalIndent(grpPlan, "", "  ")
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			expectedData, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("read expected %s: %v", expectedPath, err)
			}

			var got, expected map[string]interface{}
			if err := json.Unmarshal(gotJSON, &got); err != nil {
				t.Fatalf("unmarshal got: %v", err)
			}
			if err := json.Unmarshal(expectedData, &expected); err != nil {
				t.Fatalf("unmarshal expected: %v", err)
			}

			gotNorm, _ := json.MarshalIndent(got, "", "  ")
			expNorm, _ := json.MarshalIndent(expected, "", "  ")

			if string(gotNorm) != string(expNorm) {
				t.Errorf("golden mismatch for %s\n--- got:\n%s\n--- expected:\n%s", s.name, string(gotNorm), string(expNorm))
			}
		})
	}
}

func TestComputePlanIDDeterministic(t *testing.T) {
	repoRoot := "../../"
	absInput, err := filepath.Abs(filepath.Join(repoRoot, "testdata/golden/grp-go/input"))
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}

	result, err := goanalysis.AnalyzePath(absInput, nil)
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}

	p := NewPlanner()
	plan := p.GeneratePlan(PlanRequest{
		Target:      "testdata/golden/grp-go/input",
		Goal:        "Improve code quality",
		Diagnostics: result.Diagnostics,
		Files:       result.Files,
		Lang:        "go",
	})

	gp1 := grp.FromIRPlan(plan)
	gp1.Language = "go"
	grp.NormalizePlan(gp1)
	gp1.ID = grp.ComputePlanID(gp1)

	gp2 := grp.FromIRPlan(plan)
	gp2.Language = "go"
	grp.NormalizePlan(gp2)
	gp2.ID = grp.ComputePlanID(gp2)

	if gp1.ID != gp2.ID {
		t.Errorf("ComputePlanID not deterministic: got %q first, %q second", gp1.ID, gp2.ID)
	}
}

func TestDiagIDsDeterministic(t *testing.T) {
	repoRoot := "../../"
	absInput, err := filepath.Abs(filepath.Join(repoRoot, "testdata/golden/go-multi-diag/input"))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatalf("abs root: %v", err)
	}
	if err := os.Chdir(absRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	result1, err := goanalysis.AnalyzePath(absInput, nil)
	if err != nil {
		t.Fatalf("analyze 1: %v", err)
	}
	result2, err := goanalysis.AnalyzePath(absInput, nil)
	if err != nil {
		t.Fatalf("analyze 2: %v", err)
	}

	p := NewPlanner()
	plan1 := p.GeneratePlan(PlanRequest{
		Target: "testdata/golden/go-multi-diag/input", Goal: "Improve code quality",
		Diagnostics: result1.Diagnostics, Files: result1.Files, Lang: "go",
	})
	plan2 := p.GeneratePlan(PlanRequest{
		Target: "testdata/golden/go-multi-diag/input", Goal: "Improve code quality",
		Diagnostics: result2.Diagnostics, Files: result2.Files, Lang: "go",
	})

	gp1 := grp.FromIRPlan(plan1)
	gp1.Language = "go"
	grp.NormalizePlan(gp1)

	gp2 := grp.FromIRPlan(plan2)
	gp2.Language = "go"
	grp.NormalizePlan(gp2)

	if len(gp1.Diagnostics) != len(gp2.Diagnostics) {
		t.Fatalf("diag count mismatch: %d vs %d", len(gp1.Diagnostics), len(gp2.Diagnostics))
	}
	for i := range gp1.Diagnostics {
		if gp1.Diagnostics[i].ID != gp2.Diagnostics[i].ID {
			t.Errorf("diag[%d] ID mismatch: %q vs %q", i, gp1.Diagnostics[i].ID, gp2.Diagnostics[i].ID)
		}
	}
}

func TestStepIDsDeterministic(t *testing.T) {
	repoRoot := "../../"
	absInput, err := filepath.Abs(filepath.Join(repoRoot, "testdata/golden/react-multi-diag/input"))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatalf("abs root: %v", err)
	}
	if err := os.Chdir(absRoot); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	result1, err := analyzer.NewAnalyzer(nil).Analyze(absInput)
	if err != nil {
		t.Fatalf("analyze 1: %v", err)
	}
	result2, err := analyzer.NewAnalyzer(nil).Analyze(absInput)
	if err != nil {
		t.Fatalf("analyze 2: %v", err)
	}

	p := NewPlanner()
	plan1 := p.GeneratePlan(PlanRequest{
		Target: "testdata/golden/react-multi-diag/input", Goal: "Improve code quality",
		Diagnostics: result1.Diagnostics, Files: result1.Files, Lang: "ts",
	})
	plan2 := p.GeneratePlan(PlanRequest{
		Target: "testdata/golden/react-multi-diag/input", Goal: "Improve code quality",
		Diagnostics: result2.Diagnostics, Files: result2.Files, Lang: "ts",
	})

	gp1 := grp.FromIRPlan(plan1)
	gp1.Language = "ts"
	grp.NormalizePlan(gp1)

	gp2 := grp.FromIRPlan(plan2)
	gp2.Language = "ts"
	grp.NormalizePlan(gp2)

	if len(gp1.Steps) != len(gp2.Steps) {
		t.Fatalf("step count mismatch: %d vs %d", len(gp1.Steps), len(gp2.Steps))
	}
	for i := range gp1.Steps {
		if gp1.Steps[i].ID != gp2.Steps[i].ID {
			t.Errorf("step[%d] ID mismatch: %q vs %q", i, gp1.Steps[i].ID, gp2.Steps[i].ID)
		}
	}
}

func TestStepIDIndependentOfActionText(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoAbs, err := filepath.Abs("../../")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if err := os.Chdir(repoAbs); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	absInput, err := filepath.Abs("testdata/golden/go-multi-diag/input")
	if err != nil {
		t.Fatalf("abs input: %v", err)
	}
	result, err := goanalysis.AnalyzePath(absInput, nil)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}

	p := NewPlanner()
	base := p.GeneratePlan(PlanRequest{
		Target: "testdata/golden/go-multi-diag/input", Goal: "Improve code quality",
		Diagnostics: result.Diagnostics, Files: result.Files, Lang: "go",
	})

	modified := p.GeneratePlan(PlanRequest{
		Target: "testdata/golden/go-multi-diag/input", Goal: "Improve code quality",
		Diagnostics: result.Diagnostics, Files: result.Files, Lang: "go",
	})

	gpBase := grp.FromIRPlan(base)
	gpBase.Language = "go"
	grp.NormalizePlan(gpBase)

	gpMod := grp.FromIRPlan(modified)
	gpMod.Language = "go"

	for i := range gpMod.Steps {
		gpMod.Steps[i].Action = "MODIFIED: " + gpMod.Steps[i].Action
	}
	grp.NormalizePlan(gpMod)

	if len(gpBase.Steps) != len(gpMod.Steps) {
		t.Fatalf("step count differs after modifying action text: %d vs %d", len(gpBase.Steps), len(gpMod.Steps))
	}
	for i := range gpBase.Steps {
		if gpBase.Steps[i].ID != gpMod.Steps[i].ID {
			t.Errorf("step[%d] ID changed after action text modification: %q vs %q", i, gpBase.Steps[i].ID, gpMod.Steps[i].ID)
		}
	}
}

func TestDuplicateDiagCodesDeterministic(t *testing.T) {
	makePlan := func() *grp.Plan {
		p := &grp.Plan{
			SpecVersion: "0.1", Type: "dev.refactor.plan",
			Source: "s", Subject: ".", Language: "go", Goal: "test",
			Risk: grp.SeverityLow,
			Diagnostics: []grp.Diagnostic{
				{ID: "a", Code: "go.long-function", Severity: grp.SeverityMedium, Message: "func foo", File: "a.go", Line: 10, Symbol: &grp.Symbol{Name: "Foo"}},
				{ID: "b", Code: "go.long-function", Severity: grp.SeverityLow, Message: "func bar", File: "b.go", Line: 5, Symbol: &grp.Symbol{Name: "Bar"}},
				{ID: "c", Code: "go.long-function", Severity: grp.SeverityHigh, Message: "func baz", File: "c.go", Line: 1, Symbol: &grp.Symbol{Name: "Baz"}},
			},
		}
		return p
	}

	p1 := makePlan()
	p2 := makePlan()
	grp.NormalizePlan(p1)
	grp.NormalizePlan(p2)

	if len(p1.Diagnostics) != len(p2.Diagnostics) {
		t.Fatalf("diag count mismatch: %d vs %d", len(p1.Diagnostics), len(p2.Diagnostics))
	}
	for i := range p1.Diagnostics {
		if p1.Diagnostics[i].ID != p2.Diagnostics[i].ID {
			t.Errorf("diag[%d] ID mismatch: %q vs %q", i, p1.Diagnostics[i].ID, p2.Diagnostics[i].ID)
		}
	}
}
