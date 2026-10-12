package shared

import (
	"os"
	"path/filepath"
	"strings"
)

// DisplayTarget renders an analysis target for reports without leaking
// machine-specific absolute paths.
func DisplayTarget(target string) string {
	clean := filepath.Clean(target)
	if filepath.IsAbs(clean) {
		return filepath.Base(clean)
	}
	return filepath.ToSlash(clean)
}

// DisplayFile renders a diagnostic path for reports. Paths are relative to
// the working directory so agents can open them directly; a path that escapes
// the working directory is shown relative to the analysis target instead.
func DisplayFile(target, file string) string {
	if !strings.HasPrefix(filepath.ToSlash(file), "../") && !filepath.IsAbs(file) {
		return filepath.ToSlash(file)
	}
	abs := file
	if !filepath.IsAbs(abs) {
		cwd, err := os.Getwd()
		if err != nil {
			return filepath.ToSlash(file)
		}
		abs = filepath.Join(cwd, file)
	}
	root, err := filepath.Abs(target)
	if err != nil {
		return filepath.ToSlash(file)
	}
	if info, statErr := os.Stat(root); statErr == nil && !info.IsDir() {
		root = filepath.Dir(root)
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.Base(file)
	}
	return filepath.ToSlash(rel)
}
