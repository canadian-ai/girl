package goanalysis

import (
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// The batched importer must give the same type information importer.Default
// does: a discarded os.Remove error is recognised with high confidence.
func TestWalkGoFiles_BatchedStdImporterMatchesDefault(t *testing.T) {
	dir := t.TempDir()
	src := "package x\n\nimport \"os\"\n\nfunc Clean() {\n\t_ = os.Remove(\"a\")\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	walked, err := walkGoFiles(dir)
	if err != nil || len(walked) != 1 {
		t.Fatalf("walk: %v (%d files)", err, len(walked))
	}
	single, err := ParseGoFile(filepath.Join(dir, "x.go"))
	if err != nil {
		t.Fatal(err)
	}
	got, want := walked[0].Functions[0], single.Functions[0]
	if got.IgnoredErrs != 1 || got.IgnoredErrConfidence != "high" {
		t.Fatalf("batched importer: %d ignored, confidence %q; want 1, high", got.IgnoredErrs, got.IgnoredErrConfidence)
	}
	if got.IgnoredErrs != want.IgnoredErrs || got.IgnoredErrConfidence != want.IgnoredErrConfidence {
		t.Fatalf("batched importer disagrees with importer.Default: %+v vs %+v", got, want)
	}
}

func TestHasBlankCallAssign(t *testing.T) {
	cases := map[string]bool{
		"package x\nfunc f() { _ = g() }\nfunc g() error { return nil }\n":              true,
		"package x\nfunc f() { _, _ = g() }\nfunc g() (int, error) { return 0, nil }\n": true,
		"package x\nfunc f() { x := 1; _ = x }\n":                                       false,
		"package x\nvar _ = g()\nfunc g() int { return 0 }\n":                           false,
		"package x\nfunc f() { g() }\nfunc g() {}\n":                                    false,
	}
	for src, want := range cases {
		f, err := parser.ParseFile(token.NewFileSet(), "x.go", src, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got := hasBlankCallAssign(f); got != want {
			t.Errorf("hasBlankCallAssign(%q) = %v, want %v", src, got, want)
		}
	}
}

func TestStdExports_VendorPath(t *testing.T) {
	s := &stdExports{goroot: filepath.Clean(build.Default.GOROOT)}
	src := filepath.Join(s.goroot, "src")
	cases := []struct{ dir, want string }{
		{filepath.Join(src, "net", "http"), "vendor/golang.org/x/net/idna"},
		{filepath.Join(src, "cmd", "go", "internal", "modcmd"), "cmd/vendor/golang.org/x/net/idna"},
		{t.TempDir(), ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := s.vendorPath("golang.org/x/net/idna", c.dir); got != c.want {
			t.Errorf("vendorPath(%q) = %q, want %q", c.dir, got, c.want)
		}
	}
}
