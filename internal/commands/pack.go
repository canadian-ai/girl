package commands

import (
	"fmt"
	"strings"

	"github.com/canadian-ai/girl/internal/analyzer"
	"github.com/canadian-ai/girl/internal/packer"
	"github.com/canadian-ai/girl/internal/planner"
	"github.com/urfave/cli/v2"
)

func PackCommand() *cli.Command {
	return &cli.Command{
		Name:      "pack",
		Usage:     "Create a compact agent context pack from a plan",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "goal",
				Usage: "Refactoring goal",
			},
			&cli.IntFlag{
				Name:  "budget",
				Usage: "Token budget for context pack",
				Value: 12000,
			},
			&cli.StringFlag{
				Name:  "privacy",
				Usage: "Privacy mode: public, private",
				Value: "private",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: json (default), markdown",
				Value:   "json",
			},
		},
		Action: func(c *cli.Context) error {
			path := c.Args().First()
			if path == "" {
				path = "."
			}

			a := analyzer.NewAnalyzer(nil)
			result, err := a.Analyze(path)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}

			p := planner.NewPlanner()
			plan := p.GeneratePlan(planner.PlanRequest{
				Target:      path,
				Goal:        c.String("goal"),
				Diagnostics: result.Diagnostics,
				Files:       result.Files,
			})

			pkr := packer.NewPacker(c.Int("budget"))
			pack, err := pkr.Pack(packer.PackRequest{
				Goal:        plan.Goal,
				Files:       result.Files,
				Diagnostics: plan.Diagnostics,
				Steps:       plan.Steps,
				TokenBudget: c.Int("budget"),
			})
			if err != nil {
				return fmt.Errorf("packing failed: %w", err)
			}

			switch stringFlag(c, "output", "o") {
			case "markdown":
				fmt.Printf("# GIRL Context Pack\n\n")
				fmt.Printf("**Goal:** %s\n\n", pack.Goal)
				fmt.Printf("**Token budget:** %d\n", pack.TokenBudget)
				fmt.Printf("**Token estimate:** %d\n\n", pack.TokenEstimate)

				fmt.Printf("## Files\n\n")
				for _, f := range pack.Files {
					fmt.Printf("- `%s`\n", f)
				}
				fmt.Println()
				fmt.Printf("## Summaries\n\n")
				for _, s := range pack.Summaries {
					fmt.Printf("- `%s`: %s\n", s.Path, s.Summary)
				}
				fmt.Println()

				if len(pack.Diagnostics) > 0 {
					fmt.Printf("## Diagnostics\n\n")
					for _, d := range pack.Diagnostics {
						fmt.Printf("- [%s] %s\n", strings.ToUpper(string(d.Severity)), d.Message)
					}
					fmt.Println()
				}

				fmt.Printf("## Steps\n\n")
				for _, s := range pack.Steps {
					fmt.Printf("- %s: %s\n", s.ID, s.Action)
				}
				fmt.Println()

				if len(pack.Risks) > 0 {
					fmt.Printf("## Risks\n\n")
					for _, r := range pack.Risks {
						fmt.Printf("- %s\n", r)
					}
					fmt.Println()
				}

				if len(pack.Verification) > 0 {
					fmt.Printf("## Verification\n\n")
					for _, v := range pack.Verification {
						fmt.Printf("```bash\n%s\n```\n\n", v)
					}
				}
			default:
				printJSON(pack)
			}

			return nil
		},
	}
}
