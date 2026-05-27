package grp

import (
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

func TestFromIRPlan(t *testing.T) {
	irPlan := &ir.GrpPlan{
		PlanID: "grp_test_001",
		Goal:   "Refactor long functions",
		Risk:   ir.SeverityMedium,
		Target: ".",
		Diagnostics: []ir.Diagnostic{
			{
				Code:     "go.high-complexity",
				Severity: ir.SeverityHigh,
				Message:  "Function foo has complexity 22",
				File:     "main.go",
				Line:     10,
				EndLine:  12,
				Symbol:   "foo",
			},
			{
				Code:     "go.long-function",
				Severity: ir.SeverityLow,
				Message:  "Function bar is too long",
				File:     "util.go",
				Line:     20,
				EndLine:  30,
			},
		},
		Steps: []ir.GrpStep{
			{
				ID:     "step_001_go.high-complexity_foo",
				Recipe: "go.simplify-branches",
				Action: "Simplify branching in foo",
				File:   "main.go",
				Risk:   ir.SeverityMedium,
				Verify: []string{"go test ./..."},
			},
		},
		Verification: []string{"go test ./...", "go vet ./..."},
	}

	p := FromIRPlan(irPlan)
	if p == nil {
		t.Fatal("FromIRPlan returned nil")
	}

	if p.SpecVersion != "0.1" {
		t.Errorf("SpecVersion = %q, want %q", p.SpecVersion, "0.1")
	}
	if p.ID != "grp_test_001" {
		t.Errorf("ID = %q, want %q", p.ID, "grp_test_001")
	}
	if p.Type != "dev.refactor.plan" {
		t.Errorf("Type = %q, want %q", p.Type, "dev.refactor.plan")
	}
	if p.Source != "github.com/canadian-ai/girl" {
		t.Errorf("Source = %q, want %q", p.Source, "github.com/canadian-ai/girl")
	}
	if p.Subject != "." {
		t.Errorf("Subject = %q, want %q", p.Subject, ".")
	}
	if p.Language != "auto" {
		t.Errorf("Language = %q, want %q", p.Language, "auto")
	}
	if p.Goal != "Refactor long functions" {
		t.Errorf("Goal = %q, want %q", p.Goal, "Refactor long functions")
	}
	if p.Risk != SeverityMedium {
		t.Errorf("Risk = %q, want %q", p.Risk, SeverityMedium)
	}

	if len(p.Diagnostics) != 2 {
		t.Fatalf("len(Diagnostics) = %d, want 2", len(p.Diagnostics))
	}

	if p.Diagnostics[0].ID != "diag_0" {
		t.Errorf("Diagnostics[0].ID = %q, want %q", p.Diagnostics[0].ID, "diag_0")
	}
	if p.Diagnostics[0].Code != "go.high-complexity" {
		t.Errorf("Diagnostics[0].Code = %q, want %q", p.Diagnostics[0].Code, "go.high-complexity")
	}
	if p.Diagnostics[0].Severity != SeverityHigh {
		t.Errorf("Diagnostics[0].Severity = %q, want %q", p.Diagnostics[0].Severity, SeverityHigh)
	}
	if p.Diagnostics[0].Message != "Function foo has complexity 22" {
		t.Errorf("Diagnostics[0].Message = %q, want %q", p.Diagnostics[0].Message, "Function foo has complexity 22")
	}
	if p.Diagnostics[0].File != "main.go" {
		t.Errorf("Diagnostics[0].File = %q, want %q", p.Diagnostics[0].File, "main.go")
	}
	if p.Diagnostics[0].Symbol == nil {
		t.Fatal("Diagnostics[0].Symbol is nil")
	}
	if p.Diagnostics[0].Symbol.Name != "foo" {
		t.Errorf("Diagnostics[0].Symbol.Name = %q, want %q", p.Diagnostics[0].Symbol.Name, "foo")
	}
	if p.Diagnostics[0].Symbol.Kind != "" {
		t.Errorf("Diagnostics[0].Symbol.Kind = %q, want %q", p.Diagnostics[0].Symbol.Kind, "")
	}

	if p.Diagnostics[1].ID != "diag_1" {
		t.Errorf("Diagnostics[1].ID = %q, want %q", p.Diagnostics[1].ID, "diag_1")
	}
	if p.Diagnostics[1].Code != "go.long-function" {
		t.Errorf("Diagnostics[1].Code = %q, want %q", p.Diagnostics[1].Code, "go.long-function")
	}
	if p.Diagnostics[1].Symbol != nil {
		t.Errorf("Diagnostics[1].Symbol should be nil, got %v", p.Diagnostics[1].Symbol)
	}

	if len(p.Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(p.Steps))
	}
	if p.Steps[0].ID != "step_001_go.high-complexity_foo" {
		t.Errorf("Steps[0].ID = %q, want %q", p.Steps[0].ID, "step_001_go.high-complexity_foo")
	}
	if p.Steps[0].Recipe != "go.simplify-branches" {
		t.Errorf("Steps[0].Recipe = %q, want %q", p.Steps[0].Recipe, "go.simplify-branches")
	}
	if p.Steps[0].Action != "Simplify branching in foo" {
		t.Errorf("Steps[0].Action = %q, want %q", p.Steps[0].Action, "Simplify branching in foo")
	}
	if p.Steps[0].Target.File != "main.go" {
		t.Errorf("Steps[0].Target.File = %q, want %q", p.Steps[0].Target.File, "main.go")
	}
	if p.Steps[0].Risk != SeverityMedium {
		t.Errorf("Steps[0].Risk = %q, want %q", p.Steps[0].Risk, SeverityMedium)
	}
	if len(p.Steps[0].Verify) != 1 {
		t.Fatalf("len(Steps[0].Verify) = %d, want 1", len(p.Steps[0].Verify))
	}
	if p.Steps[0].Verify[0].Command != "go test ./..." {
		t.Errorf("Steps[0].Verify[0].Command = %q, want %q", p.Steps[0].Verify[0].Command, "go test ./...")
	}
	if !p.Steps[0].Verify[0].Required {
		t.Errorf("Steps[0].Verify[0].Required = false, want true")
	}

	if len(p.Verification) != 2 {
		t.Fatalf("len(Verification) = %d, want 2", len(p.Verification))
	}
	if p.Verification[0].Command != "go test ./..." {
		t.Errorf("Verification[0].Command = %q, want %q", p.Verification[0].Command, "go test ./...")
	}
	if !p.Verification[0].Required {
		t.Errorf("Verification[0].Required = false, want true")
	}
	if p.Verification[0].Source != "binding-default" {
		t.Errorf("Verification[0].Source = %q, want %q", p.Verification[0].Source, "binding-default")
	}
	if p.Verification[0].Confidence != "medium" {
		t.Errorf("Verification[0].Confidence = %q, want %q", p.Verification[0].Confidence, "medium")
	}
	if p.Verification[1].Command != "go vet ./..." {
		t.Errorf("Verification[1].Command = %q, want %q", p.Verification[1].Command, "go vet ./...")
	}
}

func TestDiagnosticConversion(t *testing.T) {
	irDiag := ir.Diagnostic{
		Code:       "go.high-complexity",
		Severity:   ir.SeverityHigh,
		Message:    "Function handleRequest has complexity 22",
		File:       "internal/server/handler.go",
		Line:       42,
		EndLine:    89,
		Symbol:     "handleRequest",
		Component:  "Handler",
		Kind:       ir.NodeKindFunction,
		Suggestion: "extract function",
		Package:    "server",
		Metadata: map[string]string{
			"complexity": "22",
			"threshold":  "10",
		},
		Tags: []string{"complexity", "refactor"},
		Span: &ir.Span{
			StartLine: 42,
			EndLine:   89,
			StartCol:  1,
			EndCol:    50,
		},
		Related: []ir.RelatedInfo{
			{Message: "related issue", Span: ir.Span{StartLine: 5, EndLine: 8}},
		},
		Fixes: []ir.Fix{
			{Title: "Extract function", Kind: "extract", Span: ir.Span{StartLine: 42, EndLine: 89}, Text: "extracted"},
		},
	}

	g := convertDiagnostic(irDiag, 0)

	if g.Code != "go.high-complexity" {
		t.Errorf("Code = %q, want %q", g.Code, "go.high-complexity")
	}
	if g.Severity != SeverityHigh {
		t.Errorf("Severity = %q, want %q", g.Severity, SeverityHigh)
	}
	if g.Confidence != ConfidenceHigh {
		t.Errorf("Confidence = %q, want %q", g.Confidence, ConfidenceHigh)
	}
	if g.Message != "Function handleRequest has complexity 22" {
		t.Errorf("Message = %q, want %q", g.Message, "Function handleRequest has complexity 22")
	}
	if g.File != "internal/server/handler.go" {
		t.Errorf("File = %q, want %q", g.File, "internal/server/handler.go")
	}
	if g.Line != 42 {
		t.Errorf("Line = %d, want %d", g.Line, 42)
	}
	if g.EndLine != 89 {
		t.Errorf("EndLine = %d, want %d", g.EndLine, 89)
	}

	if g.Symbol == nil {
		t.Fatal("Symbol is nil")
	}
	if g.Symbol.Name != "handleRequest" {
		t.Errorf("Symbol.Name = %q, want %q", g.Symbol.Name, "handleRequest")
	}
	if g.Symbol.Kind != "function" {
		t.Errorf("Symbol.Kind = %q, want %q", g.Symbol.Kind, "function")
	}

	if g.Metadata["complexity"] != "22" {
		t.Errorf("Metadata[complexity] = %q, want %q", g.Metadata["complexity"], "22")
	}
	if g.Metadata["threshold"] != "10" {
		t.Errorf("Metadata[threshold] = %q, want %q", g.Metadata["threshold"], "10")
	}

	if len(g.Tags) != 2 || g.Tags[0] != "complexity" || g.Tags[1] != "refactor" {
		t.Errorf("Tags = %v, want [complexity refactor]", g.Tags)
	}

	if g.Span == nil {
		t.Fatal("Span is nil")
	}
	if g.Span.StartLine != 42 {
		t.Errorf("Span.StartLine = %d, want %d", g.Span.StartLine, 42)
	}
	if g.Span.EndLine != 89 {
		t.Errorf("Span.EndLine = %d, want %d", g.Span.EndLine, 89)
	}
	if g.Span.StartColumn != 1 {
		t.Errorf("Span.StartColumn = %d, want %d", g.Span.StartColumn, 1)
	}
	if g.Span.EndColumn != 50 {
		t.Errorf("Span.EndColumn = %d, want %d", g.Span.EndColumn, 50)
	}

	if len(g.Related) != 1 {
		t.Fatalf("len(Related) = %d, want 1", len(g.Related))
	}
	if g.Related[0].Message != "related issue" {
		t.Errorf("Related[0].Message = %q, want %q", g.Related[0].Message, "related issue")
	}

	if len(g.Fixes) != 1 {
		t.Fatalf("len(Fixes) = %d, want 1", len(g.Fixes))
	}
	if g.Fixes[0].Title != "Extract function" {
		t.Errorf("Fixes[0].Title = %q, want %q", g.Fixes[0].Title, "Extract function")
	}
	if g.Fixes[0].Kind != "extract" {
		t.Errorf("Fixes[0].Kind = %q, want %q", g.Fixes[0].Kind, "extract")
	}
	if g.Fixes[0].Text != "extracted" {
		t.Errorf("Fixes[0].Text = %q, want %q", g.Fixes[0].Text, "extracted")
	}
}
