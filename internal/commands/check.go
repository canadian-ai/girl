package commands

import (
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
	"github.com/canadian-ai/girl/internal/reviewability"
	"github.com/canadian-ai/girl/internal/rustanalysis"
	"github.com/urfave/cli/v2"
)

type CheckVerificationResult struct {
	Role    string `json:"role"`
	Command string `json:"command"`
	Status  string `json:"status"`
	Output  string `json:"output,omitempty"`
}

type CheckComplexityResult struct {
	Status        string `json:"status"`
	Baseline      string `json:"baseline,omitempty"`
	BaselineFound bool   `json:"baselineFound"`
	Threshold     int    `json:"threshold"`
	Files         int    `json:"files"`
	Functions     int    `json:"functions"`
	OverThreshold int    `json:"overThreshold"`
	Regressions   int    `json:"regressions"`
}

type CheckResult struct {
	SpecVersion     string                    `json:"specversion"`
	Path            string                    `json:"path"`
	Profiles        []string                  `json:"profiles,omitempty"`
	ChangedOnly     bool                      `json:"changedOnly"`
	ChangedFiles    []string                  `json:"changedFiles,omitempty"`
	Workspaces      []string                  `json:"affectedWorkspaces,omitempty"`
	Diagnostics     []ir.Diagnostic           `json:"diagnostics,omitempty"`
	Complexity      *CheckComplexityResult    `json:"complexity,omitempty"`
	Reviewability   *ir.ReviewabilityResult   `json:"reviewability,omitempty"`
	Preflight       []PreflightCheck          `json:"preflight,omitempty"`
	Verification    []CheckVerificationResult `json:"verification,omitempty"`
	Status          string                    `json:"status"`
	VerificationRun bool                      `json:"verificationRun"`
}

func CheckCommand() *cli.Command {
	return &cli.Command{
		Name:      "check",
		Usage:     "Run the project-level GIRL quality contract",
		ArgsUsage: "[path]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "ci", Usage: "Use CI-friendly changed-file defaults"},
			&cli.BoolFlag{Name: "changed", Usage: "Force changed-file analysis"},
			&cli.BoolFlag{Name: "all", Usage: "Analyze the entire project"},
			&cli.StringFlag{Name: "base", Usage: "Git base ref for changed-file analysis and reviewability"},
			&cli.BoolFlag{Name: "no-verify", Usage: "Skip project verification commands"},
			&cli.BoolFlag{Name: "fail-fast", Usage: "Stop verification on first failed command"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "text", Usage: "Output format: text or json"},
		},
		Action: func(c *cli.Context) error {
			path := commandPath(c)
			cfg, err := loadGirlProjectConfig(path)
			if err != nil {
				return fmt.Errorf("load GIRL config: %w", err)
			}

			changedOnly := cfg.Analysis.ChangedOnly
			if c.Bool("changed") {
				changedOnly = true
			}
			if c.Bool("all") {
				changedOnly = false
			}
			base := c.String("base")
			if base == "" && c.Bool("ci") {
				base = os.Getenv("GITHUB_BASE_REF")
			}

			result := &CheckResult{
				SpecVersion:     "girl.io/check/v1",
				Path:            path,
				Profiles:        cfg.Profiles,
				ChangedOnly:     changedOnly,
				Status:          "pass",
				VerificationRun: !c.Bool("no-verify"),
			}

			if changedOnly {
				files, diffErr := gitChangedFiles(path, base)
				if diffErr == nil {
					result.ChangedFiles = files
					result.Workspaces = affectedWorkspaces(files, cfg.Workspaces)
				} else {
					changedOnly = false
					result.ChangedOnly = false
				}
			}

			analysis, err := analyzeCheckScope(path, result.ChangedFiles, changedOnly)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}
			result.Diagnostics = analysis.Diagnostics
			for _, d := range result.Diagnostics {
				if d.Severity == ir.SeverityHigh && result.Status == "pass" {
					result.Status = "warn"
			}
			}

			complexityResult, err := runCheckComplexity(path, cfg)
			if err != nil {
				return err
			}
			result.Complexity = complexityResult
			if complexityResult != nil && complexityResult.Status == "fail" {
				result.Status = "fail"
			}

			review, reviewDiagnostics, err := runCheckReviewability(path, base, cfg)
			if err != nil {
				return err
			}
			if review != nil {
				result.Reviewability = &review.Result
				result.Diagnostics = append(result.Diagnostics, reviewDiagnostics...)
				switch review.Result.Status {
				case "fail":
					result.Status = "fail"
				case "warn":
					if result.Status == "pass" {
						result.Status = "warn"
					}
				}
			}

			preflight := runPreflight(path, ProfileAuto)
			result.Preflight = preflight.Checks
			if preflight.Status == "fail" {
				result.Status = "fail"
			} else if preflight.Status == "warn" && result.Status == "pass" {
				result.Status = "warn"
			}

			if !c.Bool("no-verify") {
				result.Verification = runConfiguredVerification(path, cfg, c.Bool("fail-fast"))
				for _, v := range result.Verification {
					if v.Status == "fail" {
						result.Status = "fail"
						break
					}
				}
			}

			if stringFlag(c, "output", "o") == "json" {
				printJSON(result)
			} else {
				printCheckText(result)
			}
			if result.Status == "fail" {
				return fmt.Errorf("GIRL check failed")
			}
			return nil
		},
	}
}

func analyzeCheckScope(root string, changedFiles []string, changedOnly bool) (*ir.AnalyzerResult, error) {
	combined := &ir.AnalyzerResult{Files: []*ir.FileIR{}, Diagnostics: []ir.Diagnostic{}}
	if changedOnly {
		if len(changedFiles) == 0 {
			return combined, nil
		}
		for _, rel := range changedFiles {
			full := filepath.Join(root, filepath.FromSlash(rel))
			if !isAnalyzableFile(full) || !checkFileExists(full) {
				continue
			}
			res, err := analyzePath(full, resolveLang(full, "auto"))
			if err != nil {
				return nil, err
			}
			combined.Files = append(combined.Files, res.Files...)
			combined.Diagnostics = append(combined.Diagnostics, res.Diagnostics...)
		}
		return combined, nil
	}

	var ran bool
	if HasGoMod(root) {
		res, err := goanalysis.AnalyzePath(root, goanalysis.DefaultConfig())
		if err != nil {
			return nil, err
		}
		combined.Files = append(combined.Files, res.Files...)
		combined.Diagnostics = append(combined.Diagnostics, res.Diagnostics...)
		ran = true
	}
	if HasPackageJSON(root) {
		res, err := analyzer.NewAnalyzer(analyzer.DefaultConfig()).Analyze(root)
		if err != nil {
			return nil, err
		}
		combined.Files = append(combined.Files, res.Files...)
		combined.Diagnostics = append(combined.Diagnostics, res.Diagnostics...)
		ran = true
	}
	if HasCargoToml(root) {
		res, err := rustanalysis.AnalyzePath(root, rustanalysis.DefaultConfig())
		if err != nil {
			return nil, err
		}
		combined.Files = append(combined.Files, res.Files...)
		combined.Diagnostics = append(combined.Diagnostics, res.Diagnostics...)
		ran = true
	}
	if !ran {
		return analyzePath(root, resolveLang(root, "auto"))
	}
	return combined, nil
}

func runCheckComplexity(path string, cfg *GirlProjectConfig) (*CheckComplexityResult, error) {
	if !HasPackageJSON(path) {
		return nil, nil
	}
	report, err := complexity.Analyze(path, complexity.Options{
		Language: "auto",
		Threshold: cfg.Complexity.Max,
		Exclude: cfg.Analysis.Exclude,
	})
	if err != nil {
		return nil, fmt.Errorf("complexity analysis failed: %w", err)
	}

	result := &CheckComplexityResult{
		Status:        "pass",
		Baseline:      cfg.Complexity.Baseline,
		Threshold:     cfg.Complexity.Max,
		Files:         report.Summary.Files,
		Functions:     report.Summary.Functions,
		OverThreshold: report.Summary.OverThreshold,
	}
	baselinePath := cfg.Complexity.Baseline
	if baselinePath != "" && !filepath.IsAbs(baselinePath) {
		baselinePath = filepath.Join(path, baselinePath)
	}
	if baselinePath != "" {
		if _, statErr := os.Stat(baselinePath); statErr == nil {
			baseline, readErr := readComplexityReport(baselinePath)
			if readErr != nil {
				return nil, fmt.Errorf("read complexity baseline: %w", readErr)
			}
			complexity.Compare(report, baseline)
			result.BaselineFound = true
			if report.Comparison != nil {
				result.Regressions = report.Comparison.Regressions
			}
		}
	}

	policy := strings.ToLower(cfg.Complexity.FailOn)
	if (policy == "regression" || policy == "increase") && !result.BaselineFound {
		result.Status = "skip"
		return result, nil
	}
	failed, err := complexityPolicyFailed(policy, report)
	if err != nil {
		return nil, err
	}
	if failed {
		result.Status = "fail"
	}
	return result, nil
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

func isAnalyzableFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".rs", ".ts", ".tsx", ".js", ".jsx":
		return true
	default:
		return false
	}
}

func runConfiguredVerification(path string, cfg *GirlProjectConfig, failFast bool) []CheckVerificationResult {
	roles := []string{"typecheck", "lint", "test", "build", "architecture", "format", "other"}
	seenRole := map[string]bool{}
	var results []CheckVerificationResult
	for _, role := range roles {
		seenRole[role] = true
		for _, command := range cfg.Verify[role] {
			result := executeVerification(path, role, command)
			results = append(results, result)
			if failFast && result.Status == "fail" {
				return results
			}
		}
	}
	var extra []string
	for role := range cfg.Verify {
		if !seenRole[role] {
			extra = append(extra, role)
		}
	}
	sort.Strings(extra)
	for _, role := range extra {
		for _, command := range cfg.Verify[role] {
			result := executeVerification(path, role, command)
			results = append(results, result)
			if failFast && result.Status == "fail" {
				return results
			}
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
	status := "pass"
	if err != nil {
		status = "fail"
	}
	return CheckVerificationResult{
		Role:    role,
		Command: command,
		Status:  status,
		Output:  strings.TrimSpace(string(out)),
	}
}

func printCheckText(result *CheckResult) {
	mark := "✓"
	if result.Status == "warn" {
		mark = "!"
	} else if result.Status == "fail" {
		mark = "✗"
	}
	fmt.Printf("GIRL %s %s\n\n", mark, strings.ToUpper(result.Status))
	fmt.Printf("Changed files: %d\n", len(result.ChangedFiles))
	fmt.Printf("Affected workspaces: %d\n", len(result.Workspaces))
	fmt.Printf("Diagnostics: %d\n", len(result.Diagnostics))
	if result.Complexity != nil {
		fmt.Printf("Complexity: %s — %d functions, %d over %d, %d regressions\n",
			strings.ToUpper(result.Complexity.Status), result.Complexity.Functions,
			result.Complexity.OverThreshold, result.Complexity.Threshold, result.Complexity.Regressions)
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

	if result.VerificationRun {
		fmt.Println("Verification:")
		for _, v := range result.Verification {
			vmark := "✓"
			if v.Status == "fail" {
				vmark = "✗"
			}
			fmt.Printf("  %s %-12s %s\n", vmark, v.Role, v.Command)
			if v.Status == "fail" && v.Output != "" {
				fmt.Printf("    %s\n", strings.ReplaceAll(v.Output, "\n", "\n    "))
			}
		}
	}
}
