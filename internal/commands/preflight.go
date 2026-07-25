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

func PreflightCommand() *cli.Command {
	return &cli.Command{
		Name:      "preflight",
		Usage:     "Check CAI (Canadian AI) repo readiness",
		ArgsUsage: "[path]",
		Flags: []cli.Flag{
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

			result := runPreflight(path)

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

func runPreflight(path string) *PreflightResult {
	result := &PreflightResult{
		SpecVersion: "1.0",
		ID:          fmt.Sprintf("preflight_%d", time.Now().Unix()),
		Type:        "cai-preflight",
		Path:        path,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	checks := []PreflightCheck{}

	checks = append(checks, checkCAIDirectory(path))
	checks = append(checks, checkSIGILManifest(path))
	checks = append(checks, checkPreflightConfig(path))
	checks = append(checks, checkLaunchKit(path))
	checks = append(checks, checkTenancy(path))
	checks = append(checks, checkGirlInstalled())
	checks = append(checks, checkCAIAgents(path))

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
