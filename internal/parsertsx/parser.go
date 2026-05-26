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
	for t.pos < len(t.src) && isNumberRune(t.src[t.pos]) {
		t.pos++
	}
	return string(t.src[start:t.pos])
}

func isNumberRune(ch rune) bool {
	return (ch >= '0' && ch <= '9') || ch == '.' || ch == 'x' || ch == 'X' ||
		(ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func (t *tokenizer) skipToNewline() {
	for t.pos < len(t.src) && t.src[t.pos] != '\n' {
		t.pos++
	}
}

func parseFileContent(g *node.NodeGraph, root *node.RootNode, content string, path string) {
	lines := strings.Split(content, "\n")
	tz := newTokenizer(content)
	ctx := parseContext{g: g, root: root, content: content, lines: lines, path: path}
	var imports []string

	for tz.pos < len(tz.src) {
		if consumeTrivia(tz) {
			continue
		}
		if parseTopLevelStatement(&ctx, tz, &imports) {
			continue
		}
		tz.pos++
	}
}

type parseContext struct {
	g       *node.NodeGraph
	root    *node.RootNode
	content string
	lines   []string
	path    string
}

func consumeTrivia(tz *tokenizer) bool {
	tz.skipWhitespace()
	if tz.pos >= len(tz.src) {
		return false
	}
	if tz.peek() == '\n' {
		tz.pos++
		tz.line++
		tz.col = 1
		return true
	}
	if tz.peekN(2) == "//" || tz.peekN(2) == "/*" {
		tz.skipComment()
		return true
	}
	return false
}

func parseTopLevelStatement(ctx *parseContext, tz *tokenizer, imports *[]string) bool {
	if tz.peek() == 'i' && tz.peekN(6) == "import" {
		if imp := parseImport(tz); imp != "" {
			*imports = append(*imports, imp)
		}
		return true
	}
	if tz.peek() == 'e' && tz.peekN(6) == "export" {
		parseExportStmt(ctx, tz)
		return true
	}
	if tz.peekN(5) == "async" {
		parseAsyncFunction(ctx, tz)
		return true
	}
	if tz.peekN(8) == "function" {
		parseTopLevelFunction(ctx, tz)
		return true
	}
	if isDeclarationStart(tz) {
		parseTopLevelDeclaration(ctx, tz)
		return true
	}
	return false
}

func isDeclarationStart(tz *tokenizer) bool {
	return tz.peekN(5) == "const" || tz.peekN(3) == "let" || tz.peekN(3) == "var"
}

func addComponentByName(ctx *parseContext, name string) {
	if name == "" || !isComponentName(name) {
		return
	}
	compID := buildComponentFromBody(ctx.g, ctx.path, ctx.content, ctx.lines, name)
	if compID != "" {
		ctx.root.AddChild(node.NodeID(compID))
	}
}

func parseAsyncFunction(ctx *parseContext, tz *tokenizer) {
	tz.pos += 5
	tz.skipWhitespace()
	if tz.peekN(8) == "function" {
		addComponentByName(ctx, parseFunctionDecl(tz))
	}
}

func parseTopLevelFunction(ctx *parseContext, tz *tokenizer) {
	savePos := tz.pos
	tz.pos += 8
	tz.skipWhitespace()
	fnName := tz.readIdent()
	if fnName != "" && isComponentName(fnName) {
		addComponentByName(ctx, fnName)
		return
	}
	tz.pos = savePos
	tz.pos++
}

func parseTopLevelDeclaration(ctx *parseContext, tz *tokenizer) {
	decl := parseDeclaration(tz)
	if decl != "" && isComponentName(decl) {
		addComponentByName(ctx, decl)
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
		skipImportSpecifiers(tz)
	}

	tz.skipWhitespace()

	if tz.peekN(4) == "from" {
		return parseImportSource(tz)
	}
	if isIdentStart(tz.peek()) {
		return tz.readIdent()
	}
	return ""
}

func skipImportSpecifiers(tz *tokenizer) {
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

func parseImportSource(tz *tokenizer) string {
	tz.pos += 4
	tz.skipWhitespace()
	if tz.peek() == '"' || tz.peek() == '\'' {
		return tz.readString(tz.peek())
	}
	return ""
}

func parseExportStmt(ctx *parseContext, tz *tokenizer) {
	tz.pos += 6
	tz.skipWhitespace()

	if tz.peekN(7) == "default" {
		parseDefaultExport(ctx, tz)
		return
	}

	if tz.peekN(8) == "function" {
		addExportedFunction(ctx, tz)
		return
	}

	if tz.peekN(5) == "const" {
		addExportedConst(ctx, tz)
		return
	}
}

func parseDefaultExport(ctx *parseContext, tz *tokenizer) {
	tz.pos += 7
	tz.skipWhitespace()
	if tz.peekN(8) == "function" {
		addExportedFunction(ctx, tz)
	}
}

func addExportedFunction(ctx *parseContext, tz *tokenizer) {
	tz.pos += 8
	tz.skipWhitespace()
	addComponentByName(ctx, tz.readIdent())
}

func addExportedConst(ctx *parseContext, tz *tokenizer) {
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
		addComponentByName(ctx, name)
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
