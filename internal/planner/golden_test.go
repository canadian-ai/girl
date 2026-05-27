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
