package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDisplayTarget(t *testing.T) {
	if got := DisplayTarget("/home/someone/src/project"); got != "project" {
		t.Errorf("absolute target leaked: %q", got)
	}
	if got := DisplayTarget("./examples/app/"); got != "examples/app" {
		t.Errorf("relative target = %q", got)
	}
}

func TestDisplayFile(t *testing.T) {
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outside, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	escaping, err := filepath.Rel(cwd, filepath.Join(outside, "src", "a.go"))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct{ target, file, want string }{
		{".", "internal/x.go", "internal/x.go"},
		{outside, escaping, "src/a.go"},
		{outside, filepath.Join(outside, "src", "a.go"), "src/a.go"},
	}
	for _, c := range cases {
		if got := DisplayFile(c.target, c.file); got != c.want {
			t.Errorf("DisplayFile(%q, %q) = %q, want %q", c.target, c.file, got, c.want)
		}
	}
}
