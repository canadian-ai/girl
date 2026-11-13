// Package jsonmigration provides a conservative, compatibility-preserving JSON
// API migration. It never enables JSON v2 behavior changes automatically.
package jsonmigration

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
	"strconv"
)

type Result struct {
	Source  []byte
	Changed bool
	Review  string
}

type edit struct {
	start, end int
	text       string
}

// Migrate rewrites only direct Marshal and Unmarshal calls proven by go/types
// to reference encoding/json. A file containing other uses is left untouched.
// Type errors elsewhere do not authorize edits: every JSON reference must still
// resolve to the imported package. This also works with unbuilt local imports.
func Migrate(filename string, source []byte) (Result, error) {
	result := Result{Source: source}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, source, parser.ParseComments)
	if err != nil {
		return result, err
	}
	var target *ast.ImportSpec
	var decl *ast.GenDecl
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.IMPORT {
			continue
		}
		for _, s := range g.Specs {
			i := s.(*ast.ImportSpec)
			p, _ := strconv.Unquote(i.Path.Value)
			if p == "encoding/json/v2" {
				result.Review = "already imports JSON v2; review mixed API usage manually"
				return result, nil
			}
			if p == "encoding/json" {
				target, decl = i, g
			}
		}
	}
	if target == nil {
		return result, nil
	}
	name := "json"
	if target.Name != nil {
		name = target.Name.Name
	}
	if name == "." || name == "_" {
		result.Review = "dot or blank JSON import requires manual review"
		return result, nil
	}
	info := &types.Info{Uses: make(map[*ast.Ident]types.Object), Defs: make(map[*ast.Ident]types.Object), Implicits: make(map[ast.Node]types.Object)}
	config := types.Config{Importer: importer.Default(), Error: func(error) {}}
	_, _ = config.Check(f.Name.Name, fset, []*ast.File{f}, info)
	var object types.Object
	if target.Name != nil {
		object = info.Defs[target.Name]
	} else {
		object = info.Implicits[target]
	}
	pkg, ok := object.(*types.PkgName)
	if !ok || pkg.Imported().Path() != "encoding/json" {
		result.Review = "JSON import could not be resolved"
		return result, nil
	}
	parents := make(map[ast.Node]ast.Node)
	var stack []ast.Node
	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) > 0 {
			parents[n] = stack[len(stack)-1]
		}
		stack = append(stack, n)
		return true
	})
	var calls []*ast.CallExpr
	usedNames := make(map[string]bool)
	ast.Inspect(f, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		usedNames[id.Name] = true
		if info.Uses[id] != pkg {
			return true
		}
		sel, ok := parents[id].(*ast.SelectorExpr)
		if !ok || sel.X != id {
			result.Review = "unsupported JSON package reference"
			return true
		}
		call, ok := parents[sel].(*ast.CallExpr)
		argc := 0
		switch sel.Sel.Name {
		case "Marshal":
			argc = 1
		case "Unmarshal":
			argc = 2
		}
		if !ok || call.Fun != sel || argc == 0 || len(call.Args) != argc || call.Ellipsis.IsValid() {
			result.Review = "JSON " + sel.Sel.Name + " requires manual review (only direct Marshal/Unmarshal calls are migrated)"
			return true
		}
		calls = append(calls, call)
		return true
	})
	if result.Review != "" {
		return result, nil
	}
	if len(calls) == 0 {
		result.Review = "no proven direct JSON calls"
		return result, nil
	}
	legacy := "jsonv1"
	for i := 2; usedNames[legacy]; i++ {
		legacy = fmt.Sprintf("jsonv1%d", i)
	}
	offset := func(p token.Pos) int { return fset.Position(p).Offset }
	edits := []edit{{offset(target.Path.Pos()), offset(target.Path.End()), `"encoding/json/v2"`}}
	if target.Name == nil {
		edits = append(edits, edit{offset(target.Path.Pos()), offset(target.Path.Pos()), name + " "})
	}
	// Separate declarations also work for an original single-line import and
	// retain comments attached to that import. gofmt normalizes the result.
	edits = append(edits, edit{offset(decl.End()), offset(decl.End()), "\nimport " + legacy + ` "encoding/json"` + "\n"})
	for _, call := range calls {
		edits = append(edits, edit{offset(call.Args[len(call.Args)-1].End()), offset(call.Args[len(call.Args)-1].End()), ", " + legacy + ".DefaultOptionsV1()"})
	}
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := string(source)
	for _, e := range edits {
		out = out[:e.start] + e.text + out[e.end:]
	}
	formatted, err := format.Source([]byte(out))
	if err != nil {
		return result, fmt.Errorf("format migration: %w", err)
	}
	result.Source = formatted
	result.Changed = !bytes.Equal(source, formatted)
	return result, nil
}
