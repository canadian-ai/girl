package commands

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func gitChangedFiles(path, base string) ([]string, error) {
	args := []string{"-C", path, "diff", "--name-only", "--diff-filter=ACMR"}
	if base != "" {
		args = append(args, base+"...HEAD")
	} else {
		args = append(args, "HEAD")
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	files := splitLines(string(out))

	if base == "" {
		staged, stagedErr := exec.Command("git", "-C", path, "diff", "--cached", "--name-only", "--diff-filter=ACMR", "HEAD").Output()
		if stagedErr == nil {
			files = append(files, splitLines(string(staged))...)
		}
	}
	return uniqueSorted(files), nil
}

func gitDiffBytes(path, base string) ([]byte, error) {
	args := []string{"-C", path, "diff", "--no-ext-diff"}
	if base != "" {
		args = append(args, base+"...HEAD")
	} else {
		args = append(args, "HEAD")
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return out, nil
}

func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, filepath.ToSlash(line))
		}
	}
	return out
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
