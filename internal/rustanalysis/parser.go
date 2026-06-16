package rustanalysis

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/rust"
)

func ParseRustFile(path string) (*RustFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	lines := strings.Split(content, "\n")

	lang := rust.GetLanguage()
	p := sitter.NewParser()
	p.SetLanguage(lang)

	tree, err := p.ParseCtx(context.Background(), nil, data)
	if err != nil {
		return nil, fmt.Errorf("parse error in %s: %w", path, err)
	}
	root := tree.RootNode()

	rf := &RustFile{
		Path:    path,
		Package: detectCrateName(path),
		Lines:   len(lines),
	}

	// Find all function items and impl blocks
	implReceivers := findImplReceivers(root, data)

	rf.Functions = extractFunctions(root, data, implReceivers)

	return rf, nil
}

func detectCrateName(path string) string {
	dir := filepath.Dir(path)
	for {
		cargoPath := filepath.Join(dir, "Cargo.toml")
		if data, err := os.ReadFile(cargoPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "name") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						return strings.Trim(strings.TrimSpace(parts[1]), "\"")
					}
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// implReceiverMap maps function names to their impl receiver type
type implReceiverMap map[string]string

func findImplReceivers(root *sitter.Node, data []byte) implReceiverMap {
	receivers := implReceiverMap{}

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "impl_item" {
			// Find the type being implemented
			var typeName string
			for i := 0; i < int(n.NamedChildCount()); i++ {
				child := n.NamedChild(i)
				if child.Type() == "type_identifier" {
					typeName = string(data[child.StartByte():child.EndByte()])
					break
				}
				// Also check for generic types
				if child.Type() == "generic_type" {
					for j := 0; j < int(child.NamedChildCount()); j++ {
						gc := child.NamedChild(j)
						if gc.Type() == "type_identifier" {
							typeName = string(data[gc.StartByte():gc.EndByte()])
							break
						}
					}
					break
				}
			}
			// Find all functions inside this impl block
			for i := 0; i < int(n.NamedChildCount()); i++ {
				child := n.NamedChild(i)
				if child.Type() == "declaration_list" {
					for j := 0; j < int(child.NamedChildCount()); j++ {
						fn := child.NamedChild(j)
						if fn.Type() == "function_item" {
							nameNode := fn.ChildByFieldName("name")
							if nameNode != nil {
								fnName := string(data[nameNode.StartByte():nameNode.EndByte()])
								receivers[fnName] = typeName
							}
						}
					}
				}
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(root)
	return receivers
}

func extractFunctions(root *sitter.Node, data []byte, receivers implReceiverMap) []RustFunction {
	var functions []RustFunction

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n.Type() == "function_item" {
			fn := parseFunctionNode(n, data, receivers)
			if fn.Name != "" {
				functions = append(functions, fn)
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(root)
	return functions
}

func parseFunctionNode(n *sitter.Node, data []byte, receivers implReceiverMap) RustFunction {
	nameNode := n.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = string(data[nameNode.StartByte():nameNode.EndByte()])
	}

	startLine := int(n.StartPoint().Row) + 1
	endLine := int(n.EndPoint().Row) + 1

	fn := RustFunction{
		Name:      name,
		StartLine: startLine,
		EndLine:   endLine,
		Lines:     endLine - startLine + 1,
		Receiver:  receivers[name],
	}

	// Check for modifiers via child nodes
	// tree-sitter Rust includes pub/async/unsafe as part of the function_item node
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(i)
		switch child.Type() {
		case "visibility_modifier":
			fn.IsPub = true
		case "async_clause", "async":
			fn.IsAsync = true
		case "unsafe_clause", "unsafe":
			fn.IsUnsafe = true
		}
	}

	// Parameters
	paramsNode := n.ChildByFieldName("parameters")
	if paramsNode != nil {
		count := 0
		for i := 0; i < int(paramsNode.NamedChildCount()); i++ {
			param := paramsNode.NamedChild(i)
			if param.Type() == "parameter" || param.Type() == "self_parameter" ||
				param.Type() == "variadic_parameter" {
				count++
			}
		}
		fn.Params = count
	}

	// Return type
	returnTypeNode := n.ChildByFieldName("return_type")
	if returnTypeNode != nil {
		fn.Returns = 1
	}

	// Body
	bodyNode := n.ChildByFieldName("body")
	if bodyNode != nil {
		fn.Complexity = computeComplexity(bodyNode, data)
		fn.MaxNesting = computeNesting(bodyNode, 0)
	} else {
		fn.Complexity = 1
		fn.MaxNesting = 0
	}

	return fn
}

func computeComplexity(node *sitter.Node, data []byte) int {
	complexity := 1

	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		switch n.Type() {
		case "if_expression", "if_let_expression":
			complexity++
		case "match_expression":
			// Count match arms as branches
			for i := 0; i < int(n.NamedChildCount()); i++ {
				child := n.NamedChild(i)
				if child.Type() == "match_block" {
					for j := 0; j < int(child.NamedChildCount()); j++ {
						if child.NamedChild(j).Type() == "match_arm" {
							complexity++
						}
					}
				}
			}
		case "while_expression", "while_let_expression":
			complexity++
		case "for_expression":
			complexity++
		case "loop_expression":
			complexity++
		case "binary_expression":
			op := ""
			for i := 0; i < int(n.NamedChildCount()); i++ {
				child := n.NamedChild(i)
				ct := child.Type()
				if ct != "identifier" && ct != "call_expression" &&
					ct != "field_expression" && ct != "integer_literal" &&
					ct != "float_literal" && ct != "string_literal" &&
					ct != "boolean_literal" && ct != "parenthesized_expression" &&
					ct != "index_expression" && ct != "reference_expression" &&
					ct != "unary_expression" && ct != "binary_expression" {
					op = ct
				}
			}
			if op == "&&" || op == "||" {
				complexity++
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(node)
	return complexity
}

func computeNesting(node *sitter.Node, depth int) int {
	maxDepth := depth

	var walk func(n *sitter.Node, d int)
	walk = func(n *sitter.Node, d int) {
		if d > maxDepth {
			maxDepth = d
		}
		switch n.Type() {
		case "if_expression", "if_let_expression", "match_expression",
			"while_expression", "while_let_expression", "for_expression",
			"loop_expression", "block":
			for i := 0; i < int(n.NamedChildCount()); i++ {
				walk(n.NamedChild(i), d+1)
			}
			return
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i), d)
		}
	}
	walk(node, depth)
	return maxDepth
}

func IsRustFile(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".rs"
}

func collectRustFiles(path string) ([]*RustFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path %s: %w", path, err)
	}

	if !info.IsDir() {
		gf, err := ParseRustFile(path)
		if err != nil {
			return nil, err
		}
		return []*RustFile{gf}, nil
	}

	return walkRustFiles(path)
}

func walkRustFiles(path string) ([]*RustFile, error) {
	var files []*RustFile
	err := filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			base := filepath.Base(p)
			if base != "." && shouldSkipDir(base) {
				return filepath.SkipDir
			}
			return nil
		}
		if IsRustFile(p) {
			rf, parseErr := ParseRustFile(p)
			if parseErr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to parse %s: %v\n", p, parseErr)
			} else {
				files = append(files, rf)
			}
		}
		return nil
	})
	return files, err
}

func shouldSkipDir(name string) bool {
	switch name {
	case "target", "node_modules", ".git", "dist", "build":
		return true
	}
	return strings.HasPrefix(name, ".")
}
