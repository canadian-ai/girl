package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/canadian-ai/girl/internal/analyzer"
	"github.com/canadian-ai/girl/internal/complexity"
	"github.com/canadian-ai/girl/internal/diffstats"
	"github.com/canadian-ai/girl/internal/goanalysis"
	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/pythonanalysis"
	"github.com/canadian-ai/girl/internal/redact"
	"github.com/canadian-ai/girl/internal/reviewability"
	"github.com/canadian-ai/girl/internal/rustanalysis"
	"github.com/canadian-ai/girl/internal/shared"
	"github.com/urfave/cli/v2"
)

// Evidence verdicts. GIRL never treats the absence of a failure as proof of
// success: a check is "verified" only when every required piece of evidence
// was produced and passed.
const (
	VerdictVerified   = "verified"   // all required checks executed and passed
	VerdictFailed     = "failed"     // a required check failed
	VerdictBlocked    = "blocked"    // a prerequisite or policy requirement is missing (strict mode)
	VerdictUnverified = "unverified" // required evidence was not produced (advisory mode or explicit opt-out)
)

// Verification summary states.
const (
	VerificationVerified      = "verified"
	VerificationFailed        = "failed"
	VerificationNotConfigured = "not-configured"
	VerificationSkipped       = "skipped"
	VerificationBlocked       = "blocked"
)

// Per-command verification states.
const (
	CommandPass   = "pass"
	CommandFail   = "fail"    // ran and exited non-zero
	CommandError  = "error"   // could not start, or shell reported command not found / not executable
	CommandNotRun = "not-run" // skipped by --fail-fast after an earlier failure
)

type CheckVerificationResult struct {
	Role       string `json:"role"`
	Command    string `json:"command"`
	Status     string `json:"status"`
	ExitCode   *int   `json:"exitCode,omitempty"`
	Source     string `json:"source,omitempty"`
	Confidence string `json:"confidence,omitempty"`
	Output     string `json:"output,omitempty"`
}

type CheckVerificationSummary struct {
	Status         string `json:"status"`
	Requested      bool   `json:"requested"`
	Source         string `json:"source,omitempty"`
	DiscoveryError string `json:"discoveryError,omitempty"`
	Configured     int    `json:"configured"`
	Executed       int    `json:"executed"`
	Passed         int    `json:"passed"`
	Failed         int    `json:"failed"`
	Errored        int    `json:"errored"`
	NotRun         int    `json:"notRun"`
}

type CheckComplexityResult struct {
	Status string `json:"status"`
	// Enforcement is "enforced" (baseline ratchet applied), "threshold",
	// "unenforced" (policy could not be applied), "disabled" or "unsupported".
	Enforcement    string `json:"enforcement"`
	Engine         string `json:"engine,omitempty"`
	Policy         string `json:"policy,omitempty"`
	Reason         string `json:"reason,omitempty"`
	Remediation    string `json:"remediation,omitempty"`
	Baseline       string `json:"baseline,omitempty"`
	BaselineFound  bool   `json:"baselineFound"`
	BaselineSHA256 string `json:"baselineSha256,omitempty"`
	// BaselineSource is "base:<ref>" when the baseline was read from the
	// comparison base (so the change under review cannot edit its own
	// yardstick) or "working-tree".
	BaselineSource        string `json:"baselineSource,omitempty"`
	BaselineSchemaVersion string `json:"baselineSchemaVersion,omitempty"`
	Threshold             int    `json:"threshold"`
	Files                 int    `json:"files"`
	Functions             int    `json:"functions"`
	OverThreshold         int    `json:"overThreshold"`
	Regressions           int    `json:"regressions"`
}

// CheckScope reports what was actually analyzed, so an empty scan can never
// be mistaken for a clean one.
type CheckScope struct {
	Mode           string         `json:"mode"`
	Base           string         `json:"base,omitempty"`
	DiscoveryError string         `json:"discoveryError,omitempty"`
	Discovered     int            `json:"discovered"`
	OutsideScope   int            `json:"outsideScope,omitempty"`
	Analyzed       int            `json:"analyzed"`
	Skipped        map[string]int `json:"skipped,omitempty"`
	Exclusions     []string       `json:"exclusions,omitempty"`
	ConfigPath     string         `json:"config,omitempty"`
}

type CheckResult struct {
	SpecVersion         string                    `json:"specversion"`
	Path                string                    `json:"path"`
	Profiles            []string                  `json:"profiles,omitempty"`
	Strict              bool                      `json:"strict"`
	ChangedOnly         bool                      `json:"changedOnly"`
	ChangedFiles        []string                  `json:"changedFiles,omitempty"`
	Workspaces          []string                  `json:"affectedWorkspaces,omitempty"`
	Scope               *CheckScope               `json:"scope,omitempty"`
	Diagnostics         []ir.Diagnostic           `json:"diagnostics,omitempty"`
	BlockingDiagnostics int                       `json:"blockingDiagnostics"`
	Complexity          *CheckComplexityResult    `json:"complexity,omitempty"`
	Reviewability       *ir.ReviewabilityResult   `json:"reviewability,omitempty"`
	Preflight           []PreflightCheck          `json:"preflight,omitempty"`
	Verification        []CheckVerificationResult `json:"verification,omitempty"`
	VerificationSummary *CheckVerificationSummary `json:"verificationSummary"`
	ConfigProblems      []string                  `json:"configProblems,omitempty"`
	// PolicyFilesChanged lists gate policy files (config, baseline) modified
	// by the change under review. They are honored but flagged for review.
	PolicyFilesChanged []string `json:"policyFilesChanged,omitempty"`
	Status             string   `json:"status"`
	Verdict            string   `json:"verdict"`
	Reasons            []string `json:"reasons,omitempty"`
	// VerificationRun is true only when at least one verification command
	// actually executed. Requesting verification is not evidence.
	VerificationRun bool `json:"verificationRun"`
}

// gate accumulates the overall outcome. fail and block both produce a
// non-zero exit; warn and unverified do not.
type gate struct {
	failed, blocked, unverified, warned bool
	reasons                             []string
}

func (g *gate) fail(format string, args ...any) {
	g.failed = true
	g.reasons = append(g.reasons, "fail: "+fmt.Sprintf(format, args...))
}

func (g *gate) block(format string, args ...any) {
	g.blocked = true
	g.reasons = append(g.reasons, "blocked: "+fmt.Sprintf(format, args...))
}

func (g *gate) unverify(format string, args ...any) {
	g.unverified = true
	g.reasons = append(g.reasons, "unverified: "+fmt.Sprintf(format, args...))
}

func (g *gate) warn(format string, args ...any) {
	g.warned = true
	g.reasons = append(g.reasons, "warn: "+fmt.Sprintf(format, args...))
}

// missing records absent evidence: blocking in strict mode, otherwise a
// visible warning that also prevents a verified verdict.
func (g *gate) missing(strict bool, format string, args ...any) {
	if strict {
		g.block(format, args...)
		return
	}
	g.warned = true
	g.unverify(format, args...)
}

func (g *gate) status() string {
	switch {
	case g.failed || g.blocked:
		return "fail"
	case g.warned || g.unverified:
		return "warn"
	default:
		return "pass"
	}
}

func (g *gate) verdict() string {
	switch {
	case g.failed:
		return VerdictFailed
	case g.blocked:
		return VerdictBlocked
	case g.unverified:
		return VerdictUnverified
	default:
		return VerdictVerified
	}
}

func CheckCommand() *cli.Command {
	return &cli.Command{
		Name:      "check",
		Usage:     "Run the project-level GIRL quality contract",
		ArgsUsage: "[path]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "ci", Usage: "Use CI-friendly changed-file defaults (implies --strict)"},
			&cli.BoolFlag{Name: "strict", Usage: "Fail closed: missing verification, baselines or valid config block the gate"},
			&cli.BoolFlag{Name: "changed", Usage: "Force changed-file analysis"},
			&cli.BoolFlag{Name: "all", Usage: "Analyze the entire project"},
			&cli.StringFlag{Name: "base", Usage: "Git base ref for changed-file analysis and reviewability"},
			&cli.BoolFlag{Name: "no-verify", Usage: "Skip project verification commands (result is never 'verified')"},
			&cli.BoolFlag{Name: "fail-fast", Usage: "Stop verification on first failed command"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "text", Usage: "Output format: text or json"},
		},
		Action: func(c *cli.Context) error {
			result, err := runCheck(commandPath(c), checkOptions{
				CI:       c.Bool("ci"),
				Strict:   c.Bool("strict") || c.Bool("ci"),
				Changed:  c.Bool("changed"),
				All:      c.Bool("all"),
				Base:     c.String("base"),
				NoVerify: c.Bool("no-verify"),
				FailFast: c.Bool("fail-fast"),
			})
			if err != nil {
				return err
			}
			if stringFlag(c, "output", "o") == "json" {
				printJSON(result)
			} else {
				printCheckText(result)
			}
			if result.Status == "fail" {
				return fmt.Errorf("GIRL check failed (%s)", result.Verdict)
			}
			return nil
		},
	}
}

type checkOptions struct {
	CI, Strict, Changed, All, NoVerify, FailFast bool
	Base                                         string
}

func runCheck(path string, opts checkOptions) (*CheckResult, error) {
	cfg, err := loadGirlProjectConfig(path)
	if err != nil {
		return nil, fmt.Errorf("load GIRL config: %w", err)
	}
	policy, err := loadScopePolicy(path)
	if err != nil {
		return nil, fmt.Errorf("load GIRL scope policy: %w", err)
	}

	changedOnly := cfg.Analysis.ChangedOnly
	if opts.Changed {
		changedOnly = true
	}
	if opts.All {
		changedOnly = false
	}
	base := opts.Base
	if base == "" && opts.CI {
		base = os.Getenv("GITHUB_BASE_REF")
	}

	g := &gate{}
	result := &CheckResult{
		SpecVersion:    "girl.io/check/v1",
		Path:           path,
		Profiles:       cfg.Profiles,
		Strict:         opts.Strict,
		ChangedOnly:    changedOnly,
		ConfigProblems: cfg.Problems,
	}
	for _, p := range cfg.Problems {
		if opts.Strict {
			g.block("invalid configuration: %s", p)
		} else {
			g.warn("invalid configuration: %s", p)
		}
	}

	var changed *changedScope
	var discoveryErr error
	if changedOnly {
		changed, discoveryErr = discoverChangedFiles(path, base)
		if discoveryErr != nil {
			// Never treat failed discovery as "nothing changed": fall back to
			// a full scan (a superset) and say so.
			changedOnly = false
			result.ChangedOnly = false
			g.warn("changed-file discovery failed, analyzed the full project instead: %v", discoveryErr)
		} else {
			result.ChangedFiles = changed.Files
			result.Workspaces = affectedWorkspaces(changed.Files, cfg.Workspaces)
			baselineRel := filepath.ToSlash(filepath.Clean(cfg.Complexity.Baseline))
			for _, f := range changed.Files {
				if f == ".girl/config.yaml" || (baselineRel != "." && f == baselineRel) {
					result.PolicyFilesChanged = append(result.PolicyFilesChanged, f)
				}
			}
			if len(result.PolicyFilesChanged) > 0 {
				g.warn("gate policy file(s) modified by this change: %s; a change must not relax its own gate without human review", strings.Join(result.PolicyFilesChanged, ", "))
			}
		}
	}

	analysis, scope, err := analyzeCheckScopeWithPolicy(path, changed, changedOnly, policy)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}
	if discoveryErr != nil {
		scope.DiscoveryError = discoveryErr.Error()
	}
	result.Scope = scope
	result.Diagnostics = analysis.Diagnostics
	advisoryHigh := 0
	for _, d := range result.Diagnostics {
		if ir.IsBlocking(d) {
			result.BlockingDiagnostics++
		} else if d.Severity == ir.SeverityHigh {
			advisoryHigh++
		}
	}
	if result.BlockingDiagnostics > 0 {
		g.fail("%d file(s) could not be parsed or analyzed; repair them before refactoring", result.BlockingDiagnostics)
	}
	if advisoryHigh > 0 {
		g.warn("%d high-severity refactoring finding(s)", advisoryHigh)
	}

	complexityResult, err := runCheckComplexityWithPolicy(path, cfg, changed, changedOnly, opts.Strict, policy)
	if err != nil {
		return nil, err
	}
	result.Complexity = complexityResult
	applyComplexityGate(g, complexityResult, opts.Strict)

	review, reviewDiagnostics, err := runCheckReviewability(path, base, cfg)
	if err != nil {
		return nil, err
	}
	if review != nil {
		result.Reviewability = &review.Result
		result.Diagnostics = append(result.Diagnostics, reviewDiagnostics...)
		switch review.Result.Status {
		case "fail":
			g.fail("diff exceeds the reviewability budget")
		case "warn":
			g.warn("diff is close to the reviewability budget")
		}
	}

	preflight := runProjectPreflight(path, cfg)
	result.Preflight = preflight.Checks
	if preflight.Status == "fail" {
		g.fail("project preflight failed")
	} else if preflight.Status == "warn" {
		g.warn("project preflight has warnings")
	}

	summary := &CheckVerificationSummary{
		Requested:      !opts.NoVerify,
		Source:         cfg.VerifySource,
		DiscoveryError: cfg.VerifyDiscoveryError,
		Configured:     countVerifyCommands(cfg),
	}
	if !opts.NoVerify && cfg.VerifyDiscoveryError == "" && summary.Configured > 0 {
		result.Verification = runConfiguredVerification(path, cfg, opts.FailFast)
	}
	summarizeVerification(summary, result.Verification, opts.NoVerify)
	result.VerificationSummary = summary
	result.VerificationRun = summary.Executed > 0
	applyVerificationGate(g, summary, opts.Strict)

	result.Status = g.status()
	result.Verdict = g.verdict()
	result.Reasons = g.reasons
	return result, nil
}

func applyComplexityGate(g *gate, c *CheckComplexityResult, strict bool) {
	if c == nil {
		return
	}
	switch c.Status {
	case "fail":
		g.fail("complexity policy %q failed (%d regressions, %d over threshold)", c.Policy, c.Regressions, c.OverThreshold)
	case "blocked":
		g.block("complexity ratchet cannot be enforced: %s", c.Reason)
	case "unenforced":
		g.missing(strict, "complexity ratchet not enforced: %s", c.Reason)
	}
}

func applyVerificationGate(g *gate, s *CheckVerificationSummary, strict bool) {
	switch s.Status {
	case VerificationFailed:
		g.fail("verification failed (%d failed, %d could not run)", s.Failed, s.Errored)
	case VerificationBlocked:
		g.missing(strict, "verification command discovery failed: %s", s.DiscoveryError)
	case VerificationNotConfigured:
		g.missing(strict, "no verification commands configured or detected; add verify: commands to .girl/config.yaml")
	case VerificationSkipped:
		g.unverify("verification skipped by --no-verify")
	}
}

func countVerifyCommands(cfg *GirlProjectConfig) int {
	n := 0
	for _, cmds := range cfg.Verify {
		n += len(cmds)
	}
	return n
}

func summarizeVerification(s *CheckVerificationSummary, results []CheckVerificationResult, noVerify bool) {
	for _, r := range results {
		switch r.Status {
		case CommandPass:
			s.Executed++
			s.Passed++
		case CommandFail:
			s.Executed++
			s.Failed++
		case CommandError:
			s.Errored++
		case CommandNotRun:
			s.NotRun++
		}
	}
	switch {
	case noVerify:
		s.Status = VerificationSkipped
	case s.DiscoveryError != "":
		s.Status = VerificationBlocked
	case s.Configured == 0:
		s.Status = VerificationNotConfigured
	case s.Failed > 0 || s.Errored > 0:
		s.Status = VerificationFailed
	case s.Passed == s.Configured:
		s.Status = VerificationVerified
	default:
		// Commands were configured but not all produced evidence.
		s.Status = VerificationFailed
	}
}

// analyzeCheckScope analyzes root using the project exclusion policy that
// applies to it. changedFiles are relative to root.
func analyzeCheckScope(root string, changedFiles []string, changedOnly bool) (*ir.AnalyzerResult, error) {
	policy, err := loadScopePolicy(root)
	if err != nil {
		return nil, err
	}
	var changed *changedScope
	if changedOnly {
		changed = &changedScope{Mode: changedModeWorkingTree, Files: changedFiles, Discovered: len(changedFiles)}
	}
	res, _, err := analyzeCheckScopeWithPolicy(root, changed, changedOnly, policy)
	return res, err
}

func analyzeCheckScopeWithPolicy(root string, changed *changedScope, changedOnly bool, policy *scopePolicy) (*ir.AnalyzerResult, *CheckScope, error) {
	combined := &ir.AnalyzerResult{Files: []*ir.FileIR{}, Diagnostics: []ir.Diagnostic{}}
	scope := &CheckScope{Mode: "all", Skipped: map[string]int{}}
	if policy != nil {
		scope.Exclusions = policy.Patterns
		scope.ConfigPath = policy.ConfigPath
	}
	skip := policy.Filter()

	if changedOnly {
		if changed != nil {
			scope.Mode = changed.Mode
			scope.Base = changed.Base
			scope.Discovered = changed.Discovered
			scope.OutsideScope = changed.OutsideScope
		}
		var files []string
		if changed != nil {
			files = changed.Files
		}
		for _, rel := range files {
			full, reason := safeScopedFile(root, rel)
			switch {
			case reason != "":
			case policy.Excluded(full, false):
				reason = "excluded"
			case !isAnalyzableFile(full):
				reason = "unsupported"
			case isGeneratedFile(full):
				reason = "generated"
			}
			if reason != "" {
				scope.Skipped[reason]++
				continue
			}
			res, err := analyzePathFiltered(full, resolveLang(full, "auto"), nil)
			if err != nil {
				if errors.Is(err, pythonanalysis.ErrRuntimeMissing) {
					combined.Diagnostics = append(combined.Diagnostics, runtimeMissingDiagnostic(rel, err))
					scope.Analyzed++
					continue
				}
				return nil, nil, err
			}
			scope.Analyzed++
			combined.Files = append(combined.Files, res.Files...)
			combined.Diagnostics = append(combined.Diagnostics, res.Diagnostics...)
			source, readErr := os.ReadFile(full)
			if readErr != nil {
				return nil, nil, readErr
			}
			combined.Diagnostics = append(combined.Diagnostics, scanRepeatedDomainLiterals(rel, source)...)
		}
		finalizeScope(scope)
		return combined, scope, nil
	}

	var ran bool
	add := func(res *ir.AnalyzerResult) {
		combined.Files = append(combined.Files, res.Files...)
		combined.Diagnostics = append(combined.Diagnostics, res.Diagnostics...)
		ran = true
	}
	if HasGoMod(root) {
		cfg := goanalysis.DefaultConfig()
		cfg.Skip = skip
		res, err := goanalysis.AnalyzePath(root, cfg)
		if err != nil {
			return nil, nil, err
		}
		add(res)
	}
	if HasPackageJSON(root) {
		cfg := analyzer.DefaultConfig()
		cfg.Skip = skip
		res, err := analyzer.NewAnalyzer(cfg).Analyze(root)
		if err != nil {
			return nil, nil, err
		}
		add(res)
	}
	if HasCargoToml(root) {
		cfg := rustanalysis.DefaultConfig()
		cfg.Skip = skip
		res, err := rustanalysis.AnalyzePath(root, cfg)
		if err != nil {
			return nil, nil, err
		}
		add(res)
	}
	if HasPythonProject(root) || hasPythonFilesFiltered(root, skip) {
		cfg := pythonanalysis.DefaultConfig()
		cfg.Skip = skip
		res, err := pythonanalysis.AnalyzePath(root, cfg)
		if err != nil {
			if !errors.Is(err, pythonanalysis.ErrRuntimeMissing) {
				return nil, nil, err
			}
			combined.Diagnostics = append(combined.Diagnostics, runtimeMissingDiagnostic(".", err))
			ran = true
		} else {
			add(res)
		}
	}
	if !ran {
		res, err := analyzePathFiltered(root, resolveLang(root, "auto"), skip)
		if err != nil {
			return nil, nil, err
		}
		scope.Analyzed = len(res.Files)
		scope.Discovered = len(res.Files)
		finalizeScope(scope)
		return res, scope, nil
	}
	// Analyze the same files as the language analyzers in full-project mode.
	for _, file := range combined.Files {
		if !isAnalyzableFile(file.Path) || isGeneratedFile(file.Path) {
			continue
		}
		source, readErr := os.ReadFile(file.Path)
		if readErr != nil {
			return nil, nil, readErr
		}
		relativePath, relErr := relativeAnalyzedPath(root, file.Path)
		if relErr != nil {
			return nil, nil, relErr
		}
		combined.Diagnostics = append(combined.Diagnostics, scanRepeatedDomainLiterals(relativePath, source)...)
	}
	scope.Analyzed = len(combined.Files)
	scope.Discovered = len(combined.Files)
	finalizeScope(scope)
	return combined, scope, nil
}

func finalizeScope(scope *CheckScope) {
	if len(scope.Skipped) == 0 {
		scope.Skipped = nil
	}
}

func runtimeMissingDiagnostic(file string, err error) ir.Diagnostic {
	return ir.Diagnostic{
		Code: "py.analysis-error", Severity: ir.SeverityHigh, Confidence: "high",
		Message: err.Error(), File: file, Line: 1, EndLine: 1, Kind: ir.NodeKindFile,
		Suggestion: "Install Python 3 so GIRL can parse Python sources; unanalyzed files cannot pass the gate.",
		Tags:       []string{ir.TagBlocking, ir.TagRuntimeMissing},
		Span:       &ir.Span{StartLine: 1, EndLine: 1},
	}
}

// relativeAnalyzedPath normalizes analyzer output before deriving repository paths.
// Some language analyzers return paths relative to the process working directory,
// while others return absolute paths (notably during stdlib corpus scans).
func relativeAnalyzedPath(root, file string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absFile, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absFile)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("analyzed file %q is outside analysis root", file)
	}
	return filepath.ToSlash(rel), nil
}

func complexityRemediation(baseline string) string {
	return fmt.Sprintf("Create the baseline intentionally on a trusted revision: `girl complexity . --write-baseline %s`, review it, and commit it. GIRL never writes or resets a baseline during a gate.", baseline)
}

// runCheckComplexity is the advisory-mode entry point kept for callers that
// do not carry a scope policy.
func runCheckComplexity(path string, cfg *GirlProjectConfig, changedFiles []string, changedOnly bool) (*CheckComplexityResult, error) {
	policy, err := newScopePolicy(path, cfg.Analysis.Exclude)
	if err != nil {
		return nil, err
	}
	return runCheckComplexityWithPolicy(path, cfg, &changedScope{Mode: changedModeWorkingTree, Files: changedFiles}, changedOnly, false, policy)
}

func runCheckComplexityWithPolicy(
	path string,
	cfg *GirlProjectConfig,
	changed *changedScope,
	changedOnly bool,
	strict bool,
	policy *scopePolicy,
) (*CheckComplexityResult, error) {
	var changedFiles []string
	if changed != nil {
		changedFiles = changed.Files
	}
	policyName := strings.ToLower(cfg.Complexity.FailOn)
	result := &CheckComplexityResult{
		Engine:    "cyclomatic-ts",
		Policy:    policyName,
		Baseline:  cfg.Complexity.Baseline,
		Threshold: cfg.Complexity.Max,
	}
	if !HasPackageJSON(path) {
		// Unsupported is not clean: say which engine exists and why it did
		// not apply. Other bindings report complexity as diagnostics.
		result.Status = "not-applicable"
		result.Enforcement = "unsupported"
		result.Engine = ""
		result.Reason = "the complexity ratchet engine supports JavaScript/TypeScript projects (package.json); other languages report complexity as analyzer diagnostics"
		return result, nil
	}
	options := complexity.Options{
		Language:  "auto",
		Threshold: cfg.Complexity.Max,
		Exclude:   cfg.Analysis.Exclude,
		Skip:      policy.Filter(),
	}
	var report *complexity.Report
	var err error
	if changedOnly {
		var scoped []string
		for _, rel := range changedFiles {
			if !policy.Excluded(filepath.Join(path, filepath.FromSlash(rel)), false) {
				scoped = append(scoped, rel)
			}
		}
		report, err = complexity.AnalyzeFiles(path, scoped, options)
	} else {
		report, err = complexity.Analyze(path, options)
	}
	if err != nil {
		return nil, fmt.Errorf("complexity analysis failed: %w", err)
	}
	result.Files = report.Summary.Files
	result.Functions = report.Summary.Functions
	result.OverThreshold = report.Summary.OverThreshold

	switch policyName {
	case "never", "none", "":
		result.Status = "skip"
		result.Enforcement = "disabled"
		result.Reason = "complexity.fail_on disables enforcement"
		return result, nil
	case "threshold":
		result.Enforcement = "threshold"
		result.Status = "pass"
		if report.Summary.OverThreshold > 0 {
			result.Status = "fail"
		}
		return result, nil
	case "regression", "increase":
	default:
		result.Status = "blocked"
		result.Enforcement = "unenforced"
		result.Reason = fmt.Sprintf("unknown complexity.fail_on policy %q", cfg.Complexity.FailOn)
		return result, nil
	}

	unenforced := func(reason string) (*CheckComplexityResult, error) {
		result.Enforcement = "unenforced"
		result.Reason = reason
		result.Remediation = complexityRemediation(cfg.Complexity.Baseline)
		result.Status = "unenforced"
		if strict {
			result.Status = "blocked"
		}
		return result, nil
	}

	baselinePath := cfg.Complexity.Baseline
	if baselinePath == "" {
		return unenforced("no complexity baseline is configured")
	}
	if !filepath.IsAbs(baselinePath) {
		baselinePath = filepath.Join(path, baselinePath)
	}
	// In base (CI) mode prefer the baseline as committed on the base ref: the
	// change under review must not be able to rewrite its own yardstick.
	data, fromBase := baselineFromBase(baselinePath, changed)
	result.BaselineSource = "working-tree"
	if fromBase {
		result.BaselineSource = "base:" + changed.Base
	} else {
		var statErr error
		data, statErr = os.ReadFile(baselinePath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return unenforced(fmt.Sprintf("baseline %s does not exist", cfg.Complexity.Baseline))
			}
			return unenforced(fmt.Sprintf("baseline %s is unreadable: %v", cfg.Complexity.Baseline, statErr))
		}
	}
	sum := sha256.Sum256(data)
	result.BaselineSHA256 = hex.EncodeToString(sum[:])
	baseline, readErr := parseComplexityReport(data)
	if readErr != nil {
		return unenforced(fmt.Sprintf("baseline %s is corrupt: %v", cfg.Complexity.Baseline, readErr))
	}
	result.BaselineSchemaVersion = baseline.SchemaVersion
	if baseline.SchemaVersion != "" && baseline.SchemaVersion != complexity.SchemaVersion {
		return unenforced(fmt.Sprintf("baseline schema %q is incompatible with %q", baseline.SchemaVersion, complexity.SchemaVersion))
	}
	if baseline.Metric != "" && report.Metric != "" && baseline.Metric != report.Metric {
		return unenforced(fmt.Sprintf("baseline metric %q does not match %q", baseline.Metric, report.Metric))
	}
	complexity.Compare(report, baseline)
	result.BaselineFound = true
	result.Enforcement = "enforced"
	if report.Comparison != nil {
		result.Regressions = report.Comparison.Regressions
	}
	failed, err := complexityPolicyFailed(policyName, report)
	if err != nil {
		return nil, err
	}
	result.Status = "pass"
	if failed {
		result.Status = "fail"
	}
	return result, nil
}

// baselineFromBase reads the baseline file as committed on the comparison
// base. ok is false outside base mode or when the base has no such file.
func baselineFromBase(baselinePath string, changed *changedScope) ([]byte, bool) {
	if changed == nil || changed.Mode != changedModeBase || changed.Base == "" || changed.RepoRoot == "" {
		return nil, false
	}
	abs, err := filepath.Abs(baselinePath)
	if err != nil {
		return nil, false
	}
	if resolved, err := filepath.EvalSymlinks(filepath.Dir(abs)); err == nil {
		abs = filepath.Join(resolved, filepath.Base(abs))
	}
	rel, err := filepath.Rel(changed.RepoRoot, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, false
	}
	out, err := gitOutput(changed.RepoRoot, "show", "--end-of-options", changed.Base+":"+filepath.ToSlash(rel))
	if err != nil {
		return nil, false
	}
	return out, true
}

func runCheckReviewability(path, base string, cfg *GirlProjectConfig) (*reviewability.EvalResult, []ir.Diagnostic, error) {
	raw, err := gitDiffBytes(path, base)
	if err != nil {
		return nil, nil, nil
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, nil, nil
	}
	stats, err := diffstats.ParseDiffBytes(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("reviewability diff parse failed: %w", err)
	}
	budget := ir.ReviewabilityBudget{
		MaxDiffLines:    cfg.Reviewability.MaxDiffLines,
		MaxTouchedFiles: cfg.Reviewability.MaxTouchedFiles,
		MaxRisk:         ir.Severity(cfg.Reviewability.MaxRisk),
	}
	result := reviewability.Evaluate(stats, budget)
	return result, result.Diagnostics, nil
}

// isAnalyzableFile mirrors what full directory scans analyze, so changed-file
// runs neither crash on nor report files a whole-repo run would skip.
func isAnalyzableFile(path string) bool {
	if strings.HasSuffix(path, "_test.go") || shared.IsGeneratedName(path) {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".rs", ".py", ".ts", ".tsx", ".js", ".jsx":
		return true
	default:
		return false
	}
}

func isGeneratedFile(path string) bool {
	data, err := os.ReadFile(path)
	return err == nil && shared.IsGeneratedSource(data)
}

func verificationRoleOrder(cfg *GirlProjectConfig) []string {
	roles := []string{"typecheck", "lint", "test", "build", "architecture", "format", "other"}
	seen := map[string]bool{}
	for _, r := range roles {
		seen[r] = true
	}
	var extra []string
	for role := range cfg.Verify {
		if !seen[role] {
			extra = append(extra, role)
		}
	}
	sort.Strings(extra)
	return append(roles, extra...)
}

func runConfiguredVerification(path string, cfg *GirlProjectConfig, failFast bool) []CheckVerificationResult {
	var results []CheckVerificationResult
	stopped := false
	for _, role := range verificationRoleOrder(cfg) {
		for _, command := range cfg.Verify[role] {
			var result CheckVerificationResult
			if stopped {
				result = CheckVerificationResult{Role: role, Command: command, Status: CommandNotRun}
			} else {
				result = executeVerification(path, role, command)
				if failFast && result.Status != CommandPass {
					stopped = true
				}
			}
			if prov, ok := cfg.VerifyProvenance[command]; ok {
				result.Source, result.Confidence = prov.Source, prov.Confidence
			} else if cfg.VerifySource == "config" {
				result.Source, result.Confidence = ".girl/config.yaml", "high"
			}
			results = append(results, result)
		}
	}
	return results
}

func executeVerification(path, role, command string) CheckVerificationResult {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Dir = path
	out, err := cmd.CombinedOutput()
	result := CheckVerificationResult{
		Role:    role,
		Command: command,
		Status:  CommandPass,
		// Command output lands in JSON results and CI logs; it routinely
		// echoes environment values, so it passes the export sanitizer.
		Output: redact.String(truncateOutput(strings.TrimSpace(string(out)))),
	}
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
			result.Status = CommandFail
			// POSIX shells use 126/127 for "not executable" / "not found":
			// the check never actually ran.
			if runtime.GOOS != "windows" && (code == 126 || code == 127) {
				result.Status = CommandError
			}
		} else {
			code = -1
			result.Status = CommandError
			if result.Output == "" {
				result.Output = err.Error()
			}
		}
	}
	result.ExitCode = &code
	return result
}

const maxVerificationOutput = 64 << 10

// truncateOutput keeps the tail of very large command output (where the
// failure usually is) so results stay bounded.
func truncateOutput(s string) string {
	if len(s) <= maxVerificationOutput {
		return s
	}
	return "... (truncated)\n" + s[len(s)-maxVerificationOutput:]
}

func printCheckText(result *CheckResult) {
	mark := "✓"
	if result.Status == "warn" {
		mark = "!"
	} else if result.Status == "fail" {
		mark = "✗"
	}
	mode := ""
	if result.Strict {
		mode = " (strict)"
	}
	fmt.Printf("GIRL %s %s — %s%s\n\n", mark, strings.ToUpper(result.Status), result.Verdict, mode)
	if s := result.Scope; s != nil {
		fmt.Printf("Scope: %s — %d discovered, %d analyzed", s.Mode, s.Discovered, s.Analyzed)
		if len(s.Skipped) > 0 {
			keys := make([]string, 0, len(s.Skipped))
			for k := range s.Skipped {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := make([]string, 0, len(keys))
			for _, k := range keys {
				parts = append(parts, fmt.Sprintf("%s %d", k, s.Skipped[k]))
			}
			fmt.Printf(", skipped: %s", strings.Join(parts, ", "))
		}
		if s.OutsideScope > 0 {
			fmt.Printf(", %d outside path", s.OutsideScope)
		}
		fmt.Println()
		if s.DiscoveryError != "" {
			fmt.Printf("  changed-file discovery failed: %s\n", s.DiscoveryError)
		}
	}
	fmt.Printf("Changed files: %d\n", len(result.ChangedFiles))
	fmt.Printf("Affected workspaces: %d\n", len(result.Workspaces))
	fmt.Printf("Diagnostics: %d (%d blocking)\n", len(result.Diagnostics), result.BlockingDiagnostics)
	for _, d := range result.Diagnostics {
		if ir.IsBlocking(d) {
			fmt.Printf("  ✗ %s:%d %s — %s\n", d.File, d.Line, d.Code, d.Message)
		}
	}
	if c := result.Complexity; c != nil {
		fmt.Printf("Complexity: %s [%s] — %d functions, %d over %d, %d regressions\n",
			strings.ToUpper(c.Status), c.Enforcement, c.Functions, c.OverThreshold, c.Threshold, c.Regressions)
		if c.Reason != "" && c.Status != "pass" {
			fmt.Printf("  %s\n", c.Reason)
		}
		if c.Remediation != "" {
			fmt.Printf("  %s\n", c.Remediation)
		}
	}
	if result.Reviewability != nil && result.Reviewability.Observed != nil {
		fmt.Printf("Reviewability: %s — %d lines, %d files\n",
			strings.ToUpper(result.Reviewability.Status),
			result.Reviewability.Observed.ChangedLines, result.Reviewability.Observed.ChangedFiles)
	}

	preflightFail := 0
	preflightWarn := 0
	for _, check := range result.Preflight {
		if check.Status == "fail" {
			preflightFail++
		} else if check.Status == "warn" {
			preflightWarn++
		}
	}
	fmt.Printf("Preflight: %d failed, %d warnings\n", preflightFail, preflightWarn)

	if s := result.VerificationSummary; s != nil {
		fmt.Printf("Verification: %s — %d configured, %d executed, %d passed, %d failed, %d errored, %d not run\n",
			strings.ToUpper(s.Status), s.Configured, s.Executed, s.Passed, s.Failed, s.Errored, s.NotRun)
		if s.DiscoveryError != "" {
			fmt.Printf("  discovery error: %s\n", s.DiscoveryError)
		}
	}
	for _, v := range result.Verification {
		vmark := "✓"
		switch v.Status {
		case CommandFail, CommandError:
			vmark = "✗"
		case CommandNotRun:
			vmark = "-"
		}
		fmt.Printf("  %s %-12s %s\n", vmark, v.Role, v.Command)
		if (v.Status == CommandFail || v.Status == CommandError) && v.Output != "" {
			fmt.Printf("    %s\n", strings.ReplaceAll(v.Output, "\n", "\n    "))
		}
	}
	if len(result.Reasons) > 0 {
		fmt.Println("\nReasons:")
		for _, r := range result.Reasons {
			fmt.Printf("  - %s\n", r)
		}
	}
}
