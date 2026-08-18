package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/canadian-ai/girl/pkg/grp"
	"github.com/urfave/cli/v2"
)

func ValidateCommand() *cli.Command {
	return &cli.Command{
		Name:      "validate",
		Usage:     "Validate a GRP plan JSON file",
		ArgsUsage: "<file>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: text (default) or markdown (human-readable plan)",
				Value:   "text",
			},
		},
		Action: func(c *cli.Context) error {
			path := c.Args().First()
			if path == "" {
				return fmt.Errorf("usage: girl validate <file>")
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			var plan grp.Plan
			if err := json.Unmarshal(data, &plan); err != nil {
				return fmt.Errorf("parse JSON: %w", err)
			}

			result := grp.ValidatePlan(&plan)
			if !result.Valid {
				for _, e := range result.Errors {
					fmt.Fprintf(os.Stderr, "  %s: %s\n", e.Field, e.Message)
				}
				return fmt.Errorf("plan validation failed with %d errors", len(result.Errors))
			}

			if stringFlag(c, "output", "o") == "markdown" {
				fmt.Print(grp.RenderPlanMarkdown(&plan))
				return nil
			}

			fmt.Println("Plan is valid.")
			return nil
		},
	}
}
