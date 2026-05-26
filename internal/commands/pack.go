package commands

import (
	"fmt"
	"strings"

	"github.com/canadian-ai/girl/internal/ir"
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
			&cli.StringFlag{
				Name:  "lang",
				Usage: "Language mode: auto (default), go, ts",
				Value: "auto",
			},
		},
		Action: func(c *cli.Context) error {
			path := commandPath(c)
			lang := resolveLang(path, c.String("lang"))
			result, err := analyzePath(path, lang)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}

			p := planner.NewPlanner()
			plan := p.GeneratePlan(planner.PlanRequest{
				Target:      path,
				Goal:        c.String("goal"),
				Diagnostics: result.Diagnostics,
				Files:       result.Files,
				Lang:        lang,
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

			if stringFlag(c, "output", "o") == "markdown" {
				printPackMarkdown(pack)
			} else {
				printJSON(pack)
			}

			return nil
		},
	}
}

func printPackMarkdown(pack *ir.ContextPack) {
	fmt.Printf("# GIRL Context Pack\n\n")
	fmt.Printf("**Goal:** %s\n\n", pack.Goal)
	fmt.Printf("**Token budget:** %d\n", pack.TokenBudget)
	fmt.Printf("**Token estimate:** %d\n\n", pack.TokenEstimate)
	printPackList("Files", pack.Files, func(f string) string { return fmt.Sprintf("`%s`", f) })
	printPackList("Summaries", pack.Summaries, func(s ir.FileSummary) string {
		return fmt.Sprintf("`%s`: %s", s.Path, s.Summary)
	})
	printPackList("Diagnostics", pack.Diagnostics, func(d ir.Diagnostic) string {
		return fmt.Sprintf("[%s] %s", strings.ToUpper(string(d.Severity)), d.Message)
	})
	printPackList("Steps", pack.Steps, func(s ir.GrpStep) string {
		return fmt.Sprintf("%s: %s", s.ID, s.Action)
	})
	printPackList("Risks", pack.Risks, func(r string) string { return r })
	printPackVerification(pack.Verification)
}

func printPackList[T any](title string, rows []T, format func(T) string) {
	if len(rows) == 0 {
		return
	}
	fmt.Printf("## %s\n\n", title)
	for _, row := range rows {
		fmt.Printf("- %s\n", format(row))
	}
	fmt.Println()
}

func printPackVerification(commands []string) {
	if len(commands) == 0 {
		return
	}
	fmt.Printf("## Verification\n\n")
	for _, v := range commands {
		fmt.Printf("```bash\n%s\n```\n\n", v)
	}
}
