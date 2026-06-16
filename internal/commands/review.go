package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/canadian-ai/girl/internal/diffstats"
	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/reviewability"
	"github.com/urfave/cli/v2"
)

func ReviewCommand() *cli.Command {
	return &cli.Command{
		Name:      "review",
		Usage:     "Check diff reviewability against a budget",
		ArgsUsage: "",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "diff-file",
				Usage: "Path to unified diff file",
			},
			&cli.BoolFlag{
				Name:  "stdin",
				Usage: "Read diff from stdin",
			},
			&cli.IntFlag{
				Name:  "max-diff-lines",
				Usage: "Maximum allowed diff lines",
				Value: 1500,
			},
			&cli.IntFlag{
				Name:  "max-touched-files",
				Usage: "Maximum allowed touched files",
				Value: 12,
			},
			&cli.StringFlag{
				Name:  "max-risk",
				Usage: "Maximum risk level: low, medium, high",
				Value: "medium",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: json (default), text, markdown",
				Value:   "json",
			},
			&cli.BoolFlag{
				Name:  "fail-on-over-budget",
				Usage: "Exit with non-zero status if diff exceeds budget",
			},
		},
		Action: func(c *cli.Context) error {
			diffFile := c.String("diff-file")
			readStdin := c.Bool("stdin")

			var data []byte
			var err error

			switch {
			case diffFile != "":
				data, err = os.ReadFile(diffFile)
				if err != nil {
					return fmt.Errorf("read diff file: %w", err)
				}
			case readStdin:
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) != 0 {
					return fmt.Errorf("stdin is a terminal; pipe a diff or use --diff-file")
				}
				data, err = os.ReadFile("/dev/stdin")
				if err != nil {
					return fmt.Errorf("read stdin: %w", err)
				}
			default:
				return fmt.Errorf("provide --diff-file or --stdin")
			}

			stats, err := diffstats.ParseDiffBytes(data)
			if err != nil {
				return fmt.Errorf("parse diff: %w", err)
			}

			budget := ir.ReviewabilityBudget{
				MaxDiffLines:    c.Int("max-diff-lines"),
				MaxTouchedFiles: c.Int("max-touched-files"),
				MaxRisk:         ir.Severity(c.String("max-risk")),
			}

			result := reviewability.Evaluate(stats, budget)

			overBudget := result.Result.Status == "fail"

			switch stringFlag(c, "output", "o") {
			case "text":
				printReviewText(result)
			case "markdown":
				printReviewMarkdown(result)
			default:
				printJSON(result.Result)
			}

			if c.Bool("fail-on-over-budget") && overBudget {
				return fmt.Errorf("reviewability check FAILED")
			}
			return nil
		},
	}
}

func printReviewText(r *reviewability.EvalResult) {
	res := r.Result
	status := strings.ToUpper(res.Status)
	fmt.Printf("Reviewability: %s\n\n", status)
	if res.Observed != nil {
		fmt.Printf("  Diff lines:     %d\n", res.Observed.ChangedLines)
		fmt.Printf("  Added:          %d\n", res.Observed.AddedLines)
		fmt.Printf("  Deleted:        %d\n", res.Observed.DeletedLines)
		fmt.Printf("  Files touched:  %d\n", res.Observed.ChangedFiles)
		fmt.Printf("  Largest delta:  %d lines\n", res.Observed.LargestDelta)
	}
	if res.Budget != nil {
		fmt.Printf("\nBudget:\n")
		fmt.Printf("  Max diff lines:     %d\n", res.Budget.MaxDiffLines)
		fmt.Printf("  Max touched files:  %d\n", res.Budget.MaxTouchedFiles)
		fmt.Printf("  Max risk:           %s\n", res.Budget.MaxRisk)
	}
	if res.Recommendation != "" {
		fmt.Printf("\nRecommendation: %s\n", res.Recommendation)
	}
	if res.Reason != "" {
		fmt.Printf("Reason: %s\n", res.Reason)
	}
	if len(r.Diagnostics) > 0 {
		fmt.Printf("\nDiagnostics:\n")
		for _, d := range r.Diagnostics {
			fmt.Printf("  [%s] %s\n", strings.ToUpper(string(d.Severity)), d.Message)
		}
	}
	fmt.Println()
}

func printReviewMarkdown(r *reviewability.EvalResult) {
	res := r.Result
	status := strings.ToUpper(res.Status)
	fmt.Printf("# Reviewability Report\n\n")
	fmt.Printf("**Status:** %s\n\n", status)
	if res.Observed != nil {
		fmt.Printf("## Observed\n\n")
		fmt.Printf("| Metric | Value |\n")
		fmt.Printf("|--------|-------|\n")
		fmt.Printf("| Diff lines | %d |\n", res.Observed.ChangedLines)
		fmt.Printf("| Added | %d |\n", res.Observed.AddedLines)
		fmt.Printf("| Deleted | %d |\n", res.Observed.DeletedLines)
		fmt.Printf("| Files touched | %d |\n", res.Observed.ChangedFiles)
		fmt.Printf("| Largest file delta | %d lines |\n", res.Observed.LargestDelta)
		fmt.Println()
	}
	if res.Budget != nil {
		fmt.Printf("## Budget\n\n")
		fmt.Printf("| Limit | Value |\n")
		fmt.Printf("|-------|-------|\n")
		fmt.Printf("| Max diff lines | %d |\n", res.Budget.MaxDiffLines)
		fmt.Printf("| Max touched files | %d |\n", res.Budget.MaxTouchedFiles)
		fmt.Printf("| Max risk | %s |\n", res.Budget.MaxRisk)
		fmt.Println()
	}
	if res.Recommendation != "" {
		fmt.Printf("**Recommendation:** %s\n\n", res.Recommendation)
	}
	if res.Reason != "" {
		fmt.Printf("**Reason:** %s\n\n", res.Reason)
	}
	if len(r.Diagnostics) > 0 {
		fmt.Printf("## Diagnostics\n\n")
		for _, d := range r.Diagnostics {
			fmt.Printf("- [%s] `%s`: %s\n", strings.ToUpper(string(d.Severity)), d.Code, d.Message)
		}
		fmt.Println()
	}
}
