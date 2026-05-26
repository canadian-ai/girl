package parsertsx

import (
	"regexp"
	"strings"

	"github.com/canadian-ai/girl/internal/node"
)

var hookCallRe = regexp.MustCompile(`(use\w+)(?:<[^>]+>)?\s*\(`)
var hookKnown = map[string]bool{
	"useState": true, "useEffect": true, "useCallback": true,
	"useMemo": true, "useRef": true, "useContext": true,
	"useReducer": true, "useLayoutEffect": true,
	"useForm": true, "useController": true, "useFieldArray": true,
	"useWatch": true, "useNavigate": true, "useParams": true,
	"useSearchParams": true, "useRouter": true, "useDisclosure": true,
	"useQuery": true, "useMutation": true, "useKeyDown": true,
	"useToast": true, "useTheme": true, "useAuth": true, "useUser": true,
}

var stateRe = regexp.MustCompile(`\[(\w+),\s*(\w+)\]\s*=\s*useState(?:<[^>]+>)?\s*\(`)
var effectRe = regexp.MustCompile(`useEffect(?:<[^>]+>)?\s*\(`)
var jsxElementRe = regexp.MustCompile(`<(\w+)`)
var eventHandlerRe = regexp.MustCompile(`(?:const\s+)?(handle\w+|on\w+)\s*(?:[=:])`)
var ifRe = regexp.MustCompile(`(?m)^\s*if\s*\(`)
var ternaryRe = regexp.MustCompile(`\?\s*[^:?\n]+:`)
var loopRe = regexp.MustCompile(`(?:\.map\s*\(|\.forEach\s*\(|\.filter\s*\(|for\s*\()`)
var identRefRe = regexp.MustCompile(`\b([a-zA-Z_$][a-zA-Z0-9_$]*)\b`)

func isComponentName(name string) bool {
	if len(name) == 0 {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z' && strings.ToUpper(name) != name
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return s
}

func prevNonEmptyLine(lines []string, line int) int {
	for i := line - 2; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return i + 1
		}
	}
	return 0
}

func findFunctionBody(content string, start int) int {
	depth := 0
	inJSX := false
	jsxExprDepth := 0
	foundBrace := false
	for i := start; i < len(content); i++ {
		ch := content[i]
		if ch == '\'' || ch == '"' || ch == '`' {
			i = skipQuotedContent(content, i, ch)
			continue
		}
		if next, ok := skipCommentContent(content, i); ok {
			i = next
			continue
		}
		inJSX, jsxExprDepth = updateJSXScanState(content, i, inJSX, jsxExprDepth)
		if ch != '{' && ch != '}' {
			continue
		}
		depth, jsxExprDepth, foundBrace = updateFunctionBraceDepth(ch, depth, jsxExprDepth, inJSX, foundBrace)
		if foundBrace && depth == 0 {
			return i + 1
		}
	}
	return len(content)
}

func skipQuotedContent(content string, start int, quote byte) int {
	for i := start + 1; i < len(content); i++ {
		if content[i] == '\\' {
			i++
			continue
		}
		if content[i] == quote {
			return i
		}
	}
	return len(content) - 1
}

func skipCommentContent(content string, start int) (int, bool) {
	if start+1 >= len(content) || content[start] != '/' {
		return start, false
	}
	if content[start+1] == '/' {
		for start < len(content) && content[start] != '\n' {
			start++
		}
		return start, true
	}
	if content[start+1] == '*' {
		start += 2
		for start+1 < len(content) && !(content[start] == '*' && content[start+1] == '/') {
			start++
		}
		return start, true
	}
	return start, false
}

func updateJSXScanState(content string, pos int, inJSX bool, exprDepth int) (bool, int) {
	if startsJSXScan(content, pos, inJSX) {
		return true, 0
	}
	if endsJSXScan(content, pos, inJSX, exprDepth) {
		return false, exprDepth
	}
	return inJSX, exprDepth
}

func startsJSXScan(content string, pos int, inJSX bool) bool {
	return content[pos] == '<' && !inJSX && pos+1 < len(content) && isIdentStart(rune(content[pos+1]))
}

func endsJSXScan(content string, pos int, inJSX bool, exprDepth int) bool {
	return content[pos] == '>' && inJSX && exprDepth == 0 && pos > 0 && content[pos-1] != '?' && content[pos-1] != ':'
}

func updateFunctionBraceDepth(ch byte, depth, jsxExprDepth int, inJSX, foundBrace bool) (int, int, bool) {
	if ch == '{' {
		if inJSX {
			jsxExprDepth++
		}
		return depth + 1, jsxExprDepth, true
	}
	depth--
	if inJSX && jsxExprDepth > 0 {
		jsxExprDepth--
	}
	return depth, jsxExprDepth, foundBrace
}

func looksLikeComponentInitializer(tz *tokenizer) bool {
	tz.skipWhitespace()
	if tz.peek() != '=' {
		return false
	}
	tz.pos++
	tz.skipWhitespace()

	remaining := string(tz.src[tz.pos:])
	lineEnd := strings.IndexByte(remaining, '\n')
	firstLine := remaining
	if lineEnd >= 0 {
		firstLine = remaining[:lineEnd]
	}
	trimmed := strings.TrimSpace(firstLine)
	if trimmed == "" {
		return false
	}
	if hasAnyPrefix(trimmed, []string{"function", "async", "memo(", "React.memo(", "forwardRef(", "React.forwardRef("}) {
		return true
	}
	return strings.Contains(firstLine, "=>")
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func countLinesBefore(s string, pos int) int {
	return strings.Count(s[:pos], "\n") + 1
}

func findStartOfLine(s string, pos int) int {
	for pos > 0 && s[pos-1] != '\n' {
		pos--
	}
	return pos
}

func countLines(s string, start, end int) int {
	return strings.Count(s[start:end], "\n")
}

func skipTypeAnnotation(tz *tokenizer) {
	depth := 0
	for tz.pos < len(tz.src) {
		ch := tz.peek()
		if (ch == '=' || ch == '\n') && depth == 0 {
			return
		}
		var ok bool
		depth, ok = updateTypeAnnotationDepth(ch, depth)
		if !ok {
			return
		}
		tz.pos++
		if ch == '\n' {
			tz.line++
			tz.col = 1
		}
	}
}

func updateTypeAnnotationDepth(ch rune, depth int) (int, bool) {
	switch ch {
	case '<':
		return depth + 1, true
	case '>':
		if depth == 0 {
			return depth, false
		}
		return depth - 1, true
	case '{':
		return depth + 10, true
	case '}':
		if depth >= 10 {
			return depth - 10, true
		}
	}
	return depth, true
}

func buildComponentFromBody(g *node.NodeGraph, path, content string, lines []string, name string) string {
	idx, bodyEnd, ok := findComponentBodyRange(content, name)
	if !ok {
		return ""
	}
	body := content[idx:bodyEnd]

	compNode := newComponentGraphNode(g, path, content, name, idx, bodyEnd)
	g.AddNode(compNode)
	g.AddSymbol(name, compNode.ID())

	children := collectComponentChildren(g, path, body, name, compNode)
	g.SetChildren(compNode.ID(), children)
	return string(compNode.ID())
}

func findComponentBodyRange(content, name string) (int, int, bool) {
	idx, ok := findComponentSearchIndex(content, name)
	if !ok {
		return 0, 0, false
	}
	bodyStart := findComponentBodyStart(content, idx)
	if bodyStart < 0 || bodyStart >= len(content) {
		return 0, 0, false
	}
	return idx, findFunctionBody(content, bodyStart), true
}

func findComponentSearchIndex(content, name string) (int, bool) {
	patterns := []string{
		"export default function " + name,
		"export function " + name,
		"export const " + name,
		"function " + name,
		"const " + name,
		"var " + name,
		"let " + name,
	}
	for _, pat := range patterns {
		idx := strings.Index(content, pat)
		if idx >= 0 {
			return idx + len(pat), true
		}
	}
	idx := strings.Index(content, name)
	if idx < 0 || !isComponentName(name) {
		return 0, false
	}
	return idx + len(name), true
}

func findComponentBodyStart(content string, fnStart int) int {
	bodyStart := strings.Index(content[fnStart:], "=>")
	if bodyStart >= 0 {
		return bodyStart + fnStart + 2
	}
	parenIdx := strings.Index(content[fnStart:], "(")
	if parenIdx < 0 {
		return -1
	}
	return findBodyStartAfterParams(content, fnStart+parenIdx+1)
}

func findBodyStartAfterParams(content string, start int) int {
	depth := 1
	for j := start; j < len(content) && depth > 0; j++ {
		switch content[j] {
		case '(', ')':
			depth = updateParenDepth(content[j], depth)
			if depth == 0 {
				return findNextByte(content, j, '{')
			}
		case '\'', '"':
			j = skipQuotedContent(content, j, content[j])
		case '/':
			if next, ok := skipCommentContent(content, j); ok {
				j = next
			}
		}
	}
	return -1
}

func findNextByte(content string, start int, target byte) int {
	idx := strings.IndexByte(content[start:], target)
	if idx < 0 {
		return len(content)
	}
	return start + idx
}

func updateParenDepth(ch byte, depth int) int {
	if ch == '(' {
		return depth + 1
	}
	return depth - 1
}

func newComponentGraphNode(g *node.NodeGraph, path, content, name string, idx, bodyEnd int) *node.ComponentNode {
	startLine := countLinesBefore(content, idx)
	endLine := startLine + countLines(content, idx, bodyEnd)
	compNode := node.NewComponentNode(g.NextID("comp"), name)
	compNode.Lines = endLine - startLine + 1
	compNode.SetFile(path)
	compNode.IsExport = true
	return compNode

}

func collectComponentChildren(g *node.NodeGraph, path, body, name string, compNode *node.ComponentNode) []node.NodeID {
	var children []node.NodeID
	children = append(children, addHookNodes(g, path, body, compNode)...)
	children = append(children, addStateNodes(g, path, body, compNode)...)
	children = append(children, addEffectNodes(g, path, body, compNode)...)
	children = append(children, addJSXNodes(g, path, body, name)...)
	children = append(children, addEventNodes(g, path, body, compNode)...)
	children = append(children, addReferenceNodes(g, path, body, name)...)
	children = append(children, addConditionalNodes(g, path, body)...)
	children = append(children, addLoopNodes(g, path, body)...)
	return children
}

func addHookNodes(g *node.NodeGraph, path, body string, compNode *node.ComponentNode) []node.NodeID {
	var children []node.NodeID
	hookMatches := hookCallRe.FindAllStringSubmatch(body, -1)
	seenHooks := map[string]int{}
	for _, m := range hookMatches {
		hName := m[1]
		if !hookKnown[hName] && !strings.HasPrefix(hName, "use") {
			continue
		}
		seenHooks[hName]++
		hookNode := node.NewHookNode(g.NextID("hook"), hName)
		hookNode.SetFile(path)
		g.AddNode(hookNode)
		compNode.Hooks = append(compNode.Hooks, hookNode.ID())
		children = append(children, hookNode.ID())

	}
	return children
}

func addStateNodes(g *node.NodeGraph, path, body string, compNode *node.ComponentNode) []node.NodeID {
	var children []node.NodeID
	stateMatches := stateRe.FindAllStringSubmatch(body, -1)
	seenStates := map[string]bool{}
	for _, m := range stateMatches {
		if len(m) < 2 || seenStates[m[1]] {
			continue
		}
		seenStates[m[1]] = true
		stateNode := node.NewStateNode(g.NextID("state"), m[1])
		stateNode.SetFile(path)
		g.AddNode(stateNode)
		compNode.StateVars = append(compNode.StateVars, stateNode.ID())
		children = append(children, stateNode.ID())
	}
	return children
}

func addEffectNodes(g *node.NodeGraph, path, body string, compNode *node.ComponentNode) []node.NodeID {
	var children []node.NodeID
	effectMatches := effectRe.FindAllString(body, -1)
	for range effectMatches {
		effectNode := node.NewEffectNode(g.NextID("effect"))
		effectNode.SetFile(path)
		g.AddNode(effectNode)
		compNode.Effects = append(compNode.Effects, effectNode.ID())
		children = append(children, effectNode.ID())
	}
	return children
}

func addJSXNodes(g *node.NodeGraph, path, body, name string) []node.NodeID {
	var children []node.NodeID
	jsxMatches := jsxElementRe.FindAllStringSubmatch(body, -1)
	seenJSX := map[string]int{}
	for _, m := range jsxMatches {
		elem := m[1]
		if elem == name {
			continue
		}
		if seenJSX[elem] > 5 {
			continue
		}
		seenJSX[elem]++
		jsxNode := node.NewJSXNode(g.NextID("jsx"), elem)
		jsxNode.SetFile(path)
		jsxNode.Depth = 1
		if isComponentName(elem) {
			jsxNode.IsComponent = true
		}
		g.AddNode(jsxNode)
		children = append(children, jsxNode.ID())
	}
	return children
}

func addEventNodes(g *node.NodeGraph, path, body string, compNode *node.ComponentNode) []node.NodeID {
	var children []node.NodeID
	handlerMatches := eventHandlerRe.FindAllStringSubmatch(body, -1)
	for _, m := range handlerMatches {
		eventNode := node.NewEventNode(g.NextID("event"), m[1])
		eventNode.SetFile(path)
		g.AddNode(eventNode)
		compNode.Events = append(compNode.Events, eventNode.ID())
		children = append(children, eventNode.ID())
	}
	return children
}

func addReferenceNodes(g *node.NodeGraph, path, body, name string) []node.NodeID {
	var children []node.NodeID
	idents := identRefRe.FindAllStringSubmatch(body, -1)
	seenRefs := map[string]int{}
	for _, m := range idents {
		refName := m[1]
		if refName == name || refName == "React" || refName == "_" {
			continue
		}
		if seenRefs[refName] > 3 {
			continue
		}
		seenRefs[refName]++
		refNode := node.NewReferenceNode(g.NextID("ref"), refName, "", node.UsageRead)
		refNode.SetFile(path)
		g.AddNode(refNode)
		children = append(children, refNode.ID())
	}
	return children
}

func addConditionalNodes(g *node.NodeGraph, path, body string) []node.NodeID {
	ifCount := len(ifRe.FindAllString(body, -1))
	ternaryCount := len(ternaryRe.FindAllString(body, -1))
	condCount := ifCount + ternaryCount
	if condCount > 0 {
		condNode := node.NewConditionalNode(g.NextID("cond"))
		condNode.SetFile(path)
		g.AddNode(condNode)
		return []node.NodeID{condNode.ID()}
	}
	return nil
}

func addLoopNodes(g *node.NodeGraph, path, body string) []node.NodeID {
	var children []node.NodeID
	loopMatches := loopRe.FindAllString(body, -1)
	for range loopMatches {
		loopNode := node.NewLoopNode(g.NextID("loop"), "map")
		loopNode.SetFile(path)
		g.AddNode(loopNode)
		children = append(children, loopNode.ID())
	}
	return children
}
