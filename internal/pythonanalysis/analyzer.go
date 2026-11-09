// Package pythonanalysis maps Python AST findings into GIRL diagnostics.
// Python's standard-library ast parser is used without importing project modules.
package pythonanalysis

import (
 "bytes"
 "encoding/json"
 "fmt"
 "os"
 "os/exec"
 "path/filepath"
 "sort"
 "strings"

 "github.com/canadian-ai/girl/internal/ir"
 "github.com/canadian-ai/girl/internal/shared"
)

type Config struct { MaxFunctionLines, MaxComplexity, MaxNesting, MaxFileLines, MaxParams int }
func DefaultConfig() *Config { return &Config{80,10,4,500,5} }

type function struct {
 Name string `json:"name"`
 Start int `json:"start"`
 End int `json:"end"`
 Params int `json:"params"`
 Complexity int `json:"complexity"`
 Nesting int `json:"nesting"`
}
type parsed struct {
 Functions []function `json:"functions"`
 Error string `json:"error"`
}
// This program only parses syntax. It never executes the analyzed source.
const astScript = `
import ast, json, sys
source = sys.stdin.read()
try:
    tree = ast.parse(source)
except SyntaxError as e:
    print(json.dumps({"error": f"{e.msg} (line {e.lineno})"}))
    sys.exit(0)
found = []
branches = (ast.If, ast.For, ast.AsyncFor, ast.While, ast.Try, ast.TryStar, ast.With, ast.AsyncWith, ast.Match)
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
        count = len(params.posonlyargs) + len(params.args) + len(params.kwonlyargs)
        count += bool(params.vararg) + bool(params.kwarg)
        complexity = 1
        nesting = 0
        def walk(n, depth=0):
            nonlocal complexity, nesting
            if n is not node and isinstance(n, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef, ast.Lambda)):
                return
            if n is not node:
                if isinstance(n, (ast.If, ast.For, ast.AsyncFor, ast.While, ast.ExceptHandler, ast.IfExp, ast.Assert, ast.Match, ast.comprehension)):
                    complexity += 1
                elif isinstance(n, ast.BoolOp):
                    complexity += max(0, len(n.values) - 1)
                if isinstance(n, branches):
                    depth += 1
                    nesting = max(nesting, depth)
            for child in ast.iter_child_nodes(n):
                walk(child, depth)
        walk(node)
        found.append(dict(name=name, start=node.lineno, end=node.end_lineno or node.lineno,
                          params=count, complexity=complexity, nesting=nesting))
        self.scope.append(node.name)
        self.generic_visit(node)
        self.scope.pop()
Visitor().visit(tree)
print(json.dumps({"functions": found}))
`

func AnalyzePath(path string, cfg *Config) (*ir.AnalyzerResult, error) {
 if cfg == nil { cfg = DefaultConfig() }
 paths := []string{}
 info, err := os.Stat(path)
 if err != nil { return nil, err }
 if info.IsDir() {
  err = filepath.Walk(path, func(p string, fi os.FileInfo, walkErr error) error {
   if walkErr != nil { return walkErr }
   if fi.IsDir() {
    if p != path && (shared.ShouldSkipDir(fi.Name()) || fi.Name() == ".venv" || fi.Name() == "venv" || fi.Name() == "__pycache__" || fi.Name() == ".tox" || fi.Name() == ".mypy_cache" || fi.Name() == ".pytest_cache") { return filepath.SkipDir }
    return nil
   }
   if strings.EqualFold(filepath.Ext(p), ".py") && !shared.IsGeneratedName(p) { paths = append(paths,p) }
   return nil
  })
  if err != nil { return nil, err }
 } else if strings.EqualFold(filepath.Ext(path), ".py") { paths = append(paths,path) }
 sort.Strings(paths)
 out := &ir.AnalyzerResult{Files: []*ir.FileIR{}, Diagnostics: []ir.Diagnostic{}}
 for _, p := range paths {
  source, err := os.ReadFile(p); if err != nil { return nil, err }
  if shared.IsGeneratedSource(source) { continue }
  cmd := exec.Command("python3", "-c", astScript)
  cmd.Stdin = bytes.NewReader(source)
  raw, err := cmd.Output()
  if err != nil { return nil, fmt.Errorf("parse %s: Python 3 is required: %w", p, err) }
  var parsedFile parsed
  if err := json.Unmarshal(raw, &parsedFile); err != nil { return nil, fmt.Errorf("parse %s: %w", p, err) }
  rel := p
  if cwd,e := os.Getwd(); e == nil { if r,e := filepath.Rel(cwd,p); e == nil { rel = r } }
  lines := len(strings.Split(string(source), "\n"))
  if len(source)>0 && source[len(source)-1]=='\n' { lines-- }
  out.Files = append(out.Files, &ir.FileIR{Path:rel,Language:"python",Lines:lines})
  emit := func(code string, sev ir.Severity, msg, suggestion, symbol string, start,end int) {
   kind := ir.NodeKindFunction
   if symbol == "" { kind = ir.NodeKindFile }
   out.Diagnostics = append(out.Diagnostics, ir.Diagnostic{Code:"py."+code,Severity:sev,Confidence:"high",Message:msg,File:rel,Line:start,EndLine:end,Symbol:symbol,Kind:kind,Suggestion:suggestion,Span:&ir.Span{StartLine:start,EndLine:end}})
  }
  if parsedFile.Error != "" { emit("syntax-error",ir.SeverityHigh,parsedFile.Error,"Fix Python syntax before refactoring.","",1,lines); continue }
  if lines>cfg.MaxFileLines { emit("large-file",ir.SeverityMedium,fmt.Sprintf("File has %d lines (limit: %d)",lines,cfg.MaxFileLines),"Split independent responsibilities into modules.","",1,lines) }
  for _, fn := range parsedFile.Functions {
   span:=fn.End-fn.Start+1
   if span>cfg.MaxFunctionLines { emit("long-function",ir.SeverityMedium,fmt.Sprintf("Function %s has %d lines (limit: %d)",fn.Name,span,cfg.MaxFunctionLines),"Extract cohesive helpers.",fn.Name,fn.Start,fn.End) }
   if fn.Complexity>cfg.MaxComplexity { emit("high-complexity",ir.SeverityHigh,fmt.Sprintf("Function %s has complexity %d (limit: %d)",fn.Name,fn.Complexity,cfg.MaxComplexity),"Reduce decision branches and extract helpers.",fn.Name,fn.Start,fn.End) }
   if fn.Nesting>cfg.MaxNesting { emit("deep-nesting",ir.SeverityMedium,fmt.Sprintf("Function %s has nesting %d (limit: %d)",fn.Name,fn.Nesting,cfg.MaxNesting),"Use guard clauses and smaller helpers.",fn.Name,fn.Start,fn.End) }
   if fn.Params>cfg.MaxParams { emit("too-many-params",ir.SeverityLow,fmt.Sprintf("Function %s has %d parameters (limit: %d)",fn.Name,fn.Params,cfg.MaxParams),"Group related arguments where appropriate.",fn.Name,fn.Start,fn.End) }
  }
 }
 sort.Slice(out.Diagnostics,func(i,j int)bool { a,b:=out.Diagnostics[i],out.Diagnostics[j]; if a.File!=b.File {return a.File<b.File}; if a.Line!=b.Line{return a.Line<b.Line};return a.Code<b.Code})
 return out,nil
}
