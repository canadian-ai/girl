package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/planner"
	"github.com/canadian-ai/girl/pkg/grp"
	"github.com/urfave/cli/v2"
)

func PlanCommand() *cli.Command {
	return &cli.Command{
		Name:      "plan",
		Usage:     "Generate a structured GRP refactor plan",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "goal",
				Usage: "Refactoring goal",
			},
			&cli.StringFlag{
				Name:  "recipe",
				Usage: "Specific recipe to apply",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: json (default), markdown, grp-json",
				Value:   "json",
			},
			&cli.StringFlag{
				Name:  "lang",
				Usage: "Language mode: auto (default), go, ts",
				Value: "auto",
			},
			&cli.IntFlag{
				Name:  "budget",
				Usage: "Token budget for the plan",
				Value: 12000,
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
				Recipe:      c.String("recipe"),
				Diagnostics: result.Diagnostics,
				Files:       result.Files,
				Lang:        lang,
			})

			switch stringFlag(c, "output", "o") {
			case "markdown":
				printPlanMarkdown(plan)
			case "grp-json":
				gp := grp.FromIRPlan(plan)
				gp.Language = lang
				grp.NormalizePlan(gp)
				gp.ID = canonicalGRPPlanID(plan, lang)
				result := grp.ValidatePlan(gp)
				if !result.Valid {
					fmt.Fprintf(os.Stderr, "Warning: generated plan has validation issues:\n")
					for _, e := range result.Errors {
						fmt.Fprintf(os.Stderr, "  - %s: %s\n", e.Field, e.Message)
					}
				}
				if err := writeGRPPlanFile(path, gp); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				}
				printJSON(gp)
			default:
				printJSON(plan)
			}

			if stringFlag(c, "output", "o") != "grp-json" {
				_ = writePlanFile(path, plan)
			}
			return nil
		},
	}
}

func writeGRPPlanFile(path string, plan *grp.Plan) error {
	planDir := filepath.Join(path, ".grp")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		return fmt.Errorf("create plan directory: %w", err)
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	planFile := filepath.Join(planDir, "plan.json")
	return os.WriteFile(planFile, data, 0644)
}

func writePlanFile(path string, plan *ir.GrpPlan) error {
	planDir := filepath.Join(path, ".grp")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		return fmt.Errorf("create plan directory: %w", err)
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plan: %w", err)
	}
	planFile := filepath.Join(planDir, "plan.json")
	if err := os.WriteFile(planFile, data, 0644); err != nil {
		return fmt.Errorf("write plan: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Plan written to %s\n", planFile)
	return nil
}

func printPlanMarkdown(plan *ir.GrpPlan) {
	fmt.Printf("# GRP Refactor Plan: %s\n\n", plan.PlanID)
	fmt.Printf("**Goal:** %s\n\n", plan.Goal)
	fmt.Printf("**Risk level:** %s\n\n", strings.ToUpper(string(plan.Risk)))
	fmt.Printf("**Files to touch:** %d\n", plan.FileCount)
	fmt.Printf("**Token estimate:** ~%d\n\n", plan.TokenEstimate)

	if len(plan.Diagnostics) > 0 {
		fmt.Printf("## Diagnostics\n\n")
		for _, d := range plan.Diagnostics {
			fmt.Printf("- [%s] %s (`%s`)\n", strings.ToUpper(string(d.Severity)), d.Message, d.Code)
		}
		fmt.Println()
	}

	fmt.Printf("## Steps\n\n")
	for _, s := range plan.Steps {
		fmt.Printf("### %s: %s\n\n", s.ID, s.Action)
		fmt.Printf("- **Recipe:** `%s`\n", s.Recipe)
		fmt.Printf("- **File:** `%s`\n", s.File)
		fmt.Printf("- **Risk:** %s\n", s.Risk)
		fmt.Printf("- **Verify:** %s\n\n", strings.Join(s.Verify, ", "))
	}

	if len(plan.Verification) > 0 {
		fmt.Printf("## Verification\n\n")
		fmt.Printf("Recommended commands:\n\n")
		for _, v := range plan.Verification {
			fmt.Printf("```bash\n%s\n```\n\n", v)
		}
	}
}
