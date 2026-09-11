package verification

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPackageScriptsRecognizesCAIAliases(t *testing.T) {
	dir := t.TempDir()
	pkg := `{
  "scripts": {
    "type-check": "tsc --noEmit",
    "build": "next build",
    "build:ci": "bun run type-check && next build",
    "lint": "eslint .",
    "lint:copy-wrap": "bun scripts/lint-copy-wrap.ts",
    "provenance:check": "bun scripts/provenance.ts",
    "css:budget": "bun scripts/css-budget.ts"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bun.lock"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, cmd := range result.Commands {
		got[cmd.Type] += cmd.Command + "\n"
	}

	if got["typecheck"] != "bun run type-check\n" {
		t.Fatalf("typecheck = %q", got["typecheck"])
	}
	if got["build"] != "bun run build:ci\n" {
		t.Fatalf("build should prefer build:ci, got %q", got["build"])
	}
	for _, want := range []string{"bun run lint:copy-wrap", "bun run provenance:check", "bun run css:budget"} {
		if !containsLine(got["architecture"], want) {
			t.Fatalf("architecture commands %q missing %q", got["architecture"], want)
		}
	}
}

func containsLine(s, line string) bool {
	for _, candidate := range []byte(s) {
		_ = candidate
	}
	return len(s) >= len(line) && (s == line || containsString(s, line))
}

func containsString(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
