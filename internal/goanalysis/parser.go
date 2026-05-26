package goanalysis

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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

	gf := &GoFile{
		Path:    path,
		Package: f.Name.Name,
		Lines:   len(lines),
	}

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			start := fset.Position(d.Pos()).Line
			end := fset.Position(d.End()).Line
			if end == 0 {
				end = start
			}

			recv := ""
			if d.Recv != nil && len(d.Recv.List) > 0 {
				exprStr := typeExprString(d.Recv.List[0].Type)
				if len(d.Recv.List[0].Names) > 0 {
					recv = fmt.Sprintf("(%s %s)", d.Recv.List[0].Names[0].Name, exprStr)
				} else {
					recv = fmt.Sprintf("(%s)", exprStr)
				}
			}

			fn := GoFunction{
				Name:       d.Name.Name,
				Receiver:   recv,
				StartLine:  start,
				EndLine:    end,
				Lines:      end - start + 1,
				Params:     len(d.Type.Params.List),
				MaxNesting: computeNesting(d.Body),
			}

			if d.Type.Results != nil {
				fn.Returns = len(d.Type.Results.List)
			}

			fn.Complexity = computeComplexity(d)
			fn.IgnoredErrs = countIgnoredErrors(d)

			if d.Type.Results != nil {
				for _, ret := range d.Type.Results.List {
					if isErrorType(ret.Type) {
						fn.HasErrors = true
						break
					}
				}
			}

			gf.Functions = append(gf.Functions, fn)
		}
	}

	return gf, nil
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
	maxDepth := 0
	var walk func(n ast.Node, depth int)
	walk = func(n ast.Node, depth int) {
		if depth > maxDepth {
			maxDepth = depth
		}
		switch node := n.(type) {
		case *ast.IfStmt:
			walk(node.Body, depth+1)
			if node.Else != nil {
				if blk, ok := node.Else.(*ast.BlockStmt); ok {
					walk(blk, depth+1)
				} else {
					walk(node.Else, depth+1)
				}
			}
		case *ast.ForStmt:
			walk(node.Body, depth+1)
		case *ast.RangeStmt:
			walk(node.Body, depth+1)
		case *ast.SwitchStmt:
			walk(node.Body, depth+1)
		case *ast.TypeSwitchStmt:
			walk(node.Body, depth+1)
		case *ast.SelectStmt:
			walk(node.Body, depth+1)
		case *ast.BlockStmt:
			for _, stmt := range node.List {
				walk(stmt, depth)
			}
		default:
			ast.Inspect(n, func(c ast.Node) bool {
				if c == n {
					return false
				}
				switch c.(type) {
				case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
					walk(c, depth+1)
					return false
				}
				return true
			})
		}
	}
	walk(body, 0)
	return maxDepth
}

func countIgnoredErrors(fn *ast.FuncDecl) int {
	count := 0
	if fn.Body == nil {
		return 0
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for _, lhs := range assign.Lhs {
			if id, ok := lhs.(*ast.Ident); ok && id.Name == "_" {
				for _, rhs := range assign.Rhs {
					if isCallReturningError(rhs) {
						count++
					}
				}
			}
		}
		return true
	})
	return count
}

func isCallReturningError(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	if call.Fun == nil {
		return false
	}
	return true
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
