package diffstats

// TokenType defines the type of a lexer token, following ANTLR conventions.
type TokenType int

const (
	TokenInvalid     TokenType = 0
	TokenDiff        TokenType = 1  // diff --git a/path b/path
	TokenIndex       TokenType = 2  // index hash..hash mode
	TokenNewFile     TokenType = 3  // new file mode NNN
	TokenDeletedFile TokenType = 4  // deleted file mode NNN
	TokenOldMode     TokenType = 5  // old mode NNN
	TokenNewModeHdr  TokenType = 6  // new mode NNN
	TokenRenameFrom  TokenType = 7  // rename from path
	TokenRenameTo    TokenType = 8  // rename to path
	TokenSimilarity  TokenType = 9  // similarity index NNN%
	TokenBinaryHdr   TokenType = 10 // Binary files ... differ
	TokenOldFile     TokenType = 11 // --- a/path
	TokenNewFileHdr  TokenType = 12 // +++ b/path
	TokenHunkHdr     TokenType = 13 // @@ -n,m +n,m @@
	TokenContext     TokenType = 14 // leading space
	TokenAddition    TokenType = 15 // leading +
	TokenDeletion    TokenType = 16 // leading -
	TokenNoNewline   TokenType = 17 // \ No newline at end of file
	TokenEOF         TokenType = -1
)

type Token struct {
	Type TokenType
	Text string
	Line int
}

// grammarToken returns the token text trimmed to the grammar-relevant prefix.
func (t Token) grammarToken() string {
	if len(t.Text) > 0 {
		return string(t.Text[0])
	}
	return ""
}
