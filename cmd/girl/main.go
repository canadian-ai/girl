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
		Description: "GIRL is a project quality contract for source-grounded analysis, refactoring, verification, and AI coding agents.",
		Version:     "0.1.26",
		Commands: []*cli.Command{
			commands.InitCommand(),
			commands.CheckCommand(),
			commands.AgentCommand(),
			commands.AnalyzeCommand(),
			commands.ComplexityCommand(),
			commands.BenchmarkCommand(),
			commands.ProveCommand(),
			commands.NodesCommand(),
			commands.RefsCommand(),
			commands.PlanCommand(),
			commands.PackCommand(),
			commands.InstallCommand(),
			commands.ValidateCommand(),
			commands.ReviewCommand(),
			commands.DecomposeCommand(),
			commands.VerifyCommand(),
			commands.ReceiptCommand(),
			commands.ProveAppCommand(),
			commands.PreflightCommand(),
			commands.WorkOrderCommand(),
			commands.LaunchKitCommand(),
			commands.VersionCommand(),
			commands.UpdateCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
