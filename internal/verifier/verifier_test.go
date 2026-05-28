package verifier

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPackageManager(t *testing.T) {
	tests := []struct {
		name     string
		lockfile string
		want     string
	}{
		{"pnpm", "pnpm-lock.yaml", "pnpm"},
		{"bun lockb", "bun.lockb", "bun"},
		{"bun lock", "bun.lock", "bun"},
		{"yarn", "yarn.lock", "yarn"},
		{"npm", "package-lock.json", "npm"},
		{"go", "go.mod", "go"},
		{"no lockfile", "", "unknown"},
	}
	v := &Verifier{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.lockfile != "" {
				if err := os.WriteFile(filepath.Join(dir, tt.lockfile), nil, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if got := v.detectPackageManager(dir); got != tt.want {
				t.Errorf("detectPackageManager(%q) = %q, want %q", tt.lockfile, got, tt.want)
			}
		})
	}
}

func TestRunnerSelection(t *testing.T) {
	tests := []struct {
		name     string
		lockfile string
		want     string
	}{
		{"bun", "bun.lockb", "bun run"},
		{"pnpm", "pnpm-lock.yaml", "pnpm"},
		{"yarn", "yarn.lock", "yarn"},
		{"npm", "package-lock.json", "npm run"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tt.lockfile), nil, 0644); err != nil {
				t.Fatal(err)
			}
			scripts := map[string]string{"test": "echo test"}
			pkg := map[string]any{"scripts": scripts}
			if err := writeJSON(filepath.Join(dir, "package.json"), pkg); err != nil {
				t.Fatal(err)
			}
			v := &Verifier{}
			cmds := v.detectScripts(dir)
			if len(cmds) == 0 {
				t.Fatal("expected at least one command")
			}
			for _, c := range cmds {
				if c.Source != "package.json" {
					t.Errorf("command %q source = %q, want %q", c.Name, c.Source, "package.json")
				}
				if c.Confidence != "high" {
					t.Errorf("command %q confidence = %q, want %q", c.Name, c.Confidence, "high")
				}
			}
			first := cmds[0]
			got, want := first.Command[:len(tt.want)], tt.want
			if got != want {
				t.Errorf("command prefix = %q, want %q (runner %q)", got, want, tt.want)
			}
		})
	}
}

func TestScriptDetection(t *testing.T) {
	dir := t.TempDir()
	scripts := map[string]string{
		"typecheck": "tsc --noEmit",
		"build":     "vite build",
		"test":      "vitest run",
	}
	pkg := map[string]any{"scripts": scripts}
	if err := writeJSON(filepath.Join(dir, "package.json"), pkg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	v := &Verifier{}
	cmds := v.detectScripts(dir)

	got := map[string]*CommandCheck{}
	for i, c := range cmds {
		got[c.Name] = &cmds[i]
	}

	if c, ok := got["typecheck"]; !ok {
		t.Error("expected typecheck script")
	} else if c.Command != "npm run typecheck" {
		t.Errorf("typecheck command = %q, want %q", c.Command, "npm run typecheck")
	} else if !c.Required {
		t.Error("typecheck should be required")
	}

	if c, ok := got["build"]; !ok {
		t.Error("expected build script")
	} else if !c.Required {
		t.Error("build should be required")
	}

	if c, ok := got["test"]; !ok {
		t.Error("expected test script")
	} else if c.Required {
		t.Error("test should not be required")
	}

	if _, ok := got["lint"]; ok {
		t.Error("lint should not be reported (missing from package.json)")
	}
	if _, ok := got["format"]; ok {
		t.Error("format should not be reported (missing from package.json)")
	}
}

func TestGoCommands(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	v := &Verifier{}
	cmds := v.detectGoCommands(dir, "go")

	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(cmds))
	}

	expected := []struct {
		name    string
		command string
	}{
		{"Go build", "go build ./..."},
		{"Go vet", "go vet ./..."},
		{"Go test", "go test ./..."},
	}
	for i, e := range expected {
		if cmds[i].Name != e.name {
			t.Errorf("command[%d].Name = %q, want %q", i, cmds[i].Name, e.name)
		}
		if cmds[i].Command != e.command {
			t.Errorf("command[%d].Command = %q, want %q", i, cmds[i].Command, e.command)
		}
		if cmds[i].Source != "go.mod" {
			t.Errorf("command[%d].Source = %q, want %q", i, cmds[i].Source, "go.mod")
		}
		if cmds[i].Confidence != "high" {
			t.Errorf("command[%d].Confidence = %q, want %q", i, cmds[i].Confidence, "high")
		}
		if !cmds[i].Required {
			t.Errorf("command[%d].Required = false, want true", i)
		}
	}
}

func TestGoCommandsSkippedWithoutGoMod(t *testing.T) {
	v := &Verifier{}
	cmds := v.detectGoCommands("", "")
	if len(cmds) != 0 {
		t.Errorf("expected no commands for non-go PM, got %d", len(cmds))
	}
}

func TestPackageJsonSourceAndConfidence(t *testing.T) {
	dir := t.TempDir()
	scripts := map[string]string{"test": "echo test"}
	pkg := map[string]any{"scripts": scripts}
	if err := writeJSON(filepath.Join(dir, "package.json"), pkg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	v := &Verifier{}
	cmds := v.detectScripts(dir)

	if len(cmds) == 0 {
		t.Fatal("expected commands")
	}
	for _, c := range cmds {
		if c.Source != "package.json" {
			t.Errorf("command %q source = %q, want %q", c.Name, c.Source, "package.json")
		}
		if c.Confidence != "high" {
			t.Errorf("command %q confidence = %q, want %q", c.Name, c.Confidence, "high")
		}
	}
}

func TestGoModSourceAndConfidence(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	v := &Verifier{}
	cmds := v.detectGoCommands(dir, "go")

	for _, c := range cmds {
		if c.Source != "go.mod" {
			t.Errorf("command %q source = %q, want %q", c.Name, c.Source, "go.mod")
		}
	}
}

func TestSuggestedSourceWhenUnknownPm(t *testing.T) {
	dir := t.TempDir()
	scripts := map[string]string{"test": "echo test"}
	pkg := map[string]any{"scripts": scripts}
	if err := writeJSON(filepath.Join(dir, "package.json"), pkg); err != nil {
		t.Fatal(err)
	}
	// No lockfile — PM will be "unknown"

	v := &Verifier{}
	cmds := v.detectScripts(dir)

	if len(cmds) == 0 {
		t.Fatal("expected commands")
	}
	for _, c := range cmds {
		if c.Source != "suggested" {
			t.Errorf("command %q source = %q, want %q", c.Name, c.Source, "suggested")
		}
		if c.Confidence != "low" {
			t.Errorf("command %q confidence = %q, want %q", c.Name, c.Confidence, "low")
		}
	}
}

func TestMakefileConfidenceIsHigh(t *testing.T) {
	dir := t.TempDir()
	makefile := "test:\n\techo test\n"
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatal(err)
	}

	v := &Verifier{}
	cmds := v.detectOptionalCommands(dir)

	found := false
	for _, c := range cmds {
		if c.Name == "make test" {
			found = true
			if c.Confidence != "high" {
				t.Errorf("make test confidence = %q, want %q", c.Confidence, "high")
			}
			if c.Source != "Makefile" {
				t.Errorf("make test source = %q, want %q", c.Source, "Makefile")
			}
		}
	}
	if !found {
		t.Error("expected make test command")
	}
}

func TestConfidenceFor(t *testing.T) {
	v := &Verifier{}
	tests := []struct {
		pm   string
		want string
	}{
		{"bun", "high"},
		{"pnpm", "high"},
		{"yarn", "high"},
		{"npm", "high"},
		{"go", "high"},
		{"unknown", "low"},
		{"", "low"},
	}
	for _, tt := range tests {
		if got := v.confidenceFor(tt.pm); got != tt.want {
			t.Errorf("confidenceFor(%q) = %q, want %q", tt.pm, got, tt.want)
		}
	}
}

func writeJSON(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
