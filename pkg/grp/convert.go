package grp

import (
	"fmt"

	"github.com/canadian-ai/girl/internal/ir"
)

func FromIRPlan(irPlan *ir.GrpPlan) *Plan {
	if irPlan == nil {
		return nil
	}

	p := &Plan{
		SpecVersion:  "0.1",
		ID:           irPlan.PlanID,
		Type:         "dev.refactor.plan",
		Source:       "github.com/canadian-ai/girl",
		Subject:      irPlan.Target,
		Language:     "auto",
		Goal:         irPlan.Goal,
		Risk:         Severity(irPlan.Risk),
		Diagnostics:  make([]Diagnostic, len(irPlan.Diagnostics)),
		Steps:        make([]Step, len(irPlan.Steps)),
		Verification: convertVerification(irPlan.Verification),
	}

	for i, d := range irPlan.Diagnostics {
		p.Diagnostics[i] = convertDiagnostic(d, i)
	}

	for i, s := range irPlan.Steps {
		p.Steps[i] = convertStep(s)
	}

	return p
}

func convertVerification(cmds []string) []Verification {
	v := make([]Verification, len(cmds))
	for i, cmd := range cmds {
		v[i] = Verification{
			Command:    cmd,
			Required:   true,
			Source:     "auto",
			Confidence: "high",
		}
	}
	return v
}

func convertDiagnostic(d ir.Diagnostic, index int) Diagnostic {
	g := Diagnostic{
		ID:         fmt.Sprintf("diag_%d", index),
		Code:       d.Code,
		Severity:   Severity(d.Severity),
		Confidence: ConfidenceHigh,
		Message:    d.Message,
		File:       d.File,
		Line:       d.Line,
		EndLine:    d.EndLine,
		Metadata:   d.Metadata,
		Tags:       d.Tags,
		Related:    make([]RelatedInfo, len(d.Related)),
		Fixes:      make([]Fix, len(d.Fixes)),
	}

	if d.Span != nil {
		g.Span = &Span{
			StartLine:   d.Span.StartLine,
			StartColumn: d.Span.StartCol,
			EndLine:     d.Span.EndLine,
			EndColumn:   d.Span.EndCol,
		}
	}

	if d.Symbol != "" || d.Component != "" || d.Kind != "" {
		sym := &Symbol{}
		if d.Symbol != "" {
			sym.Name = d.Symbol
		} else if d.Component != "" {
			sym.Name = d.Component
		}
		if d.Kind != "" {
			sym.Kind = string(d.Kind)
		}
		g.Symbol = sym
	}

	for i, r := range d.Related {
		g.Related[i] = RelatedInfo{
			Message: r.Message,
			Span: Span{
				StartLine:   r.Span.StartLine,
				StartColumn: r.Span.StartCol,
				EndLine:     r.Span.EndLine,
				EndColumn:   r.Span.EndCol,
			},
		}
	}

	for i, f := range d.Fixes {
		g.Fixes[i] = Fix{
			Title: f.Title,
			Kind:  f.Kind,
			Span: Span{
				StartLine:   f.Span.StartLine,
				StartColumn: f.Span.StartCol,
				EndLine:     f.Span.EndLine,
				EndColumn:   f.Span.EndCol,
			},
			Text: f.Text,
		}
	}

	return g
}

func convertStep(s ir.GrpStep) Step {
	title := s.Recipe
	if title == "" {
		title = s.Action
	}

	return Step{
		ID:     s.ID,
		Recipe: s.Recipe,
		Title:  title,
		Action: s.Action,
		Target: Target{
			File: s.File,
		},
		Risk:   Severity(s.Risk),
		Verify: convertVerification(s.Verify),
	}
}
