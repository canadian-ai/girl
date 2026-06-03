package verifier

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyPackageManagersAndScripts(t *testing.T) {
	tests := []struct {
		name    string
		lock    string
		wantPM  string
		wantCmd string
	}{
		{"pnpm", "pnpm-lock.yaml", "pnpm", "pnpm typecheck"},
		{"bun", "bun.lockb", "bun", "bun run typecheck"},
		{"yarn", "yarn.lock", "yarn", "yarn typecheck"},
		{"npm", "package-lock.json", "npm", "npm run typecheck"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeJSON(t, filepath.Join(dir, "package.json"), map[string]any{
				"scripts": map[string]string{"typecheck": "tsc --noEmit"},
			})
			if err := os.WriteFile(filepath.Join(dir, tt.lock), nil, 0644); err != nil {
				t.Fatal(err)
			}

			result, err := NewVerifier().Verify(dir)
			if err != nil {
				t.Fatal(err)
			}
			if result.PackageManager != tt.wantPM {
				t.Fatalf("PackageManager = %q, want %q", result.PackageManager, tt.wantPM)
			}
			cmd := findCommand(result.Commands, tt.wantCmd)
			if cmd == nil {
				t.Fatalf("missing command %q in %#v", tt.wantCmd, result.Commands)
			}
			if cmd.Source != "package.json" || cmd.Confidence != "high" {
				t.Fatalf("source/confidence = %q/%q, want package.json/high", cmd.Source, cmd.Confidence)
			}
		})
	}
}

func TestVerifyMissingScriptsNotReported(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "package.json"), map[string]any{
		"scripts": map[string]string{"typecheck": "tsc --noEmit", "build": "vite build"},
	})
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := NewVerifier().Verify(dir)
	if err != nil {
		t.Fatal(err)
	}
	if findCommand(result.Commands, "pnpm typecheck") == nil || findCommand(result.Commands, "pnpm build") == nil {
		t.Fatalf("expected typecheck and build commands, got %#v", result.Commands)
	}
	if findCommand(result.Commands, "pnpm lint") != nil || findCommand(result.Commands, "pnpm test") != nil {
		t.Fatalf("missing scripts should not be reported: %#v", result.Commands)
	}
}

func TestVerifyGoCommands(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := NewVerifier().Verify(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"go build ./...", "go vet ./...", "go test ./..."} {
		cmd := findCommand(result.Commands, want)
		if cmd == nil {
			t.Fatalf("missing command %q in %#v", want, result.Commands)
		}
		if cmd.Source != "go.mod" || cmd.Confidence != "high" {
			t.Fatalf("go command source/confidence = %q/%q, want go.mod/high", cmd.Source, cmd.Confidence)
		}
	}
}

func TestVerifyPackageJSONFallbackToNPM(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "package.json"), map[string]any{
		"scripts": map[string]string{"typecheck": "tsc --noEmit"},
	})

	result, err := NewVerifier().Verify(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.PackageManager != "npm" {
		t.Fatalf("PackageManager = %q, want npm", result.PackageManager)
	}
	if findCommand(result.Commands, "npm run typecheck") == nil {
		t.Fatalf("missing npm fallback command in %#v", result.Commands)
	}
}

func TestVerifyMakefileConfidenceIsHigh(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("test:\n\tgo test ./...\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := NewVerifier().Verify(dir)
	if err != nil {
		t.Fatal(err)
	}
	cmd := findCommand(result.Commands, "make test")
	if cmd == nil {
		t.Fatalf("missing make test command in %#v", result.Commands)
	}
	if cmd.Source != "Makefile" || cmd.Confidence != "high" {
		t.Fatalf("make test source/confidence = %q/%q, want Makefile/high", cmd.Source, cmd.Confidence)
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func findCommand(cmds []CommandCheck, command string) *CommandCheck {
	for i := range cmds {
		if cmds[i].Command == command {
			return &cmds[i]
		}
	}
	return nil
}
