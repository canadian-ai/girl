// Package pythonanalysis maps Python AST findings into GIRL diagnostics.
// Python's standard-library ast parser is used without importing project modules.
package pythonanalysis

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/shared"
)

// Codes emitted by this binding. Every code must have a registered GRP recipe
// (see internal/recipes) so findings are never silently dropped by the planner.
const (
	CodeSyntaxError    = "py.syntax-error"
	CodeAnalysisError  = "py.analysis-error"
	CodeLargeFile      = "py.large-file"
	CodeLongFunction   = "py.long-function"
	CodeHighComplexity = "py.high-complexity"
	CodeDeepNesting    = "py.deep-nesting"
	CodeTooManyParams  = "py.too-many-params"
)

// EmittedCodes lists every diagnostic code the Python binding can produce.
func EmittedCodes() []string {
	return []string{CodeSyntaxError, CodeAnalysisError, CodeLargeFile, CodeLongFunction,
		CodeHighComplexity, CodeDeepNesting, CodeTooManyParams}
}

// ErrRuntimeMissing is returned when no python3 interpreter is available. It is
// distinct from a parse failure: GIRL could not analyze anything.
var ErrRuntimeMissing = errors.New("python3 runtime not found on PATH; Python files cannot be analyzed")

type Config struct {
	MaxFunctionLines, MaxComplexity, MaxNesting, MaxFileLines, MaxParams int
	// Skip excludes paths (project exclusions) before any file is read.
	Skip shared.PathFilter
}

func DefaultConfig() *Config {
	return &Config{MaxFunctionLines: 80, MaxComplexity: 10, MaxNesting: 4, MaxFileLines: 500, MaxParams: 5}
}

type function struct {
	Name       string `json:"name"`
	Start      int    `json:"start"`
	End        int    `json:"end"`
	Params     int    `json:"params"`
	Complexity int    `json:"complexity"`
	Nesting    int    `json:"nesting"`
}

type parsed struct {
	Functions []function `json:"functions"`
	// Kind is "syntax" for a SyntaxError (including unsupported syntax for
	// the running interpreter version) or "analysis" for other parser
	// failures such as null bytes or recursion limits.
	Kind    string `json:"kind"`
	Error   string `json:"error"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Version string `json:"version"`
}

// This program only parses syntax. It never executes the analyzed source.
// Failures are reported in a JSON envelope with exit 0 so the Go side can
// emit a structured, blocking diagnostic; transport success is not a pass.
const astScript = `
import ast, json, sys
version = "%d.%d" % sys.version_info[:2]
source = sys.stdin.buffer.read()
try:
    tree = ast.parse(source)
except SyntaxError as e:
    print(json.dumps({"kind": "syntax", "error": e.msg or "invalid syntax", "line": e.lineno or 1, "column": e.offset or 0, "version": version}))
    sys.exit(0)
except (ValueError, RecursionError, MemoryError) as e:
    print(json.dumps({"kind": "analysis", "error": str(e) or type(e).__name__, "line": 1, "column": 0, "version": version}))
    sys.exit(0)
found = []
branches = tuple(getattr(ast, n) for n in ("If", "For", "AsyncFor", "While", "Try", "TryStar", "With", "AsyncWith", "Match") if hasattr(ast, n))
decisions = tuple(getattr(ast, n) for n in ("If", "For", "AsyncFor", "While", "ExceptHandler", "IfExp", "Assert", "Match", "comprehension") if hasattr(ast, n))
class Visitor(ast.NodeVisitor):
    def __init__(self):
        self.scope = []
    def visit_ClassDef(self, node):
        self.scope.append(node.name)
        self.generic_visit(node)
        self.scope.pop()
    def visit_FunctionDef(self, node):
        self.function(node)
    visit_AsyncFunctionDef = visit_FunctionDef
    def function(self, node):
        name = ".".join(self.scope + [node.name])
        params = node.args
        count = len(getattr(params, "posonlyargs", [])) + len(params.args) + len(params.kwonlyargs)
        count += bool(params.vararg) + bool(params.kwarg)
        complexity = 1
        nesting = 0
        def walk(n, depth=0):
            nonlocal complexity, nesting
            if n is not node and isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef, ast.Lambda)):
                return
            if n is not node:
                if isinstance(n, decisions):
                    complexity += 1
                elif isinstance(n, ast.BoolOp):
                    complexity += max(0, len(n.values) - 1)
                if isinstance(n, branches):
                    depth += 1
                    nesting = max(nesting, depth)
            for child in ast.iter_child_nodes(n):
                walk(child, depth)
        walk(node)
        found.append(dict(name=name, start=node.lineno, end=getattr(node, "end_lineno", None) or node.lineno,
                          params=count, complexity=complexity, nesting=nesting))
        self.scope.append(node.name)
        self.generic_visit(node)
        self.scope.pop()
Visitor().visit(tree)
print(json.dumps({"functions": found, "version": version}))
`

func skipPythonDir(name string) bool {
	switch name {
	case ".venv", "venv", "__pycache__", ".tox", ".mypy_cache", ".pytest_cache":
		return true
	}
	return shared.ShouldSkipDir(name)
}

func collectPaths(path string, skip shared.PathFilter) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if strings.EqualFold(filepath.Ext(path), ".py") {
			return []string{path}, nil
		}
		return nil, nil
	}
	var paths []string
	err = filepath.Walk(path, func(p string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if p == path {
			return nil
		}
		if fi.IsDir() {
			if skipPythonDir(fi.Name()) || skip.Skip(p, true) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(p), ".py") && !shared.IsGeneratedName(p) && !shared.IsSymlink(fi) && !skip.Skip(p, false) {
			paths = append(paths, p)
		}
		return nil
	})
	sort.Strings(paths)
	return paths, err
}

func countLines(source []byte) int {
	lines := len(strings.Split(string(source), "\n"))
	if len(source) > 0 && source[len(source)-1] == '\n' {
		lines--
	}
	return lines
}

func runParser(source []byte) (*parsed, error) {
	if _, err := exec.LookPath("python3"); err != nil {
		return nil, ErrRuntimeMissing
	}
	cmd := exec.Command("python3", "-I", "-c", astScript)
	cmd.Stdin = bytes.NewReader(source)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	raw, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("python3 parser failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var out parsed
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("python3 parser returned invalid output: %w", err)
	}
	return &out, nil
}

func AnalyzePath(path string, cfg *Config) (*ir.AnalyzerResult, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	paths, err := collectPaths(path, cfg.Skip)
	if err != nil {
		return nil, err
	}
	out := &ir.AnalyzerResult{Files: []*ir.FileIR{}, Diagnostics: []ir.Diagnostic{}}
	for _, p := range paths {
		source, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		if shared.IsGeneratedSource(source) {
			continue
		}
		parsedFile, err := runParser(source)
		if err != nil {
			if errors.Is(err, ErrRuntimeMissing) {
				return nil, err
			}
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		rel := p
		if cwd, e := os.Getwd(); e == nil {
			if r, e := filepath.Rel(cwd, p); e == nil {
				rel = r
			}
		}
		lines := countLines(source)
		fileIR := &ir.FileIR{Path: rel, Language: "python", Lines: lines}
		out.Files = append(out.Files, fileIR)
		emit := func(code string, sev ir.Severity, msg, suggestion, symbol string, start, end int) {
			kind := ir.NodeKindFunction
			if symbol == "" {
				kind = ir.NodeKindFile
			}
			out.Diagnostics = append(out.Diagnostics, ir.Diagnostic{Code: code, Severity: sev, Confidence: "high", Message: msg, File: rel, Line: start, EndLine: end, Symbol: symbol, Kind: kind, Suggestion: suggestion, Span: &ir.Span{StartLine: start, EndLine: end}})
		}
		switch parsedFile.Kind {
		case "syntax":
			se := ir.SyntaxError{Line: parsedFile.Line, Column: parsedFile.Column,
				Message: fmt.Sprintf("%s (Python %s parser; newer syntax may be unsupported by this interpreter)", parsedFile.Error, parsedFile.Version)}
			fileIR.SyntaxError = &se
			d := ir.SyntaxErrorDiagnostic("py", rel, se)
			d.Metadata = map[string]string{"pythonVersion": parsedFile.Version}
			out.Diagnostics = append(out.Diagnostics, d)
			continue
		case "analysis":
			out.Diagnostics = append(out.Diagnostics, ir.Diagnostic{
				Code: CodeAnalysisError, Severity: ir.SeverityHigh, Confidence: "high",
				Message: fmt.Sprintf("Python parser could not analyze file: %s", parsedFile.Error),
				File:    rel, Line: 1, EndLine: 1, Kind: ir.NodeKindFile,
				Suggestion: "Make the file parseable (remove null bytes, reduce pathological nesting) before refactoring.",
				Tags:       []string{ir.TagBlocking, ir.TagAnalysisError},
				Metadata:   map[string]string{"pythonVersion": parsedFile.Version},
				Span:       &ir.Span{StartLine: 1, EndLine: 1},
			})
			continue
		}
		if lines > cfg.MaxFileLines {
			emit(CodeLargeFile, ir.SeverityMedium, fmt.Sprintf("File has %d lines (limit: %d)", lines, cfg.MaxFileLines), "Split independent responsibilities into modules.", "", 1, lines)
		}
		for _, fn := range parsedFile.Functions {
			span := fn.End - fn.Start + 1
			if span > cfg.MaxFunctionLines {
				emit(CodeLongFunction, ir.SeverityMedium, fmt.Sprintf("Function %s has %d lines (limit: %d)", fn.Name, span, cfg.MaxFunctionLines), "Extract cohesive helpers.", fn.Name, fn.Start, fn.End)
			}
			if fn.Complexity > cfg.MaxComplexity {
				emit(CodeHighComplexity, ir.SeverityHigh, fmt.Sprintf("Function %s has complexity %d (limit: %d)", fn.Name, fn.Complexity, cfg.MaxComplexity), "Reduce decision branches and extract helpers.", fn.Name, fn.Start, fn.End)
			}
			if fn.Nesting > cfg.MaxNesting {
				emit(CodeDeepNesting, ir.SeverityMedium, fmt.Sprintf("Function %s has nesting %d (limit: %d)", fn.Name, fn.Nesting, cfg.MaxNesting), "Use guard clauses and smaller helpers.", fn.Name, fn.Start, fn.End)
			}
			if fn.Params > cfg.MaxParams {
				emit(CodeTooManyParams, ir.SeverityLow, fmt.Sprintf("Function %s has %d parameters (limit: %d)", fn.Name, fn.Params, cfg.MaxParams), "Group related arguments where appropriate.", fn.Name, fn.Start, fn.End)
			}
		}
	}
	sort.Slice(out.Diagnostics, func(i, j int) bool {
		a, b := out.Diagnostics[i], out.Diagnostics[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Code < b.Code
	})
	return out, nil
}
