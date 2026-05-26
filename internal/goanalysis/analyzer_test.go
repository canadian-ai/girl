package goanalysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

func TestDetectLargeFile_SetsNewFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.go")
	content := "package testdata\n\n" + strings.Repeat("// comment\n", 510) + "\nfunc f() {}\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	gf, err := ParseGoFile(path)
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	diags := detectLargeFile(gf, cfg)
	if len(diags) == 0 {
		t.Fatal("expected diagnostic for large file")
	}
	d := diags[0]

	if d.Kind != ir.NodeKindFile {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindFile)
	}
	if d.Symbol != relPath(path) {
		t.Errorf("Symbol = %q, want %q", d.Symbol, relPath(path))
	}
	if d.EndLine != gf.Lines {
		t.Errorf("EndLine = %d, want %d", d.EndLine, gf.Lines)
	}
	if d.File != relPath(path) {
		t.Errorf("File = %q, want %q", d.File, relPath(path))
	}
}

func TestDetectLongFunction_SetsNewFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "longfn.go")

	var sb strings.Builder
	sb.WriteString("package testdata\n\nfunc longFunc() {\n")
	for i := 0; i < 85; i++ {
		sb.WriteString("\t_ = 1\n")
	}
	sb.WriteString("}\n")

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}

	gf, err := ParseGoFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(gf.Functions) == 0 {
		t.Fatal("no functions parsed")
	}

	cfg := DefaultConfig()
	diags := detectLongFunction(gf, gf.Functions[0], cfg)
	if len(diags) == 0 {
		t.Fatal("expected diagnostic for long function")
	}
	d := diags[0]

	if d.Kind != ir.NodeKindFunction {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindFunction)
	}
	if d.Symbol != fnName(gf.Functions[0]) {
		t.Errorf("Symbol = %q, want %q", d.Symbol, fnName(gf.Functions[0]))
	}
	if d.EndLine != gf.Functions[0].EndLine {
		t.Errorf("EndLine = %d, want %d", d.EndLine, gf.Functions[0].EndLine)
	}
	if d.Span == nil {
		t.Fatal("Span is nil")
	}
	if d.Span.StartLine != gf.Functions[0].StartLine {
		t.Errorf("Span.StartLine = %d, want %d", d.Span.StartLine, gf.Functions[0].StartLine)
	}
	if d.Span.EndLine != gf.Functions[0].EndLine {
		t.Errorf("Span.EndLine = %d, want %d", d.Span.EndLine, gf.Functions[0].EndLine)
	}
	if d.File != relPath(path) {
		t.Errorf("File = %q, want %q", d.File, relPath(path))
	}
}

func TestDetectHighComplexity_SetsNewFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "complex.go")

	var sb strings.Builder
	sb.WriteString("package testdata\n\nfunc complexFunc() {\n")
	for i := 0; i < 15; i++ {
		sb.WriteString("\tif true { _ = 1 }\n")
	}
	sb.WriteString("}\n")

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}

	gf, err := ParseGoFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(gf.Functions) == 0 {
		t.Fatal("no functions parsed")
	}

	cfg := DefaultConfig()
	diags := detectHighComplexity(gf, gf.Functions[0], cfg)
	if len(diags) == 0 {
		t.Fatal("expected diagnostic for high complexity")
	}
	d := diags[0]

	if d.Kind != ir.NodeKindFunction {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindFunction)
	}
	if d.Symbol != fnName(gf.Functions[0]) {
		t.Errorf("Symbol = %q, want %q", d.Symbol, fnName(gf.Functions[0]))
	}
	if d.File != relPath(path) {
		t.Errorf("File = %q, want %q", d.File, relPath(path))
	}
}

func TestDetectDeepNesting_SetsNewFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nesting.go")

	content := `package testdata

func deepNest() {
	if true {
		if true {
			if true {
				if true {
					if true {
						_ = 1
					}
				}
			}
		}
	}
}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	gf, err := ParseGoFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(gf.Functions) == 0 {
		t.Fatal("no functions parsed")
	}

	cfg := DefaultConfig()
	diags := detectDeepNesting(gf, gf.Functions[0], cfg)
	if len(diags) == 0 {
		t.Fatal("expected diagnostic for deep nesting")
	}
	d := diags[0]

	if d.Kind != ir.NodeKindFunction {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindFunction)
	}
	if d.Symbol != fnName(gf.Functions[0]) {
		t.Errorf("Symbol = %q, want %q", d.Symbol, fnName(gf.Functions[0]))
	}
	if d.File != relPath(path) {
		t.Errorf("File = %q, want %q", d.File, relPath(path))
	}
}

func TestDetectTooManyParams_SetsNewFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "params.go")

	content := `package testdata

func manyParams(a int, b int, c int, d int, e int, f int) {}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	gf, err := ParseGoFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(gf.Functions) == 0 {
		t.Fatal("no functions parsed")
	}

	cfg := DefaultConfig()
	diags := detectTooManyParams(gf, gf.Functions[0], cfg)
	if len(diags) == 0 {
		t.Fatal("expected diagnostic for too many params")
	}
	d := diags[0]

	if d.Kind != ir.NodeKindFunction {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindFunction)
	}
	if d.Symbol != fnName(gf.Functions[0]) {
		t.Errorf("Symbol = %q, want %q", d.Symbol, fnName(gf.Functions[0]))
	}
	if d.File != relPath(path) {
		t.Errorf("File = %q, want %q", d.File, relPath(path))
	}
}

func TestDetectIgnoredErrors_SetsNewFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ignorederr.go")

	content := `package testdata

func ignoredErrs() {
	_ = someFunc()
	_ = someFunc()
}

func someFunc() error { return nil }
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	gf, err := ParseGoFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// The second function is someFunc, first is ignoredErrs
	if len(gf.Functions) < 1 {
		t.Fatal("no functions parsed")
	}

	fn := gf.Functions[0]
	diags := detectIgnoredErrors(gf, fn)
	if len(diags) == 0 {
		t.Fatal("expected diagnostic for ignored errors")
	}
	d := diags[0]

	if d.Kind != ir.NodeKindFunction {
		t.Errorf("Kind = %q, want %q", d.Kind, ir.NodeKindFunction)
	}
	if d.Symbol != fnName(fn) {
		t.Errorf("Symbol = %q, want %q", d.Symbol, fnName(fn))
	}
	if d.File != relPath(path) {
		t.Errorf("File = %q, want %q", d.File, relPath(path))
	}
}

func TestDetectLargeFile_UnderLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "small.go")
	content := "package testdata\n\nfunc f() {}\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	gf, err := ParseGoFile(path)
	if err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	diags := detectLargeFile(gf, cfg)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostic, got %d", len(diags))
	}
}

func TestDetectLongFunction_UnderLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shortfn.go")
	content := "package testdata\n\nfunc shortFunc() { _ = 1 }\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	gf, err := ParseGoFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(gf.Functions) == 0 {
		t.Fatal("no functions parsed")
	}

	cfg := DefaultConfig()
	diags := detectLongFunction(gf, gf.Functions[0], cfg)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostic, got %d", len(diags))
	}
}
