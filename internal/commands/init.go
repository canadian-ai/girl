package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

func InitCommand() *cli.Command {
	return &cli.Command{
		Name:      "init",
		Usage:     "Initialize GIRL for a project",
		ArgsUsage: "[path]",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "force", Usage: "Replace an existing .girl/config.yaml"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Print the generated configuration without writing it"},
		},
		Action: func(c *cli.Context) error {
			path := commandPath(c)
			cfg, err := defaultGirlProjectConfig(path)
			if err != nil {
				return fmt.Errorf("detect project: %w", err)
			}
			content := renderGirlProjectConfig(cfg)
			configPath := filepath.Join(path, ".girl", "config.yaml")

			if c.Bool("dry-run") {
				fmt.Print(content)
				return nil
			}
			if _, err := os.Stat(configPath); err == nil && !c.Bool("force") {
				return fmt.Errorf("%s already exists; use --force to replace it", configPath)
			}
			if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
				return fmt.Errorf("create .girl directory: %w", err)
			}
			if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			fmt.Printf("Initialized GIRL at %s\n", configPath)
			if len(cfg.Profiles) > 0 {
				fmt.Printf("Profiles: %v\n", cfg.Profiles)
			}
			fmt.Println("Next: run 'girl check'.")
			return nil
		},
	}
}
