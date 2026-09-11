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
}

func TestGirlProjectConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &GirlProjectConfig{
		Version:    "1",
		Profiles:   []string{"cai", "next"},
		Workspaces: []string{"."},
		Analysis:   GirlAnalysisConfig{ChangedOnly: true, Exclude: []string{"node_modules", ".next"}},
		Complexity: GirlComplexityConfig{Max: 10, Baseline: ".girl/complexity-baseline.json", FailOn: "regression"},
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
	if got := loaded.Verify["build"]; len(got) != 1 || got[0] != "bun run build:ci" {
		t.Fatalf("build commands = %v", got)
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
