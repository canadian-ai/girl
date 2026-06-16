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
		Description: "GIRL analyzes code, detects refactoring opportunities, and generates structured GRP plans for AI coding agents. Ships with agents/skills for OpenCode, Claude Code, Codex, Pi, OpenRewrite, RTK, GritQL, and Rust-LSP.",
		Version:     "0.1.7",
		Commands: []*cli.Command{
			commands.AnalyzeCommand(),
			commands.NodesCommand(),
			commands.RefsCommand(),
			commands.PlanCommand(),
			commands.PackCommand(),
			commands.InstallCommand(),
			commands.ValidateCommand(),
			commands.ReviewCommand(),
			commands.DecomposeCommand(),
			commands.VerifyCommand(),
			commands.VersionCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
