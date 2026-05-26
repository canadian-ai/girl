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
		if ch == '\'' || ch == '"' {
			quote := ch
			i++
			for i < len(content) && content[i] != quote {
				if content[i] == '\\' {
					i++
				}
				i++
			}
			continue
		}
		if ch == '`' {
			i++
			for i < len(content) && content[i] != '`' {
				if content[i] == '\\' {
					i++
				}
				i++
			}
			continue
		}
		if ch == '/' && i+1 < len(content) {
			if content[i+1] == '/' {
				for i < len(content) && content[i] != '\n' {
					i++
				}
				continue
			}
			if content[i+1] == '*' {
				i += 2
				for i+1 < len(content) && !(content[i] == '*' && content[i+1] == '/') {
					i++
				}
				continue
			}
		}
		if ch == '<' {
			if !inJSX && i+1 < len(content) && isIdentStart(rune(content[i+1])) {
				inJSX = true
				jsxExprDepth = 0
			}
		}
		if ch == '>' && inJSX && jsxExprDepth == 0 {
			if content[i-1] != '?' && content[i-1] != ':' {
				inJSX = false
			}
		}
		if ch == '{' {
			if inJSX {
				jsxExprDepth++
			}
			depth++
			foundBrace = true
		}
		if ch == '}' {
			depth--
			if inJSX && jsxExprDepth > 0 {
				jsxExprDepth--
			}
			if foundBrace && depth == 0 {
				return i + 1
			}
		}
	}
	return len(content)
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
	if strings.HasPrefix(trimmed, "function") || strings.HasPrefix(trimmed, "async") || strings.HasPrefix(trimmed, "memo(") || strings.HasPrefix(trimmed, "React.memo(") || strings.HasPrefix(trimmed, "forwardRef(") || strings.HasPrefix(trimmed, "React.forwardRef(") {
		return true
	}
	if strings.Contains(firstLine, "=>") {
		return true
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
		if ch == '=' || ch == '\n' {
			if depth == 0 {
				return
			}
		}
		if ch == '<' {
			depth++
		}
		if ch == '>' {
			if depth > 0 {
				depth--
			} else {
				return
			}
		}
		if ch == '{' {
			depth += 10
		}
		if ch == '}' {
			if depth >= 10 {
				depth -= 10
			}
		}
		tz.pos++
		if ch == '\n' {
			tz.line++
			tz.col = 1
		}
	}
}

func buildComponentFromBody(g *node.NodeGraph, path, content string, lines []string, name string) string {
	patterns := []string{
		"export default function " + name,
		"export function " + name,
		"export const " + name,
		"function " + name,
		"const " + name,
		"var " + name,
		"let " + name,
	}

	var idx int
	found := false
	for _, pat := range patterns {
		idx = strings.Index(content, pat)
		if idx >= 0 {
			found = true
			idx += len(pat)
			break
		}
	}
	if !found {
		idx = strings.Index(content, name)
		if idx < 0 || !isComponentName(name) {
			return ""
		}
		idx += len(name)
	}

	fnStart := idx
	bodyStart := strings.Index(content[fnStart:], "=>")
	if bodyStart >= 0 {
		bodyStart += fnStart + 2
	} else {
		parenIdx := strings.Index(content[fnStart:], "(")
		if parenIdx < 0 {
			return ""
		}
		depth := 1
		j := fnStart + parenIdx + 1
		for j < len(content) && depth > 0 {
			switch content[j] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					for j < len(content) && content[j] != '{' {
						j++
					}
					bodyStart = j
				}
			case '\'':
				j++
				for j < len(content) && content[j] != '\'' {
					j++
				}
			case '"':
				j++
				for j < len(content) && content[j] != '"' {
					j++
				}
			case '/':
				if j+1 < len(content) && content[j+1] == '/' {
					for j < len(content) && content[j] != '\n' {
						j++
					}
				}
			}
			j++
		}
	}
	if bodyStart < 0 || bodyStart >= len(content) {
		return ""
	}

	bodyEnd := findFunctionBody(content, bodyStart)
	body := content[idx:bodyEnd]

	startLine := countLinesBefore(content, idx)
	endLine := startLine + countLines(content, idx, bodyEnd)
	compLines := endLine - startLine + 1

	compNode := node.NewComponentNode(g.NextID("comp"), name)
	compNode.Lines = compLines
	compNode.SetFile(path)
	compNode.IsExport = true
	g.AddNode(compNode)
	g.AddSymbol(name, compNode.ID())

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

	effectMatches := effectRe.FindAllString(body, -1)
	for range effectMatches {
		effectNode := node.NewEffectNode(g.NextID("effect"))
		effectNode.SetFile(path)
		g.AddNode(effectNode)
		compNode.Effects = append(compNode.Effects, effectNode.ID())
		children = append(children, effectNode.ID())
	}

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

	handlerMatches := eventHandlerRe.FindAllStringSubmatch(body, -1)
	for _, m := range handlerMatches {
		eventNode := node.NewEventNode(g.NextID("event"), m[1])
		eventNode.SetFile(path)
		g.AddNode(eventNode)
		compNode.Events = append(compNode.Events, eventNode.ID())
		children = append(children, eventNode.ID())
	}

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

	ifCount := len(ifRe.FindAllString(body, -1))
	ternaryCount := len(ternaryRe.FindAllString(body, -1))
	condCount := ifCount + ternaryCount
	if condCount > 0 {
		condNode := node.NewConditionalNode(g.NextID("cond"))
		condNode.SetFile(path)
		g.AddNode(condNode)
		children = append(children, condNode.ID())
	}

	loopMatches := loopRe.FindAllString(body, -1)
	for range loopMatches {
		loopNode := node.NewLoopNode(g.NextID("loop"), "map")
		loopNode.SetFile(path)
		g.AddNode(loopNode)
		children = append(children, loopNode.ID())
	}

	g.SetChildren(compNode.ID(), children)
	return string(compNode.ID())
}
