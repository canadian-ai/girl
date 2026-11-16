package shared

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// FirstSyntaxError returns the 1-based line/column and kind of the first
// ERROR or MISSING node in a tree-sitter tree, or ok=false when the tree is
// clean. Tree-sitter recovers from errors silently, so callers must check this
// before trusting metrics derived from the tree.
func FirstSyntaxError(root *sitter.Node) (line, col int, missing bool, ok bool) {
	if root == nil || !root.HasError() {
		return 0, 0, false, false
	}
	var walk func(n *sitter.Node) bool
	walk = func(n *sitter.Node) bool {
		if n.IsError() || n.IsMissing() {
			p := n.StartPoint()
			line, col, missing = int(p.Row)+1, int(p.Column)+1, n.IsMissing()
			return true
		}
		if !n.HasError() {
			return false
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			if c := n.Child(i); c != nil && walk(c) {
				return true
			}
		}
		return false
	}
	if walk(root) {
		return line, col, missing, true
	}
	// HasError was set but no explicit node was found; report the root.
	return 1, 1, false, true
}

// SyntaxErrorMessage renders the result of FirstSyntaxError.
func SyntaxErrorMessage(missing bool) string {
	if missing {
		return "parser inserted a missing token (incomplete construct)"
	}
	return "unexpected or unparseable tokens"
}
