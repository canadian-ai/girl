package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/canadian-ai/girl/internal/analyzer"
	"github.com/canadian-ai/girl/internal/goanalysis"
	"github.com/canadian-ai/girl/internal/ir"
	"github.com/urfave/cli/v2"
)

type AnalyzerResult = ir.AnalyzerResult
type GrpPlan = ir.GrpPlan
type GrpStep = ir.GrpStep
type ContextPack = ir.ContextPack

func commandPath(c *cli.Context) string {
	path := c.Args().First()
	if path == "" {
		return "."
	}
	return path
}

func analyzePath(path, lang string) (*ir.AnalyzerResult, error) {
	if lang == "go" {
		return goanalysis.AnalyzePath(path, nil)
	}
	return analyzer.NewAnalyzer(nil).Analyze(path)
}

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

func resolveLang(path string, lang string) string {
	if lang != "auto" {
		return lang
	}
	info, err := os.Stat(path)
	if err != nil {
		return "ts"
	}
	if info.IsDir() {
		goModInfo, err := os.Stat(filepath.Join(path, "go.mod"))
		if err == nil && !goModInfo.IsDir() {
			return "go"
		}
		hasGo, hasTS := detectLangFiles(path)
		if hasGo && !hasTS {
			return "go"
		}
		return "ts"
	}
	if goanalysis.IsGoFile(path) {
		return "go"
	}
	return "ts"
}

func detectLangFiles(path string) (bool, bool) {
	hasGo := false
	hasTS := false
	err := filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		if goanalysis.IsGoFile(p) {
			hasGo = true
		}
		if isScriptFile(p) {
			hasTS = true
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: language auto-detection incomplete:", err)
	}
	return hasGo, hasTS
}

func isScriptFile(path string) bool {
	switch filepath.Ext(path) {
	case ".ts", ".tsx", ".js", ".jsx":
		return true
	default:
		return false
	}
}
