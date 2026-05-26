package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canadian-ai/girl/internal/analyzer"
	"github.com/canadian-ai/girl/internal/goanalysis"
	"github.com/canadian-ai/girl/internal/ir"
	"github.com/urfave/cli/v2"
)

func AnalyzeCommand() *cli.Command {
	return &cli.Command{
		Name:      "analyze",
		Usage:     "Analyze a file or directory for refactoring opportunities",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: json (default), text, markdown",
				Value:   "json",
			},
			&cli.StringFlag{
				Name:  "lang",
				Usage: "Language mode: auto (default), go, ts",
				Value: "auto",
			},
			&cli.IntFlag{
				Name:  "max-lines",
				Usage: "Maximum component lines before warning",
				Value: 200,
			},
			&cli.StringSliceFlag{
				Name:  "exclude",
				Usage: "Directories to exclude",
			},
		},
		Action: func(c *cli.Context) error {
			path := c.Args().First()
			if path == "" {
				path = "."
			}

			lang := resolveLang(path, c.String("lang"))

			var result *ir.AnalyzerResult
			var err error

			if lang == "go" {
				cfg := goanalysis.DefaultConfig()
				result, err = goanalysis.AnalyzePath(path, cfg)
			} else {
				cfg := analyzer.DefaultConfig()
				cfg.MaxComponentLines = c.Int("max-lines")
				cfg.ExcludeDirs = c.StringSlice("exclude")
				a := analyzer.NewAnalyzer(cfg)
				result, err = a.Analyze(path)
			}
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}

			switch stringFlag(c, "output", "o") {
			case "text":
				printText(result)
			case "markdown":
				printMarkdown(result)
			default:
				printJSON(result)
			}

			return nil
		},
	}
}

func printJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("{\"error\":%q}\n", err)
		return
	}
	fmt.Println(string(data))
}

func printText(result *ir.AnalyzerResult) {
	if len(result.Diagnostics) == 0 {
		fmt.Println("No issues found.")
		return
	}

	diags := result.Diagnostics
	fmt.Printf("Found %d issue(s):\n\n", len(diags))

	high, med, low := 0, 0, 0
	for _, d := range diags {
		switch d.Severity {
		case ir.SeverityHigh:
			high++
		case ir.SeverityMedium:
			med++
		case ir.SeverityLow:
			low++
		}
	}
	fmt.Printf("  High:   %d\n", high)
	fmt.Printf("  Medium: %d\n", med)
	fmt.Printf("  Low:    %d\n\n", low)

	for _, d := range diags {
		sev := strings.ToUpper(string(d.Severity))
		fmt.Printf("[%s] %s\n", sev, d.Message)
		if d.Suggestion != "" {
			fmt.Printf("      %s\n\n", d.Suggestion)
		}
	}
}

func printMarkdown(result *ir.AnalyzerResult) {
	if len(result.Diagnostics) == 0 {
		fmt.Println("No issues found.")
		return
	}

	fmt.Printf("# GIRL Analysis Report\n\n")
	fmt.Printf("**Files analyzed:** %d\n\n", len(result.Files))

	high, med, low := 0, 0, 0
	for _, d := range result.Diagnostics {
		switch d.Severity {
		case ir.SeverityHigh:
			high++
		case ir.SeverityMedium:
			med++
		case ir.SeverityLow:
			low++
		}
	}

	fmt.Printf("| Severity | Count |\n")
	fmt.Printf("|----------|-------|\n")
	fmt.Printf("| High     | %d     |\n", high)
	fmt.Printf("| Medium   | %d     |\n", med)
	fmt.Printf("| Low      | %d     |\n\n", low)

	fmt.Printf("## Diagnostics\n\n")
	for _, d := range result.Diagnostics {
		badge := map[string]string{
			"high":   "🔴",
			"medium": "🟡",
			"low":    "🟢",
		}
		fmt.Printf("### %s %s\n\n", badge[string(d.Severity)], d.Message)
		fmt.Printf("- **File:** `%s`\n", d.File)
		fmt.Printf("- **Code:** `%s`\n", d.Code)
		fmt.Printf("- **Suggestion:** %s\n\n", d.Suggestion)
	}
}
