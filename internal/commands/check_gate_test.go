package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canadian-ai/girl/internal/complexity"
	"github.com/canadian-ai/girl/internal/ir"
)

// Fail-closed gate contract (issues #119, #120, #125): every scenario that
// used to look like success without evidence is pinned here.

func hasBlocking(diags []ir.Diagnostic, code string) *ir.Diagnostic {
	for i, d := range diags {
		if d.Code == code && ir.IsBlocking(d) {
			return &diags[i]
		}
	}
	return nil
}

func TestCheckFailsOnMalformedPythonEvenWithoutVerification(t *testing.T) {
	requirePython3(t)
	dir := t.TempDir()
	writeConfig(t, dir, "analysis:\n  changed_only: false\n")
	writeFixture(t, dir, "ok.py", "def fine():\n    return 1\n")
	writeFixture(t, dir, "bad.py", "def ok():\n    return 1\n\ndef bad(:\n    pass\n")
	res, err := runCheck(dir, checkOptions{All: true, NoVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "fail" || res.Verdict != VerdictFailed {
		t.Fatalf("status=%s verdict=%s reasons=%v", res.Status, res.Verdict, res.Reasons)
	}
	d := hasBlocking(res.Diagnostics, "py.syntax-error")
	if d == nil {
		t.Fatalf("missing blocking py.syntax-error: %+v", res.Diagnostics)
	}
	if d.Line != 4 || d.Span == nil || d.Span.StartCol == 0 {
		t.Fatalf("syntax error location not preserved: line=%d span=%+v", d.Line, d.Span)
	}
	if res.BlockingDiagnostics != 1 {
		t.Fatalf("blocking count = %d", res.BlockingDiagnostics)
	}
}

func TestCheckFailsOnMalformedGo(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "analysis:\n  changed_only: false\n")
	writeFixture(t, dir, "go.mod", "module example.com/x\n\ngo 1.22\n")
	writeFixture(t, dir, "main.go", "package main\n\nfunc main() {\n\tif {\n}\n")
	res, err := runCheck(dir, checkOptions{All: true, NoVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	d := hasBlocking(res.Diagnostics, "go.syntax-error")
	if res.Status != "fail" || d == nil || d.Line < 4 {
		t.Fatalf("status=%s diags=%+v", res.Status, res.Diagnostics)
	}
}

func TestRecoveringGrammarFindingsAreAdvisoryNotBlocking(t *testing.T) {
	for _, tc := range []struct{ name, marker, file, src, code string }{
		{"ts malformed", "package.json", "app.ts", "export function f( {\n  return 1\n", "ts.parse-recovery"},
		{"rust malformed", "Cargo.toml", "src/lib.rs", "pub fn f( {\n    1\n", "rust.parse-recovery"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, "analysis:\n  changed_only: false\nverify:\n  test:\n    - \"true\"\n")
			writeFixture(t, dir, tc.marker, "{}")
			if tc.marker == "Cargo.toml" {
				writeFixture(t, dir, tc.marker, "[package]\nname = \"x\"\nversion = \"0.1.0\"\n")
			}
			writeFixture(t, dir, tc.file, tc.src)
			res, err := runCheck(dir, checkOptions{All: true})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, d := range res.Diagnostics {
				if d.Code == tc.code {
					found = true
					if ir.IsBlocking(d) || d.Confidence != "low" {
						t.Fatalf("parse recovery must be advisory and low confidence: %+v", d)
					}
				}
			}
			if !found || res.BlockingDiagnostics != 0 || res.Status == "fail" {
				t.Fatalf("found=%v status=%s reasons=%v", found, res.Status, res.Reasons)
			}
		})
	}
}

func TestValidJSXWithBareAmpersandDoesNotFail(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "analysis:\n  changed_only: false\nverify:\n  test:\n    - \"true\"\n")
	writeFixture(t, dir, "package.json", "{}")
	writeFixture(t, dir, "page.tsx", "export default function Page() {\n  return <h2>Billing & usage</h2>\n}\n")
	res, err := runCheck(dir, checkOptions{All: true, Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.BlockingDiagnostics != 0 || res.Verdict == VerdictFailed {
		t.Fatalf("valid JSX failed the gate: verdict=%s reasons=%v", res.Verdict, res.Reasons)
	}
}

func TestCheckValidPythonWithAdvisoryFindingsDoesNotFail(t *testing.T) {
	requirePython3(t)
	dir := t.TempDir()
	writeConfig(t, dir, "analysis:\n  changed_only: false\nverify:\n  test:\n    - \"true\"\n")
	var b strings.Builder
	b.WriteString("def busy(x):\n")
	for i := 0; i < 12; i++ {
		b.WriteString("    if x == " + string(rune('a'+i)) + "':\n        return 1\n")
	}
	src := strings.ReplaceAll(b.String(), "== ", "== '")
	writeFixture(t, dir, "busy.py", src+"    return 0\n")
	res, err := runCheck(dir, checkOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status == "fail" || res.BlockingDiagnostics != 0 {
		t.Fatalf("advisory findings must not fail: status=%s reasons=%v", res.Status, res.Reasons)
	}
	found := false
	for _, d := range res.Diagnostics {
		found = found || (d.Code == "py.high-complexity" && d.Severity == ir.SeverityHigh)
	}
	if !found {
		t.Fatalf("fixture did not produce the advisory finding: %+v", res.Diagnostics)
	}
}

func TestVerificationStates(t *testing.T) {
	cases := []struct {
		name         string
		verify       string
		opts         checkOptions
		wantSummary  string
		wantRun      bool
		wantStatus   string
		wantVerdict  string
		wantExecuted int
	}{
		{"zero commands advisory", "", checkOptions{All: true}, VerificationNotConfigured, false, "warn", VerdictUnverified, 0},
		{"zero commands strict", "", checkOptions{All: true, Strict: true}, VerificationNotConfigured, false, "fail", VerdictBlocked, 0},
		{"no-verify", "verify:\n  test:\n    - \"true\"\n", checkOptions{All: true, NoVerify: true}, VerificationSkipped, false, "warn", VerdictUnverified, 0},
		{"success", "verify:\n  test:\n    - \"true\"\n", checkOptions{All: true}, VerificationVerified, true, "", "", 1},
		{"failure", "verify:\n  test:\n    - \"exit 3\"\n", checkOptions{All: true}, VerificationFailed, true, "fail", VerdictFailed, 1},
		{"missing command", "verify:\n  test:\n    - \"girl-definitely-missing-cmd\"\n", checkOptions{All: true}, VerificationFailed, false, "fail", VerdictFailed, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, "analysis:\n  changed_only: false\n"+tc.verify)
			res, err := runCheck(dir, tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			s := res.VerificationSummary
			if s.Status != tc.wantSummary || res.VerificationRun != tc.wantRun || s.Executed != tc.wantExecuted {
				t.Fatalf("summary=%+v verificationRun=%v", s, res.VerificationRun)
			}
			if tc.wantStatus != "" && res.Status != tc.wantStatus {
				t.Fatalf("status=%s want %s reasons=%v", res.Status, tc.wantStatus, res.Reasons)
			}
			if tc.wantVerdict != "" && res.Verdict != tc.wantVerdict {
				t.Fatalf("verdict=%s want %s reasons=%v", res.Verdict, tc.wantVerdict, res.Reasons)
			}
			if tc.opts.NoVerify && res.Verdict == VerdictVerified {
				t.Fatal("--no-verify must never yield a verified verdict")
			}
		})
	}
}

func TestVerificationRecordsExitCodesAndFailFastRemainder(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "analysis:\n  changed_only: false\nverify:\n  typecheck:\n    - \"true\"\n  lint:\n    - \"exit 4\"\n  test:\n    - \"true\"\n")
	res, err := runCheck(dir, checkOptions{All: true, FailFast: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Verification) != 3 {
		t.Fatalf("expected all configured commands to be reported: %+v", res.Verification)
	}
	got := []string{res.Verification[0].Status, res.Verification[1].Status, res.Verification[2].Status}
	if got[0] != CommandPass || got[1] != CommandFail || got[2] != CommandNotRun {
		t.Fatalf("statuses = %v", got)
	}
	if res.Verification[1].ExitCode == nil || *res.Verification[1].ExitCode != 4 {
		t.Fatalf("exit code not recorded: %+v", res.Verification[1])
	}
	if res.Verification[0].Source != ".girl/config.yaml" {
		t.Fatalf("provenance missing: %+v", res.Verification[0])
	}
	s := res.VerificationSummary
	if s.Executed != 2 || s.NotRun != 1 || s.Status != VerificationFailed {
		t.Fatalf("summary = %+v", s)
	}
}

func complexityFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, dir, "package.json", "{}")
	writeFixture(t, dir, "app.ts", "export function value(flag: boolean) { if (flag) return 1; return 0 }\n")
	return dir
}

func regressionConfig(baseline string) *GirlProjectConfig {
	return &GirlProjectConfig{
		Analysis:   GirlAnalysisConfig{Exclude: []string{"node_modules"}},
		Complexity: GirlComplexityConfig{Max: 10, Baseline: baseline, FailOn: "regression"},
	}
}

func TestComplexityMissingBaselineIsUnenforcedNotSkipped(t *testing.T) {
	dir := complexityFixture(t)
	policy, _ := newScopePolicy(dir, nil)
	advisory, err := runCheckComplexityWithPolicy(dir, regressionConfig(".girl/missing.json"), nil, false, false, policy)
	if err != nil {
		t.Fatal(err)
	}
	if advisory.Status != "unenforced" || advisory.Enforcement != "unenforced" || advisory.Remediation == "" {
		t.Fatalf("advisory = %+v", advisory)
	}
	strict, err := runCheckComplexityWithPolicy(dir, regressionConfig(".girl/missing.json"), nil, false, true, policy)
	if err != nil {
		t.Fatal(err)
	}
	if strict.Status != "blocked" {
		t.Fatalf("strict = %+v", strict)
	}
	if _, err := os.Stat(filepath.Join(dir, ".girl", "missing.json")); !os.IsNotExist(err) {
		t.Fatal("a gate must never write a baseline")
	}
	g := &gate{}
	applyComplexityGate(g, strict, true)
	if g.status() != "fail" || g.verdict() != VerdictBlocked {
		t.Fatalf("strict missing baseline must block: %s/%s", g.status(), g.verdict())
	}
}

func TestComplexityCorruptAndIncompatibleBaselines(t *testing.T) {
	dir := complexityFixture(t)
	policy, _ := newScopePolicy(dir, nil)
	writeFixture(t, dir, ".girl/corrupt.json", "{not json")
	writeFixture(t, dir, ".girl/old.json", `{"schemaVersion":"girl.io/complexity/v0","metric":"cyclomatic","functions":[]}`)
	for _, name := range []string{"corrupt", "old"} {
		res, err := runCheckComplexityWithPolicy(dir, regressionConfig(".girl/"+name+".json"), nil, false, true, policy)
		if err != nil {
			t.Fatalf("%s: corrupt baselines must be reported, not crash: %v", name, err)
		}
		if res.Status != "blocked" || res.Reason == "" {
			t.Fatalf("%s: %+v", name, res)
		}
	}
}

func TestComplexityBaselineEnforcedWithProvenance(t *testing.T) {
	dir := complexityFixture(t)
	policy, _ := newScopePolicy(dir, nil)
	cfg := regressionConfig(".girl/baseline.json")
	writeBaselineForTest(t, dir, ".girl/baseline.json")
	res, err := runCheckComplexityWithPolicy(dir, cfg, nil, false, true, policy)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "pass" || res.Enforcement != "enforced" || res.BaselineSHA256 == "" || !res.BaselineFound {
		t.Fatalf("enforced = %+v", res)
	}
	// Increase complexity: a regression must fail.
	writeFixture(t, dir, "app.ts", "export function value(a: boolean, b: boolean, c: boolean) { if (a) return 1; if (b) return 2; if (c) return 3; return 0 }\n")
	res, err = runCheckComplexityWithPolicy(dir, cfg, nil, false, true, policy)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "fail" || res.Regressions == 0 {
		t.Fatalf("regression not detected: %+v", res)
	}
}

func TestComplexityWithoutPackageJSONIsNotApplicableNotClean(t *testing.T) {
	dir := t.TempDir()
	policy, _ := newScopePolicy(dir, nil)
	res, err := runCheckComplexityWithPolicy(dir, regressionConfig(".girl/x.json"), nil, false, true, policy)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.Status != "not-applicable" || res.Enforcement != "unsupported" || res.Reason == "" {
		t.Fatalf("result = %+v", res)
	}
}

func TestCheckReportsDiscoveryFailureAndFallsBackToFullScan(t *testing.T) {
	requireGit(t)
	dir := t.TempDir() // not a repository
	writeConfig(t, dir, "analysis:\n  changed_only: true\nverify:\n  test:\n    - \"true\"\n")
	writeFixture(t, dir, "go.mod", "module example.com/x\n\ngo 1.22\n")
	writeFixture(t, dir, "a.go", "package x\n")
	res, err := runCheck(dir, checkOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Scope.DiscoveryError == "" || res.ChangedOnly || res.Scope.Analyzed == 0 || res.Status == "pass" {
		t.Fatalf("discovery failure hidden: scope=%+v status=%s", res.Scope, res.Status)
	}
}

func TestChangedScopeSkipsSymlinksAndCountsReasons(t *testing.T) {
	dir := initRepo(t)
	writeFixture(t, dir, "README.md", "x\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "init")
	outside := t.TempDir()
	writeFixture(t, outside, "secret.go", "package secret\n")
	if err := os.Symlink(filepath.Join(outside, "secret.go"), filepath.Join(dir, "link.go")); err != nil {
		t.Skip("symlinks unsupported")
	}
	writeFixture(t, dir, "notes.txt", "x\n")
	writeFixture(t, dir, "real.go", "package x\n")
	scope, err := discoverChangedFiles(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	policy, _ := newScopePolicy(dir, nil)
	res, cs, err := analyzeCheckScopeWithPolicy(dir, scope, true, policy)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Files {
		if strings.Contains(f.Path, "secret") || strings.HasSuffix(f.Path, "link.go") {
			t.Fatalf("followed symlink outside scope: %s", f.Path)
		}
	}
	if cs.Skipped["symlink"] != 1 || cs.Skipped["unsupported"] < 1 || cs.Analyzed != 1 {
		t.Fatalf("scope = %+v", cs)
	}
}

func TestStrictModeRejectsInvalidConfiguration(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "analysis:\n  changed_only: false\ncomplexity:\n  max: lots\nverify:\n  test:\n    - \"true\"\n")
	res, err := runCheck(dir, checkOptions{All: true, Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ConfigProblems) == 0 || res.Status != "fail" || res.Verdict != VerdictBlocked {
		t.Fatalf("problems=%v status=%s verdict=%s", res.ConfigProblems, res.Status, res.Verdict)
	}
}

func writeBaselineForTest(t *testing.T, dir, rel string) {
	t.Helper()
	report, err := complexity.Analyze(dir, complexity.Options{Language: "auto", Threshold: 10})
	if err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeComplexityReport(full, report); err != nil {
		t.Fatal(err)
	}
}
