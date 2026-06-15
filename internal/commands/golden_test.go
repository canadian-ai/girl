package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/canadian-ai/girl/internal/analyzer"
	"github.com/canadian-ai/girl/internal/goanalysis"
	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/packer"
	"github.com/canadian-ai/girl/internal/planner"
	"github.com/canadian-ai/girl/pkg/grp"
)

const cmdRepoRoot = "../../"

func chdirRepoRoot(t *testing.T) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	repoAbs, err := filepath.Abs(cmdRepoRoot)
	if err != nil {
		t.Fatalf("repo abs: %v", err)
	}
	if err := os.Chdir(repoAbs); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
}

func readGoldenOrUpdate(t *testing.T, path string, data []byte) []byte {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		return data
	}
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	return expected
}

func assertJSONEqual(t *testing.T, name string, got, expected []byte) {
	t.Helper()
	var gotV, expV map[string]interface{}
	if err := json.Unmarshal(got, &gotV); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if err := json.Unmarshal(expected, &expV); err != nil {
		t.Fatalf("unmarshal expected: %v", err)
	}
	gotNorm, _ := json.MarshalIndent(gotV, "", "  ")
	expNorm, _ := json.MarshalIndent(expV, "", "  ")
	if string(gotNorm) != string(expNorm) {
		t.Errorf("golden mismatch for %s\n--- got:\n%s\n--- expected:\n%s", name, string(gotNorm), string(expNorm))
	}
}

func TestGoldenAnalyze(t *testing.T) {
	chdirRepoRoot(t)

	scenarios := []struct {
		name string
		path string
		lang string
	}{
		{name: "analyze-go", path: "testdata/golden/commands/simple-go", lang: "go"},
		{name: "analyze-react", path: "testdata/golden/commands/simple-react", lang: "ts"},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			expectedPath := filepath.Join("testdata/golden/commands", s.name+".expected.json")

			var result *ir.AnalyzerResult
			var err error
			if s.lang == "go" {
				result, err = goanalysis.AnalyzePath(s.path, nil)
			} else {
				result, err = analyzer.NewAnalyzer(nil).Analyze(s.path)
			}
			if err != nil {
				t.Fatalf("analyze failed: %v", err)
			}

			gotJSON, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			expectedData := readGoldenOrUpdate(t, expectedPath, gotJSON)
			assertJSONEqual(t, s.name, gotJSON, expectedData)
		})
	}
}

func TestGoldenPlanGRPJSON(t *testing.T) {
	chdirRepoRoot(t)
	goal := "Improve code quality"

	scenarios := []struct {
		name string
		path string
		lang string
	}{
		{name: "plan-go", path: "testdata/golden/commands/simple-go", lang: "go"},
		{name: "plan-react", path: "testdata/golden/commands/simple-react", lang: "ts"},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			expectedPath := filepath.Join("testdata/golden/commands", s.name+".expected.json")

			var result *ir.AnalyzerResult
			var err error
			if s.lang == "go" {
				result, err = goanalysis.AnalyzePath(s.path, nil)
			} else {
				result, err = analyzer.NewAnalyzer(nil).Analyze(s.path)
			}
			if err != nil {
				t.Fatalf("analyze failed: %v", err)
			}

			p := planner.NewPlanner()
			plan := p.GeneratePlan(planner.PlanRequest{
				Target:      s.path,
				Goal:        goal,
				Diagnostics: result.Diagnostics,
				Files:       result.Files,
				Lang:        s.lang,
			})

			gp := grp.FromIRPlan(plan)
			gp.Language = s.lang
			grp.NormalizePlan(gp)
			gp.ID = grp.ComputePlanID(gp)

			gotJSON, err := json.MarshalIndent(gp, "", "  ")
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			expectedData := readGoldenOrUpdate(t, expectedPath, gotJSON)
			assertJSONEqual(t, s.name, gotJSON, expectedData)
		})
	}
}

func TestGoldenPackGRPContextJSON(t *testing.T) {
	chdirRepoRoot(t)
	goal := "Improve code quality"

	scenarios := []struct {
		name   string
		path   string
		lang   string
		budget int
	}{
		{name: "pack-go", path: "testdata/golden/commands/simple-go", lang: "go", budget: 4000},
		{name: "pack-react", path: "testdata/golden/commands/simple-react", lang: "ts", budget: 4000},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			expectedPath := filepath.Join("testdata/golden/commands", s.name+".expected.json")

			var result *ir.AnalyzerResult
			var err error
			if s.lang == "go" {
				result, err = goanalysis.AnalyzePath(s.path, nil)
			} else {
				result, err = analyzer.NewAnalyzer(nil).Analyze(s.path)
			}
			if err != nil {
				t.Fatalf("analyze failed: %v", err)
			}

			p := planner.NewPlanner()
			plan := p.GeneratePlan(planner.PlanRequest{
				Target:      s.path,
				Goal:        goal,
				Diagnostics: result.Diagnostics,
				Files:       result.Files,
				Lang:        s.lang,
			})

			pkr := packer.NewPacker(s.budget)
			pack, err := pkr.Pack(packer.PackRequest{
				Goal:        plan.Goal,
				Files:       result.Files,
				Diagnostics: plan.Diagnostics,
				Steps:       plan.Steps,
				TokenBudget: s.budget,
				PlanID:      plan.PlanID,
				TargetPath:  s.path,
				PrivacyMode: "private",
			})
			if err != nil {
				t.Fatalf("pack failed: %v", err)
			}

			planID := canonicalGRPPlanID(plan, s.lang)
			gcp := pkr.GrpContextPack(pack, planID)

			gotJSON, err := json.MarshalIndent(gcp, "", "  ")
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			expectedData := readGoldenOrUpdate(t, expectedPath, gotJSON)
			assertJSONEqual(t, s.name, gotJSON, expectedData)
		})
	}
}

func TestGoldenValidate(t *testing.T) {
	chdirRepoRoot(t)

	t.Run("validate-valid-plan", func(t *testing.T) {
		plan := &grp.Plan{
			SpecVersion: "0.1",
			ID:          "grp_test1234",
			Type:        "dev.refactor.plan",
			Source:      "github.com/canadian-ai/girl",
			Subject:     "testdata/golden/commands/simple-go",
			Language:    "go",
			Goal:        "Improve code quality",
			Risk:        grp.SeverityLow,
			Diagnostics: []grp.Diagnostic{
				{
					ID:         "diag_001",
					Code:       "go.long-function",
					Severity:   grp.SeverityLow,
					Confidence: grp.ConfidenceHigh,
					Message:    "Function longFunc is too long",
					File:       "main.go",
				},
			},
			Steps: []grp.Step{
				{
					ID:     "step_001_extract-main",
					Recipe: "extract-function",
					Title:  "Extract function main",
					Action: "Extract longFunc into smaller functions",
					Target: grp.Target{File: "main.go"},
					Risk:   grp.SeverityLow,
					Verify: []grp.Verification{
						{Command: "go build ./...", Required: true, Source: "binding-default", Confidence: "medium"},
					},
				},
			},
			Verification: []grp.Verification{
				{Command: "go test ./...", Required: true, Source: "binding-default", Confidence: "medium"},
			},
		}

		result := grp.ValidatePlan(plan)
		if !result.Valid {
			t.Fatalf("expected valid plan, got errors: %v", result.Errors)
		}

		expectedPath := "testdata/golden/commands/validate-valid-plan.expected.json"
		gotJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		expectedData := readGoldenOrUpdate(t, expectedPath, gotJSON)
		assertJSONEqual(t, "validate-valid-plan", gotJSON, expectedData)
	})

	t.Run("validate-invalid-plan", func(t *testing.T) {
		plan := &grp.Plan{
			SpecVersion: "0.1",
			ID:          "",
			Type:        "",
			Goal:        "",
			Diagnostics: []grp.Diagnostic{
				{ID: "", Code: "", Severity: grp.SeverityLow, Confidence: grp.ConfidenceHigh, Message: "", File: ""},
			},
			Steps: []grp.Step{
				{
					ID:     "",
					Title:  "",
					Action: "",
					Target: grp.Target{File: ""},
					Risk:   grp.SeverityLow,
				},
			},
		}

		result := grp.ValidatePlan(plan)
		if result.Valid {
			t.Fatal("expected invalid plan")
		}

		expectedPath := "testdata/golden/commands/validate-invalid-plan.expected.json"
		gotJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		expectedData := readGoldenOrUpdate(t, expectedPath, gotJSON)
		assertJSONEqual(t, "validate-invalid-plan", gotJSON, expectedData)
	})
}
