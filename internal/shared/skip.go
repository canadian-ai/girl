package shared

import (
	"os"
	"strings"
)

// ShouldSkipDir is the single source of truth for which directories to skip
// during file tree walks. Add new entries here rather than duplicating logic
// in individual walkers.
func ShouldSkipDir(base string) bool {
	switch base {
	case ".git", ".grp", "node_modules", "vendor", "dist", "build", ".next", ".turbo", ".cache", "out", ".vercel", "coverage":
		return true
	}
	return strings.HasPrefix(base, ".")
}

// IsSymlink reports whether a walk entry is a symbolic link. Walkers skip
// links so a link inside a repository can never cause a file outside the
// analysis root to be read (filepath.Walk does not descend into linked
// directories, but a linked file would otherwise be opened by name).
func IsSymlink(fi os.FileInfo) bool {
	return fi != nil && fi.Mode()&os.ModeSymlink != 0
}

// PathFilter reports whether a path should be excluded from analysis. It is
// called with the absolute or walk-relative path exactly as the walker sees it
// and must be consistent for directories (isDir=true) and files. A nil filter
// excludes nothing. Analyzers consult it before reading any file contents.
type PathFilter func(path string, isDir bool) bool

// Skip is a nil-safe call of f.
func (f PathFilter) Skip(path string, isDir bool) bool {
	return f != nil && f(path, isDir)
}
