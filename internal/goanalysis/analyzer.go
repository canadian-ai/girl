package goanalysis

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/shared"
)

func AnalyzePath(path string, cfg *Config) (*ir.AnalyzerResult, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	goFiles, err := collectGoFiles(path)
	if err != nil {
		return nil, err
	}

	diags := detectDiagnostics(goFiles, cfg)
	if diags == nil {
		diags = []ir.Diagnostic{}
	}
	sortDiagnostics(diags)

	return &ir.AnalyzerResult{
		Files:       fileIRs(goFiles),
		Diagnostics: diags,
	}, nil
}

func collectGoFiles(path string) ([]*GoFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path %s: %w", path, err)
	}

	if !info.IsDir() {
		gf, err := ParseGoFile(path)
		if err != nil {
			return nil, err
		}
		return []*GoFile{gf}, nil
	}

	return walkGoFiles(path)
}

func walkGoFiles(path string) ([]*GoFile, error) {
	var goFiles []*GoFile
	err := filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if shouldSkipWalkEntry(p, fi) {
			return filepath.SkipDir
		}
		if IsGoFile(p) {
			if gf, parseErr := ParseGoFile(p); parseErr == nil {
				goFiles = append(goFiles, gf)
			}
		}
		return nil
	})
	return goFiles, err
}

func shouldSkipWalkEntry(path string, fi os.FileInfo) bool {
	return fi.IsDir() && filepath.Base(path) != "." && shared.ShouldSkipDir(filepath.Base(path))
}

func fileIRs(goFiles []*GoFile) []*ir.FileIR {
	files := make([]*ir.FileIR, len(goFiles))
	for i, gf := range goFiles {
		files[i] = &ir.FileIR{
			Path:     gf.Path,
			Language: "go",
			Lines:    gf.Lines,
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

func detectDiagnostics(files []*GoFile, cfg *Config) []ir.Diagnostic {
	var diags []ir.Diagnostic
	for _, f := range files {
		diags = append(diags, detectLargeFile(f, cfg)...)
		for _, fn := range f.Functions {
			diags = append(diags, detectLongFunction(f, fn, cfg)...)
			diags = append(diags, detectHighComplexity(f, fn, cfg)...)
			diags = append(diags, detectDeepNesting(f, fn, cfg)...)
			diags = append(diags, detectTooManyParams(f, fn, cfg)...)
			diags = append(diags, detectIgnoredErrors(f, fn, cfg)...)
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

func fnName(fn GoFunction) string {
	if fn.Receiver != "" {
		return fmt.Sprintf("%s.%s", fn.Receiver, fn.Name)
	}
	return fn.Name
}

func detectLargeFile(f *GoFile, cfg *Config) []ir.Diagnostic {
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
		Code:       "go.large-file",
		Severity:   sev,
		Message:    fmt.Sprintf("File %q is %d lines (limit: %d)", relPath(f.Path), f.Lines, cfg.MaxFileLines),
		File:       relPath(f.Path),
		Suggestion: "Split this file by responsibility into smaller packages or files.",
		Kind:       ir.NodeKindFile,
		Symbol:     relPath(f.Path),
		EndLine:    f.Lines,
	}}
}

func detectLongFunction(f *GoFile, fn GoFunction, cfg *Config) []ir.Diagnostic {
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
		Code:       "go.long-function",
		Severity:   sev,
		Message:    fmt.Sprintf("Function %s in %q is %d lines (limit: %d)", fnName(fn), relPath(f.Path), fn.Lines, cfg.MaxFunctionLines),
		File:       relPath(f.Path),
		Line:       fn.StartLine,
		Suggestion: "Extract smaller helper functions or simplify the logic. Aim for functions under 50 lines.",
		Kind:       ir.NodeKindFunction,
		Symbol:     fnName(fn),
		EndLine:    fn.EndLine,
		Span:       &ir.Span{StartLine: fn.StartLine, EndLine: fn.EndLine},
	}}
}

func detectHighComplexity(f *GoFile, fn GoFunction, cfg *Config) []ir.Diagnostic {
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
		Code:       "go.high-complexity",
		Severity:   sev,
		Message:    fmt.Sprintf("Function %s has cyclomatic complexity %d (limit: %d)", fnName(fn), fn.Complexity, cfg.MaxComplexity),
		File:       relPath(f.Path),
		Line:       fn.StartLine,
		Suggestion: "Reduce branching with early returns, guard clauses, or table-driven tests.",
		Kind:       ir.NodeKindFunction,
		Symbol:     fnName(fn),
	}}
}

func detectDeepNesting(f *GoFile, fn GoFunction, cfg *Config) []ir.Diagnostic {
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
		Code:       "go.deep-nesting",
		Severity:   sev,
		Message:    fmt.Sprintf("Function %s has nesting depth %d (limit: %d)", fnName(fn), fn.MaxNesting, cfg.MaxNesting),
		File:       relPath(f.Path),
		Line:       fn.StartLine,
		Suggestion: "Extract inner logic into helper functions or use guard clauses to flatten.",
		Kind:       ir.NodeKindFunction,
		Symbol:     fnName(fn),
	}}
}

func detectTooManyParams(f *GoFile, fn GoFunction, cfg *Config) []ir.Diagnostic {
	if fn.Params <= cfg.MaxParams {
		return nil
	}
	return []ir.Diagnostic{{
		Code:       "go.too-many-params",
		Severity:   ir.SeverityLow,
		Message:    fmt.Sprintf("Function %s has %d parameters (limit: %d)", fnName(fn), fn.Params, cfg.MaxParams),
		File:       relPath(f.Path),
		Line:       fn.StartLine,
		Suggestion: "Group related parameters into a config/options struct.",
		Kind:       ir.NodeKindFunction,
		Symbol:     fnName(fn),
	}}
}

func detectIgnoredErrors(f *GoFile, fn GoFunction, cfg *Config) []ir.Diagnostic {
	if fn.IgnoredErrs == 0 {
		return nil
	}
	if cfg.IgnoredErrorConfidence == "off" {
		return nil
	}
	confidence := fn.IgnoredErrConfidence
	if cfg.IgnoredErrorConfidence == "always" {
		confidence = "high"
	}
	return []ir.Diagnostic{{
		Code:       "go.ignored-error",
		Severity:   ir.SeverityMedium,
		Message:    fmt.Sprintf("Function %s ignores %d error(s) with _", fnName(fn), fn.IgnoredErrs),
		File:       relPath(f.Path),
		Line:       fn.StartLine,
		Suggestion: "Check each error explicitly or use a helper like `must` for expected failures.",
		Kind:       ir.NodeKindFunction,
		Symbol:     fnName(fn),
		Metadata: map[string]string{
			"reason":     "error return value ignored with _",
			"confidence": confidence,
		},
	}}
}
