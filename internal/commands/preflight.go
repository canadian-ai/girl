package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

type PreflightCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type PreflightResult struct {
	SpecVersion string          `json:"specversion"`
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Path        string          `json:"path"`
	Profile     string          `json:"profile,omitempty"`
	Status      string          `json:"status"`
	Checks      []PreflightCheck `json:"checks"`
	Summary     struct {
		Total int `json:"total"`
		Pass  int `json:"pass"`
		Warn  int `json:"warn"`
		Fail  int `json:"fail"`
	} `json:"summary"`
	Timestamp string `json:"timestamp"`
}

type PreflightProfile string

const (
	ProfileAuto          PreflightProfile = "auto"
	ProfileGeneric       PreflightProfile = "generic"
	ProfileCAINextConvex PreflightProfile = "cai-next-convex"
)

type profileChecks struct {
	Name        string
	Description string
	Checks      func(path string) []PreflightCheck
}

var profileRegistry = map[PreflightProfile]*profileChecks{
	ProfileGeneric: {
		Name:        "Generic",
		Description: "Generic repository readiness checks",
		Checks: func(path string) []PreflightCheck {
			return []PreflightCheck{
				checkPackageManager(path),
				checkReadme(path),
				checkGitIgnore(path),
				checkLicense(path),
			}
		},
	},
	ProfileCAINextConvex: {
		Name:        "CAI Next.js + Convex",
		Description: "CAI agent-safety checks for Next.js + Convex projects",
		Checks: func(path string) []PreflightCheck {
			return []PreflightCheck{
				checkCAIDirectory(path),
				checkSIGILManifest(path),
				checkPreflightConfig(path),
				checkLaunchKit(path),
				checkTenancy(path),
				checkGirlInstalled(),
				checkCAIAgents(path),
				checkNextJSConfig(path),
				checkConvexConfig(path),
				checkClerkConfig(path),
				checkVercelConfig(path),
				checkAgentsMd(path),
				checkGrpDir(path),
				checkVerificationScripts(path),
				checkSecretFiles(path),
			}
		},
	},
}

func detectProfile(path string) PreflightProfile {
	caiDir := filepath.Join(path, ".cai")
	if info, err := os.Stat(caiDir); err == nil && info.IsDir() {
		nextConfig := checkFileExists(filepath.Join(path, "next.config.js")) || checkFileExists(filepath.Join(path, "next.config.ts"))
		convexConfig := checkFileExists(filepath.Join(path, "convex.json")) || dirExists(filepath.Join(path, "convex"))
		if nextConfig || convexConfig {
			return ProfileCAINextConvex
		}
	}
	return ProfileGeneric
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func PreflightCommand() *cli.Command {
	return &cli.Command{
		Name:      "preflight",
		Usage:     "Check CAI (Canadian AI) repo readiness",
		ArgsUsage: "[path]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "profile",
				Aliases: []string{"p"},
				Usage:   "Preflight profile: auto (default), generic, cai-next-convex",
				Value:   "auto",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: json (default), text, markdown",
				Value:   "json",
			},
			&cli.BoolFlag{
				Name:    "strict",
				Aliases: []string{"s"},
				Usage:   "Fail on warnings, not just errors",
			},
		},
		Action: func(c *cli.Context) error {
			path := c.Args().First()
			if path == "" {
				path = "."
			}

			profileStr := c.String("profile")
			if profileStr == "" {
				profileStr = "auto"
			}
			profile := PreflightProfile(profileStr)

			result := runPreflight(path, profile)

			if c.Bool("strict") && result.Status == "warn" {
				result.Status = "fail"
			}

			switch stringFlag(c, "output", "o") {
			case "text":
				printPreflightText(result)
			case "markdown":
				printPreflightMarkdown(result)
			default:
				printJSON(result)
			}

			if result.Status == "fail" {
				return fmt.Errorf("preflight failed")
			}
			return nil
		},
	}
}

func runPreflight(path string, profile PreflightProfile) *PreflightResult {
	result := &PreflightResult{
		SpecVersion: "1.0",
		ID:          fmt.Sprintf("preflight_%d", time.Now().Unix()),
		Type:        "cai-preflight",
		Path:        path,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	if profile == ProfileAuto {
		profile = detectProfile(path)
	}
	result.Profile = string(profile)

	var checks []PreflightCheck

	if p, ok := profileRegistry[profile]; ok {
		checks = p.Checks(path)
	} else {
		checks = profileRegistry[ProfileGeneric].Checks(path)
	}

	result.Checks = checks

	var pass, warn, fail int
	for _, ch := range checks {
		switch ch.Status {
		case "pass":
			pass++
		case "warn":
			warn++
		case "fail":
			fail++
		}
	}

	result.Summary.Total = pass + warn + fail
	result.Summary.Pass = pass
	result.Summary.Warn = warn
	result.Summary.Fail = fail

	switch {
	case fail > 0:
		result.Status = "fail"
	case warn > 0:
		result.Status = "warn"
	default:
		result.Status = "pass"
	}

	return result
}

func checkFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func checkFileValidJSON(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var v interface{}
	return json.Unmarshal(data, &v) == nil
}

func checkPackageManager(path string) PreflightCheck {
	if checkFileExists(filepath.Join(path, "go.mod")) {
		return PreflightCheck{
			Name:    "Package Manager",
			Status:  "pass",
			Message: "Go module detected (go.mod)",
		}
	}
	if checkFileExists(filepath.Join(path, "package.json")) {
		return PreflightCheck{
			Name:    "Package Manager",
			Status:  "pass",
			Message: "Node.js project detected (package.json)",
		}
	}
	if checkFileExists(filepath.Join(path, "Cargo.toml")) {
		return PreflightCheck{
			Name:    "Package Manager",
			Status:  "pass",
			Message: "Rust project detected (Cargo.toml)",
		}
	}
	if checkFileExists(filepath.Join(path, "requirements.txt")) || checkFileExists(filepath.Join(path, "pyproject.toml")) {
		return PreflightCheck{
			Name:    "Package Manager",
			Status:  "pass",
			Message: "Python project detected",
		}
	}
	return PreflightCheck{
		Name:    "Package Manager",
		Status:  "warn",
		Message: "No recognized package manager manifest found",
	}
}

func checkReadme(path string) PreflightCheck {
	if checkFileExists(filepath.Join(path, "README.md")) {
		return PreflightCheck{
			Name:    "README",
			Status:  "pass",
			Message: "README.md exists",
		}
	}
	return PreflightCheck{
		Name:    "README",
		Status:  "warn",
		Message: "README.md not found",
	}
}

func checkGitIgnore(path string) PreflightCheck {
	if checkFileExists(filepath.Join(path, ".gitignore")) {
		return PreflightCheck{
			Name:    ".gitignore",
			Status:  "pass",
			Message: ".gitignore exists",
		}
	}
	return PreflightCheck{
		Name:    ".gitignore",
		Status:  "warn",
		Message: ".gitignore not found",
	}
}

func checkLicense(path string) PreflightCheck {
	if checkFileExists(filepath.Join(path, "LICENSE")) {
		return PreflightCheck{
			Name:    "License",
			Status:  "pass",
			Message: "LICENSE exists",
		}
	}
	return PreflightCheck{
		Name:    "License",
		Status:  "warn",
		Message: "LICENSE not found",
	}
}

func checkNextJSConfig(path string) PreflightCheck {
	if checkFileExists(filepath.Join(path, "next.config.js")) {
		return PreflightCheck{
			Name:    "Next.js Config",
			Status:  "pass",
			Message: "next.config.js exists",
		}
	}
	if checkFileExists(filepath.Join(path, "next.config.ts")) {
		return PreflightCheck{
			Name:    "Next.js Config",
			Status:  "pass",
			Message: "next.config.ts exists",
		}
	}
	return PreflightCheck{
		Name:    "Next.js Config",
		Status:  "warn",
		Message: "Next.js config not found",
	}
}

func checkConvexConfig(path string) PreflightCheck {
	if checkFileExists(filepath.Join(path, "convex.json")) {
		return PreflightCheck{
			Name:    "Convex Config",
			Status:  "pass",
			Message: "convex.json exists",
		}
	}
	if convexDir := filepath.Join(path, "convex"); dirExists(convexDir) {
		return PreflightCheck{
			Name:    "Convex Config",
			Status:  "pass",
			Message: "convex/ directory exists",
		}
	}
	return PreflightCheck{
		Name:    "Convex Config",
		Status:  "warn",
		Message: "No convex.json or convex/ directory found",
	}
}

func checkClerkConfig(path string) PreflightCheck {
	mwFile := filepath.Join(path, "middleware.ts")
	if !checkFileExists(mwFile) {
		mwFile = filepath.Join(path, "src", "middleware.ts")
	}
	if !checkFileExists(mwFile) {
		return PreflightCheck{
			Name:    "Clerk Config",
			Status:  "warn",
			Message: "middleware.ts not found",
		}
	}
	data, err := os.ReadFile(mwFile)
	if err != nil {
		return PreflightCheck{
			Name:    "Clerk Config",
			Status:  "warn",
			Message: fmt.Sprintf("cannot read middleware.ts: %v", err),
		}
	}
	content := string(data)
	if strings.Contains(content, "clerkMiddleware") || strings.Contains(content, "@clerk/nextjs") {
		return PreflightCheck{
			Name:    "Clerk Config",
			Status:  "pass",
			Message: "middleware.ts has Clerk references",
		}
	}
	return PreflightCheck{
		Name:    "Clerk Config",
		Status:  "warn",
		Message: "middleware.ts found but no Clerk references detected",
	}
}

func checkVercelConfig(path string) PreflightCheck {
	if checkFileExists(filepath.Join(path, "vercel.json")) {
		return PreflightCheck{
			Name:    "Vercel Config",
			Status:  "pass",
			Message: "vercel.json exists",
		}
	}
	return PreflightCheck{
		Name:    "Vercel Config",
		Status:  "warn",
		Message: "vercel.json not found",
	}
}

func checkAgentsMd(path string) PreflightCheck {
	if checkFileExists(filepath.Join(path, "AGENTS.md")) {
		return PreflightCheck{
			Name:    "AGENTS.md",
			Status:  "pass",
			Message: "AGENTS.md exists",
		}
	}
	opencodeAgents := filepath.Join(path, ".opencode", "agents")
	if info, err := os.Stat(opencodeAgents); err == nil && info.IsDir() {
		return PreflightCheck{
			Name:    "AGENTS.md",
			Status:  "pass",
			Message: ".opencode/agents/ directory exists",
		}
	}
	return PreflightCheck{
		Name:    "AGENTS.md",
		Status:  "warn",
		Message: "No AGENTS.md or .opencode/agents/ found",
	}
}

func checkGrpDir(path string) PreflightCheck {
	grpDir := filepath.Join(path, ".grp")
	if info, err := os.Stat(grpDir); err == nil && info.IsDir() {
		return PreflightCheck{
			Name:    ".grp Directory",
			Status:  "pass",
			Message: ".grp/ directory exists",
		}
	}
	return PreflightCheck{
		Name:    ".grp Directory",
		Status:  "warn",
		Message: ".grp/ directory not found",
	}
}

func checkVerificationScripts(path string) PreflightCheck {
	pkgPath := filepath.Join(path, "package.json")
	if !checkFileExists(pkgPath) {
		return PreflightCheck{
			Name:    "Verification Scripts",
			Status:  "warn",
			Message: "package.json not found, cannot verify scripts",
		}
	}
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return PreflightCheck{
			Name:    "Verification Scripts",
			Status:  "warn",
			Message: fmt.Sprintf("cannot read package.json: %v", err),
		}
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil || pkg.Scripts == nil {
		return PreflightCheck{
			Name:    "Verification Scripts",
			Status:  "warn",
			Message: "package.json has no scripts section",
		}
	}
	verificationCmds := []string{"lint", "test", "typecheck", "build", "check"}
	found := []string{}
	for _, cmd := range verificationCmds {
		if _, ok := pkg.Scripts[cmd]; ok {
			found = append(found, cmd)
		}
	}
	if len(found) > 0 {
		return PreflightCheck{
			Name:    "Verification Scripts",
			Status:  "pass",
			Message: fmt.Sprintf("Found verification scripts: %s", strings.Join(found, ", ")),
		}
	}
	return PreflightCheck{
		Name:    "Verification Scripts",
		Status:  "warn",
		Message: "No standard verification scripts (lint, test, typecheck, build, check) found",
	}
}

func checkSecretFiles(path string) PreflightCheck {
	secretFiles := []string{".env", ".env.local", ".env.production", ".env.development"}
	existing := []string{}
	for _, f := range secretFiles {
		if checkFileExists(filepath.Join(path, f)) {
			existing = append(existing, f)
		}
	}
	if len(existing) > 0 {
		return PreflightCheck{
			Name:    "Secret Files",
			Status:  "pass",
			Message: fmt.Sprintf("Found secret files: %s", strings.Join(existing, ", ")),
		}
	}
	return PreflightCheck{
		Name:    "Secret Files",
		Status:  "warn",
		Message: "No .env files found (might be expected for some projects)",
	}
}

func checkCAIDirectory(path string) PreflightCheck {
	caiDir := filepath.Join(path, ".cai")
	info, err := os.Stat(caiDir)
	if err != nil || !info.IsDir() {
		return PreflightCheck{
			Name:    "CAI Directory",
			Status:  "fail",
			Message: ".cai/ directory not found",
		}
	}

	tenancyExists := checkFileExists(filepath.Join(caiDir, "tenancy.yaml"))
	launchkitExists := checkFileExists(filepath.Join(caiDir, "launchkit.yaml"))

	if tenancyExists && launchkitExists {
		return PreflightCheck{
			Name:    "CAI Directory",
			Status:  "pass",
			Message: ".cai/ exists with tenancy.yaml and launchkit.yaml",
		}
	}

	missing := []string{}
	if !tenancyExists {
		missing = append(missing, "tenancy.yaml")
	}
	if !launchkitExists {
		missing = append(missing, "launchkit.yaml")
	}
	return PreflightCheck{
		Name:    "CAI Directory",
		Status:  "warn",
		Message: fmt.Sprintf(".cai/ exists but missing: %s", strings.Join(missing, ", ")),
	}
}

func checkSIGILManifest(path string) PreflightCheck {
	caiSigil := filepath.Join(path, ".cai", "sigil.json")
	rootSigil := filepath.Join(path, "sigil.json")

	if checkFileExists(caiSigil) && checkFileValidJSON(caiSigil) {
		return PreflightCheck{
			Name:    "SIGIL Manifest",
			Status:  "pass",
			Message: ".cai/sigil.json exists and is valid JSON",
		}
	}
	if checkFileExists(rootSigil) && checkFileValidJSON(rootSigil) {
		return PreflightCheck{
			Name:    "SIGIL Manifest",
			Status:  "pass",
			Message: "sigil.json exists and is valid JSON",
		}
	}

	if checkFileExists(caiSigil) || checkFileExists(rootSigil) {
		return PreflightCheck{
			Name:    "SIGIL Manifest",
			Status:  "fail",
			Message: "SIGIL manifest found but contains invalid JSON",
		}
	}

	return PreflightCheck{
		Name:    "SIGIL Manifest",
		Status:  "warn",
		Message: "no sigil.json found in .cai/ or project root",
	}
}

func checkPreflightConfig(path string) PreflightCheck {
	caiPreflight := filepath.Join(path, ".cai", "preflight.yaml")
	if checkFileExists(caiPreflight) {
		return PreflightCheck{
			Name:    "Preflight Config",
			Status:  "pass",
			Message: ".cai/preflight.yaml exists",
		}
	}

	caiPreflightJSON := filepath.Join(path, ".cai", "preflight.json")
	if checkFileExists(caiPreflightJSON) {
		return PreflightCheck{
			Name:    "Preflight Config",
			Status:  "pass",
			Message: ".cai/preflight.json exists",
		}
	}

	return PreflightCheck{
		Name:    "Preflight Config",
		Status:  "warn",
		Message: "no preflight config found in .cai/",
	}
}

func checkLaunchKit(path string) PreflightCheck {
	launchkitPath := filepath.Join(path, ".cai", "launchkit.yaml")
	if !checkFileExists(launchkitPath) {
		return PreflightCheck{
			Name:    "Launch Kit",
			Status:  "fail",
			Message: ".cai/launchkit.yaml not found",
		}
	}

	data, err := os.ReadFile(launchkitPath)
	if err != nil {
		return PreflightCheck{
			Name:    "Launch Kit",
			Status:  "fail",
			Message: fmt.Sprintf("cannot read launchkit.yaml: %v", err),
		}
	}

	content := string(data)
	if strings.Contains(content, "gates:") || strings.Contains(content, "gate:") {
		return PreflightCheck{
			Name:    "Launch Kit",
			Status:  "pass",
			Message: ".cai/launchkit.yaml exists with gates defined",
		}
	}

	return PreflightCheck{
		Name:    "Launch Kit",
		Status:  "warn",
		Message: ".cai/launchkit.yaml exists but no gates defined",
	}
}

func checkTenancy(path string) PreflightCheck {
	tenancyPath := filepath.Join(path, ".cai", "tenancy.yaml")
	if !checkFileExists(tenancyPath) {
		return PreflightCheck{
			Name:    "Tenancy",
			Status:  "fail",
			Message: ".cai/tenancy.yaml not found",
		}
	}

	data, err := os.ReadFile(tenancyPath)
	if err != nil {
		return PreflightCheck{
			Name:    "Tenancy",
			Status:  "fail",
			Message: fmt.Sprintf("cannot read tenancy.yaml: %v", err),
		}
	}

	content := string(data)
	hasEnvironment := strings.Contains(content, "environment:") || strings.Contains(content, "env:")

	var hasIsolation bool
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "isolation") || strings.Contains(trimmed, "sandbox") || strings.Contains(trimmed, "tenant") {
			hasIsolation = true
			break
		}
	}

	if hasEnvironment && hasIsolation {
		return PreflightCheck{
			Name:    "Tenancy",
			Status:  "pass",
			Message: ".cai/tenancy.yaml exists with environment and isolation settings",
		}
	}

	missing := []string{}
	if !hasEnvironment {
		missing = append(missing, "environment setting")
	}
	if !hasIsolation {
		missing = append(missing, "isolation setting")
	}
	return PreflightCheck{
		Name:    "Tenancy",
		Status:  "warn",
		Message: fmt.Sprintf(".cai/tenancy.yaml exists but missing: %s", strings.Join(missing, ", ")),
	}
}

func checkGirlInstalled() PreflightCheck {
	_, err := exec.LookPath("girl")
	if err != nil {
		return PreflightCheck{
			Name:    "Girl Installed",
			Status:  "warn",
			Message: "girl not found on PATH",
		}
	}
	return PreflightCheck{
		Name:    "Girl Installed",
		Status:  "pass",
		Message: "girl is available on PATH",
	}
}

func checkCAIAgents(path string) PreflightCheck {
	claudePath := filepath.Join(path, "CLAUDE.md")
	if !checkFileExists(claudePath) {
		return PreflightCheck{
			Name:    "CAI Agents",
			Status:  "warn",
			Message: "CLAUDE.md not found",
		}
	}

	data, err := os.ReadFile(claudePath)
	if err != nil {
		return PreflightCheck{
			Name:    "CAI Agents",
			Status:  "warn",
			Message: fmt.Sprintf("cannot read CLAUDE.md: %v", err),
		}
	}

	content := string(data)
	if strings.Contains(content, "CAI") || strings.Contains(content, "cai") || strings.Contains(content, "Canadian AI") {
		return PreflightCheck{
			Name:    "CAI Agents",
			Status:  "pass",
			Message: "CLAUDE.md found with CAI references",
		}
	}

	return PreflightCheck{
		Name:    "CAI Agents",
		Status:  "warn",
		Message: "CLAUDE.md found but no CAI references detected",
	}
}

func printPreflightText(result *PreflightResult) {
	fmt.Printf("Preflight result for %s\n", result.Path)
	fmt.Printf("Profile: %s\n", result.Profile)
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Timestamp: %s\n", result.Timestamp)
	fmt.Println()

	for _, ch := range result.Checks {
		mark := "✓"
		switch ch.Status {
		case "pass":
			mark = "✓"
		case "warn":
			mark = "⚠"
		case "fail":
			mark = "✗"
		}
		fmt.Printf("  %s %s\n", mark, ch.Name)
		if ch.Message != "" {
			fmt.Printf("         %s\n", ch.Message)
		}
	}

	fmt.Println()
	fmt.Printf("Summary: %d total — %d pass, %d warn, %d fail\n",
		result.Summary.Total, result.Summary.Pass, result.Summary.Warn, result.Summary.Fail)
}

func printPreflightMarkdown(result *PreflightResult) {
	fmt.Printf("# Preflight Result\n\n")
	fmt.Printf("- **Path:** %s\n", result.Path)
	fmt.Printf("- **Profile:** %s\n", result.Profile)
	fmt.Printf("- **Status:** %s\n", result.Status)
	fmt.Printf("- **Timestamp:** %s\n", result.Timestamp)
	fmt.Printf("- **ID:** %s\n", result.ID)
	fmt.Println()

	for _, ch := range result.Checks {
		icon := ":white_check_mark:"
		switch ch.Status {
		case "pass":
			icon = ":white_check_mark:"
		case "warn":
			icon = ":warning:"
		case "fail":
			icon = ":x:"
		}
		fmt.Printf("- %s **%s** (%s)", icon, ch.Name, ch.Status)
		if ch.Message != "" {
			fmt.Printf(" — %s", ch.Message)
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Printf("**Summary:** %d total — %d pass, %d warn, %d fail\n",
		result.Summary.Total, result.Summary.Pass, result.Summary.Warn, result.Summary.Fail)
}
