package main

import (
	"log"
	"os"

	"github.com/canadian-ai/girl/internal/commands"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:        "girl",
		Usage:       "Grammar-Informed Refactoring Language",
		Description: "GIRL analyzes code, detects refactoring opportunities, and generates structured GRP plans for AI coding agents.",
		Version:     "0.1.0",
		Commands: []*cli.Command{
			commands.AnalyzeCommand(),
			commands.NodesCommand(),
			commands.RefsCommand(),
			commands.PlanCommand(),
			commands.PackCommand(),
			commands.ValidateCommand(),
			commands.VerifyCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
