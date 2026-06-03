package verification

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPackageManagers(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		wantPM   string
		wantCmds []string
	}{
		{name: "bun", files: []string{"bun.lock", "package.json"}, wantPM: "bun", wantCmds: []string{"bun run typecheck"}},
		{name: "bun legacy", files: []string{"bun.lockb", "package.json"}, wantPM: "bun", wantCmds: []string{"bun run typecheck"}},
		{name: "pnpm", files: []string{"pnpm-lock.yaml", "package.json"}, wantPM: "pnpm", wantCmds: []string{"pnpm typecheck"}},
		{name: "yarn", files: []string{"yarn.lock", "package.json"}, wantPM: "yarn", wantCmds: []string{"yarn typecheck"}},
		{name: "npm", files: []string{"package-lock.json", "package.json"}, wantPM: "npm", wantCmds: []string{"npm run typecheck"}},
		{name: "package fallback", files: []string{"package.json"}, wantPM: "npm", wantCmds: []string{"npm run typecheck"}},
		{name: "unknown", files: nil, wantPM: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, file := range tt.files {
				if file == "package.json" {
					writePackageJSON(t, dir, map[string]string{"typecheck": "tsc --noEmit"})
					continue
				}
				if err := os.WriteFile(filepath.Join(dir, file), nil, 0644); err != nil {
					t.Fatal(err)
				}
			}
			result, err := Detect(dir)
			if err != nil {
				t.Fatal(err)
			}
			if result.PackageManager != tt.wantPM {
				t.Fatalf("PackageManager = %q, want %q", result.PackageManager, tt.wantPM)
			}
			for _, want := range tt.wantCmds {
				if !hasCommand(result.Commands, want) {
					t.Fatalf("missing command %q in %#v", want, result.Commands)
				}
			}
		})
	}
}

func TestDetectPackageScriptsOnlyExisting(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir, map[string]string{"typecheck": "tsc --noEmit", "lint": "eslint ."})
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !hasCommand(result.Commands, "pnpm typecheck") || !hasCommand(result.Commands, "pnpm lint") {
		t.Fatalf("expected typecheck and lint commands, got %#v", result.Commands)
	}
	if hasCommand(result.Commands, "pnpm test") || hasCommand(result.Commands, "pnpm build") {
		t.Fatalf("missing scripts should not be emitted: %#v", result.Commands)
	}
	for _, cmd := range result.Commands {
		if cmd.Source != "package.json" || cmd.Confidence != "high" {
			t.Fatalf("script command source/confidence = %q/%q, want package.json/high", cmd.Source, cmd.Confidence)
		}
	}
}

func TestDetectGoCommands(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0644); err != nil {
		t.Fatal(err)
	}
	result, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"go build ./...", "go vet ./...", "go test ./..."} {
		if !hasCommand(result.Commands, want) {
			t.Fatalf("missing command %q in %#v", want, result.Commands)
		}
	}
	for _, cmd := range result.Commands {
		if cmd.Source != "go.mod" || cmd.Confidence != "high" {
			t.Fatalf("go command source/confidence = %q/%q, want go.mod/high", cmd.Source, cmd.Confidence)
		}
	}
}

func TestDetectMakefileAndGolangCILint(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("test:\n\tgo test ./...\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".golangci.yml"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	result, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasMakefile || !result.HasGolangCILint {
		t.Fatalf("expected Makefile and golangci-lint config flags")
	}
	if !hasCommand(result.Commands, "make test") || !hasCommand(result.Commands, "golangci-lint run") {
		t.Fatalf("expected optional commands, got %#v", result.Commands)
	}
}

func writePackageJSON(t *testing.T, dir string, scripts map[string]string) {
	t.Helper()
	data, err := json.Marshal(map[string]any{"scripts": scripts})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func hasCommand(cmds []Command, want string) bool {
	for _, cmd := range cmds {
		if cmd.Command == want {
			return true
		}
	}
	return false
}
