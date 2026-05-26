package ir

import "testing"

func TestDiagnosticTarget_Symbol(t *testing.T) {
	d := Diagnostic{
		Code:      "TEST001",
		Severity:  SeverityHigh,
		Message:   "symbol test",
		File:      "test.ts",
		Line:      10,
		Component: "OldComponent",
		Suggestion: "rename it",
		Symbol:    "newSymbol",
	}
	if got := d.DiagnosticTarget(); got != "newSymbol" {
		t.Errorf("DiagnosticTarget() = %q, want %q", got, "newSymbol")
	}
}

func TestDiagnosticTarget_Component(t *testing.T) {
	d := Diagnostic{
		Code:      "TEST002",
		Severity:  SeverityMedium,
		Message:   "component test",
		File:      "test.ts",
		Line:      20,
		Component: "MyComponent",
	}
	if got := d.DiagnosticTarget(); got != "MyComponent" {
		t.Errorf("DiagnosticTarget() = %q, want %q", got, "MyComponent")
	}
}

func TestDiagnosticTarget_File(t *testing.T) {
	d := Diagnostic{
		Code:     "TEST003",
		Severity: SeverityLow,
		Message:  "file test",
		File:     "fallback.ts",
		Line:     30,
	}
	if got := d.DiagnosticTarget(); got != "fallback.ts" {
		t.Errorf("DiagnosticTarget() = %q, want %q", got, "fallback.ts")
	}
}

func TestSpan_Basic(t *testing.T) {
	s := Span{StartLine: 1, EndLine: 10, StartCol: 2, EndCol: 15}
	if s.StartLine != 1 {
		t.Errorf("StartLine = %d, want 1", s.StartLine)
	}
	if s.EndLine != 10 {
		t.Errorf("EndLine = %d, want 10", s.EndLine)
	}
	if s.StartCol != 2 {
		t.Errorf("StartCol = %d, want 2", s.StartCol)
	}
	if s.EndCol != 15 {
		t.Errorf("EndCol = %d, want 15", s.EndCol)
	}
}

func TestRelatedInfo_Basic(t *testing.T) {
	r := RelatedInfo{
		Message: "see here",
		Span:    Span{StartLine: 5, EndLine: 8},
	}
	if r.Message != "see here" {
		t.Errorf("Message = %q, want %q", r.Message, "see here")
	}
	if r.Span.StartLine != 5 {
		t.Errorf("Span.StartLine = %d, want 5", r.Span.StartLine)
	}
	if r.Span.EndLine != 8 {
		t.Errorf("Span.EndLine = %d, want 8", r.Span.EndLine)
	}
}

func TestFix_Kinds(t *testing.T) {
	fixes := []Fix{
		{Title: "Rename", Kind: "rename", Span: Span{StartLine: 1, EndLine: 1}, Text: "newName"},
		{Title: "Delete", Kind: "delete", Span: Span{StartLine: 10, EndLine: 20}},
		{Title: "Move", Kind: "move", Span: Span{StartLine: 5, EndLine: 5}, Text: "targetFile"},
	}
	if len(fixes) != 3 {
		t.Fatalf("expected 3 fixes, got %d", len(fixes))
	}
	for _, f := range fixes {
		if f.Title == "" {
			t.Errorf("Fix with kind %q has empty Title", f.Kind)
		}
		if f.Kind == "" {
			t.Errorf("Fix has empty Kind")
		}
	}
	if fixes[0].Kind != "rename" || fixes[0].Text != "newName" {
		t.Errorf("rename fix: kind=%q text=%q", fixes[0].Kind, fixes[0].Text)
	}
	if fixes[1].Kind != "delete" || fixes[1].Text != "" {
		t.Errorf("delete fix: kind=%q text=%q", fixes[1].Kind, fixes[1].Text)
	}
	if fixes[2].Kind != "move" || fixes[2].Text != "targetFile" {
		t.Errorf("move fix: kind=%q text=%q", fixes[2].Kind, fixes[2].Text)
	}
}
