package rustanalysis

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/canadian-ai/girl/internal/ir"
)

func AnalyzePath(path string, cfg *Config) (*ir.AnalyzerResult, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	rustFiles, err := collectRustFiles(path)
	if err != nil {
		return nil, err
	}

	diags := detectDiagnostics(rustFiles, cfg)
	if diags == nil {
		diags = []ir.Diagnostic{}
	}
	sortDiagnostics(diags)

	return &ir.AnalyzerResult{
		Files:       fileIRs(rustFiles),
		Diagnostics: diags,
	}, nil
}

func fileIRs(rustFiles []*RustFile) []*ir.FileIR {
	files := make([]*ir.FileIR, len(rustFiles))
	for i, rf := range rustFiles {
		files[i] = &ir.FileIR{
			Path:     rf.Path,
			Language: "rust",
			Lines:    rf.Lines,
		}
	}
	return files
}

func sortDiagnostics(diags []ir.Diagnostic) {
	severityOrder := map[ir.Severity]int{ir.SeverityHigh: 0, ir.SeverityMedium: 1, ir.SeverityLow: 2}
	sort.Slice(diags, func(i, j int) bool {
		if diags[i].Severity != diags[j].Severity {
			return severityOrder[diags[i].Severity] < severityOrder[diags[j].Severity]
		}
		return diags[i].Code < diags[j].Code
	})
}

func detectDiagnostics(files []*RustFile, cfg *Config) []ir.Diagnostic {
	var diags []ir.Diagnostic
	for _, f := range files {
		diags = append(diags, detectLargeFile(f, cfg)...)
		for _, fn := range f.Functions {
			diags = append(diags, detectLongFunction(f, fn, cfg)...)
			diags = append(diags, detectHighComplexity(f, fn, cfg)...)
			diags = append(diags, detectDeepNesting(f, fn, cfg)...)
			diags = append(diags, detectTooManyParams(f, fn, cfg)...)
		}
	}
	return diags
}

func relPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}
	return rel
}

func fnName(fn RustFunction) string {
	if fn.Receiver != "" {
		return fmt.Sprintf("%s::%s", fn.Receiver, fn.Name)
	}
	return fn.Name
}

func detectLargeFile(f *RustFile, cfg *Config) []ir.Diagnostic {
	if f.Lines <= cfg.MaxFileLines {
		return nil
	}
	sev := ir.SeverityLow
	if f.Lines > cfg.MaxFileLines*2 {
		sev = ir.SeverityHigh
	} else if f.Lines > cfg.MaxFileLines*3/2 {
		sev = ir.SeverityMedium
	}
	return []ir.Diagnostic{{
		Code:       "rust.large-file",
		Severity:   sev,
		Message:    fmt.Sprintf("File %q is %d lines (limit: %d)", relPath(f.Path), f.Lines, cfg.MaxFileLines),
		File:       relPath(f.Path),
		Suggestion: "Split this file by responsibility into smaller modules.",
		Kind:       ir.NodeKindFile,
		Symbol:     relPath(f.Path),
		EndLine:    f.Lines,
	}}
}

func detectLongFunction(f *RustFile, fn RustFunction, cfg *Config) []ir.Diagnostic {
	if fn.Lines <= cfg.MaxFunctionLines {
		return nil
	}
	sev := ir.SeverityLow
	if fn.Lines > cfg.MaxFunctionLines*2 {
		sev = ir.SeverityHigh
	} else if fn.Lines > cfg.MaxFunctionLines*3/2 {
		sev = ir.SeverityMedium
	}
	return []ir.Diagnostic{{
		Code:       "rust.long-function",
		Severity:   sev,
		Message:    fmt.Sprintf("Function %s in %q is %d lines (limit: %d)", fnName(fn), relPath(f.Path), fn.Lines, cfg.MaxFunctionLines),
		File:       relPath(f.Path),
		Line:       fn.StartLine,
		Suggestion: "Extract smaller helper functions or simplify the logic.",
		Kind:       ir.NodeKindFunction,
		Symbol:     fnName(fn),
		EndLine:    fn.EndLine,
		Span:       &ir.Span{StartLine: fn.StartLine, EndLine: fn.EndLine},
	}}
}

func detectHighComplexity(f *RustFile, fn RustFunction, cfg *Config) []ir.Diagnostic {
	if fn.Complexity <= cfg.MaxComplexity {
		return nil
	}
	sev := ir.SeverityLow
	if fn.Complexity > cfg.MaxComplexity*2 {
		sev = ir.SeverityHigh
	} else if fn.Complexity > cfg.MaxComplexity*3/2 {
		sev = ir.SeverityMedium
	}
	return []ir.Diagnostic{{
		Code:       "rust.high-complexity",
		Severity:   sev,
		Message:    fmt.Sprintf("Function %s has cyclomatic complexity %d (limit: %d)", fnName(fn), fn.Complexity, cfg.MaxComplexity),
		File:       relPath(f.Path),
		Line:       fn.StartLine,
		Suggestion: "Reduce branching with early returns, guard clauses, or match arms.",
		Kind:       ir.NodeKindFunction,
		Symbol:     fnName(fn),
	}}
}

func detectDeepNesting(f *RustFile, fn RustFunction, cfg *Config) []ir.Diagnostic {
	if fn.MaxNesting <= cfg.MaxNesting {
		return nil
	}
	sev := ir.SeverityLow
	if fn.MaxNesting > cfg.MaxNesting+3 {
		sev = ir.SeverityHigh
	} else if fn.MaxNesting > cfg.MaxNesting+1 {
		sev = ir.SeverityMedium
	}
	return []ir.Diagnostic{{
		Code:       "rust.deep-nesting",
		Severity:   sev,
		Message:    fmt.Sprintf("Function %s has nesting depth %d (limit: %d)", fnName(fn), fn.MaxNesting, cfg.MaxNesting),
		File:       relPath(f.Path),
		Line:       fn.StartLine,
		Suggestion: "Extract inner logic into helper functions or use guard clauses to flatten.",
		Kind:       ir.NodeKindFunction,
		Symbol:     fnName(fn),
	}}
}

func detectTooManyParams(f *RustFile, fn RustFunction, cfg *Config) []ir.Diagnostic {
	if fn.Params <= cfg.MaxParams {
		return nil
	}
	sev := ir.SeverityLow
	if fn.Params > cfg.MaxParams*2 {
		sev = ir.SeverityMedium
	}
	return []ir.Diagnostic{{
		Code:       "rust.too-many-params",
		Severity:   sev,
		Message:    fmt.Sprintf("Function %s has %d parameters (limit: %d)", fnName(fn), fn.Params, cfg.MaxParams),
		File:       relPath(f.Path),
		Line:       fn.StartLine,
		Suggestion: "Group related parameters into a config/options struct.",
		Kind:       ir.NodeKindFunction,
		Symbol:     fnName(fn),
	}}
}
