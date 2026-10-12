package analyzer

import (
	"fmt"
	"path/filepath"
	"sort"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/canadian-ai/girl/internal/complexity"
	"github.com/canadian-ai/girl/internal/ir"
)

// metricSink collects per-file function metrics from parser workers.
type metricSink struct {
	threshold int
	mu        sync.Mutex
	byPath    map[string][]complexity.FunctionMetric
}

func newMetricSink(threshold int) *metricSink {
	return &metricSink{threshold: threshold, byPath: map[string][]complexity.FunctionMetric{}}
}

func (s *metricSink) visit(fir *ir.FileIR, root *sitter.Node, data []byte) {
	metrics, err := complexity.MeasureTree(fir.Path, fir.Path, root, data, s.threshold)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.byPath[fir.Path] = metrics
	s.mu.Unlock()
}

// detectFunctionSmells reports long and branch-heavy TS/JS functions. The
// React rules only look at components, so without this a plain TypeScript
// codebase (a CLI, a server, an editor) would look spotless to an agent.
func (a *Analyzer) detectFunctionSmells(path string, files []*ir.FileIR, sink *metricSink) []ir.Diagnostic {
	root := path
	if len(files) == 1 && filepath.Clean(files[0].Path) == filepath.Clean(path) {
		root = filepath.Dir(path)
	}
	type fileMetrics struct {
		rel     string
		path    string
		metrics []complexity.FunctionMetric
	}
	ordered := make([]fileMetrics, 0, len(files))
	for _, f := range files {
		rel, err := filepath.Rel(root, f.Path)
		if err != nil {
			rel = filepath.Base(f.Path)
		}
		ordered = append(ordered, fileMetrics{rel: filepath.ToSlash(rel), path: f.Path, metrics: sink.byPath[f.Path]})
	}
	// Same order complexity.Analyze reports: repo-relative path, line, symbol.
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].rel < ordered[j].rel })

	var diags []ir.Diagnostic
	for _, fm := range ordered {
		fns := append([]complexity.FunctionMetric(nil), fm.metrics...)
		sort.Slice(fns, func(i, j int) bool {
			if fns[i].StartLine != fns[j].StartLine {
				return fns[i].StartLine < fns[j].StartLine
			}
			return fns[i].Symbol < fns[j].Symbol
		})
		file := relPath(fm.path)
		for _, fn := range fns {
			if fn.Kind == "react-component" {
				continue
			}
			diags = append(diags, a.functionDiagnostics(fn, file)...)
		}
	}
	return diags
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
