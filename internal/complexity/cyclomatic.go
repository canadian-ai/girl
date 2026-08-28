package complexity

import sitter "github.com/smacker/go-tree-sitter"

// CyclomaticForFunction applies a source-level McCabe count compatible with
// modern JavaScript/TypeScript control flow. It starts at one and excludes
// nested functions, which are measured independently.
func CyclomaticForFunction(function *sitter.Node, data []byte) int {
	value := 1
	walkComplexity(function, function, data, &value)
	return value
}

// CyclomaticForNodes is used when a caller already has separate parameter and
// body nodes, as the React component parser does.
func CyclomaticForNodes(params, body *sitter.Node, data []byte) int {
	value := 1
	if params != nil {
		walkComplexity(params, params, data, &value)
	}
	if body != nil {
		walkComplexity(body, body, data, &value)
	}
	return value
}

func walkComplexity(node, root *sitter.Node, data []byte, value *int) {
	if node != root && isFunctionNode(node) {
		return
	}
	*value += decisionIncrement(node, data)
	for i := 0; i < int(node.NamedChildCount()); i++ {
		walkComplexity(node.NamedChild(i), root, data, value)
	}
}

func decisionIncrement(node *sitter.Node, data []byte) int {
	switch node.Type() {
	case "if_statement", "switch_case", "for_statement", "for_in_statement",
		"while_statement", "do_statement", "catch_clause", "ternary_expression":
		return 1
	case "assignment_pattern":
		return 1
	case "required_parameter", "optional_parameter":
		if node.ChildByFieldName("value") != nil {
			return 1
		}
	case "binary_expression", "logical_expression", "augmented_assignment_expression":
		if hasLogicalOperator(node) {
			return 1
		}
	case "optional_chain":
		return 1
	}
	return 0
}

func hasLogicalOperator(node *sitter.Node) bool {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		op := child.Type()
		if op == "&&" || op == "||" || op == "??" || op == "&&=" || op == "||=" || op == "??=" {
			return true
		}
	}
	return false
}
