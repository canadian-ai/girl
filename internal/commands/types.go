package commands

import (
	"os"
	"strings"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/urfave/cli/v2"
)

// Re-export package types at command level for convenience
type AnalyzerResult = ir.AnalyzerResult
type GrpPlan = ir.GrpPlan
type GrpStep = ir.GrpStep
type ContextPack = ir.ContextPack

func stringFlag(c *cli.Context, name string, aliases ...string) string {
	if c.IsSet(name) {
		return c.String(name)
	}
	keys := append([]string{name}, aliases...)
	args := os.Args
	for i, arg := range args {
		for _, key := range keys {
			long := "--" + key
			short := "-" + key
			if strings.HasPrefix(arg, long+"=") {
				return strings.TrimPrefix(arg, long+"=")
			}
			if (arg == long || arg == short) && i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return c.String(name)
}
