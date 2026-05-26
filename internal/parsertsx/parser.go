package parsertsx

import (
	"os"
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
