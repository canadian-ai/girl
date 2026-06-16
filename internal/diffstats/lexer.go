package diffstats

import (
	"bufio"
	"io"
	"strings"
)

// Lexer tokenizes a unified diff input into a token stream.
// It implements a state-machine lexer following the UnifiedDiff grammar.
type Lexer struct {
	tokens []Token
	pos    int
	lineNo int
}

// Lex reads all input and produces a token slice.
func Lex(r io.Reader) (*Lexer, error) {
	l := &Lexer{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)

	for scanner.Scan() {
		l.lineNo++
		line := scanner.Text()
		l.emit(line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	l.tokens = append(l.tokens, Token{Type: TokenEOF, Line: l.lineNo})
	return l, nil
}

// LexString is a convenience wrapper around Lex.
func LexString(input string) (*Lexer, error) {
	return Lex(strings.NewReader(input))
}

// Next consumes and returns the next token.
func (l *Lexer) Next() Token {
	if l.pos >= len(l.tokens) {
		return Token{Type: TokenEOF, Line: l.lineNo}
	}
	t := l.tokens[l.pos]
	l.pos++
	return t
}

// Peek returns the next token without consuming it.
func (l *Lexer) Peek() Token {
	if l.pos >= len(l.tokens) {
		return Token{Type: TokenEOF, Line: l.lineNo}
	}
	return l.tokens[l.pos]
}

// HasMore returns true if there are unconsumed tokens.
func (l *Lexer) HasMore() bool {
	return l.pos < len(l.tokens)
}

// emit classifies a single line and appends the corresponding token.
func (l *Lexer) emit(line string) {
	// Order matters: more specific patterns checked first.
	switch {
	case l.isHunkHeader(line):
		l.tokens = append(l.tokens, Token{Type: TokenHunkHdr, Text: line, Line: l.lineNo})

	case l.isDiffHeader(line):
		l.tokens = append(l.tokens, Token{Type: TokenDiff, Text: line, Line: l.lineNo})

	case l.isBinaryHeader(line):
		l.tokens = append(l.tokens, Token{Type: TokenBinaryHdr, Text: line, Line: l.lineNo})

	case l.isNewFileMode(line):
		l.tokens = append(l.tokens, Token{Type: TokenNewFile, Text: line, Line: l.lineNo})

	case l.isDeletedFileMode(line):
		l.tokens = append(l.tokens, Token{Type: TokenDeletedFile, Text: line, Line: l.lineNo})

	case l.isOldMode(line):
		l.tokens = append(l.tokens, Token{Type: TokenOldMode, Text: line, Line: l.lineNo})

	case l.isNewModeHeader(line):
		l.tokens = append(l.tokens, Token{Type: TokenNewModeHdr, Text: line, Line: l.lineNo})

	case l.isRenameFrom(line):
		l.tokens = append(l.tokens, Token{Type: TokenRenameFrom, Text: line, Line: l.lineNo})

	case l.isRenameTo(line):
		l.tokens = append(l.tokens, Token{Type: TokenRenameTo, Text: line, Line: l.lineNo})

	case l.isSimilarity(line):
		l.tokens = append(l.tokens, Token{Type: TokenSimilarity, Text: line, Line: l.lineNo})

	case l.isIndex(line):
		l.tokens = append(l.tokens, Token{Type: TokenIndex, Text: line, Line: l.lineNo})

	case l.isOldFile(line):
		l.tokens = append(l.tokens, Token{Type: TokenOldFile, Text: line, Line: l.lineNo})

	case l.isNewFileHeader(line):
		l.tokens = append(l.tokens, Token{Type: TokenNewFileHdr, Text: line, Line: l.lineNo})

	case l.isNoNewline(line):
		l.tokens = append(l.tokens, Token{Type: TokenNoNewline, Text: line, Line: l.lineNo})

	case l.isAddition(line):
		l.tokens = append(l.tokens, Token{Type: TokenAddition, Text: line, Line: l.lineNo})

	case l.isDeletion(line):
		l.tokens = append(l.tokens, Token{Type: TokenDeletion, Text: line, Line: l.lineNo})

	case l.isContext(line):
		l.tokens = append(l.tokens, Token{Type: TokenContext, Text: line, Line: l.lineNo})

	// Empty lines between diffs or unknown lines are silently skipped.
	}
}

// ── Pattern matchers (grammar-informed, no regex) ──────────────────────────

func (*Lexer) isDiffHeader(s string) bool {
	return strings.HasPrefix(s, "diff --git a/")
}

func (*Lexer) isIndex(s string) bool {
	return strings.HasPrefix(s, "index ") && len(s) > 6 && isHexRange(s[6:])
}

func isHexRange(s string) bool {
	if len(s) < 3 {
		return false
	}
	dotCount := 0
	for _, c := range s {
		if c == '.' {
			dotCount++
		} else if !isHexDigit(c) && c != ' ' {
			return false
		}
	}
	return dotCount == 2
}

func isHexDigit(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func (*Lexer) isNewFileMode(s string) bool {
	return strings.HasPrefix(s, "new file mode ")
}

func (*Lexer) isDeletedFileMode(s string) bool {
	return strings.HasPrefix(s, "deleted file mode ")
}

func (*Lexer) isOldMode(s string) bool {
	return strings.HasPrefix(s, "old mode ")
}

func (*Lexer) isNewModeHeader(s string) bool {
	return strings.HasPrefix(s, "new mode ")
}

func (*Lexer) isRenameFrom(s string) bool {
	return strings.HasPrefix(s, "rename from ")
}

func (*Lexer) isRenameTo(s string) bool {
	return strings.HasPrefix(s, "rename to ")
}

func (*Lexer) isSimilarity(s string) bool {
	return strings.HasPrefix(s, "similarity index ")
}

func (*Lexer) isBinaryHeader(s string) bool {
	return strings.HasPrefix(s, "Binary files ") && strings.HasSuffix(s, " differ")
}

func (*Lexer) isOldFile(s string) bool {
	return strings.HasPrefix(s, "--- ")
}

func (*Lexer) isNewFileHeader(s string) bool {
	return strings.HasPrefix(s, "+++ ")
}

func (*Lexer) isHunkHeader(s string) bool {
	return strings.HasPrefix(s, "@@ -")
}

func (*Lexer) isNoNewline(s string) bool {
	return strings.HasPrefix(s, "\\ No newline at end of file")
}

func (*Lexer) isAddition(s string) bool {
	return len(s) > 0 && s[0] == '+' && !strings.HasPrefix(s, "+++ ")
}

func (*Lexer) isDeletion(s string) bool {
	return len(s) > 0 && s[0] == '-' && !strings.HasPrefix(s, "--- ")
}

func (*Lexer) isContext(s string) bool {
	return len(s) > 0 && s[0] == ' '
}
