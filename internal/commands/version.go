package commands

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
)

var (
	Version    = "0.1.2"
	Commit     = "unknown"
	BuiltAt    = "unknown"
	GrpVersion = "0.1"
)

type versionInfo struct {
	Girl    string `json:"girl"`
	Grp     string `json:"grp"`
	Commit  string `json:"commit"`
	BuiltAt string `json:"builtAt"`
}

func VersionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print version information",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: text (default), json",
				Value:   "text",
			},
		},
		Action: func(c *cli.Context) error {
			info := versionInfo{
				Girl:    Version,
				Grp:     GrpVersion,
				Commit:  Commit,
				BuiltAt: BuiltAt,
			}

			switch c.String("output") {
			case "json":
				data, err := json.MarshalIndent(info, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal version: %w", err)
				}
				fmt.Println(string(data))
			default:
				fmt.Printf("girl  %s\n", info.Girl)
				fmt.Printf("grp   %s\n", info.Grp)
				fmt.Printf("commit %s\n", info.Commit)
				fmt.Printf("built  %s\n", info.BuiltAt)
			}

			return nil
		},
	}
}
