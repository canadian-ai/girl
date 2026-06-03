package packer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

func TestPacker_PacksEmptyRequest(t *testing.T) {
	p := NewPacker(1000)
	pack, err := p.Pack(PackRequest{
		Goal: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if pack.Goal != "test" {
		t.Errorf("expected goal 'test', got %q", pack.Goal)
	}
	if pack.TokenBudget != 1000 {
		t.Errorf("expected budget 1000, got %d", pack.TokenBudget)
	}
	if len(pack.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(pack.Files))
	}
	if len(pack.SelectedSnippets) != 0 {
		t.Errorf("expected 0 snippets, got %d", len(pack.SelectedSnippets))
	}
	if pack.TokenEstimate != 0 {
		t.Errorf("expected 0 estimate, got %d", pack.TokenEstimate)
	}
	if pack.DiagnosticCounts == nil {
		t.Error("expected non-nil DiagnosticCounts")
	}
}

func TestPacker_CreatesSummary(t *testing.T) {
	p := NewPacker(5000)
	pack, err := p.Pack(PackRequest{
		Goal: "test",
		Files: []*ir.FileIR{
			{
				Path:     "testdata/testfile.ts",
				Language: "ts",
				Lines:    100,
				Components: []ir.ComponentIR{
					{Name: "App", StartLine: 1, EndLine: 50, Lines: 50},
					{Name: "Header", StartLine: 51, EndLine: 80, Lines: 30},
				},
				Hooks:   []ir.HookIR{{Name: "useState", Line: 5}},
				Imports: []ir.ImportIR{{Source: "react", Names: []string{"useState"}}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(pack.Summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(pack.Summaries))
	}
	s := pack.Summaries[0]
	if !strings.Contains(s.Summary, "App") || !strings.Contains(s.Summary, "Header") {
		t.Errorf("summary should contain component names: %s", s.Summary)
	}
	if !strings.Contains(s.Summary, "100 lines") {
		t.Errorf("summary should contain line count: %s", s.Summary)
	}
}

func TestPacker_DiagnosticRanges(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.ts")
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i+1))
	}
	if err := os.WriteFile(tmpFile, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewPacker(5000)

	tests := []struct {
		name               string
		diag               ir.Diagnostic
		wantStart, wantEnd int
	}{
		{
			name:      "span takes priority",
			diag:      ir.Diagnostic{Line: 5, EndLine: 12, Span: &ir.Span{StartLine: 3, EndLine: 8}},
			wantStart: 3,
			wantEnd:   8,
		},
		{
			name:      "endLine fallback",
			diag:      ir.Diagnostic{Line: 5, EndLine: 12},
			wantStart: 5,
			wantEnd:   12,
		},
		{
			name:      "line-only fallback window",
			diag:      ir.Diagnostic{Line: 5},
			wantStart: 5,
			wantEnd:   15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snippet := p.createDiagnosticSnippet(tmpFile, tmpFile, tt.diag, 5000)
			if snippet.StartLine != tt.wantStart || snippet.EndLine != tt.wantEnd {
				t.Errorf("createDiagnosticSnippet range = [%d, %d], want [%d, %d]",
					snippet.StartLine, snippet.EndLine, tt.wantStart, tt.wantEnd)
			}
			if snippet.Content == "" {
				t.Error("snippet content should not be empty")
			}
			if !strings.Contains(snippet.Content, "contentHash:") {
				t.Error("snippet should contain content hash")
			}
		})
	}
}

func TestPacker_DiagnosticRanges_Bounds(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "small.ts")
	if err := os.WriteFile(tmpFile, []byte("line 1\nline 2\nline 3"), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewPacker(5000)

	t.Run("clamps to file bounds", func(t *testing.T) {
		d := ir.Diagnostic{Line: 1, EndLine: 100}
		snippet := p.createDiagnosticSnippet(tmpFile, tmpFile, d, 5000)
		if snippet.EndLine != 3 {
			t.Errorf("expected endLine clamped to 3, got %d", snippet.EndLine)
		}
		if snippet.StartLine != 1 {
			t.Errorf("expected startLine 1, got %d", snippet.StartLine)
		}
	})

	t.Run("negative line clamped to 1", func(t *testing.T) {
		d := ir.Diagnostic{Line: -5}
		snippet := p.createDiagnosticSnippet(tmpFile, tmpFile, d, 5000)
		if snippet.StartLine != 1 {
			t.Errorf("expected startLine clamped to 1, got %d", snippet.StartLine)
		}
		if snippet.EndLine > 3 {
			t.Errorf("expected endLine clamped to file length, got %d", snippet.EndLine)
		}
	})
}

func TestPacker_TokenBudget(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "long.ts")
	var longLines []string
	for i := 0; i < 100; i++ {
		longLines = append(longLines, "line "+fmt.Sprint(i)+" "+strings.Repeat("x", 200))
	}
	if err := os.WriteFile(tmpFile, []byte(strings.Join(longLines, "\n")), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewPacker(200)
	pack, err := p.Pack(PackRequest{
		Goal: "test",
		Files: []*ir.FileIR{
			{
				Path:     tmpFile,
				Language: "ts",
				Lines:    100,
				Components: []ir.ComponentIR{
					{Name: "BigComp", StartLine: 1, EndLine: 100, Lines: 100},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if pack.TokenEstimate > pack.TokenBudget {
		t.Errorf("token estimate %d exceeds budget %d", pack.TokenEstimate, pack.TokenBudget)
	}
	if len(pack.SelectedSnippets) == 0 {
		t.Error("expected at least one snippet")
	}
	for _, sn := range pack.SelectedSnippets {
		if sn.Tokens > pack.TokenBudget {
			t.Errorf("snippet token count %d exceeds budget %d", sn.Tokens, pack.TokenBudget)
		}
	}
}

func TestPacker_TokenBudget_LargeComponent(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large.ts")
	var hugeLines []string
	for i := 0; i < 500; i++ {
		hugeLines = append(hugeLines, "line "+fmt.Sprint(i)+" "+strings.Repeat("data", 50))
	}
	if err := os.WriteFile(tmpFile, []byte(strings.Join(hugeLines, "\n")), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewPacker(50)
	snippet := p.createSnippet(tmpFile, tmpFile, ir.ComponentIR{Name: "Huge", StartLine: 1, EndLine: 500, Lines: 500}, 50)
	if snippet.Tokens > 50 {
		t.Errorf("snippet tokens %d should not exceed budget 50", snippet.Tokens)
	}
	if !strings.Contains(snippet.Content, "truncated") {
		t.Error("expected truncated content for oversized component")
	}
}

func TestPacker_RelativePaths(t *testing.T) {
	p := NewPacker(5000)
	pack, err := p.Pack(PackRequest{
		Goal: "test",
		Files: []*ir.FileIR{
			{
				Path:     "/tmp/abs/test.ts",
				Language: "ts",
				Lines:    10,
				Components: []ir.ComponentIR{
					{Name: "App", StartLine: 1, EndLine: 5, Lines: 5},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range pack.Files {
		if filepath.IsAbs(f) {
			t.Errorf("absolute path leak in Files: %s", f)
		}
	}
	for _, s := range pack.Summaries {
		if filepath.IsAbs(s.Path) {
			t.Errorf("absolute path leak in Summary: %s", s.Path)
		}
	}
	for _, sn := range pack.SelectedSnippets {
		if filepath.IsAbs(sn.File) {
			t.Errorf("absolute path leak in Snippet: %s", sn.File)
		}
	}
}

func TestPacker_DiagnosticSnippetsFallback(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "diag.ts")
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i+1))
	}
	if err := os.WriteFile(tmpFile, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		t.Fatal(err)
	}

	p := NewPacker(5000)
	pack, err := p.Pack(PackRequest{
		Goal: "test",
		Files: []*ir.FileIR{
			{
				Path:     tmpFile,
				Language: "ts",
				Lines:    30,
			},
		},
		Diagnostics: []ir.Diagnostic{
			{
				File:     tmpFile,
				Line:     10,
				EndLine:  15,
				Code:     "E001",
				Message:  "test diagnostic",
				Severity: ir.SeverityHigh,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	hasDiagnosticSnippet := false
	for _, sn := range pack.SelectedSnippets {
		if strings.Contains(sn.Content, "contentHash:") {
			hasDiagnosticSnippet = true
			if sn.StartLine != 10 || sn.EndLine != 15 {
				t.Errorf("diagnostic snippet range = [%d, %d], want [10, 15]", sn.StartLine, sn.EndLine)
			}
			break
		}
	}
	if !hasDiagnosticSnippet {
		t.Error("expected a diagnostic-based snippet when no components exist")
	}

	if pack.DiagnosticCounts["high"] != 1 {
		t.Errorf("expected 1 high diagnostic, got %v", pack.DiagnosticCounts)
	}
	if len(pack.TopCodes) != 1 || pack.TopCodes[0] != "E001" {
		t.Errorf("expected top code [E001], got %v", pack.TopCodes)
	}
}

func TestRedactPath(t *testing.T) {
	tests := []struct {
		path     string
		homeDir  string
		expected string
	}{
		{"/home/user/project/src/app.ts", "", "<redacted>/src/app.ts"},
		{"/tmp/x.ts", "", "<redacted>/tmp/x.ts"},
		{"src/app.ts", "", "src/app.ts"},
		{"relative/path/file.go", "", "relative/path/file.go"},
	}
	for _, tc := range tests {
		got := redactPath(tc.path, tc.homeDir)
		if got != tc.expected {
			t.Errorf("redactPath(%q, %q) = %q, want %q", tc.path, tc.homeDir, got, tc.expected)
		}
	}
}

func TestRedactPathHomeDirRelative(t *testing.T) {
	got := redactPath("users/ola/project/file.ts", "users/ola")
	if got != "~/project/file.ts" {
		t.Errorf("expected ~/project/file.ts, got %q", got)
	}
}

func TestSanitizePublicPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"src/app.ts", "src/app.ts"},
		{"src/private/api.ts", "src/synthetic/api.ts"},
		{"internal/config/secret.key", "internal/config/synthetic.key"},
		{"components/private/routes.ts", "components/synthetic/routes.ts"},
	}
	for _, tc := range tests {
		got := sanitizePublicPath(tc.path)
		if got != tc.expected {
			t.Errorf("sanitizePublicPath(%q) = %q, want %q", tc.path, got, tc.expected)
		}
	}
}

func TestRedactContent(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"secret = \"sk-test1234567890abcdefghij\""},
		{"Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0"},
		{"API_KEY=abcdefghijklmnopqrstuvwxyz1234"},
		{"contact user@example.com for access"},
		{"PASSWORD=supersecretvalue123"},
	}
	for _, tc := range tests {
		got := redactContent(tc.input)
		if got == tc.input {
			t.Errorf("redactContent(%q) did not redact anything", tc.input)
		}
	}
}

func TestRedactContentPreservesNormalCode(t *testing.T) {
	input := `func add(a, b int) int { return a + b }`
	got := redactContent(input)
	if got != input {
		t.Errorf("redactContent should not modify normal code: %q -> %q", input, got)
	}
}

func TestPacker_DiagnosticCountsAndTopCodes(t *testing.T) {
	p := NewPacker(5000)
	pack, err := p.Pack(PackRequest{
		Goal: "test",
		Diagnostics: []ir.Diagnostic{
			{Code: "E001", Severity: ir.SeverityHigh, Message: "err1"},
			{Code: "E001", Severity: ir.SeverityHigh, Message: "err2"},
			{Code: "W001", Severity: ir.SeverityMedium, Message: "warn1"},
			{Code: "I001", Severity: ir.SeverityLow, Message: "info1"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if pack.DiagnosticCounts["high"] != 2 {
		t.Errorf("expected 2 high, got %d", pack.DiagnosticCounts["high"])
	}
	if pack.DiagnosticCounts["medium"] != 1 {
		t.Errorf("expected 1 medium, got %d", pack.DiagnosticCounts["medium"])
	}
	if pack.DiagnosticCounts["low"] != 1 {
		t.Errorf("expected 1 low, got %d", pack.DiagnosticCounts["low"])
	}

	if len(pack.TopCodes) < 1 || pack.TopCodes[0] != "E001" {
		t.Errorf("expected E001 as top code, got %v", pack.TopCodes)
	}
}
