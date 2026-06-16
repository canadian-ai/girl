package commands

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

//go:embed install-files/opencode install-files/claude install-files/codex install-files/pi install-files/openrewrite install-files/rtk install-files/gritql install-files/rust-lsp
var installFS embed.FS

type frameworkTarget struct {
	EmbedDir string
	DestDir  string
	Files    []string
}

var frameworkTargets = map[string]frameworkTarget{
	"opencode": {
		EmbedDir: "install-files/opencode",
		DestDir:  ".opencode/agents",
		Files:    []string{"girl-planner.md", "girl-implementer.md", "girl-reviewer.md"},
	},
	"claude": {
		EmbedDir: "install-files/claude",
		DestDir:  ".claude",
		Files:    []string{"CLAUDE.md", "skills/girl/SKILL.md"},
	},
	"codex": {
		EmbedDir: "install-files/codex",
		DestDir:  ".codex",
		Files:    []string{"skills/girl/SKILL.md"},
	},
	"pi": {
		EmbedDir: "install-files/pi",
		DestDir:  ".pi/agent",
		Files:    []string{"skills/girl/SKILL.md"},
	},
	"openrewrite": {
		EmbedDir: "install-files/openrewrite",
		DestDir:  ".openrewrite",
		Files:    []string{"skills/SKILL.md"},
	},
	"rtk": {
		EmbedDir: "install-files/rtk",
		DestDir:  ".rtk",
		Files:    []string{"skills/SKILL.md"},
	},
	"gritql": {
		EmbedDir: "install-files/gritql",
		DestDir:  ".gritql",
		Files:    []string{"skills/SKILL.md"},
	},
	"rust-lsp": {
		EmbedDir: "install-files/rust-lsp",
		DestDir:  ".rust-lsp",
		Files:    []string{"skills/SKILL.md"},
	},
}

func InstallCommand() *cli.Command {
	return &cli.Command{
		Name:      "install",
		Usage:     "Install GIRL agents/skills for an AI coding framework",
		ArgsUsage: "[framework]",
		Description: `Install GIRL configuration files for your AI coding framework of choice.

Frameworks:
  opencode     Copy agents to .opencode/agents/ (girl-planner, girl-implementer, girl-reviewer)
  claude       Copy skill to .claude/ (skills/girl/SKILL.md + CLAUDE.md)
  codex        Copy skill to .codex/ (skills/girl/SKILL.md)
  pi           Copy skill to .pi/agent/ (skills/girl/SKILL.md)
  openrewrite  Copy skill to .openrewrite/ (skills/SKILL.md)
  rtk          Copy skill to .rtk/ (skills/SKILL.md)
  gritql       Copy skill to .gritql/ (skills/SKILL.md)
  rust-lsp     Copy skill to .rust-lsp/ (skills/SKILL.md)

If no framework is specified, prints available frameworks.`,
		Action: func(c *cli.Context) error {
			framework := c.Args().First()
			if framework == "" {
				fmt.Fprintln(os.Stderr, "Available frameworks: opencode, claude, codex, pi, openrewrite, rtk, gritql, rust-lsp")
				fmt.Fprintln(os.Stderr, "Usage: girl install <framework>")
				return nil
			}

			target, ok := frameworkTargets[framework]
			if !ok {
				return fmt.Errorf("unknown framework %q; supported: opencode, claude, codex, pi, openrewrite, rtk, gritql, rust-lsp", framework)
			}

			for _, f := range target.Files {
				embedPath := filepath.ToSlash(filepath.Join(target.EmbedDir, f))
				dst := filepath.Join(target.DestDir, f)

				data, err := installFS.ReadFile(embedPath)
				if err != nil {
					return fmt.Errorf("read embedded %s: %w", embedPath, err)
				}

				if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
					return fmt.Errorf("create directory %s: %w", filepath.Dir(dst), err)
				}

				if err := os.WriteFile(dst, data, 0644); err != nil {
					return fmt.Errorf("write destination %s: %w", dst, err)
				}

				fmt.Fprintf(os.Stderr, "Installed %s -> %s\n", f, dst)
			}

			fmt.Fprintf(os.Stderr, "GIRL %s integration installed. Use 'girl analyze/plan/pack/verify' in your project.\n", framework)
			return nil
		},
	}
}
