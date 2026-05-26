package parsertsx

import (
	"os"
	"regexp"
	"strings"

	"github.com/canadian-ai/girl/internal/node"
)

type Parser struct{}

func New() *Parser {
	return &Parser{}
}

func (p *Parser) ParseFile(path string) (*node.NodeGraph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	g := node.NewNodeGraph()
	root := node.NewRootNode(g.NextID("root"))
	root.SetFile(path)
	g.AddNode(root)
	g.SetFileNode(path, root.ID())
	parseFileContent(g, root, content, path)
	g.SetChildren(root.ID(), root.Children())
	return g, nil
}

type tokenizer struct {
	src  []rune
	pos  int
	line int
	col  int
}

type token struct {
	kind  tokenKind
	value string
	line  int
	col   int
}

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokIdent
	tokString
	tokNumber
	tokLParen
	tokRParen
	tokLBrace
	tokRBrace
	tokLBracket
	tokRBracket
	tokLAngle
	tokRAngle
	tokComma
	tokDot
	tokColon
	tokSemicolon
	tokArrow
	tokEquals
	tokAsterisk
	tokAmpersand
	tokPipe
	tokQuestion
	tokBang
	tokSlash
	tokMinus
	tokPlus
	tokBacktick
	tokAt
	tokHash
	tokNewline
	tokKeyword
	tokJSXString
	tokJSXIdent
	tokJSXBrace
	tokTemplateStart
	tokTemplateEnd
	tokTemplateMid
)

var keywords = map[string]bool{
	"import": true, "export": true, "default": true, "from": true,
	"function": true, "const": true, "let": true, "var": true,
	"return": true, "if": true, "else": true, "for": true, "while": true,
	"do": true, "switch": true, "case": true, "break": true,
	"continue": true, "new": true, "delete": true, "typeof": true,
	"async": true, "await": true, "yield": true, "class": true,
	"extends": true, "implements": true, "interface": true, "type": true,
	"enum": true, "as": true, "in": true, "of": true, "try": true,
	"catch": true, "finally": true, "throw": true, "this": true,
	"super": true, "true": true, "false": true, "null": true,
	"undefined": true, "void": true, "declare": true, "namespace": true,
}

func newTokenizer(src string) *tokenizer {
	return &tokenizer{
		src:  []rune(src),
		pos:  0,
		line: 1,
		col:  1,
	}
}

func (t *tokenizer) peek() rune {
	if t.pos >= len(t.src) {
		return 0
	}
	return t.src[t.pos]
}

func (t *tokenizer) peekN(n int) string {
	if t.pos+n > len(t.src) {
		return ""
	}
	return string(t.src[t.pos : t.pos+n])
}

func (t *tokenizer) advance() rune {
	ch := t.src[t.pos]
	t.pos++
	if ch == '\n' {
		t.line++
		t.col = 1
	} else {
		t.col++
	}
	return ch
}

func (t *tokenizer) skipWhitespace() {
	for t.pos < len(t.src) {
		ch := t.src[t.pos]
		if ch == ' ' || ch == '\t' || ch == '\r' {
			t.pos++
			t.col++
		} else if ch == '\n' {
			return
		} else {
			break
		}
	}
}

func (t *tokenizer) skipComment() {
	if t.peekN(2) == "//" {
		for t.pos < len(t.src) && t.src[t.pos] != '\n' {
			t.pos++
		}
		return
	}
	if t.peekN(2) == "/*" {
		t.pos += 2
		for t.pos+1 < len(t.src) {
			if t.src[t.pos] == '*' && t.src[t.pos+1] == '/' {
				t.pos += 2
				return
			}
			if t.src[t.pos] == '\n' {
				t.line++
			}
			t.pos++
		}
	}
}

func isIdentStart(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || ch == '$'
}

func isIdentPart(ch rune) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}

func (t *tokenizer) readIdent() string {
	start := t.pos
	for t.pos < len(t.src) && isIdentPart(t.src[t.pos]) {
		t.pos++
	}
	return string(t.src[start:t.pos])
}

func (t *tokenizer) readString(quote rune) string {
	start := t.pos
	t.pos++
	for t.pos < len(t.src) {
		ch := t.src[t.pos]
		if ch == '\\' {
			t.pos += 2
			continue
		}
		if ch == quote {
			t.pos++
			return string(t.src[start:t.pos])
		}
		if ch == '\n' {
			return string(t.src[start:t.pos])
		}
		t.pos++
	}
	return string(t.src[start:t.pos])
}

func (t *tokenizer) readNumber() string {
	start := t.pos
	for t.pos < len(t.src) {
		ch := t.src[t.pos]
		if (ch >= '0' && ch <= '9') || ch == '.' || ch == 'x' || ch == 'X' ||
			(ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
			t.pos++
		} else {
			break
		}
	}
	return string(t.src[start:t.pos])
}

func (t *tokenizer) skipToNewline() {
	for t.pos < len(t.src) && t.src[t.pos] != '\n' {
		t.pos++
	}
}

func parseFileContent(g *node.NodeGraph, root *node.RootNode, content string, path string) {
	lines := strings.Split(content, "\n")
	tz := newTokenizer(content)

	var imports []string

	for tz.pos < len(tz.src) {
		tz.skipWhitespace()
		tz.skipComment()
		if tz.pos >= len(tz.src) {
			break
		}

		if tz.peek() == '\n' {
			tz.pos++
			tz.line++
			tz.col = 1
			continue
		}

		if tz.peekN(2) == "//" || tz.peekN(2) == "/*" {
			tz.skipComment()
			continue
		}

		if tz.peek() == 'i' && tz.peekN(6) == "import" {
			imp := parseImport(tz)
			if imp != "" {
				imports = append(imports, imp)
			}
			continue
		}

		if tz.peek() == 'e' && tz.peekN(6) == "export" {
			parseExportStmt(g, root, tz, content, lines, path)
			continue
		}

		if isIdentStart(tz.peek()) || tz.peek() == 'a' {
			name := tz.peekN(5)
			if name == "async" {
				// could be async function
				tz.pos += 5
				tz.skipWhitespace()
				if tz.peekN(8) == "function" {
					fn := parseFunctionDecl(tz)
					if fn != "" && isComponentName(fn) {
						comp := buildComponentFromBody(g, path, content, lines, fn)
						if comp != "" {
							root.AddChild(node.NodeID(comp))
						}
					}
				}
				continue
			}
		}

		if tz.peekN(8) == "function" {
			savePos := tz.pos
			tz.pos += 8
			tz.skipWhitespace()
			fnName := tz.readIdent()
			if fnName != "" && isComponentName(fnName) {
				compID := buildComponentFromBody(g, path, content, lines, fnName)
				if compID != "" {
					root.AddChild(node.NodeID(compID))
				}
			} else {
				tz.pos = savePos
				tz.pos++
			}
			continue
		}

		if tz.peekN(5) == "const" || tz.peekN(3) == "let" || tz.peekN(3) == "var" {
			decl := parseDeclaration(tz)
			if decl != "" && isComponentName(decl) {
				compID := buildComponentFromBody(g, path, content, lines, decl)
				if compID != "" {
					root.AddChild(node.NodeID(compID))
				}
			}
			continue
		}

		tz.pos++
	}
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

func parseImport(tz *tokenizer) string {
	tz.pos += 6
	tz.skipWhitespace()

	if tz.peekN(4) == "type" {
		tz.pos += 4
		tz.skipWhitespace()
	}

	if tz.peek() == '{' {
		tz.pos++
		for tz.pos < len(tz.src) && tz.peek() != '}' {
			if tz.peek() == '\n' {
				tz.line++
				tz.col = 1
			}
			tz.pos++
		}
		if tz.peek() == '}' {
			tz.pos++
		}
	}

	tz.skipWhitespace()

	if tz.peekN(4) == "from" {
		tz.pos += 4
		tz.skipWhitespace()
		if tz.peek() == '"' || tz.peek() == '\'' {
			return tz.readString(tz.peek())
		}
	} else {
		if isIdentStart(tz.peek()) {
			module := tz.readIdent()
			return module
		}
	}
	return ""
}

func parseExportStmt(g *node.NodeGraph, root *node.RootNode, tz *tokenizer, content string, lines []string, path string) {
	tz.pos += 6
	tz.skipWhitespace()

	if tz.peekN(7) == "default" {
		tz.pos += 7
		tz.skipWhitespace()
		if tz.peekN(8) == "function" {
			tz.pos += 8
			tz.skipWhitespace()
			name := tz.readIdent()
			if name != "" && isComponentName(name) {
				compID := buildComponentFromBody(g, path, content, lines, name)
				if compID != "" {
					root.AddChild(node.NodeID(compID))
				}
			}
		}
		return
	}

	if tz.peekN(8) == "function" {
		tz.pos += 8
		tz.skipWhitespace()
		name := tz.readIdent()
		if name != "" && isComponentName(name) {
			compID := buildComponentFromBody(g, path, content, lines, name)
			if compID != "" {
				root.AddChild(node.NodeID(compID))
			}
		}
		return
	}

	if tz.peekN(5) == "const" {
		tz.pos += 5
		tz.skipWhitespace()
		if tz.peek() == '{' {
			tz.skipToNewline()
			return
		}
		name := tz.readIdent()
		tz.skipWhitespace()
		if tz.peek() == ':' {
			tz.pos++
			tz.skipWhitespace()
			skipTypeAnnotation(tz)
			tz.skipWhitespace()
		}
		if name != "" && isComponentName(name) && looksLikeComponentInitializer(tz) {
			compID := buildComponentFromBody(g, path, content, lines, name)
			if compID != "" {
				root.AddChild(node.NodeID(compID))
			}
		}
		return
	}
}

func parseFunctionDecl(tz *tokenizer) string {
	if tz.peekN(8) == "function" {
		tz.pos += 8
		tz.skipWhitespace()
		return tz.readIdent()
	}
	return ""
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

func parseDeclaration(tz *tokenizer) string {
	if tz.peekN(5) == "const" {
		tz.pos += 5
	} else if tz.peekN(3) == "let" {
		tz.pos += 3
	} else if tz.peekN(3) == "var" {
		tz.pos += 3
	} else {
		tz.skipToNewline()
		return ""
	}

	tz.skipWhitespace()

	if tz.peek() == '{' {
		tz.skipToNewline()
		return ""
	}

	name := tz.readIdent()
	tz.skipWhitespace()

	if tz.peek() == ':' {
		tz.pos++
		tz.skipWhitespace()
		skipTypeAnnotation(tz)
		tz.skipWhitespace()
	}

	if looksLikeComponentInitializer(tz) {
		return name
	}
	return ""
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
