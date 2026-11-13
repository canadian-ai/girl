// jsonv2-migrate conservatively migrates direct encoding/json calls.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/canadian-ai/girl/internal/jsonmigration"
)

func main() {
	write := flag.Bool("write", false, "write migrations (default: report only)")
	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}
	failed := false
	seen := make(map[string]bool)
	for _, root := range paths {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor" || d.Name() == "node_modules") {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(path, ".go") {
				return nil
			}
			absolute, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if seen[absolute] {
				return nil
			}
			seen[absolute] = true
			display := reportPath(path)
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			result, err := jsonmigration.Migrate(display, source)
			if err != nil {
				return fmt.Errorf("%s: %w", display, err)
			}
			if result.Review != "" {
				fmt.Printf("REVIEW %s: %s\n", display, result.Review)
				return nil
			}
			if !result.Changed {
				return nil
			}
			if !*write {
				fmt.Printf("WOULD MIGRATE %s\n", display)
				return nil
			}
			stat, err := os.Stat(path)
			if err != nil {
				return err
			}
			if err := os.WriteFile(path, result.Source, stat.Mode().Perm()); err != nil {
				return err
			}
			fmt.Printf("MIGRATED %s\n", display)
			return nil
		})
		if err != nil {
			var pathErr *os.PathError
			if errors.As(err, &pathErr) {
				fmt.Fprintf(os.Stderr, "%s: %v\n", reportPath(pathErr.Path), pathErr.Err)
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

// Reports stay relative even when input paths are absolute.
func reportPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Base(path)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return filepath.Base(path)
	}
	relative, err := filepath.Rel(cwd, absolute)
	if err != nil {
		return filepath.Base(path)
	}
	return relative
}
