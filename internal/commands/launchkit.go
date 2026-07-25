package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

// CAI Launch Kit JSON Schema (draft-07)
//
// See schemas/cai-launchkit.v0.1.schema.json for the authoritative schema.
//
// Schema summary:
//
//	{
//	  "$schema": "http://json-schema.org/draft-07/schema#",
//	  "title": "CAI Launch Kit",
//	  "type": "object",
//	  "properties": {
//	    "launch_kit": {
//	      "type": "object",
//	      "properties": {
//	        "version":  { "type": "string", "pattern": "^\\d+\\.\\d+$" },
//	        "gates":    { "type": "object" },
//	        "receipt":  {
//	          "type": "object",
//	          "properties": {
//	            "enabled": { "type": "boolean" },
//	            "format":  { "type": "string", "enum": ["json", "yaml"] }
//	          }
//	        }
//	      },
//	      "required": ["version", "gates"]
//	    }
//	  }
//	}

type launchKitGateConfig struct {
	Required bool   `json:"required"`
	Command  string `json:"command"`
}

type launchKitReceiptConfig struct {
	Enabled bool   `json:"enabled"`
	Format  string `json:"format"`
}

type launchKitConfig struct {
	Version string                          `json:"version,omitempty"`
	Gates   map[string]launchKitGateConfig   `json:"gates"`
	Receipt launchKitReceiptConfig           `json:"receipt,omitempty"`
}

type LaunchKitGateResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Command string `json:"command,omitempty"`
}

type LaunchKitValidation struct {
	SpecVersion string                `json:"specversion"`
	ID          string                `json:"id"`
	Type        string                `json:"type"`
	Path        string                `json:"path"`
	Valid       bool                  `json:"valid"`
	Gates       []LaunchKitGateResult `json:"gates"`
	Timestamp   string                `json:"timestamp"`
}

func LaunchKitCommand() *cli.Command {
	return &cli.Command{
		Name:  "launchkit",
		Usage: "Launch kit quality gate management",
		Subcommands: []*cli.Command{
			launchKitValidateCommand(),
		},
	}
}

func launchKitValidateCommand() *cli.Command {
	return &cli.Command{
		Name:      "validate",
		Usage:     "Validate launch kit quality gates",
		ArgsUsage: "[path]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: json (default), text, markdown",
				Value:   "json",
			},
			&cli.StringSliceFlag{
				Name:    "gate",
				Aliases: []string{"g"},
				Usage:   "Specific gate to validate (preflight, review, launchkit_validate, prove_app). May be specified multiple times.",
			},
			&cli.BoolFlag{
				Name:  "fail-fast",
				Usage: "Stop on first gate failure",
			},
		},
		Action: func(c *cli.Context) error {
			path := c.Args().First()
			if path == "" {
				path = "."
			}

			cfg, filePath, err := findLaunchKit(path)
			if err != nil {
				return fmt.Errorf("launch kit not found: %w", err)
			}

			gates := validateLaunchKitGates(cfg, c.StringSlice("gate"), c.Bool("fail-fast"))

			valid := true
			for _, g := range gates {
				if g.Status == "fail" {
					valid = false
					break
				}
			}

			result := &LaunchKitValidation{
				SpecVersion: "0.1",
				ID:          "launchkit-validate-" + fmt.Sprintf("%d", time.Now().Unix()),
				Type:        "launchkit_validation",
				Path:        filePath,
				Valid:       valid,
				Gates:       gates,
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
			}

			switch c.String("output") {
			case "text":
				printLaunchKitText(result)
			case "markdown":
				printLaunchKitMarkdown(result)
			default:
				printJSON(result)
			}

			if !valid {
				return fmt.Errorf("launch kit validation failed")
			}
			return nil
		},
	}
}

func findLaunchKit(path string) (*launchKitConfig, string, error) {
	candidates := []string{
		filepath.Join(path, ".cai", "launchkit.json"),
		filepath.Join(path, ".cai", "launchkit.yaml"),
		filepath.Join(path, ".cai", "launchkit.yml"),
		filepath.Join(path, "launchkit.json"),
		filepath.Join(path, "launchkit.yaml"),
		filepath.Join(path, "launchkit.yml"),
	}

	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}

		switch filepath.Ext(candidate) {
		case ".json":
			cfg, err := parseLaunchKitJSON(data)
			if err == nil {
				return cfg, candidate, nil
			}
		case ".yaml", ".yml":
			cfg, err := parseLaunchKitYAML(data)
			if err == nil {
				return cfg, candidate, nil
			}
		}
	}

	return nil, "", fmt.Errorf("no launch kit file found in %s (looked for .cai/launchkit.{json,yaml,yml} or launchkit.{json,yaml,yml})", path)
}

func parseLaunchKitJSON(data []byte) (*launchKitConfig, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	if lk, ok := raw["launch_kit"]; ok {
		var cfg launchKitConfig
		if err := json.Unmarshal(lk, &cfg); err != nil {
			return nil, err
		}
		if cfg.Gates == nil {
			cfg.Gates = make(map[string]launchKitGateConfig)
		}
		return &cfg, nil
	}

	var cfg launchKitConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Gates == nil {
		cfg.Gates = make(map[string]launchKitGateConfig)
	}
	return &cfg, nil
}

func parseLaunchKitYAML(data []byte) (*launchKitConfig, error) {
	cfg := &launchKitConfig{
		Gates:   make(map[string]launchKitGateConfig),
		Receipt: launchKitReceiptConfig{},
	}

	lines := strings.Split(string(data), "\n")
	var inGates, inReceipt bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if trimmed == "launch_kit:" {
			continue
		}
		if trimmed == "gates:" {
			inGates = true
			inReceipt = false
			continue
		}
		if trimmed == "receipt:" {
			inReceipt = true
			inGates = false
			continue
		}

		if !strings.Contains(trimmed, ":") {
			continue
		}

		parts := strings.SplitN(trimmed, ":", 2)
		key := strings.TrimSpace(parts[0])
		rest := strings.TrimSpace(parts[1])

		if inGates {
			if strings.HasPrefix(rest, "{") && strings.HasSuffix(rest, "}") {
				inner := rest[1 : len(rest)-1]
				m := parseInlineYAMLMap(inner)
				cfg.Gates[key] = launchKitGateConfig{
					Required: m["required"] == "true",
					Command:  strings.Trim(m["command"], "\""),
				}
			}
			continue
		}

		if inReceipt {
			switch key {
			case "enabled":
				cfg.Receipt.Enabled = rest == "true" || rest == "yes"
			case "format":
				cfg.Receipt.Format = strings.Trim(rest, "\"")
			}
			continue
		}

		if key == "version" {
			cfg.Version = strings.Trim(rest, "\"")
		}
	}

	if cfg.Version == "" && len(cfg.Gates) == 0 {
		return nil, fmt.Errorf("could not parse launch kit from YAML")
	}

	return cfg, nil
}

func parseInlineYAMLMap(s string) map[string]string {
	m := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if kv := strings.SplitN(part, ":", 2); len(kv) == 2 {
			m[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return m
}

func validateLaunchKitGates(cfg *launchKitConfig, gateFilter []string, failFast bool) []LaunchKitGateResult {
	filterSet := make(map[string]bool)
	for _, f := range gateFilter {
		if f != "" {
			filterSet[f] = true
		}
	}

	var results []LaunchKitGateResult

	// --- Schema-level validation (always runs, not filtered) ---

	// 1. Version format validation (must be semver-like, e.g., "0.1")
	if cfg.Version != "" {
		if matched, _ := regexp.MatchString(`^\d+\.\d+$`, cfg.Version); !matched {
			results = append(results, LaunchKitGateResult{
				Name:    "version",
				Status:  "fail",
				Message: fmt.Sprintf("version %q is not valid semver (expected format: MAJOR.MINOR, e.g. \"0.1\")", cfg.Version),
			})
			if failFast {
				return results
			}
		}
	}

	// 2. Receipt config validation (if enabled, format must be "json" or "yaml")
	if cfg.Receipt.Enabled && cfg.Receipt.Format != "" {
		if cfg.Receipt.Format != "json" && cfg.Receipt.Format != "yaml" {
			results = append(results, LaunchKitGateResult{
				Name:    "receipt",
				Status:  "warn",
				Message: fmt.Sprintf("receipt format %q is not valid; expected \"json\" or \"yaml\"", cfg.Receipt.Format),
			})
		}
	}

	// 3. Duplicate gate detection (case-insensitive key collision)
	seenGates := make(map[string]string)
	for name := range cfg.Gates {
		lower := strings.ToLower(name)
		if orig, ok := seenGates[lower]; ok {
			results = append(results, LaunchKitGateResult{
				Name:    name,
				Status:  "warn",
				Message: fmt.Sprintf("gate %q is a case-insensitive duplicate of %q", name, orig),
			})
		}
		seenGates[lower] = name
	}

	// 4. Unknown gate name detection
	knownGates := map[string]bool{"preflight": true, "review": true, "launchkit_validate": true, "prove_app": true}
	for name := range cfg.Gates {
		if !knownGates[name] {
			results = append(results, LaunchKitGateResult{
				Name:    name,
				Status:  "warn",
				Message: fmt.Sprintf("unknown gate %q; expected one of: preflight, review, launchkit_validate, prove_app", name),
			})
		}
	}

	// --- Per-gate validation ---
	allGates := []struct {
		Name     string
		Validate func(*launchKitConfig) (status, msg string)
	}{
		{
			Name: "preflight",
			Validate: func(cfg *launchKitConfig) (string, string) {
				g, ok := cfg.Gates["preflight"]
				if !ok {
					return "fail", "preflight gate not configured"
				}
				if !g.Required {
					return "warn", "preflight gate exists but is not marked required"
				}
				if g.Command == "" {
					return "warn", "preflight gate has no command"
				}
				if !strings.HasPrefix(g.Command, "girl ") {
					return "warn", fmt.Sprintf("preflight command %q does not start with \"girl \"", g.Command)
				}
				if !strings.Contains(g.Command, "preflight") {
					return "warn", fmt.Sprintf("preflight command %q does not reference preflight", g.Command)
				}
				return "pass", "preflight gate configured correctly"
			},
		},
		{
			Name: "review",
			Validate: func(cfg *launchKitConfig) (string, string) {
				g, ok := cfg.Gates["review"]
				if !ok {
					return "fail", "review gate not configured"
				}
				if !g.Required {
					return "warn", "review gate exists but is not marked required"
				}
				if g.Command == "" {
					return "warn", "review gate has no command"
				}
				if !strings.HasPrefix(g.Command, "girl ") {
					return "warn", fmt.Sprintf("review command %q does not start with \"girl \"", g.Command)
				}
				return "pass", "review gate configured correctly"
			},
		},
		{
			Name: "launchkit_validate",
			Validate: func(cfg *launchKitConfig) (string, string) {
				g, ok := cfg.Gates["launchkit_validate"]
				if !ok {
					return "fail", "launchkit_validate gate not configured"
				}
				if !g.Required {
					return "warn", "launchkit_validate gate exists but is not marked required"
				}
				if g.Command == "" {
					return "warn", "launchkit_validate gate has no command"
				}
				if !strings.HasPrefix(g.Command, "girl ") {
					return "warn", fmt.Sprintf("launchkit_validate command %q does not start with \"girl \"", g.Command)
				}
				if cfg.Version == "" {
					return "fail", "launch kit has no version"
				}
				return "pass", "launch kit self-validation passed"
			},
		},
		{
			Name: "prove_app",
			Validate: func(cfg *launchKitConfig) (string, string) {
				g, ok := cfg.Gates["prove_app"]
				if !ok {
					return "fail", "prove_app gate not configured"
				}
				if !g.Required {
					return "warn", "prove_app gate exists but is not marked required"
				}
				if g.Command == "" {
					return "warn", "prove_app gate has no command"
				}
				if !strings.HasPrefix(g.Command, "girl ") {
					return "warn", fmt.Sprintf("prove_app command %q does not start with \"girl \"", g.Command)
				}
				return "pass", "prove_app gate configured correctly"
			},
		},
	}

	for _, k := range allGates {
		if len(filterSet) > 0 && !filterSet[k.Name] {
			continue
		}

		status, message := k.Validate(cfg)
		g := cfg.Gates[k.Name]

		results = append(results, LaunchKitGateResult{
			Name:    k.Name,
			Status:  status,
			Message: message,
			Command: g.Command,
		})

		if failFast && status == "fail" {
			break
		}
	}

	return results
}

func printLaunchKitText(v *LaunchKitValidation) {
	fmt.Printf("Launch Kit Validation: %s\n", filepath.Base(v.Path))
	fmt.Printf("  Spec Version: %s\n", v.SpecVersion)
	fmt.Printf("  ID: %s\n", v.ID)
	fmt.Printf("  Timestamp: %s\n", v.Timestamp)
	fmt.Printf("  Valid: %v\n\n", v.Valid)

	fmt.Println("  Gates:")
	for _, g := range v.Gates {
		icon := map[string]string{"pass": "✓", "warn": "!", "fail": "✗"}
		fmt.Printf("    %s %s: %s\n", icon[g.Status], g.Name, g.Message)
		if g.Command != "" {
			fmt.Printf("      Command: %s\n", g.Command)
		}
	}

	fmt.Println()
	if v.Valid {
		fmt.Println("  Result: PASS")
	} else {
		fmt.Println("  Result: FAIL")
	}
}

func printLaunchKitMarkdown(v *LaunchKitValidation) {
	fmt.Printf("# Launch Kit Validation\n\n")
	fmt.Printf("- **File:** `%s`\n", filepath.Base(v.Path))
	fmt.Printf("- **Spec Version:** %s\n", v.SpecVersion)
	fmt.Printf("- **ID:** `%s`\n", v.ID)
	fmt.Printf("- **Timestamp:** %s\n", v.Timestamp)
	fmt.Printf("- **Valid:** %v\n\n", v.Valid)

	fmt.Println("## Gates")
	fmt.Println()
	fmt.Println("| Gate | Status | Message | Command |")
	fmt.Println("|------|--------|---------|---------|")
	for _, g := range v.Gates {
		icon := map[string]string{"pass": "✅ Pass", "warn": "⚠️ Warn", "fail": "❌ Fail"}
		fmt.Printf("| %s | %s | %s | `%s` |\n", g.Name, icon[g.Status], g.Message, g.Command)
	}
	fmt.Println()
	if v.Valid {
		fmt.Println("**Result: PASS** ✅")
	} else {
		fmt.Println("**Result: FAIL** ❌")
	}
}
