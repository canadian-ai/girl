package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Changed-file discovery modes.
const (
	// changedModeWorkingTree compares the working tree with HEAD: staged,
	// unstaged and untracked (non-ignored) files. Renames report the new path.
	changedModeWorkingTree = "working-tree"
	// changedModeBase compares committed history: merge-base(base, HEAD)..HEAD.
	// This is the CI mode; uncommitted files are not included.
	changedModeBase = "base"
)

// changedScope is the result of changed-file discovery. Files are relative to
// the requested analysis root (not the Git root) and constrained to it.
//
// Deleted files are never reported (there is nothing to analyze). Ignored
// files (.gitignore) are never reported. Paths are read NUL-delimited so
// spaces, Unicode and unusual characters survive intact.
type changedScope struct {
	Mode     string   `json:"mode"`
	Base     string   `json:"base,omitempty"`
	RepoRoot string   `json:"-"`
	Files    []string `json:"-"`
	// Discovered counts changed paths anywhere in the repository; OutsideScope
	// counts those discarded because they are outside the analysis root.
	Discovered   int `json:"discovered"`
	OutsideScope int `json:"outsideScope"`
}

func gitOutput(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out, nil
}

// gitRepoRoot returns the absolute, symlink-resolved repository root for dir.
func gitRepoRoot(dir string) (string, error) {
	out, err := gitOutput(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(string(out))
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	return root, nil
}

func splitNUL(b []byte) []string {
	var out []string
	for _, part := range bytes.Split(b, []byte{0}) {
		if len(part) > 0 {
			out = append(out, string(part))
		}
	}
	return out
}

// resolveBaseRef accepts a ref as given, falling back to origin/<ref> (the
// usual shape of GITHUB_BASE_REF in a CI checkout).
func resolveBaseRef(root, base string) (string, error) {
	// A ref is passed to git as an argument; never let it be read as an option.
	if strings.HasPrefix(base, "-") || strings.ContainsAny(base, "\x00\n\r") {
		return "", fmt.Errorf("invalid base ref %q", base)
	}
	for _, candidate := range []string{base, "origin/" + base} {
		if _, err := gitOutput(root, "rev-parse", "--verify", "--quiet", "--end-of-options", candidate+"^{commit}"); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("base ref %q not found (fetch it, or pass --base with an existing ref)", base)
}

// discoverChangedFiles enumerates changed files for the analysis root path.
// A failure is returned as an error and must never be treated as "no changes".
func discoverChangedFiles(path, base string) (*changedScope, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = resolved
	}
	root, err := gitRepoRoot(absPath)
	if err != nil {
		return nil, err
	}
	scope := &changedScope{RepoRoot: root}
	var repoPaths []string
	if base != "" {
		ref, err := resolveBaseRef(root, base)
		if err != nil {
			return nil, err
		}
		scope.Mode, scope.Base = changedModeBase, ref
		out, err := gitOutput(root, "diff", "--name-only", "-z", "--no-ext-diff", "--diff-filter=ACMR", ref+"...HEAD")
		if err != nil {
			return nil, err
		}
		repoPaths = splitNUL(out)
	} else {
		scope.Mode = changedModeWorkingTree
		var out []byte
		if _, headErr := gitOutput(root, "rev-parse", "--verify", "--quiet", "HEAD"); headErr == nil {
			out, err = gitOutput(root, "diff", "--name-only", "-z", "--no-ext-diff", "--diff-filter=ACMR", "HEAD")
		} else {
			// Unborn branch: everything staged is new.
			out, err = gitOutput(root, "diff", "--cached", "--name-only", "-z", "--no-ext-diff", "--diff-filter=ACMR")
		}
		if err != nil {
			return nil, err
		}
		repoPaths = splitNUL(out)
		untracked, err := gitOutput(root, "ls-files", "--others", "--exclude-standard", "-z", "--full-name")
		if err != nil {
			return nil, err
		}
		repoPaths = append(repoPaths, splitNUL(untracked)...)
	}
	repoPaths = uniqueSorted(repoPaths)
	scope.Discovered = len(repoPaths)

	prefix, err := filepath.Rel(root, absPath)
	if err != nil {
		return nil, err
	}
	prefix = filepath.ToSlash(prefix)
	if prefix == ".." || strings.HasPrefix(prefix, "../") {
		return nil, fmt.Errorf("analysis path %s is outside repository %s", path, root)
	}
	for _, rp := range repoPaths {
		rel, ok := withinPrefix(rp, prefix)
		if !ok {
			scope.OutsideScope++
			continue
		}
		scope.Files = append(scope.Files, rel)
	}
	return scope, nil
}

// withinPrefix converts a repo-relative path to one relative to prefix using
// segment boundaries ("app" does not contain "apple/x").
func withinPrefix(repoPath, prefix string) (string, bool) {
	repoPath = filepath.ToSlash(repoPath)
	if prefix == "." || prefix == "" {
		return repoPath, true
	}
	if strings.HasPrefix(repoPath, prefix+"/") {
		return strings.TrimPrefix(repoPath, prefix+"/"), true
	}
	return "", false
}

// gitChangedFiles is the path-only form used by callers that do not report
// discovery metadata.
func gitChangedFiles(path, base string) ([]string, error) {
	scope, err := discoverChangedFiles(path, base)
	if err != nil {
		return nil, err
	}
	return scope.Files, nil
}

func gitDiffBytes(path, base string) ([]byte, error) {
	args := []string{"diff", "--no-ext-diff"}
	if base != "" {
		root, err := gitRepoRoot(path)
		if err != nil {
			return nil, err
		}
		ref, err := resolveBaseRef(root, base)
		if err != nil {
			return nil, err
		}
		args = append(args, ref+"...HEAD")
	} else {
		args = append(args, "HEAD")
	}
	// Restrict the diff to the analysis root so reviewability agrees with
	// changed-file scope for nested workspaces.
	args = append(args, "--", ".")
	return gitOutput(path, args...)
}

// safeScopedFile resolves rel under root and refuses anything that escapes
// the root, including through symlinks. It returns the absolute path and a
// skip reason ("" when the file may be read).
func safeScopedFile(root, rel string) (string, string) {
	full := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Lstat(full)
	if err != nil {
		return full, "missing"
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return full, "symlink"
	}
	if !info.Mode().IsRegular() {
		return full, "not-regular"
	}
	// Resolve symlinked parent directories too, so a link inside the tree
	// cannot redirect a read outside it.
	absRoot, err1 := filepath.EvalSymlinks(root)
	absFull, err2 := filepath.EvalSymlinks(full)
	if err1 != nil || err2 != nil {
		return full, "unresolvable"
	}
	absRoot, err1 = filepath.Abs(absRoot)
	absFull, err2 = filepath.Abs(absFull)
	if err1 != nil || err2 != nil {
		return full, "unresolvable"
	}
	if r, err := filepath.Rel(absRoot, absFull); err != nil || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return full, "outside-scope"
	}
	return full, ""
}

func uniqueSorted(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func affectedWorkspaces(files, workspaces []string) []string {
	if len(files) == 0 {
		return nil
	}
	var affected []string
	for _, ws := range workspaces {
		if ws == "." {
			affected = append(affected, ws)
			continue
		}
		prefix := filepath.ToSlash(ws)
		if idx := strings.Index(prefix, "*"); idx >= 0 {
			prefix = prefix[:idx]
		}
		prefix = strings.TrimSuffix(prefix, "/")
		for _, file := range files {
			if file == prefix || strings.HasPrefix(file, prefix+"/") {
				affected = append(affected, ws)
				break
			}
		}
	}
	return uniqueSorted(affected)
}
