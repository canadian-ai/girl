package commands

import (
	"encoding/json"
	"fmt"

	"github.com/canadian-ai/girl/internal/proof"
	"github.com/urfave/cli/v2"
)

func BenchmarkCommand() *cli.Command {
	return &cli.Command{
		Name:      "benchmark",
		Usage:     "Summarize GIRL findings across a repo",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "lang", Usage: "Language mode: auto (default), go, ts, rust", Value: "auto"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text (default), json, markdown", Value: "text"},
			&cli.IntFlag{Name: "top-files", Usage: "Number of worst files to show", Value: 10},
		},
		Action: func(c *cli.Context) error {
			path := commandPath(c)
			lang := resolveLang(path, c.String("lang"))
			result, err := analyzePath(path, lang)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}
			report := proof.NewBenchmarkReport(path, result, c.Int("top-files"))
			switch stringFlag(c, "output", "o") {
			case "json":
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
			case "markdown":
				fmt.Print(proof.MarkdownBenchmark(report))
			default:
				fmt.Print(proof.TextBenchmark(report))
			}
			return nil
		},
	}
}
