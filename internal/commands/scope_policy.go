package commands

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/canadian-ai/girl/internal/shared"
)

// scopePolicy is the single source of truth for which files a GIRL command may
// read. It is applied before any file is opened, by check, next and every
// language analyzer, in all/changed/single-file modes.
//
// Exclusion pattern semantics (analysis.exclude in .girl/config.yaml):
//
//   - A pattern without "/" (e.g. "generated", "*.pb.go") matches any single
//     path segment at any depth: a directory with that name excludes its whole
//     subtree; a file whose base name matches is excluded.
//   - A pattern containing "/" (e.g. "src/legacy", "packages/*/fixtures") is
//     anchored at the directory holding .girl/config.yaml and excludes that
//     path and everything beneath it. Each segment may use path.Match globs.
//   - A leading "./" or trailing "/" is ignored; "**" matches any number of
//     segments.
//   - Matching is on path-segment boundaries: "gen" never excludes "generic".
//
// Built-in skips (VCS metadata, dependency and build directories) always apply
// in addition to configured exclusions; configuration cannot re-include them.
type scopePolicy struct {
	// Root is the absolute directory exclusion patterns are relative to.
	Root     string
	Patterns []string
	// ConfigPath is the config file the patterns came from ("" for defaults).
	ConfigPath string
}

func newScopePolicy(root string, patterns []string) (*scopePolicy, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var cleaned []string
	for _, p := range patterns {
		p = strings.TrimSpace(filepath.ToSlash(p))
		p = strings.TrimPrefix(p, "./")
		p = strings.TrimSuffix(p, "/")
		if p == "" || p == "." {
			continue
		}
		if _, err := path.Match(strings.ReplaceAll(p, "**", "*"), ""); err != nil {
			return nil, fmt.Errorf("invalid exclusion pattern %q: %w", p, err)
		}
		cleaned = append(cleaned, p)
	}
	return &scopePolicy{Root: abs, Patterns: cleaned}, nil
}

// relative returns p relative to the policy root in slash form, and false when
// p lies outside the root.
func (s *scopePolicy) relative(p string) (string, bool) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(s.Root, abs)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

// Excluded reports whether p (absolute or relative to the working directory)
// is excluded by configuration. Paths outside the policy root are not
// excluded by it (they are rejected separately by scope checks).
func (s *scopePolicy) Excluded(p string, isDir bool) bool {
	if s == nil || len(s.Patterns) == 0 {
		return false
	}
	rel, ok := s.relative(p)
	if !ok || rel == "." {
		return false
	}
	return s.ExcludedRel(rel)
}

// ExcludedRel is Excluded for a root-relative slash path.
func (s *scopePolicy) ExcludedRel(rel string) bool {
	if s == nil {
		return false
	}
	segs := strings.Split(strings.Trim(rel, "/"), "/")
	for _, pat := range s.Patterns {
		if !strings.Contains(pat, "/") {
			for _, seg := range segs {
				if ok, _ := path.Match(pat, seg); ok {
					return true
				}
			}
			continue
		}
		if matchAnchoredPrefix(strings.Split(pat, "/"), segs) {
			return true
		}
	}
	return false
}

// matchAnchoredPrefix reports whether pattern segments match a prefix of segs.
func matchAnchoredPrefix(pat, segs []string) bool {
	if len(pat) == 0 {
		return true
	}
	if pat[0] == "**" {
		for i := 0; i <= len(segs); i++ {
			if matchAnchoredPrefix(pat[1:], segs[i:]) {
				return true
			}
		}
		return false
	}
	if len(segs) == 0 {
		return false
	}
	if ok, _ := path.Match(pat[0], segs[0]); !ok {
		return false
	}
	return matchAnchoredPrefix(pat[1:], segs[1:])
}

// Filter adapts the policy for analyzers.
func (s *scopePolicy) Filter() shared.PathFilter {
	if s == nil || len(s.Patterns) == 0 {
		return nil
	}
	return s.Excluded
}

// findProjectConfigDir walks up from start to the nearest directory holding
// .girl/config.yaml, stopping at the Git repository root (or filesystem root).
// It returns "" when no configuration applies.
func findProjectConfigDir(start string) string {
	abs, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	stop, err := gitRepoRoot(abs)
	if err != nil {
		// Outside a repository there is no trustworthy boundary: only the
		// target directory itself may supply configuration, never a parent
		// such as the user's home directory.
		if checkFileExists(filepath.Join(abs, ".girl", "config.yaml")) {
			return abs
		}
		return ""
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		if checkFileExists(filepath.Join(dir, ".girl", "config.yaml")) {
			return dir
		}
		resolved := dir
		if r, rerr := filepath.EvalSymlinks(dir); rerr == nil {
			resolved = r
		}
		if resolved == stop || dir == filepath.Dir(dir) {
			return ""
		}
	}
}

// loadScopePolicy loads the exclusion policy that applies to target: the
// nearest enclosing project configuration, or GIRL's defaults rooted at target.
func loadScopePolicy(target string) (*scopePolicy, error) {
	dir := findProjectConfigDir(target)
	root := target
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		root = filepath.Dir(target)
	}
	if dir != "" {
		root = dir
	}
	cfg, err := loadGirlProjectConfig(root)
	if err != nil {
		return nil, err
	}
	return policyFromConfig(root, cfg)
}

func policyFromConfig(root string, cfg *GirlProjectConfig) (*scopePolicy, error) {
	policy, err := newScopePolicy(root, cfg.Analysis.Exclude)
	if err != nil {
		return nil, err
	}
	if cfg.sourcePath != "" {
		policy.ConfigPath = ".girl/config.yaml"
	}
	return policy, nil
}
