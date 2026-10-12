package analyzer

import (
	"fmt"
	"path/filepath"

	"github.com/canadian-ai/girl/internal/complexity"
	"github.com/canadian-ai/girl/internal/ir"
)

// detectFunctionSmells reports long and branch-heavy TS/JS functions. The
// React rules only look at components, so without this a plain TypeScript
// codebase (a CLI, a server, an editor) would look spotless to an agent.
func (a *Analyzer) detectFunctionSmells(path string, files []*ir.FileIR) ([]ir.Diagnostic, error) {
	if len(files) == 0 {
		return nil, nil
	}
	report, err := complexity.Analyze(path, complexity.Options{
		Language:  "all",
		Threshold: a.config.MaxComplexity,
		Exclude:   a.config.ExcludeDirs,
	})
	if err != nil {
		return nil, err
	}

	// Only report on files the parser kept, so generated files stay excluded.
	kept := make(map[string]bool, len(files))
	for _, f := range files {
		kept[filepath.Clean(f.Path)] = true
	}
	root := path
	if len(files) == 1 && filepath.Clean(files[0].Path) == filepath.Clean(path) {
		root = filepath.Dir(path)
	}

	var diags []ir.Diagnostic
	for _, fn := range report.Functions {
		if fn.Kind == "react-component" {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(fn.File))
		if !kept[filepath.Clean(full)] {
			continue
		}
		diags = append(diags, a.functionDiagnostics(fn, relPath(full))...)
	}
	return diags, nil
}

func (a *Analyzer) functionDiagnostics(fn complexity.FunctionMetric, file string) []ir.Diagnostic {
	var diags []ir.Diagnostic
	span := &ir.Span{StartLine: fn.StartLine, EndLine: fn.EndLine}
	if limit := a.config.MaxComplexity; fn.Complexity > limit {
		diags = append(diags, ir.Diagnostic{
			Code:       "ts.high-complexity",
			Severity:   scaledSeverity(fn.Complexity, limit),
			Message:    fmt.Sprintf("Function %s has cyclomatic complexity %d (limit: %d)", fn.Symbol, fn.Complexity, limit),
			File:       file,
			Line:       fn.StartLine,
			EndLine:    fn.EndLine,
			Span:       span,
			Suggestion: "Reduce branching with guard clauses, early returns, lookup tables, or extracted helpers.",
			Kind:       ir.NodeKindFunction,
			Symbol:     fn.Symbol,
		})
	}
	if limit := a.config.MaxFunctionLines; fn.Lines > limit {
		diags = append(diags, ir.Diagnostic{
			Code:       "ts.long-function",
			Severity:   scaledSeverity(fn.Lines, limit),
			Message:    fmt.Sprintf("Function %s is %d lines (limit: %d)", fn.Symbol, fn.Lines, limit),
			File:       file,
			Line:       fn.StartLine,
			EndLine:    fn.EndLine,
			Span:       span,
			Suggestion: "Extract cohesive blocks into well-named helper functions.",
			Kind:       ir.NodeKindFunction,
			Symbol:     fn.Symbol,
		})
	}
	return diags
}

// scaledSeverity mirrors the Go analyzer: over the limit is low, over 1.5x is
// medium, over 2x is high.
func scaledSeverity(value, limit int) ir.Severity {
	switch {
	case value > limit*2:
		return ir.SeverityHigh
	case value > limit*3/2:
		return ir.SeverityMedium
	default:
		return ir.SeverityLow
	}
}
