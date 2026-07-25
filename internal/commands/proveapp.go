package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/canadian-ai/girl/internal/verification"
	"github.com/canadian-ai/girl/internal/verifier"
	"github.com/urfave/cli/v2"
)

const proveSpecVersion = "girl.io/proveapp/v1"
const proveType = "prove-app"

type ReadinessCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	Required bool   `json:"required"`
}

type ProveAppResult struct {
	SpecVersion string           `json:"specversion"`
	ID          string           `json:"id"`
	Type        string           `json:"type"`
	Path        string           `json:"path"`
	Status      string           `json:"status"`
	Checks      []ReadinessCheck `json:"checks"`
	Summary     struct {
		Total int `json:"total"`
		Pass  int `json:"pass"`
		Warn  int `json:"warn"`
		Fail  int `json:"fail"`
	} `json:"summary"`
	Timestamp string `json:"timestamp"`
}

type proveOpts struct {
	Format       string
	OutputFile   string
	CheckDir     string
	SkipBuild    bool
	SkipTest     bool
	RequireDocker bool
	RequireCI     bool
}

func ProveAppCommand() *cli.Command {
	return &cli.Command{
		Name:      "prove-app",
		Usage:     "Check if a generated app is deployment-ready",
		ArgsUsage: "[path]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: json (default), text, markdown",
				Value:   "json",
			},
			&cli.StringFlag{
				Name:  "check-dir",
				Usage: "Directory to check",
			},
			&cli.StringFlag{
				Name:    "output-file",
				Aliases: []string{"f"},
				Usage:   "Write prove-app result to file (e.g., .grp/prove-app.json)",
			},
			&cli.BoolFlag{
				Name:  "skip-build",
				Usage: "Skip build check",
			},
			&cli.BoolFlag{
				Name:  "skip-test",
				Usage: "Skip test check",
			},
			&cli.BoolFlag{
				Name:  "require-docker",
				Usage: "Require Dockerfile presence",
			},
			&cli.BoolFlag{
				Name:  "require-ci",
				Usage: "Require CI workflow",
			},
		},
		Action: func(c *cli.Context) error {
			path := commandPath(c)

			opts := proveOpts{
				Format:        stringFlag(c, "output", "o"),
				OutputFile:    c.String("output-file"),
				CheckDir:      c.String("check-dir"),
				SkipBuild:     c.Bool("skip-build"),
				SkipTest:      c.Bool("skip-test"),
				RequireDocker: c.Bool("require-docker"),
				RequireCI:     c.Bool("require-ci"),
			}

			if opts.CheckDir == "" {
				opts.CheckDir = path
			}

			result := runReadinessChecks(path, opts)

			if opts.OutputFile != "" {
				if err := writeJSONFile(opts.OutputFile, result); err != nil {
					return fmt.Errorf("write prove-app file: %w", err)
				}
			}

			switch opts.Format {
			case "text":
				printProveAppText(result)
			case "markdown":
				printProveAppMarkdown(result)
			default:
				printJSON(result)
			}

			return nil
		},
	}
}

func runReadinessChecks(path string, opts proveOpts) *ProveAppResult {
	v := verifier.NewVerifier()
	vr, err := v.Verify(path)
	if err != nil {
		vr = &verification.Result{
			WorkDir: path,
		}
	}

	checks := []ReadinessCheck{}

	checks = append(checks, checkConfig(path, vr))

	if !opts.SkipBuild {
		checks = append(checks, checkBuild(path, vr))
	}

	if !opts.SkipTest {
		checks = append(checks, checkTests(path, vr))
	}

	checks = append(checks, checkTypeCheck(path, vr))
	checks = append(checks, checkLint(path, vr))

	reqDocker := opts.RequireDocker
	checks = append(checks, checkDocker(path, reqDocker))

	reqCI := opts.RequireCI
	checks = append(checks, checkCI(path, reqCI))

	checks = append(checks, checkHealth(path, vr))

	id := "prove_" + time.Now().UTC().Format("20060102T150405Z")
	ts := time.Now().UTC().Format(time.RFC3339)

	total := len(checks)
	passCount := 0
	warnCount := 0
	failCount := 0
	hasRequiredFail := false
	hasOptionalWarn := false

	for _, ch := range checks {
		switch ch.Status {
		case "pass":
			passCount++
		case "warn":
			warnCount++
			if !ch.Required {
				hasOptionalWarn = true
			}
		case "fail":
			failCount++
			if ch.Required {
				hasRequiredFail = true
			}
		}
	}

	var overallStatus string
	switch {
	case hasRequiredFail:
		overallStatus = "fail"
	case hasOptionalWarn:
		overallStatus = "warn"
	default:
		overallStatus = "pass"
	}

	result := &ProveAppResult{
		SpecVersion: proveSpecVersion,
		ID:          id,
		Type:        proveType,
		Path:        path,
		Status:      overallStatus,
		Checks:      checks,
		Timestamp:   ts,
	}
	result.Summary.Total = total
	result.Summary.Pass = passCount
	result.Summary.Warn = warnCount
	result.Summary.Fail = failCount

	return result
}

func checkBuild(path string, vr *verification.Result) ReadinessCheck {
	hasBuild := false
	sources := []string{}

	for _, cmd := range vr.Commands {
		if !cmd.Exists {
			continue
		}
		lower := strings.ToLower(cmd.Name)
		cmdLower := strings.ToLower(cmd.Command)
		if lower == "build" || cmd.Type == "build" {
			hasBuild = true
			sources = append(sources, cmd.Source)
		}
		if strings.Contains(cmdLower, "go build") || strings.Contains(cmdLower, "build") {
			hasBuild = true
			if cmd.Source != "" {
				sources = append(sources, cmd.Source)
			}
		}
	}

	if !hasBuild {
		if HasGoMod(path) {
			hasBuild = true
			sources = append(sources, "go.mod")
		}
		if HasPackageJSON(path) {
			hasBuild = true
			if len(sources) == 0 || sources[len(sources)-1] != "go.mod" {
				sources = append(sources, "package.json")
			}
		}
	}

	sources = uniqueStrings(sources)
	if hasBuild {
		return ReadinessCheck{
			Name:     "Build",
			Status:   "pass",
			Message:  fmt.Sprintf("Build configured (%s)", strings.Join(sources, ", ")),
			Required: true,
		}
	}

	return ReadinessCheck{
		Name:     "Build",
		Status:   "fail",
		Message:  "No build configuration found",
		Required: true,
	}
}

func uniqueStrings(s []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(s))
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

func checkTests(path string, vr *verification.Result) ReadinessCheck {
	hasTest := false
	sources := []string{}

	for _, cmd := range vr.Commands {
		if !cmd.Exists {
			continue
		}
		lower := strings.ToLower(cmd.Name)
		cmdLower := strings.ToLower(cmd.Command)
		if lower == "test" || cmd.Type == "test" {
			hasTest = true
			sources = append(sources, cmd.Source)
		}
		if strings.Contains(cmdLower, "go test") || strings.Contains(cmdLower, "test") {
			if cmd.Source != "" {
				sources = append(sources, cmd.Source)
			}
		}
	}

	sources = uniqueStrings(sources)
	if !hasTest {
		if HasGoMod(path) {
			hasTest = true
			sources = append(sources, "go.mod")
		}
		if HasPackageJSON(path) {
			hasTest = true
			if len(sources) == 0 || sources[len(sources)-1] != "go.mod" {
				sources = append(sources, "package.json")
			}
		}
	}

	sources = uniqueStrings(sources)
	if hasTest {
		return ReadinessCheck{
			Name:     "Test",
			Status:   "pass",
			Message:  fmt.Sprintf("Tests configured (%s)", strings.Join(sources, ", ")),
			Required: true,
		}
	}

	return ReadinessCheck{
		Name:     "Test",
		Status:   "warn",
		Message:  "No test configuration found",
		Required: true,
	}
}

func checkTypeCheck(path string, vr *verification.Result) ReadinessCheck {
	hasTypeCheck := false
	sources := []string{}

	if vr.HasConfig {
		hasTypeCheck = true
		sources = append(sources, "tsconfig.json")
	}

	for _, cmd := range vr.Commands {
		if !cmd.Exists {
			continue
		}
		lower := strings.ToLower(cmd.Name)
		cmdLower := strings.ToLower(cmd.Command)
		if lower == "typecheck" || strings.Contains(cmdLower, "go vet") || strings.Contains(cmdLower, "typecheck") || strings.Contains(cmdLower, "tsc") {
			hasTypeCheck = true
			sources = append(sources, cmd.Source)
		}
	}

	sources = uniqueStrings(sources)
	if !hasTypeCheck {
		if fileExists(filepath.Join(path, "tsconfig.json")) {
			hasTypeCheck = true
			sources = append(sources, "tsconfig.json")
		}
		if HasGoMod(path) {
			hasTypeCheck = true
			sources = append(sources, "go.mod")
		}
	}

	sources = uniqueStrings(sources)
	if hasTypeCheck {
		return ReadinessCheck{
			Name:     "Type Check",
			Status:   "pass",
			Message:  fmt.Sprintf("Type checking configured (%s)", strings.Join(sources, ", ")),
			Required: true,
		}
	}

	return ReadinessCheck{
		Name:     "Type Check",
		Status:   "warn",
		Message:  "No type checking configuration found",
		Required: true,
	}
}

func checkLint(path string, vr *verification.Result) ReadinessCheck {
	hasLint := false
	sources := []string{}

	if vr.HasGolangCILint {
		hasLint = true
		sources = append(sources, ".golangci.yml")
	}

	for _, cmd := range vr.Commands {
		if !cmd.Exists {
			continue
		}
		lower := strings.ToLower(cmd.Name)
		cmdLower := strings.ToLower(cmd.Command)
		if lower == "lint" || cmd.Type == "lint" {
			hasLint = true
			sources = append(sources, cmd.Source)
		}
		if strings.Contains(cmdLower, "golangci-lint") || strings.Contains(cmdLower, "eslint") {
			if cmd.Source != "" {
				sources = append(sources, cmd.Source)
			}
		}
	}

	if !hasLint {
		if fileExists(filepath.Join(path, ".eslintrc")) ||
			fileExists(filepath.Join(path, ".eslintrc.json")) ||
			fileExists(filepath.Join(path, ".eslintrc.yaml")) ||
			fileExists(filepath.Join(path, ".eslintrc.yml")) ||
			fileExists(filepath.Join(path, ".eslintrc.js")) ||
			fileExists(filepath.Join(path, ".golangci.yml")) ||
			fileExists(filepath.Join(path, ".golangci.yaml")) {
			hasLint = true
			sources = append(sources, "config-file")
		}
	}

	if hasLint {
		return ReadinessCheck{
			Name:     "Lint",
			Status:   "pass",
			Message:  fmt.Sprintf("Linting configured (%s)", strings.Join(sources, ", ")),
			Required: false,
		}
	}

	return ReadinessCheck{
		Name:     "Lint",
		Status:   "warn",
		Message:  "No lint configuration found",
		Required: false,
	}
}

func checkDocker(path string, required bool) ReadinessCheck {
	if fileExists(filepath.Join(path, "Dockerfile")) {
		return ReadinessCheck{
			Name:     "Docker",
			Status:   "pass",
			Message:  "Dockerfile found",
			Required: required,
		}
	}

	status := "warn"
	if required {
		status = "fail"
	}

	return ReadinessCheck{
		Name:     "Docker",
		Status:   status,
		Message:  "No Dockerfile found",
		Required: required,
	}
}

func checkCI(path string, required bool) ReadinessCheck {
	if fileExists(filepath.Join(path, ".github", "workflows")) {
		return ReadinessCheck{
			Name:     "CI",
			Status:   "pass",
			Message:  "CI workflow found",
			Required: required,
		}
	}

	status := "warn"
	if required {
		status = "fail"
	}

	return ReadinessCheck{
		Name:     "CI",
		Status:   status,
		Message:  "No CI workflow found",
		Required: required,
	}
}

func checkHealth(path string, vr *verification.Result) ReadinessCheck {
	indicators := []string{
		"health", "healthz", "readyz", "livez",
		"health.go", "health.ts", "health.js",
		"probes", "readiness",
	}

	found := []string{}
	for _, ind := range indicators {
		pattern := filepath.Join(path, ind)
		if fileExists(pattern) {
			found = append(found, ind)
		}
	}

	if !fileExists(filepath.Join(path, "health")) {
		entries, err := os.ReadDir(path)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := strings.ToLower(entry.Name())
				for _, ind := range indicators {
					if strings.Contains(name, ind) {
						found = append(found, entry.Name())
						break
					}
				}
			}
			entries = nil
		}
	}

	if HasPackageJSON(path) {
		data, err := os.ReadFile(filepath.Join(path, "package.json"))
		if err == nil {
			content := strings.ToLower(string(data))
			if strings.Contains(content, "health") || strings.Contains(content, "healthz") || strings.Contains(content, "readinessprobe") {
				found = append(found, "package.json")
			}
		}
	}

	if HasGoMod(path) {
		data, err := os.ReadFile(filepath.Join(path, "go.mod"))
		if err == nil {
			content := strings.ToLower(string(data))
			if strings.Contains(content, "health") {
				found = append(found, "go.mod")
			}
		}
	}

	if len(found) > 0 {
		return ReadinessCheck{
			Name:     "Health",
			Status:   "pass",
			Message:  fmt.Sprintf("Health check indicators found (%s)", strings.Join(found, ", ")),
			Required: false,
		}
	}

	return ReadinessCheck{
		Name:     "Health",
		Status:   "warn",
		Message:  "No health check endpoints or probes found",
		Required: false,
	}
}

func checkConfig(path string, vr *verification.Result) ReadinessCheck {
	found := []string{}

	if HasGoMod(path) {
		found = append(found, "go.mod")
	}
	if HasPackageJSON(path) {
		found = append(found, "package.json")
	}
	if fileExists(filepath.Join(path, "tsconfig.json")) {
		found = append(found, "tsconfig.json")
	}
	if fileExists(filepath.Join(path, "Makefile")) {
		found = append(found, "Makefile")
	}

	if len(found) > 0 {
		return ReadinessCheck{
			Name:     "Config",
			Status:   "pass",
			Message:  fmt.Sprintf("Essential config files present (%s)", strings.Join(found, ", ")),
			Required: true,
		}
	}

	return ReadinessCheck{
		Name:     "Config",
		Status:   "fail",
		Message:  "No essential config files found (go.mod, package.json, tsconfig.json, Makefile)",
		Required: true,
	}
}

func printProveAppText(r *ProveAppResult) {
	fmt.Printf("ProveApp Result: %s\n\n", strings.ToUpper(r.Status))
	fmt.Printf("  Path:      %s\n", r.Path)
	fmt.Printf("  ID:        %s\n", r.ID)
	fmt.Printf("  Timestamp: %s\n\n", r.Timestamp)

	fmt.Printf("Checks (%d total, %d pass, %d warn, %d fail):\n\n", r.Summary.Total, r.Summary.Pass, r.Summary.Warn, r.Summary.Fail)

	for _, ch := range r.Checks {
		var mark string
		switch ch.Status {
		case "pass":
			mark = "✓"
		case "warn":
			mark = "!"
		case "fail":
			mark = "✗"
		}
		req := ""
		if ch.Required {
			req = " [required]"
		}
		fmt.Printf("  %s %s%s\n", mark, ch.Name, req)
		if ch.Message != "" {
			fmt.Printf("    %s\n", ch.Message)
		}
	}
}

func printProveAppMarkdown(r *ProveAppResult) {
	fmt.Printf("# ProveApp Result\n\n")
	fmt.Printf("**Status:** %s\n\n", strings.ToUpper(r.Status))
	fmt.Printf("**Path:** `%s`\n\n", r.Path)
	fmt.Printf("**ID:** `%s`\n\n", r.ID)
	fmt.Printf("**Timestamp:** %s\n\n", r.Timestamp)

	fmt.Printf("## Summary\n\n")
	fmt.Printf("- Total: %d\n", r.Summary.Total)
	fmt.Printf("- Pass:  %d\n", r.Summary.Pass)
	fmt.Printf("- Warn:  %d\n", r.Summary.Warn)
	fmt.Printf("- Fail:  %d\n\n", r.Summary.Fail)

	fmt.Printf("## Checks\n\n")
	for _, ch := range r.Checks {
		var icon string
		switch ch.Status {
		case "pass":
			icon = "✅"
		case "warn":
			icon = "⚠️"
		case "fail":
			icon = "❌"
		}
		req := ""
		if ch.Required {
			req = " (*required*)"
		}
		fmt.Printf("- %s **%s**%s", icon, ch.Name, req)
		if ch.Message != "" {
			fmt.Printf(": %s", ch.Message)
		}
		fmt.Printf("\n")
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info != nil
}
