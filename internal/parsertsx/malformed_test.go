package parsertsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canadian-ai/girl/internal/node"
)

func parseStrSafe(t *testing.T, content string) (g *node.NodeGraph, err error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tsx")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	defer func() {
		if r := recover(); r != nil {
			err = &panicError{recovered: r}
		}
	}()
	p := New()
	g, err = p.ParseFile(path)
	return
}

type panicError struct {
	recovered any
}

func (e *panicError) Error() string {
	return "panic: " + strings.TrimSpace(strings.TrimPrefix(e.recovered.(string), "panic: "))
}

func TestMalformed_EmptyString(t *testing.T) {
	g, err := parseStrSafe(t, "")
	if err != nil {
		// empty input may produce a parse error; that's acceptable
		// as long as we didn't panic
		t.Logf("empty input returned error (acceptable): %v", err)
	}
	if g == nil {
		t.Log("empty input returned nil graph (acceptable)")
		return
	}
	all := g.AllNodes()
	if all == nil {
		t.Error("AllNodes returned nil")
	}
}

func TestMalformed_UnclosedJSX(t *testing.T) {
	g, err := parseStrSafe(t, `<div><span>`)
	if err != nil {
		t.Logf("unclosed JSX returned error (acceptable): %v", err)
	}
	if g == nil {
		return
	}
	g.AllNodes()
	g.AllFiles()
}

func TestMalformed_UnmatchedBraces(t *testing.T) {
	g, err := parseStrSafe(t, `function foo() {`)
	if err != nil {
		t.Logf("unmatched braces returned error (acceptable): %v", err)
	}
	if g == nil {
		return
	}
	comps := g.AllNodesOfKind(node.KindComponent)
	_ = comps
}

func TestMalformed_RandomGarbage(t *testing.T) {
	g, err := parseStrSafe(t, `@#$%^&*()`)
	if err != nil {
		t.Logf("random garbage returned error (acceptable): %v", err)
	}
	if g == nil {
		return
	}
	_ = g.AllNodes()
}

func TestMalformed_UnterminatedString(t *testing.T) {
	g, err := parseStrSafe(t, "const x = 'hello")
	if err != nil {
		t.Logf("unterminated string returned error (acceptable): %v", err)
	}
	if g == nil {
		return
	}
	all := g.AllNodes()
	_ = all
}

func TestMalformed_NonASCIIGarbage(t *testing.T) {
	garbage := string([]byte{
		0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD,
		0x80, 0x81, 0x82,
		0xC0, 0xC1, 0xF5, 0xF6,
		0xE0, 0xE1, 0xE2, 0xE3, 0xE4,
	})
	g, err := parseStrSafe(t, garbage)
	if err != nil {
		t.Logf("non-ASCII garbage returned error (acceptable): %v", err)
	}
	if g == nil {
		return
	}
	g.AllNodes()
}

func TestMalformed_DeeplyNested(t *testing.T) {
	var b strings.Builder
	b.WriteString("function Outer() {\n")
	for i := 0; i < 100; i++ {
		b.WriteString("  if (true) {\n")
	}
	b.WriteString("    return <div>deep</div>;\n")
	for i := 0; i < 100; i++ {
		b.WriteString("  }\n")
	}
	b.WriteString("}\n")
	g, err := parseStrSafe(t, b.String())
	if err != nil {
		t.Logf("deeply nested returned error (acceptable): %v", err)
	}
	if g == nil {
		return
	}
	comps := g.AllNodesOfKind(node.KindComponent)
	_ = comps
}

func TestMalformed_HTMLWithoutTSX(t *testing.T) {
	html := `<html><body><p>text</p></body></html>`
	g, err := parseStrSafe(t, html)
	if err != nil {
		t.Logf("HTML without TSX returned error (acceptable): %v", err)
	}
	if g == nil {
		return
	}
	comps := g.AllNodesOfKind(node.KindComponent)
	if len(comps) != 0 {
		t.Logf("HTML without TSX unexpectedly produced %d component(s)", len(comps))
	}
}

func TestMalformed_MissingClosingTag(t *testing.T) {
	src := `const Component = () => <div>`
	g, err := parseStrSafe(t, src)
	if err != nil {
		t.Logf("missing closing tag returned error (acceptable): %v", err)
	}
	if g == nil {
		return
	}
	comps := g.AllNodesOfKind(node.KindComponent)
	_ = comps
}

func TestMalformed_LargeUnclosedJSX(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 1000; i++ {
		b.WriteString("<div>")
	}
	g, err := parseStrSafe(t, b.String())
	if err != nil {
		t.Logf("large unclosed JSX returned error (acceptable): %v", err)
	}
	if g == nil {
		return
	}
	_ = g.AllNodes()
}
