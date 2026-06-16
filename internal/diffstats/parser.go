package diffstats

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ParseDiff parses a unified git diff from an io.Reader into DiffStats.
// Uses a grammar-based lexer+parser instead of regex.
func ParseDiff(r io.Reader) (*DiffStats, error) {
	lexer, err := Lex(r)
	if err != nil {
		return nil, fmt.Errorf("lex diff: %w", err)
	}
	return parseDiff(lexer)
}

// ParseDiffBytes parses a unified git diff from a byte slice.
func ParseDiffBytes(data []byte) (*DiffStats, error) {
	return ParseDiff(strings.NewReader(string(data)))
}

// ── Grammar: diff → file* EOF ──────────────────────────────────────────────

func parseDiff(l *Lexer) (*DiffStats, error) {
	stats := &DiffStats{}

	for l.HasMore() {
		tok := l.Peek()
		switch tok.Type {
		case TokenEOF:
			PostProcess(stats)
			return stats, nil
		case TokenDiff:
			file, err := parseFile(l)
			if err != nil {
				return nil, err
			}
			stats.Files = append(stats.Files, *file)
		default:
			l.Next()
		}
	}

	PostProcess(stats)
	return stats, nil
}

// ── Grammar: file → header metadata* body ──────────────────────────────────

func parseFile(l *Lexer) (*FileStat, error) {
	f := &FileStat{}

	// header → DIFF
	diffTok := l.Next()
	if diffTok.Type != TokenDiff {
		return nil, fmt.Errorf("expected diff header, got token %d at line %d", diffTok.Type, diffTok.Line)
	}
	f.Path, f.OldPath = extractPaths(diffTok.Text)

	// metadata* — consume optional metadata lines
	for l.HasMore() {
		tok := l.Peek()
		switch tok.Type {
		case TokenIndex, TokenOldMode, TokenNewModeHdr, TokenSimilarity:
			l.Next()

		case TokenNewFile:
			l.Next()

		case TokenDeletedFile:
			l.Next()

		case TokenRenameFrom:
			f.OldPath = extractRenamePath(l.Next().Text)
			f.IsRename = true

		case TokenRenameTo:
			if p := extractRenamePath(l.Next().Text); p != "" {
				f.Path = p
			}
			f.IsRename = true

		case TokenBinaryHdr:
			f.IsBinary = true
			l.Next()
			return f, nil

		case TokenOldFile:
			l.Next()
			return parseBody(l, f)

		case TokenNewFileHdr:
			l.Next()
			return parseBody(l, f)

		case TokenHunkHdr:
			return parseBody(l, f)

		default:
			return f, nil
		}
	}

	return f, nil
}

// ── Grammar: body → (OLD_FILE NEW_FILE)? hunk+ ─────────────────────────────

func parseBody(l *Lexer, f *FileStat) (*FileStat, error) {
	// Consume optional +++ line (--- already consumed above)
	for l.HasMore() {
		tok := l.Peek()
		switch tok.Type {
		case TokenNewFileHdr:
			l.Next()
		case TokenHunkHdr:
			goto hunks
		case TokenOldFile:
			l.Next()
		case TokenDiff, TokenEOF:
			return f, nil
		default:
			return f, nil
		}
	}

	return f, nil

hunks:
	for l.HasMore() {
		tok := l.Peek()
		switch tok.Type {
		case TokenHunkHdr:
			if err := parseHunk(l, f); err != nil {
				return nil, err
			}
		case TokenDiff, TokenEOF:
			return f, nil
		default:
			l.Next()
		}
	}

	return f, nil
}

// ── Grammar: hunk → HUNK_HDR hunkLine* ─────────────────────────────────────

func parseHunk(l *Lexer, f *FileStat) error {
	l.Next() // consume HUNK_HDR

	for l.HasMore() {
		tok := l.Peek()
		switch tok.Type {
		case TokenAddition:
			f.AddedLines++
			l.Next()
		case TokenDeletion:
			f.DeletedLines++
			l.Next()
		case TokenContext, TokenNoNewline:
			l.Next()
		case TokenHunkHdr, TokenDiff, TokenEOF:
			return nil
		default:
			return nil
		}
	}
	return nil
}

// ── Path extraction helpers ─────────────────────────────────────────────────

func extractPaths(diffLine string) (newPath, oldPath string) {
	rest := strings.TrimPrefix(diffLine, "diff --git a/")
	if idx := strings.Index(rest, " b/"); idx >= 0 {
		oldPath = rest[:idx]
		newPath = rest[idx+3:]
	}
	return newPath, oldPath
}

func extractRenamePath(line string) string {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) == 3 {
		return strings.TrimSpace(parts[2])
	}
	return ""
}

// ── Post-processing (unchanged from regex version) ─────────────────────────

var generatedPrefixes = []string{
	"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lock",
	"go.sum", "Cargo.lock", "Gemfile.lock",
	"poetry.lock", "composer.lock",
}

var generatedDirs = []string{"node_modules", ".next", "dist", "build", ".cache"}

var lockfileNames = map[string]bool{
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true,
	"bun.lock": true, "go.sum": true, "Cargo.lock": true, "Gemfile.lock": true,
}

func isGenerated(path string) bool {
	base := filepath.Base(path)
	for _, p := range generatedPrefixes {
		if base == p {
			return true
		}
	}
	dir := filepath.Dir(path)
	for _, d := range generatedDirs {
		if strings.HasPrefix(dir, d) || dir == d {
			return true
		}
	}
	return false
}

func isLockfile(path string) bool {
	return lockfileNames[filepath.Base(path)]
}

func categoryForFile(path string) string {
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	switch {
	case strings.HasPrefix(dir, "internal/") || strings.HasPrefix(dir, "pkg/"):
		return "library"
	case ext == ".go":
		return "go"
	case ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx":
		return "typescript"
	case ext == ".css" || ext == ".scss" || ext == ".less":
		return "style"
	case ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".toml":
		return "config"
	case ext == ".md" || ext == ".txt":
		return "documentation"
	case strings.HasPrefix(path, "test") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".test.ts"):
		return "test"
	}
	return "other"
}

// PostProcess computes aggregate fields on DiffStats after parsing all files.
func PostProcess(stats *DiffStats) {
	for i := range stats.Files {
		f := &stats.Files[i]
		f.ChangedLines = f.AddedLines + f.DeletedLines
		f.IsGenerated = isGenerated(f.Path)
		f.IsLockfile = isLockfile(f.Path)
		stats.TotalAdded += f.AddedLines
		stats.TotalDeleted += f.DeletedLines
		stats.TotalChanged += f.ChangedLines
		if f.ChangedLines > stats.LargestDelta {
			stats.LargestDelta = f.ChangedLines
		}
	}
	stats.TotalFiles = len(stats.Files)

	catSet := map[string]bool{}
	for _, f := range stats.Files {
		cat := categoryForFile(f.Path)
		if cat != "" {
			catSet[cat] = true
		}
		if f.IsBinary {
			stats.HasBinary = true
		}
		if f.IsGenerated {
			stats.HasGenerated = true
		}
		if f.IsLockfile {
			stats.HasLockfile = true
		}
		if f.IsRename {
			stats.HasRename = true
		}
	}
	for c := range catSet {
		stats.Categories = append(stats.Categories, c)
	}
}
