package shared

import "strings"

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
