package commands

import (
	"encoding/json"
	"fmt"

	"github.com/canadian-ai/girl/internal/proof"
	"github.com/urfave/cli/v2"
)

func ProveCommand() *cli.Command {
	return &cli.Command{
		Name:      "prove",
		Usage:     "Generate a shareable repository health proof report",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "lang", Usage: "Language mode: auto (default), go, ts, rust", Value: "auto"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text (default), json, markdown", Value: "text"},
		},
		Action: func(c *cli.Context) error {
			path := commandPath(c)
			lang := resolveLang(path, c.String("lang"))
			result, err := analyzePath(path, lang)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}
			report := proof.NewProofReport(path, result, 10)
			switch stringFlag(c, "output", "o") {
			case "json":
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
			case "markdown":
				fmt.Print(proof.MarkdownProof(report))
			default:
				fmt.Print(proof.TextProof(report))
			}
			return nil
		},
	}
}
