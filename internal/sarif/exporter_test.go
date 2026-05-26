package sarif

import (
	"encoding/json"
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

func TestExportDiagnostics_Empty(t *testing.T) {
	out, err := ExportDiagnostics(nil, "girl", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal([]byte(out), &log); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}

	if log.Schema == "" {
		t.Error("missing $schema")
	}
	if log.Version != "2.1.0" {
		t.Errorf("expected version 2.1.0, got %s", log.Version)
	}
	if len(log.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(log.Runs))
	}

	run := log.Runs[0]
	if len(run.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(run.Results))
	}
	if len(run.Tool.Driver.Rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(run.Tool.Driver.Rules))
	}
	if run.Tool.Driver.Name != "girl" {
		t.Errorf("expected tool name 'girl', got %s", run.Tool.Driver.Name)
	}
	if run.Tool.Driver.Version != "1.0.0" {
		t.Errorf("expected tool version '1.0.0', got %s", run.Tool.Driver.Version)
	}
	if run.ColumnKind != "utf16CodeUnits" {
		t.Errorf("expected columnKind 'utf16CodeUnits', got %s", run.ColumnKind)
	}
	if run.Properties.DiagnosticCount != 0 {
		t.Errorf("expected diagnosticCount 0, got %d", run.Properties.DiagnosticCount)
	}
}

func TestExportDiagnostics_Single(t *testing.T) {
	diags := []ir.Diagnostic{
		{
			Code:       "GIRL-HIGH",
			Severity:   ir.SeverityHigh,
			Message:    "Component exceeds 200 lines. This affects maintainability.",
			Suggestion: "Extract smaller sub-components.",
			File:       "src/Component.tsx",
			Line:       10,
			EndLine:    50,
			Span: &ir.Span{
				StartLine: 10,
				EndLine:   50,
				StartCol:  1,
				EndCol:    80,
			},
		},
	}

	out, err := ExportDiagnostics(diags, "girl", "2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal([]byte(out), &log); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}

	run := log.Runs[0]
	if len(run.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(run.Results))
	}
	if len(run.Tool.Driver.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(run.Tool.Driver.Rules))
	}

	r := run.Results[0]
	if r.Level != "error" {
		t.Errorf("expected level 'error', got %s", r.Level)
	}
	if r.RuleID != "GIRL-HIGH" {
		t.Errorf("expected ruleId 'GIRL-HIGH', got %s", r.RuleID)
	}
	if r.RuleIndex != 0 {
		t.Errorf("expected ruleIndex 0, got %d", r.RuleIndex)
	}

	if len(r.Locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(r.Locations))
	}

	loc := r.Locations[0]
	if loc.PhysicalLocation.ArtifactLocation.URI != "src/Component.tsx" {
		t.Errorf("expected URI 'src/Component.tsx', got %s", loc.PhysicalLocation.ArtifactLocation.URI)
	}
	if loc.PhysicalLocation.ArtifactLocation.URIBaseID != "%SRCROOT%" {
		t.Errorf("expected uriBaseId '%%SRCROOT%%', got %s", loc.PhysicalLocation.ArtifactLocation.URIBaseID)
	}

	reg := loc.PhysicalLocation.Region
	if reg.StartLine != 10 {
		t.Errorf("expected startLine 10, got %d", reg.StartLine)
	}
	if reg.EndLine != 50 {
		t.Errorf("expected endLine 50, got %d", reg.EndLine)
	}
	if reg.StartColumn != 1 {
		t.Errorf("expected startColumn 1, got %d", reg.StartColumn)
	}
	if reg.EndColumn != 80 {
		t.Errorf("expected endColumn 80, got %d", reg.EndColumn)
	}

	rule := run.Tool.Driver.Rules[0]
	if rule.ID != "GIRL-HIGH" {
		t.Errorf("expected rule id 'GIRL-HIGH', got %s", rule.ID)
	}
	if rule.DefaultConfiguration.Level != "error" {
		t.Errorf("expected rule level 'error', got %s", rule.DefaultConfiguration.Level)
	}
	if rule.Properties.Severity != "high" {
		t.Errorf("expected severity 'high', got %s", rule.Properties.Severity)
	}
	if rule.ShortDescription.Text != "Component exceeds 200 lines." {
		t.Errorf("shortDescription mismatch: %s", rule.ShortDescription.Text)
	}
	if rule.FullDescription.Text != "Component exceeds 200 lines. This affects maintainability.\n\nSuggestion: Extract smaller sub-components." {
		t.Errorf("fullDescription mismatch: %s", rule.FullDescription.Text)
	}

	if run.Properties.DiagnosticCount != 1 {
		t.Errorf("expected diagnosticCount 1, got %d", run.Properties.DiagnosticCount)
	}
	if run.Properties.ErrorCount != 1 {
		t.Errorf("expected errorCount 1, got %d", run.Properties.ErrorCount)
	}
	if run.Properties.WarningCount != 0 {
		t.Errorf("expected warningCount 0, got %d", run.Properties.WarningCount)
	}
	if run.Properties.NoteCount != 0 {
		t.Errorf("expected noteCount 0, got %d", run.Properties.NoteCount)
	}
}

func TestExportDiagnostics_Multiple(t *testing.T) {
	diags := []ir.Diagnostic{
		{
			Code:     "GIRL-HIGH",
			Severity: ir.SeverityHigh,
			Message:  "High severity issue.",
			File:     "a.ts",
			Line:     1,
		},
		{
			Code:     "GIRL-MED",
			Severity: ir.SeverityMedium,
			Message:  "Medium severity issue.",
			File:     "b.ts",
			Line:     2,
		},
		{
			Code:     "GIRL-LOW",
			Severity: ir.SeverityLow,
			Message:  "Low severity issue.",
			File:     "c.ts",
			Line:     3,
		},
	}

	out, err := ExportDiagnostics(diags, "girl", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal([]byte(out), &log); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}

	run := log.Runs[0]
	if len(run.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(run.Results))
	}
	if len(run.Tool.Driver.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(run.Tool.Driver.Rules))
	}

	if run.Properties.DiagnosticCount != 3 {
		t.Errorf("expected diagnosticCount 3, got %d", run.Properties.DiagnosticCount)
	}
	if run.Properties.ErrorCount != 1 {
		t.Errorf("expected errorCount 1, got %d", run.Properties.ErrorCount)
	}
	if run.Properties.WarningCount != 1 {
		t.Errorf("expected warningCount 1, got %d", run.Properties.WarningCount)
	}
	if run.Properties.NoteCount != 1 {
		t.Errorf("expected noteCount 1, got %d", run.Properties.NoteCount)
	}

	levels := make(map[string]bool)
	for _, r := range run.Results {
		levels[r.Level] = true
	}
	if !levels["error"] {
		t.Error("missing 'error' level result")
	}
	if !levels["warning"] {
		t.Error("missing 'warning' level result")
	}
	if !levels["note"] {
		t.Error("missing 'note' level result")
	}
}

func TestExportDiagnostics_DeduplicatesRules(t *testing.T) {
	diags := []ir.Diagnostic{
		{
			Code:     "GIRL-SAME",
			Severity: ir.SeverityHigh,
			Message:  "First occurrence.",
			File:     "a.ts",
			Line:     1,
		},
		{
			Code:     "GIRL-SAME",
			Severity: ir.SeverityMedium,
			Message:  "Second occurrence.",
			File:     "b.ts",
			Line:     2,
		},
	}

	out, err := ExportDiagnostics(diags, "girl", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal([]byte(out), &log); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}

	run := log.Runs[0]
	if len(run.Results) != 2 {
		t.Errorf("expected 2 results, got %d", len(run.Results))
	}
	if len(run.Tool.Driver.Rules) != 1 {
		t.Fatalf("expected 1 rule (deduped), got %d", len(run.Tool.Driver.Rules))
	}

	for _, r := range run.Results {
		if r.RuleIndex != 0 {
			t.Errorf("expected ruleIndex 0 for deduped rule, got %d", r.RuleIndex)
		}
	}
}

func TestExportDiagnostics_LevelMapping(t *testing.T) {
	tests := []struct {
		severity  ir.Severity
		wantLevel string
	}{
		{ir.SeverityHigh, "error"},
		{ir.SeverityMedium, "warning"},
		{ir.SeverityLow, "note"},
	}

	for _, tt := range tests {
		out, err := ExportDiagnostics([]ir.Diagnostic{
			{Code: "TEST", Severity: tt.severity, Message: "test", File: "f.ts", Line: 1},
		}, "girl", "1.0.0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var log sarifLog
		if err := json.Unmarshal([]byte(out), &log); err != nil {
			t.Fatalf("invalid SARIF JSON: %v", err)
		}

		run := log.Runs[0]
		if len(run.Results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(run.Results))
		}
		if run.Results[0].Level != tt.wantLevel {
			t.Errorf("severity %s: expected level %s, got %s", tt.severity, tt.wantLevel, run.Results[0].Level)
		}
	}
}

func TestExportDiagnostics_RoundTrip(t *testing.T) {
	diags := []ir.Diagnostic{
		{
			Code:       "GIRL-BIG",
			Severity:   ir.SeverityHigh,
			Message:    "Component too large. Refactor into smaller pieces.",
			Suggestion: "Break into sub-components.",
			File:       "src/Foo.tsx",
			Line:       1,
			EndLine:    200,
			Span: &ir.Span{
				StartLine: 1,
				EndLine:   200,
				StartCol:  1,
				EndCol:    100,
			},
			Symbol:    "Foo",
			Component: "Foo",
		},
		{
			Code:       "GIRL-SMALL",
			Severity:   ir.SeverityLow,
			Message:    "Small component.",
			File:       "src/Bar.tsx",
			Line:       5,
			Suggestion: "",
		},
	}

	out, err := ExportDiagnostics(diags, "girl", "1.5.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal([]byte(out), &log); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}

	run := log.Runs[0]

	if run.Tool.Driver.Name != "girl" {
		t.Errorf("expected tool name 'girl', got %s", run.Tool.Driver.Name)
	}
	if run.Tool.Driver.Version != "1.5.0" {
		t.Errorf("expected tool version '1.5.0', got %s", run.Tool.Driver.Version)
	}
	if run.Tool.Driver.InformationURI != "https://github.com/canadian-ai/girl" {
		t.Errorf("expected informationUri 'https://github.com/canadian-ai/girl', got %s", run.Tool.Driver.InformationURI)
	}

	if len(run.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(run.Results))
	}
	if len(run.Tool.Driver.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(run.Tool.Driver.Rules))
	}

	if run.Properties.DiagnosticCount != 2 {
		t.Errorf("expected diagnosticCount 2, got %d", run.Properties.DiagnosticCount)
	}
	if run.Properties.ErrorCount != 1 {
		t.Errorf("expected errorCount 1, got %d", run.Properties.ErrorCount)
	}
	if run.Properties.NoteCount != 1 {
		t.Errorf("expected noteCount 1, got %d", run.Properties.NoteCount)
	}

	if run.ColumnKind != "utf16CodeUnits" {
		t.Errorf("expected columnKind 'utf16CodeUnits', got %s", run.ColumnKind)
	}

	result0 := run.Results[0]
	if result0.RuleID != "GIRL-BIG" {
		t.Errorf("expected ruleId 'GIRL-BIG', got %s", result0.RuleID)
	}
	if result0.Level != "error" {
		t.Errorf("expected level 'error', got %s", result0.Level)
	}
	reg0 := result0.Locations[0].PhysicalLocation.Region
	if reg0.StartLine != 1 || reg0.EndLine != 200 || reg0.StartColumn != 1 || reg0.EndColumn != 100 {
		t.Errorf("span mismatch: got (%d,%d,%d,%d)", reg0.StartLine, reg0.EndLine, reg0.StartColumn, reg0.EndColumn)
	}

	result1 := run.Results[1]
	if result1.RuleID != "GIRL-SMALL" {
		t.Errorf("expected ruleId 'GIRL-SMALL', got %s", result1.RuleID)
	}
	if result1.Level != "note" {
		t.Errorf("expected level 'note', got %s", result1.Level)
	}
	reg1 := result1.Locations[0].PhysicalLocation.Region
	if reg1.StartLine != 5 || reg1.EndLine != 5 {
		t.Errorf("line-only span mismatch: got (%d,%d)", reg1.StartLine, reg1.EndLine)
	}
	if reg1.StartColumn != 0 || reg1.EndColumn != 0 {
		t.Errorf("expected no columns for line-only diagnostic, got (%d,%d)", reg1.StartColumn, reg1.EndColumn)
	}

	rule0 := run.Tool.Driver.Rules[0]
	if rule0.Name != "Foo" {
		t.Errorf("expected rule name 'Foo' (from Symbol), got '%s'", rule0.Name)
	}
}

func TestExportDiagnostics_EndLineWithoutSpan(t *testing.T) {
	diags := []ir.Diagnostic{
		{
			Code:     "GIRL-END",
			Severity: ir.SeverityMedium,
			Message:  "Issue with range.",
			File:     "test.ts",
			Line:     5,
			EndLine:  10,
		},
	}

	out, err := ExportDiagnostics(diags, "girl", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal([]byte(out), &log); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}

	run := log.Runs[0]
	reg := run.Results[0].Locations[0].PhysicalLocation.Region
	if reg.StartLine != 5 {
		t.Errorf("expected startLine 5, got %d", reg.StartLine)
	}
	if reg.EndLine != 10 {
		t.Errorf("expected endLine 10, got %d", reg.EndLine)
	}
	if reg.StartColumn != 0 {
		t.Errorf("expected no startColumn, got %d", reg.StartColumn)
	}
}

func TestExportDiagnostics_FirstSentence(t *testing.T) {
	diags := []ir.Diagnostic{
		{
			Code:     "GIRL-DOT",
			Severity: ir.SeverityLow,
			Message:  "First sentence. Second sentence. Third.",
			File:     "f.ts",
			Line:     1,
		},
	}

	out, err := ExportDiagnostics(diags, "girl", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var log sarifLog
	if err := json.Unmarshal([]byte(out), &log); err != nil {
		t.Fatalf("invalid SARIF JSON: %v", err)
	}

	rule := log.Runs[0].Tool.Driver.Rules[0]
	if rule.ShortDescription.Text != "First sentence." {
		t.Errorf("expected shortDescription 'First sentence.', got '%s'", rule.ShortDescription.Text)
	}
}
