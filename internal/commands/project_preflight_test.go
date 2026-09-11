package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunProjectPreflightUsesSelectedProfiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"workspaces":["apps/*"]}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "next.config.ts"), []byte("export default {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &GirlProjectConfig{Profiles: []string{"next", "monorepo"}}
	result := runProjectPreflight(dir, cfg)
	if result.Profile != "monorepo,next" {
		t.Fatalf("profile = %q", result.Profile)
	}
	if !hasPreflightCheck(result.Checks, "Next.js Config") {
		t.Fatalf("missing Next.js check: %#v", result.Checks)
	}
	if !hasPreflightCheck(result.Checks, "Monorepo Workspaces") {
		t.Fatalf("missing monorepo check: %#v", result.Checks)
	}
	if hasPreflightCheck(result.Checks, "Convex Config") {
		t.Fatalf("unexpected Convex check: %#v", result.Checks)
	}
}

func TestRunProjectPreflightGenericFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	result := runProjectPreflight(dir, &GirlProjectConfig{})
	if result.Profile != "generic" {
		t.Fatalf("profile = %q", result.Profile)
	}
	if !hasPreflightCheck(result.Checks, "Package Manager") {
		t.Fatalf("missing generic package check: %#v", result.Checks)
	}
}

func hasPreflightCheck(checks []PreflightCheck, name string) bool {
	for _, check := range checks {
		if check.Name == name {
			return true
		}
	}
	return false
}
