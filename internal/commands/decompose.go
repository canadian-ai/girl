package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/canadian-ai/girl/internal/decomposer"
	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/structural"
	"github.com/urfave/cli/v2"
)

func DecomposeCommand() *cli.Command {
	return &cli.Command{
		Name:      "decompose",
		Usage:     "Decompose a large diff into smaller reviewable tasks",
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
				Usage: "Maximum allowed diff lines per task",
				Value: 400,
			},
			&cli.IntFlag{
				Name:  "max-touched-files",
				Usage: "Maximum allowed touched files per task",
				Value: 5,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: json (default), markdown",
				Value:   "json",
			},
			&cli.StringFlag{
				Name:    "output-file",
				Aliases: []string{"f"},
				Usage:   "Write decomposition to file (e.g., .grp/decomposition.json)",
			},
		},
		Action: func(c *cli.Context) error {
			stats, err := parseDiffFromFlags(c)
			if err != nil {
				return err
			}

			// Compute structural clusters for guided decomposition
			class := structural.Classify(stats)
			suggestedClusters := class.Cohesion.SuggestedClusters

			decomp := decomposer.Decompose(&decomposer.DecomposeRequest{
				DiffStats:         stats,
				SuggestedClusters: suggestedClusters,
			})

			outputFile := c.String("output-file")

			switch stringFlag(c, "output", "o") {
			case "markdown":
				printDecompositionMarkdown(decomp)
			default:
				if outputFile != "" {
					if err := writeJSONFile(outputFile, decomp); err != nil {
						return fmt.Errorf("write decomposition: %w", err)
					}
				}
				printJSON(decomp)
			}

			if len(decomp.Tasks) == 0 {
				fmt.Fprintf(os.Stderr, "Warning: no decomposition tasks generated (diff may be too small)\n")
			}

			return nil
		},
	}
}

func printDecompositionMarkdown(d *ir.Decomposition) {
	fmt.Printf("# GIRL Decomposition\n\n")
	fmt.Printf("**Strategy:** %s\n\n", d.Strategy)
	if d.ParentPlan != "" {
		fmt.Printf("**Parent plan:** `%s`\n\n", d.ParentPlan)
	}
	fmt.Printf("**Tasks:** %d\n\n", len(d.Tasks))

	for _, task := range d.Tasks {
		fmt.Printf("## %s: %s\n\n", task.ID, task.Goal)
		fmt.Printf("- **Max diff lines:** %d\n", task.MaxDiffLines)
		fmt.Printf("- **Parallelizable:** %t\n", task.Parallelizable)
		if len(task.DependsOn) > 0 {
			fmt.Printf("- **Depends on:** %s\n", formatList(task.DependsOn))
		}
		if len(task.AllowedFiles) > 0 {
			fmt.Printf("- **Files:**\n")
			for _, f := range task.AllowedFiles {
				fmt.Printf("  - `%s`\n", f)
			}
		}
		if len(task.Verification) > 0 {
			fmt.Printf("- **Verification:**\n")
			for _, v := range task.Verification {
				fmt.Printf("  - `%s`\n", v)
			}
		}
		fmt.Println()
	}
}

func formatList(items []string) string {
	return strings.Join(items, ", ")
}
