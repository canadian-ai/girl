package commands

import (
	"fmt"
	"os"

	"github.com/canadian-ai/girl/internal/diffstats"
	"github.com/urfave/cli/v2"
)

func parseDiffFromFlags(c *cli.Context) (*diffstats.DiffStats, error) {
	diffFile := c.String("diff-file")
	readStdin := c.Bool("stdin")

	var data []byte
	var err error

	switch {
	case diffFile != "":
		data, err = os.ReadFile(diffFile)
		if err != nil {
			return nil, fmt.Errorf("read diff file: %w", err)
		}
	case readStdin:
		stat, err := os.Stdin.Stat()
		if err != nil {
			return nil, fmt.Errorf("stat stdin: %w", err)
		}
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return nil, fmt.Errorf("stdin is a terminal; pipe a diff or use --diff-file")
		}
		data, err = os.ReadFile("/dev/stdin")
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
	default:
		return nil, fmt.Errorf("provide --diff-file or --stdin")
	}

	stats, err := diffstats.ParseDiffBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parse diff: %w", err)
	}

	return stats, nil
}
