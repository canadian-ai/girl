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

func TestStepForDiagnostic_OpenRewriteRecipes(t *testing.T) {
	cases := []struct {
		code       string
		wantRecipe string
		wantVerify string
	}{
		{"openrewrite.refactor-opportunity", "openrewrite.export-yaml-recipe", "mvn rewrite:run"},
		{"openrewrite.export-yaml", "openrewrite.export-yaml-recipe", "mvn rewrite:run"},
	}
	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			diag := ir.Diagnostic{
				Code:     c.code,
				Severity: ir.SeverityMedium,
				Symbol:   "LegacyService",
				File:     "src/main/java/com/example/LegacyService.java",
			}
			step := StepForDiagnostic(diag)
			if step.Recipe != c.wantRecipe {
				t.Errorf("StepForDiagnostic(%q).Recipe = %q, want %q", c.code, step.Recipe, c.wantRecipe)
			}
			if step.Action == "" {
				t.Errorf("StepForDiagnostic(%q).Action should not be empty", c.code)
			}
			if len(step.Verify) == 0 || step.Verify[0] != c.wantVerify {
				t.Errorf("StepForDiagnostic(%q).Verify[0] = %q, want %q", c.code, step.Verify[0], c.wantVerify)
			}
		})
	}
}

func TestStepForDiagnostic_RTKRecipes(t *testing.T) {
	cases := []struct {
		code       string
		wantRecipe string
	}{
		{"rtk.optimize-commands", "rtk.pipe-through-proxy"},
		{"rtk.hook-configure", "rtk.pipe-through-proxy"},
	}
	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			diag := ir.Diagnostic{
				Code:   c.code,
				Symbol: "girl",
				File:   ".claude/settings.json",
			}
			step := StepForDiagnostic(diag)
			if step.Recipe != c.wantRecipe {
				t.Errorf("StepForDiagnostic(%q).Recipe = %q, want %q", c.code, step.Recipe, c.wantRecipe)
			}
			if step.Action == "" {
				t.Errorf("StepForDiagnostic(%q).Action should not be empty", c.code)
			}
			if len(step.Verify) == 0 || step.Verify[0] != "rtk version" {
				t.Errorf("StepForDiagnostic(%q).Verify[0] = %q, want 'rtk version'", c.code, step.Verify[0])
			}
			if step.File != ".claude/settings.json" {
				t.Errorf("StepForDiagnostic(%q).File = %q, want '.claude/settings.json'", c.code, step.File)
			}
		})
	}
}

func TestStepForDiagnostic_GritQLRecipes(t *testing.T) {
	cases := []struct {
		code       string
		wantRecipe string
		wantSeverity ir.Severity
	}{
		{"gritql.pattern-available", "gritql.generate-pattern", ir.SeverityMedium},
		{"gritql.generate-pattern", "gritql.generate-pattern", ir.SeverityLow},
		{"gritql.apply-transform", "gritql.generate-pattern", ir.SeverityMedium},
	}
	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			diag := ir.Diagnostic{
				Code:     c.code,
				Severity: ir.SeverityMedium,
				Symbol:   "extractMethod",
				File:     "src/transform.ts",
			}
			step := StepForDiagnostic(diag)
			if step.Recipe != c.wantRecipe {
				t.Errorf("StepForDiagnostic(%q).Recipe = %q, want %q", c.code, step.Recipe, c.wantRecipe)
			}
			if step.Action == "" {
				t.Errorf("StepForDiagnostic(%q).Action should not be empty", c.code)
			}
		})
	}
}

func TestStepForDiagnostic_RustLSPRecipes(t *testing.T) {
	cases := []struct {
		code       string
		wantRecipe string
	}{
		{"rust-lsp.long-function", "rust-lsp.export-diagnostics"},
		{"rust-lsp.high-complexity", "rust-lsp.export-diagnostics"},
		{"rust-lsp.deep-nesting", "rust-lsp.export-diagnostics"},
		{"rust-lsp.large-file", "rust-lsp.export-diagnostics"},
		{"rust-lsp.export-diagnostics", "rust-lsp.export-diagnostics"},
	}
	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			diag := ir.Diagnostic{
				Code:     c.code,
				Severity: ir.SeverityHigh,
				Symbol:   "process_data",
				File:     "src/main.rs",
			}
			step := StepForDiagnostic(diag)
			if step.Recipe != c.wantRecipe {
				t.Errorf("StepForDiagnostic(%q).Recipe = %q, want %q", c.code, step.Recipe, c.wantRecipe)
			}
			if step.Action == "" {
				t.Errorf("StepForDiagnostic(%q).Action should not be empty", c.code)
			}
			if step.File != "src/main.rs" {
				t.Errorf("StepForDiagnostic(%q).File = %q, want 'src/main.rs'", c.code, step.File)
			}
		})
	}
}

func TestStepForDiagnostic_RustLSP_SeverityPassthrough(t *testing.T) {
	cases := []struct {
		name     string
		code     string
		severity ir.Severity
	}{
		{"low", "rust-lsp.long-function", ir.SeverityLow},
		{"medium", "rust-lsp.high-complexity", ir.SeverityMedium},
		{"high", "rust-lsp.deep-nesting", ir.SeverityHigh},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			diag := ir.Diagnostic{Code: c.code, Severity: c.severity, Symbol: "test", File: "test.rs"}
			step := StepForDiagnostic(diag)
			if step.Risk != c.severity {
				t.Errorf("StepForDiagnostic(%q).Risk = %q, want %q", c.code, step.Risk, c.severity)
			}
		})
	}
}

func TestStepForDiagnostic_RustLSP_Verify(t *testing.T) {
	diag := ir.Diagnostic{
		Code:   "rust-lsp.long-function",
		Symbol: "process_data",
		File:   "src/main.rs",
	}
	step := StepForDiagnostic(diag)
	if len(step.Verify) == 0 {
		t.Fatal("expected non-empty Verify")
	}
	if step.Verify[0] != "cargo check" {
		t.Errorf("expected verify[0] = 'cargo check', got %q", step.Verify[0])
	}
}

func TestStepForDiagnostic_RustLSP_HighComplexityVerify(t *testing.T) {
	diag := ir.Diagnostic{
		Code:   "rust-lsp.high-complexity",
		Symbol: "computeScore",
		File:   "score.rs",
	}
	step := StepForDiagnostic(diag)
	if len(step.Verify) < 2 {
		t.Fatal("expected at least 2 verify commands for high-complexity")
	}
	if step.Verify[0] != "cargo check" {
		t.Errorf("expected verify[0] = 'cargo check', got %q", step.Verify[0])
	}
	if step.Verify[1] != "cargo clippy" {
		t.Errorf("expected verify[1] = 'cargo clippy', got %q", step.Verify[1])
	}
}

func TestStepForDiagnostic_OpenRewriteVerifys(t *testing.T) {
	diag := ir.Diagnostic{
		Code:   "openrewrite.refactor-opportunity",
		Symbol: "LegacyService",
		File:   "LegacyService.java",
	}
	step := StepForDiagnostic(diag)
	if len(step.Verify) < 2 {
		t.Fatal("expected 2 verify commands for openrewrite")
	}
	expected := []string{"mvn rewrite:run", "gradle rewriteRun"}
	for i, v := range expected {
		if step.Verify[i] != v {
			t.Errorf("expected verify[%d] = %q, got %q", i, v, step.Verify[i])
		}
	}
}

func TestStepForDiagnostic_GritQLVerifyDefault(t *testing.T) {
	diag := ir.Diagnostic{
		Code:   "gritql.apply-transform",
		Symbol: "refactorMe",
		File:   "target.ts",
	}
	step := StepForDiagnostic(diag)
	if len(step.Verify) < 2 {
		t.Fatal("expected 2 verify commands for gritql.apply-transform")
	}
	if step.Verify[0] != "grit apply" {
		t.Errorf("expected verify[0] = 'grit apply', got %q", step.Verify[0])
	}
	if step.Verify[1] != "grit check" {
		t.Errorf("expected verify[1] = 'grit check', got %q", step.Verify[1])
	}
}

func TestRegistered_NewRecipeCount(t *testing.T) {
	registered := Registered()
	// 24 existing (7 react + 6 go + 8 agent + 2 openrewrite + 2 rtk + 3 gritql + 5 rust-lsp)
	if len(registered) < 28 {
		t.Errorf("expected at least 28 registered recipes with new tool diagnostics, got %d", len(registered))
	}
	// Verify new ones are present
	expectedCodes := []string{
		"openrewrite.refactor-opportunity",
		"openrewrite.export-yaml",
		"rtk.optimize-commands",
		"rtk.hook-configure",
		"gritql.pattern-available",
		"gritql.generate-pattern",
		"gritql.apply-transform",
		"rust-lsp.long-function",
		"rust-lsp.high-complexity",
		"rust-lsp.deep-nesting",
		"rust-lsp.large-file",
		"rust-lsp.export-diagnostics",
	}
	codeSet := map[string]bool{}
	for _, r := range registered {
		codeSet[r.Code] = true
	}
	for _, code := range expectedCodes {
		if !codeSet[code] {
			t.Errorf("expected registered recipe code %q not found", code)
		}
	}
}
