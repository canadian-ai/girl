package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultGirlProjectConfigDetectsCAIStack(t *testing.T) {
	dir := t.TempDir()
	pkg := `{
  "workspaces": ["apps/*", "packages/*"],
  "dependencies": {
    "next": "16.2.0",
    "convex": "1.34.1",
    "@clerk/nextjs": "7.0.8"
  },
  "scripts": {
    "type-check": "tsc --noEmit",
    "build:ci": "next build",
    "lint": "eslint ."
  }
}`
	mustWriteTestFile(t, filepath.Join(dir, "package.json"), pkg)
	mustWriteTestFile(t, filepath.Join(dir, "bun.lock"), "")
	if err := os.MkdirAll(filepath.Join(dir, ".cai"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".opencode"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "convex"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg, err := defaultGirlProjectConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"cai", "clerk", "convex", "monorepo", "next"} {
		if !hasString(cfg.Profiles, profile) {
			t.Fatalf("profiles %v missing %q", cfg.Profiles, profile)
		}
	}
	if len(cfg.Workspaces) != 2 {
		t.Fatalf("workspaces = %v", cfg.Workspaces)
	}
	if !cfg.Agents["opencode"] {
		t.Fatal("expected opencode agent detection")
	}
	if got := cfg.Verify["typecheck"]; len(got) != 1 || got[0] != "bun run type-check" {
		t.Fatalf("typecheck commands = %v", got)
	}
	if cfg.Reviewability.MaxDiffLines != 1500 || cfg.Reviewability.MaxTouchedFiles != 12 {
		t.Fatalf("reviewability defaults = %#v", cfg.Reviewability)
	}
}

func TestGirlProjectConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &GirlProjectConfig{
		Version:       "1",
		Profiles:      []string{"cai", "next"},
		Workspaces:    []string{"."},
		Analysis:      GirlAnalysisConfig{ChangedOnly: true, Exclude: []string{"node_modules", ".next"}},
		Complexity:    GirlComplexityConfig{Max: 10, Baseline: ".girl/complexity-baseline.json", FailOn: "regression"},
		Reviewability: GirlReviewabilityConfig{MaxDiffLines: 1200, MaxTouchedFiles: 10, MaxRisk: "medium"},
		Verify: map[string][]string{
			"typecheck": {"bun run type-check"},
			"build":     {"bun run build:ci"},
		},
		Agents: map[string]bool{"opencode": true, "codex": true, "claude": false},
	}
	configPath := filepath.Join(dir, ".girl", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(renderGirlProjectConfig(cfg)), 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := loadGirlProjectConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Analysis.ChangedOnly || loaded.Complexity.Max != 10 {
		t.Fatalf("loaded config = %#v", loaded)
	}
	if loaded.Reviewability.MaxDiffLines != 1200 || loaded.Reviewability.MaxTouchedFiles != 10 {
		t.Fatalf("loaded reviewability = %#v", loaded.Reviewability)
	}
	if got := loaded.Verify["build"]; len(got) != 1 || got[0] != "bun run build:ci" {
		t.Fatalf("build commands = %v", got)
	}
}

func TestAnalyzeCheckScopeCleanChangedOnlyDoesNotScanWholeRepo(t *testing.T) {
	result, err := analyzeCheckScope(t.TempDir(), nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 0 || len(result.Diagnostics) != 0 {
		t.Fatalf("expected empty changed-only result, got %#v", result)
	}
}

func TestCheckComplexitySkipsRegressionWithoutBaseline(t *testing.T) {
	dir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(dir, "package.json"), `{}`)
	mustWriteTestFile(t, filepath.Join(dir, "app.ts"), "export function value(flag: boolean) { if (flag) return 1; return 0 }\n")
	cfg := &GirlProjectConfig{
		Analysis:   GirlAnalysisConfig{Exclude: []string{"node_modules"}},
		Complexity: GirlComplexityConfig{Max: 10, Baseline: ".girl/missing.json", FailOn: "regression"},
	}
	result, err := runCheckComplexity(dir, cfg, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Status != "skip" || result.BaselineFound {
		t.Fatalf("complexity result = %#v", result)
	}
}

func TestCheckComplexityChangedOnlyScopesToChangedFiles(t *testing.T) {
	dir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(dir, "package.json"), `{}`)
	mustWriteTestFile(t, filepath.Join(dir, "changed.ts"), "export function changed(flag: boolean) { if (flag) return 1; return 0 }\n")
	mustWriteTestFile(t, filepath.Join(dir, "unchanged.ts"), `
export function unchanged(a: boolean, b: boolean, c: boolean) {
  if (a) {}
  if (b) {}
  if (c) {}
  return 0
}
`)
	cfg := &GirlProjectConfig{
		Analysis:   GirlAnalysisConfig{Exclude: []string{"node_modules"}},
		Complexity: GirlComplexityConfig{Max: 2, FailOn: "threshold"},
	}
	result, err := runCheckComplexity(dir, cfg, []string{"changed.ts"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.Files != 1 || result.OverThreshold != 0 {
		t.Fatalf("changed-only complexity result = %#v", result)
	}
}

func TestSyncManagedBlockPreservesProjectInstructions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	mustWriteTestFile(t, path, "# Project\n\nKeep this line.\n")
	if _, err := syncManagedBlock(path, girlAgentManagedBlock(), false); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "Keep this line.") || !strings.Contains(content, girlManagedStart) {
		t.Fatalf("managed block did not preserve existing instructions:\n%s", content)
	}
}

func mustWriteTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func hasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
