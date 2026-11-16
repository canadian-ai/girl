package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Changed-file discovery and exclusion policy (issues #121, #122).

func writeFixture(t *testing.T, root, rel, content string) string {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return full
}

func writeConfig(t *testing.T, root, body string) {
	t.Helper()
	writeFixture(t, root, ".girl/config.yaml", "version: \"1\"\n"+body)
}

func requirePython3(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.invalid",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.invalid", "GIT_CONFIG_GLOBAL=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func initRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	return dir
}

func TestChangedFilesIncludeUntrackedAndRespectIgnore(t *testing.T) {
	dir := initRepo(t)
	writeFixture(t, dir, ".gitignore", "ignored.go\n")
	writeFixture(t, dir, "committed.go", "package x\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "init")
	writeFixture(t, dir, "committed.go", "package x\n\nfunc A() {}\n")
	writeFixture(t, dir, "staged.go", "package x\n")
	git(t, dir, "add", "staged.go")
	writeFixture(t, dir, "untracked new.go", "package x\n")
	writeFixture(t, dir, "ignored.go", "package x\n")
	scope, err := discoverChangedFiles(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"committed.go": true, "staged.go": true, "untracked new.go": true}
	if len(scope.Files) != len(want) {
		t.Fatalf("files = %q", scope.Files)
	}
	for _, f := range scope.Files {
		if !want[f] {
			t.Fatalf("unexpected %q in %q", f, scope.Files)
		}
	}
	if scope.Mode != changedModeWorkingTree {
		t.Fatalf("mode = %s", scope.Mode)
	}
}

func TestChangedFilesFromSubdirectoryAreRelativeToAnalysisRoot(t *testing.T) {
	dir := initRepo(t)
	writeFixture(t, dir, "README.md", "x\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "init")
	writeFixture(t, dir, "apps/web/src/new.ts", "export const x = 1\n")
	writeFixture(t, dir, "apps/webby/other.ts", "export const y = 1\n")
	writeFixture(t, dir, "root.go", "package x\n")
	scope, err := discoverChangedFiles(filepath.Join(dir, "apps", "web"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(scope.Files) != 1 || scope.Files[0] != "src/new.ts" {
		t.Fatalf("files = %q", scope.Files)
	}
	if scope.Discovered != 3 || scope.OutsideScope != 2 {
		t.Fatalf("counts discovered=%d outside=%d", scope.Discovered, scope.OutsideScope)
	}
	// check and next agree on the nested workspace.
	full := filepath.Join(dir, "apps", "web", filepath.FromSlash(scope.Files[0]))
	if _, err := os.Stat(full); err != nil {
		t.Fatalf("resolved path does not exist: %v", err)
	}
}

func TestChangedFilesBaseModeRenamesAndMissingBase(t *testing.T) {
	dir := initRepo(t)
	writeFixture(t, dir, "old_name.go", "package x\n\nfunc Keep() int { return 1 }\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "init")
	git(t, dir, "checkout", "-q", "-b", "feature")
	git(t, dir, "mv", "old_name.go", "new_name.go")
	git(t, dir, "commit", "-q", "-m", "rename")
	scope, err := discoverChangedFiles(dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	if scope.Mode != changedModeBase || len(scope.Files) != 1 || scope.Files[0] != "new_name.go" {
		t.Fatalf("scope = %+v", scope)
	}
	if _, err := discoverChangedFiles(dir, "does-not-exist"); err == nil {
		t.Fatal("a missing base must be an error, not an empty scan")
	}
}

func TestChangedFilesEmptyDiffAndNonRepo(t *testing.T) {
	dir := initRepo(t)
	writeFixture(t, dir, "a.go", "package x\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "init")
	scope, err := discoverChangedFiles(dir, "")
	if err != nil || len(scope.Files) != 0 {
		t.Fatalf("clean tree: files=%v err=%v", scope.Files, err)
	}
	if _, err := discoverChangedFiles(t.TempDir(), ""); err == nil {
		t.Fatal("discovery outside a repository must fail")
	}
}

func TestScopePolicyPatternSemantics(t *testing.T) {
	p, err := newScopePolicy("/repo", []string{"generated", "src/legacy", "packages/*/fixtures", "./build/", "**/snapshots"})
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]bool{
		"generated/a.go":              true,
		"pkg/generated/b.go":          true,
		"generic/c.go":                false, // boundary: not a prefix match
		"src/legacy/old.ts":           true,
		"src/legacy":                  true,
		"lib/src/legacy/x.ts":         false, // anchored pattern
		"src/legacyish/x.ts":          false,
		"packages/ui/fixtures/f.ts":   true,
		"packages/ui/src/f.ts":        false,
		"build/out.js":                true,
		"deep/a/b/snapshots/s.ts":     true,
		"src/app.ts":                  false,
		"packages/ui/fixtures-x/a.ts": false,
	}
	for rel, want := range cases {
		if got := p.ExcludedRel(rel); got != want {
			t.Errorf("%s: excluded=%v want %v", rel, got, want)
		}
	}
	if _, err := newScopePolicy("/repo", []string{"[bad"}); err == nil {
		t.Fatal("invalid pattern must be rejected")
	}
}

func TestBaseRefCannotInjectGitOptions(t *testing.T) {
	dir := initRepo(t)
	writeFixture(t, dir, "a.go", "package x\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "init")
	marker := filepath.Join(dir, "pwned")
	for _, base := range []string{"--output=" + marker, "-h", "--upload-pack=touch " + marker} {
		if _, err := discoverChangedFiles(dir, base); err == nil {
			t.Fatalf("base %q accepted", base)
		}
		if _, err := gitDiffBytes(dir, base); err == nil {
			t.Fatalf("diff base %q accepted", base)
		}
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("git option injection wrote a file")
	}
}

func TestConfigIsNotInheritedFromParentsOutsideARepository(t *testing.T) {
	parent := t.TempDir() // not a git repository
	writeConfig(t, parent, "analysis:\n  exclude:\n    - child\n")
	child := filepath.Join(parent, "child")
	writeFixture(t, child, "a.go", "package x\n")
	if got := findProjectConfigDir(child); got != "" {
		t.Fatalf("picked up config from unrelated parent %s", got)
	}
}
