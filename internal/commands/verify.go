package commands

import (
	"fmt"

	"github.com/canadian-ai/girl/internal/verifier"
	"github.com/urfave/cli/v2"
)

func VerifyCommand() *cli.Command {
	return &cli.Command{
		Name:      "verify",
		Usage:     "Detect available verification commands for a project",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: json (default), text",
				Value:   "json",
			},
		},
		Action: func(c *cli.Context) error {
			path := c.Args().First()
			if path == "" {
				path = "."
			}

			v := verifier.NewVerifier()
			result, err := v.Verify(path)
			if err != nil {
				return fmt.Errorf("verification failed: %w", err)
			}

			switch stringFlag(c, "output", "o") {
			case "text":
				fmt.Printf("Package manager: %s\n", result.PackageManager)
				fmt.Printf("Work directory: %s\n\n", result.WorkDir)

				if len(result.Commands) > 0 {
					fmt.Println("Available verification commands:")
					for _, cmd := range result.Commands {
						fmt.Printf("  ✓ %s (%s)\n", cmd.Command, cmd.Name)
					}
				} else {
					fmt.Println("No standard verification commands found.")
				}

				fmt.Println()
				checks := []struct {
					name string
					val  bool
				}{
					{"TypeScript config", result.HasConfig},
					{"Convex project", result.HasConvex},
					{"Dockerfile", result.HasDocker},
					{"CI workflow", result.HasCI},
				}
				for _, c := range checks {
					mark := "✗"
					if c.val {
						mark = "✓"
					}
					fmt.Printf("  %s %s\n", mark, c.name)
				}
			default:
				printJSON(result)
			}

			return nil
		},
	}
}
