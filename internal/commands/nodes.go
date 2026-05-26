package commands

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/canadian-ai/girl/internal/node"
	"github.com/canadian-ai/girl/internal/parsertsx"
	"github.com/urfave/cli/v2"
)

type nodeSummary struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	File     string `json:"file"`
	Children int    `json:"children"`
}

type refSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Usage  string `json:"usage"`
	Target string `json:"target,omitempty"`
	File   string `json:"file"`
}

func NodesCommand() *cli.Command {
	return &cli.Command{
		Name:      "nodes",
		Usage:     "Parse TS/TSX files and list semantic nodes",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: json (default), text", Value: "json"},
			&cli.StringFlag{Name: "kind", Usage: "Filter by node kind (component, hook, state, jsx, reference, ...)"},
		},
		Action: func(c *cli.Context) error {
			path := c.Args().First()
			if path == "" {
				path = "."
			}
			rows, err := collectNodes(path, node.NodeKind(stringFlag(c, "kind")))
			if err != nil {
				return err
			}
			if stringFlag(c, "output", "o") == "text" {
				for _, row := range rows {
					fmt.Printf("%s\t%s\t%s\t%s\tchildren=%d\n", row.Kind, row.ID, row.Name, row.File, row.Children)
				}
				return nil
			}
			return printJSONRows(rows)
		},
	}
}

func RefsCommand() *cli.Command {
	return &cli.Command{
		Name:      "refs",
		Usage:     "Parse TS/TSX files and list reference nodes",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: json (default), text", Value: "json"},
			&cli.StringFlag{Name: "symbol", Aliases: []string{"s"}, Usage: "Only show references with this symbol name"},
		},
		Action: func(c *cli.Context) error {
			path := c.Args().First()
			if path == "" {
				path = "."
			}
			rows, err := collectRefs(path, stringFlag(c, "symbol", "s"))
			if err != nil {
				return err
			}
			if stringFlag(c, "output", "o") == "text" {
				for _, row := range rows {
					fmt.Printf("%s\t%s\t%s\t%s\t%s\n", row.Usage, row.ID, row.Name, row.Target, row.File)
				}
				return nil
			}
			return printJSONRows(rows)
		},
	}
}

func collectNodes(path string, kind node.NodeKind) ([]nodeSummary, error) {
	var rows []nodeSummary
	err := walkTSX(path, func(file string) error {
		g, err := parsertsx.New().ParseFile(file)
		if err != nil {
			return err
		}
		for _, n := range g.AllNodes() {
			if kind != "" && n.Kind() != kind {
				continue
			}
			rows = append(rows, nodeSummary{
				ID:       string(n.ID()),
				Kind:     string(n.Kind()),
				Name:     n.Name(),
				File:     file,
				Children: len(g.ChildrenOf(n.ID())),
			})
		}
		return nil
	})
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].File == rows[j].File {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].File < rows[j].File
	})
	return rows, err
}

func collectRefs(path, symbol string) ([]refSummary, error) {
	var rows []refSummary
	err := walkTSX(path, func(file string) error {
		g, err := parsertsx.New().ParseFile(file)
		if err != nil {
			return err
		}
		for _, n := range g.AllNodesOfKind(node.KindReference) {
			ref := n.(*node.ReferenceNode)
			if symbol != "" && ref.Name() != symbol {
				continue
			}
			rows = append(rows, refSummary{
				ID:     string(ref.ID()),
				Name:   ref.Name(),
				Usage:  string(ref.Usage),
				Target: string(ref.Target),
				File:   file,
			})
		}
		return nil
	})
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].File == rows[j].File {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].File < rows[j].File
	})
	return rows, err
}

func walkTSX(path string, visit func(file string) error) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return visitTSXFile(path, visit)
	}
	return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return skipTSXDir(p)
		}
		return visitTSXFile(p, visit)
	})
}

func visitTSXFile(path string, visit func(file string) error) error {
	if !isTSXFile(path) {
		return nil
	}
	return visit(path)
}

func skipTSXDir(path string) error {
	if skippedTSXDirs[filepath.Base(path)] {
		return filepath.SkipDir
	}
	return nil
}

var skippedTSXDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"dist":         true,
	"build":        true,
	".next":        true,
	"coverage":     true,
}

func isTSXFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx"
}

func printJSONRows(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
