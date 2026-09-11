package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

const (
	girlManagedStart = "<!-- GIRL:BEGIN -->"
	girlManagedEnd   = "<!-- GIRL:END -->"
)

func AgentCommand() *cli.Command {
	return &cli.Command{
		Name:  "agent",
		Usage: "Manage project-local GIRL agent integration",
		Subcommands: []*cli.Command{
			{
				Name:  "sync",
				Usage: "Sync canonical GIRL skills without replacing project instructions",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{Name: "framework", Aliases: []string{"f"}, Usage: "Framework to sync (opencode, codex, claude)"},
					&cli.BoolFlag{Name: "dry-run", Usage: "Show what would change without writing files"},
				},
				Action: syncAgentAction,
			},
		},
	}
}

func syncAgentAction(c *cli.Context) error {
	path := commandPath(c)
	cfg, err := loadGirlProjectConfig(path)
	if err != nil {
		return err
	}
	frameworks := c.StringSlice("framework")
	if len(frameworks) == 0 {
		for _, name := range []string{"opencode", "codex", "claude"} {
			if cfg.Agents[name] {
				frameworks = append(frameworks, name)
			}
		}
	}
	if len(frameworks) == 0 {
		frameworks = []string{"opencode", "codex"}
	}

	dryRun := c.Bool("dry-run")
	for _, framework := range frameworks {
		target, ok := frameworkTargets[framework]
		if !ok || (framework != "opencode" && framework != "codex" && framework != "claude") {
			return fmt.Errorf("unsupported agent framework %q", framework)
		}
		for _, file := range target.Files {
			if strings.EqualFold(filepath.Base(file), "CLAUDE.md") {
				continue
			}
			embedPath := filepath.ToSlash(filepath.Join(target.EmbedDir, file))
			data, err := installFS.ReadFile(embedPath)
			if err != nil {
				return err
			}
			dest := filepath.Join(path, target.DestDir, file)
			changed, err := syncOwnedFile(dest, data, dryRun)
			if err != nil {
				return err
			}
			if changed {
				verb := "Synced"
				if dryRun {
					verb = "Would sync"
				}
				fmt.Printf("%s %s\n", verb, dest)
			}
		}
	}

	agentsPath := filepath.Join(path, "AGENTS.md")
	block := girlAgentManagedBlock()
	changed, err := syncManagedBlock(agentsPath, block, dryRun)
	if err != nil {
		return err
	}
	if changed {
		verb := "Updated"
		if dryRun {
			verb = "Would update"
		}
		fmt.Printf("%s %s managed GIRL block\n", verb, agentsPath)
	}
	return nil
}

func girlAgentManagedBlock() string {
	return girlManagedStart + `
## GIRL

Use GIRL as the project quality contract:

1. Run ` + "`girl check`" + ` before handing off or finishing a change.
2. Prefer ` + "`girl check --changed`" + ` while iterating and ` + "`girl check --ci`" + ` in CI.
3. Use ` + "`girl plan`" + ` / ` + "`girl pack`" + ` when a structured refactor plan or agent context pack is needed.
4. Do not expose secrets or bypass repository auth, tenancy, storage, or deployment boundaries.
5. Report files changed, verification commands, failures, and remaining risks.
` + girlManagedEnd
}

func syncOwnedFile(path string, content []byte, dryRun bool) (bool, error) {
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, content) {
		return false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if dryRun {
		return true, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, content, 0644)
}

func syncManagedBlock(path, block string, dryRun bool) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	existing := string(data)
	updated := existing
	start := strings.Index(updated, girlManagedStart)
	end := strings.Index(updated, girlManagedEnd)
	if start >= 0 && end >= start {
		end += len(girlManagedEnd)
		updated = updated[:start] + block + updated[end:]
	} else if strings.TrimSpace(updated) == "" {
		updated = "# Repository Instructions\n\n" + block + "\n"
	} else {
		updated = strings.TrimRight(updated, "\n") + "\n\n" + block + "\n"
	}
	if updated == existing {
		return false, nil
	}
	if dryRun {
		return true, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && filepath.Dir(path) != "." {
		return false, err
	}
	return true, os.WriteFile(path, []byte(updated), 0644)
}
