package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Exclusion parity between check and next, and gate hardening.

func exclusionFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeConfig(t, dir, "analysis:\n  changed_only: false\n  exclude:\n    - src/legacy\n    - fixtures\nverify:\n  test:\n    - \"true\"\n")
	writeFixture(t, dir, "go.mod", "module example.com/x\n\ngo 1.22\n")
	big := "package x\n\nfunc Big(a, b, c, d, e, f, g int) int {\n"
	for i := 0; i < 90; i++ {
		big += "\tif a > b {\n\t\ta++\n\t}\n"
	}
	big += "\treturn a\n}\n"
	writeFixture(t, dir, "src/legacy/old/big.go", big)
	writeFixture(t, dir, "testing/fixtures/big.go", big)
	writeFixture(t, dir, "src/app/keep.go", big)
	return dir
}

func TestExclusionsApplyToCheckAndNext(t *testing.T) {
	dir := exclusionFixture(t)
	res, err := runCheck(dir, checkOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	assertNoExcluded := func(where string, files []string) {
		t.Helper()
		for _, f := range files {
			slash := filepath.ToSlash(f)
			if strings.Contains(slash, "src/legacy") || strings.Contains(slash, "fixtures") {
				t.Fatalf("%s reported excluded file %s", where, f)
			}
		}
	}
	var checkFiles []string
	keep := false
	for _, d := range res.Diagnostics {
		checkFiles = append(checkFiles, d.File)
		if strings.Contains(filepath.ToSlash(d.File), "src/app/keep.go") {
			keep = true
		}
	}
	assertNoExcluded("check", checkFiles)
	if !keep {
		t.Fatalf("non-excluded file lost: %+v", res.Diagnostics)
	}
	if len(res.Scope.Exclusions) != 2 || res.Scope.ConfigPath == "" {
		t.Fatalf("effective exclusions not reported: %+v", res.Scope)
	}

	next, err := buildNext(dir, nextOptions{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	var nextFiles []string
	for _, item := range next.Items {
		nextFiles = append(nextFiles, item.File)
	}
	assertNoExcluded("next", nextFiles)
	if len(nextFiles) == 0 {
		t.Fatal("next lost non-excluded findings")
	}
}

func TestExclusionsApplyToNestedWorkspaceTarget(t *testing.T) {
	requireGit(t)
	dir := exclusionFixture(t)
	// Config inheritance is bounded by the repository root.
	git(t, dir, "init", "-q", "-b", "main")
	// Analyze a nested directory: the root config's anchored exclusion still applies.
	res, err := analyzeCheckScope(filepath.Join(dir, "src"), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Files {
		if strings.Contains(filepath.ToSlash(f.Path), "legacy") {
			t.Fatalf("nested target ignored root exclusions: %s", f.Path)
		}
	}
}

func TestFullScanDoesNotReadThroughSymlinks(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "analysis:\n  changed_only: false\n")
	writeFixture(t, dir, "go.mod", "module example.com/x\n\ngo 1.22\n")
	writeFixture(t, dir, "real.go", "package x\n")
	outside := t.TempDir()
	// Malformed on purpose: if it were read it would raise a blocking diagnostic.
	writeFixture(t, outside, "outside.go", "package secret\n\nfunc (\n")
	if err := os.Symlink(filepath.Join(outside, "outside.go"), filepath.Join(dir, "link.go")); err != nil {
		t.Skip("symlinks unsupported")
	}
	res, err := analyzeCheckScope(dir, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != 1 || len(res.Diagnostics) != 0 {
		t.Fatalf("symlinked file was analyzed: files=%d diags=%+v", len(res.Files), res.Diagnostics)
	}
}

func TestChangeCannotRewriteItsOwnComplexityBaseline(t *testing.T) {
	dir := initRepo(t)
	writeFixture(t, dir, "package.json", "{}")
	writeFixture(t, dir, "app.ts", "export function value(flag: boolean) { if (flag) return 1; return 0 }\n")
	writeBaselineForTest(t, dir, ".girl/complexity-baseline.json")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "trusted baseline")
	git(t, dir, "checkout", "-q", "-b", "feature")
	// The change makes the function more complex AND regenerates the baseline to hide it.
	writeFixture(t, dir, "app.ts", "export function value(a: boolean, b: boolean, c: boolean) { if (a) return 1; if (b) return 2; if (c) return 3; return 0 }\n")
	writeBaselineForTest(t, dir, ".girl/complexity-baseline.json")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "sneaky")

	changed, err := discoverChangedFiles(dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	policy, _ := newScopePolicy(dir, nil)
	res, err := runCheckComplexityWithPolicy(dir, regressionConfig(".girl/complexity-baseline.json"), changed, true, true, policy)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "fail" || res.BaselineSource != "base:main" || res.Regressions == 0 {
		t.Fatalf("rewritten baseline hid the regression: %+v", res)
	}

	writeConfig(t, dir, "analysis:\n  changed_only: true\nverify:\n  test:\n    - \"true\"\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "config")
	full, err := runCheck(dir, checkOptions{Base: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if len(full.PolicyFilesChanged) != 2 {
		t.Fatalf("policy file changes not flagged: %v", full.PolicyFilesChanged)
	}
}

func TestVerificationOutputIsSanitized(t *testing.T) {
	dir := t.TempDir()
	secret := strings.Join([]string{"Synthetic", "-Pa55word-", "Value"}, "")
	writeConfig(t, dir, "analysis:\n  changed_only: false\nverify:\n  test:\n    - \"echo DB_PASSWORD="+secret+"; exit 1\"\n")
	res, err := runCheck(dir, checkOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Verification) != 1 || res.Verification[0].Status != CommandFail {
		t.Fatalf("verification = %+v", res.Verification)
	}
	if strings.Contains(res.Verification[0].Output, secret) || !strings.Contains(res.Verification[0].Output, "DB_PASSWORD") {
		t.Fatalf("output not sanitized: %q", res.Verification[0].Output)
	}
}
