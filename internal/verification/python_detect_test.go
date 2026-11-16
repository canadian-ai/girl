package verification

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPythonCommandsUsesDeclaredToolsOnly(t *testing.T) {
	dir := t.TempDir()
	if r, _ := Detect(dir); len(r.Commands) != 0 {
		t.Fatalf("empty dir must not invent commands: %+v", r.Commands)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("pyproject.toml", "[project]\nname='x'\n\n[tool.pytest.ini_options]\n\n[tool.ruff]\n")
	r, err := Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	byType := map[string]Command{}
	for _, c := range r.Commands {
		byType[c.Type] = c
	}
	if byType["build"].Source != "pyproject.toml" || byType["test"].Command != "python3 -m pytest" || byType["lint"].Command != "ruff check ." {
		t.Fatalf("commands = %+v", r.Commands)
	}
	if _, ok := byType["typecheck"]; ok {
		t.Fatal("mypy is not declared and must not be proposed")
	}
}
