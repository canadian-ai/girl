package shared

import "strings"

func ShouldSkipDir(base string) bool {
	switch base {
	case ".git", ".grp", "node_modules", "vendor", "dist", "build", ".next", ".turbo", ".cache", "out", ".vercel":
		return true
	}
	return strings.HasPrefix(base, ".")
}
