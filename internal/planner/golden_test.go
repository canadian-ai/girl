package planner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/canadian-ai/girl/internal/goanalysis"
	"github.com/canadian-ai/girl/pkg/grp"
)

type goldenScenario struct {
	name   string
	input  string
	lang   string
}

func TestGoldenGRPPlans(t *testing.T) {
	repoRoot := "../../"

	scenarios := []goldenScenario{
		{name: "minimal-core", input: "testdata/golden/minimal-core/input", lang: "go"},
		{name: "grp-go", input: "testdata/golden/grp-go/input", lang: "go"},
		{name: "grp-react", input: "testdata/golden/grp-react/input", lang: "ts"},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			absInput, err := filepath.Abs(filepath.Join(repoRoot, s.input))
			if err != nil {
				t.Fatalf("abs path: %v", err)
			}
			expectedPath := filepath.Join(repoRoot, "testdata/golden", s.name, "expected.plan.json")

			result, err := goanalysis.AnalyzePath(absInput, nil)
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

			delete(got, "id")
			delete(expected, "id")

			gotNorm, _ := json.MarshalIndent(got, "", "  ")
			expNorm, _ := json.MarshalIndent(expected, "", "  ")

			if string(gotNorm) != string(expNorm) {
				t.Errorf("golden mismatch for %s\n--- got:\n%s\n--- expected:\n%s", s.name, string(gotNorm), string(expNorm))
			}
		})
	}
}
