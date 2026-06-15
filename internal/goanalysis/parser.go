package goanalysis

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
)

func ParseGoFile(path string) (*GoFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	lines := strings.Split(content, "\n")

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse error in %s: %w", path, err)
	}

	conf := types.Config{Importer: importer.Default()}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	var typesInfo *types.Info
	if _, err := conf.Check(f.Name.Name, fset, []*ast.File{f}, info); err != nil {
		typesInfo = nil
	} else {
		typesInfo = info
	}

	gf := &GoFile{
		Path:      path,
		Package:   f.Name.Name,
		Lines:     len(lines),
		typesInfo: typesInfo,
	}

	for _, decl := range f.Decls {
		if d, ok := decl.(*ast.FuncDecl); ok {
			gf.Functions = append(gf.Functions, newGoFunction(fset, d, typesInfo))
		}
	}

	return gf, nil
}

func newGoFunction(fset *token.FileSet, d *ast.FuncDecl, info *types.Info) GoFunction {
	start := fset.Position(d.Pos()).Line
	end := fset.Position(d.End()).Line
	if end == 0 {
		end = start
	}

	ignoredErrs, confidence := countIgnoredErrors(d, info)
	fn := GoFunction{
		Name:                 d.Name.Name,
		Receiver:             receiverName(d),
		StartLine:            start,
		EndLine:              end,
		Lines:                end - start + 1,
		Params:               len(d.Type.Params.List),
		MaxNesting:           computeNesting(d.Body),
		Complexity:           computeComplexity(d),
		IgnoredErrs:          ignoredErrs,
		IgnoredErrConfidence: confidence,
	}
	if d.Type.Results != nil {
		fn.Returns = len(d.Type.Results.List)
		fn.HasErrors = hasErrorResult(d.Type.Results.List)
	}
	return fn
}

func receiverName(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return ""
	}
	recv := d.Recv.List[0]
	exprStr := typeExprString(recv.Type)
	if len(recv.Names) > 0 {
		return fmt.Sprintf("(%s %s)", recv.Names[0].Name, exprStr)
	}
	return fmt.Sprintf("(%s)", exprStr)
}

func hasErrorResult(results []*ast.Field) bool {
	for _, ret := range results {
		if isErrorType(ret.Type) {
			return true
		}
	}
	return false
}

func computeComplexity(fn *ast.FuncDecl) int {
	c := 1
	if fn.Body == nil {
		return c
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt:
			c++
		case *ast.ForStmt:
			c++
		case *ast.RangeStmt:
			c++
		case *ast.CaseClause:
			c++
		case *ast.CommClause:
			c++
		case *ast.BinaryExpr:
			be := n.(*ast.BinaryExpr)
			if be.Op == token.LAND || be.Op == token.LOR {
				c++
			}
		}
		return true
	})
	return c
}

func computeNesting(body *ast.BlockStmt) int {
	if body == nil {
		return 0
	}
	w := &nestingWalker{}
	w.walk(body, 0)
	return w.maxDepth
}

type nestingWalker struct {
	maxDepth int
}

func (w *nestingWalker) walk(n ast.Node, depth int) {
	if depth > w.maxDepth {
		w.maxDepth = depth
	}
	if block, ok := n.(*ast.BlockStmt); ok {
		w.walkBlock(block, depth)
		return
	}
	if w.walkControlNode(n, depth) {
		return
	}
	w.walkNestedControlNodes(n, depth)
}

func (w *nestingWalker) walkBlock(block *ast.BlockStmt, depth int) {
	for _, stmt := range block.List {
		w.walk(stmt, depth)
	}
}

func (w *nestingWalker) walkControlNode(n ast.Node, depth int) bool {
	switch node := n.(type) {
	case *ast.IfStmt:
		w.walk(node.Body, depth+1)
		w.walkIfElse(node.Else, depth+1)
	case *ast.ForStmt:
		w.walk(node.Body, depth+1)
	case *ast.RangeStmt:
		w.walk(node.Body, depth+1)
	case *ast.SwitchStmt:
		w.walk(node.Body, depth+1)
	case *ast.TypeSwitchStmt:
		w.walk(node.Body, depth+1)
	case *ast.SelectStmt:
		w.walk(node.Body, depth+1)
	default:
		return false
	}
	return true
}

func (w *nestingWalker) walkIfElse(node ast.Node, depth int) {
	if node == nil {
		return
	}
	w.walk(node, depth)
}

func (w *nestingWalker) walkNestedControlNodes(n ast.Node, depth int) {
	ast.Inspect(n, func(c ast.Node) bool {
		if c == n {
			return false
		}
		if isNestingControlNode(c) {
			w.walk(c, depth+1)
			return false
		}
		return true
	})
}

func isNestingControlNode(n ast.Node) bool {
	switch n.(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return true
	default:
		return false
	}
}

func countIgnoredErrors(fn *ast.FuncDecl, info *types.Info) (int, string) {
	count := 0
	if fn.Body == nil {
		return 0, "high"
	}
	hasTypeInfo := info != nil

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, lhs := range assign.Lhs {
			if id, ok := lhs.(*ast.Ident); ok && id.Name == "_" {
				for _, rhs := range assign.Rhs {
					if isCallReturningError(rhs, info) {
						count++
					}
				}
			}
		}
		return true
	})

	if count == 0 {
		return 0, "high"
	}
	if hasTypeInfo {
		return count, "high"
	}
	return count, "low"
}

func isCallReturningError(expr ast.Expr, info *types.Info) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	if call.Fun == nil {
		return false
	}
	if info != nil {
		return callReturnsErrorType(call, info)
	}
	return isProbablyErrorReturning(call)
}

func callReturnsErrorType(call *ast.CallExpr, info *types.Info) bool {
	tv, ok := info.Types[call]
	if !ok || tv.Type == nil {
		return false
	}
	if isGoErrorType(tv.Type) {
		return true
	}
	if tuple, ok := tv.Type.(*types.Tuple); ok && tuple.Len() > 0 {
		return isGoErrorType(tuple.At(tuple.Len() - 1).Type())
	}
	return false
}

func isGoErrorType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	return named.Obj().Name() == "error" && named.Obj().Pkg() == nil
}

var likelyErrorFuncs = map[string]bool{
	"Atoi": true, "Open": true, "Create": true, "Read": true, "Write": true,
	"Exec": true, "Stat": true, "Chdir": true, "Mkdir": true, "MkdirAll": true,
	"Remove": true, "RemoveAll": true, "Rename": true, "Chmod": true, "Chown": true,
	"Parse": true, "Dial": true, "Listen": true, "Accept": true,
}

var likelyNonErrorFuncs = map[string]bool{
	"Print": true, "Printf": true, "Println": true,
	"Sprintf": true, "Log": true, "Logf": true, "Logln": true,
}

func isProbablyErrorReturning(call *ast.CallExpr) bool {
	name := callName(call)
	if likelyNonErrorFuncs[name] {
		return false
	}
	if likelyErrorFuncs[name] {
		return true
	}
	return false
}

func callName(call *ast.CallExpr) string {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

func isErrorType(expr ast.Expr) bool {
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name == "error"
	}
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		return isErrorType(sel.Sel)
	}
	return false
}

func typeExprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeExprString(t.X)
	case *ast.SelectorExpr:
		return typeExprString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + typeExprString(t.Elt)
	case *ast.MapType:
		return "map[" + typeExprString(t.Key) + "]" + typeExprString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func IsGoFile(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}
