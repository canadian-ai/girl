package goanalysis

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/canadian-ai/girl/internal/shared"
)

// stdExports maps standard-library import paths (including GOROOT-vendored
// ones such as "vendor/golang.org/x/net/...") to export data files.
//
// importer.Default resolves each standard package with its own
// `go list -export` subprocess. On a large repository that is hundreds of
// sequential process launches per run; resolving them in one batch keeps
// type information identical while removing almost all of the wait.
type stdExports struct {
	goroot  string
	exports map[string]string
}

// resolveStdExports lists export data for the standard packages that the
// type-checked files import, plus their dependencies, in a single call.
func resolveStdExports(imports []string) *stdExports {
	s := &stdExports{goroot: filepath.Clean(build.Default.GOROOT), exports: map[string]string{}}
	var pkgs []string
	for _, p := range imports {
		if p != "C" && p != "unsafe" && p != "" {
			pkgs = append(pkgs, p)
		}
	}
	if len(pkgs) == 0 || s.goroot == "" {
		return s
	}
	// Vendored standard dependencies are listed under their vendor/ path.
	cmd := exec.Command(filepath.Join(s.goroot, "bin", "go"), append([]string{"list", "-e", "-deps", "-export",
		"-f", "{{if and .Standard .Export}}{{.ImportPath}}\t{{.Export}}{{end}}"}, pkgs...)...)
	cmd.Dir = s.goroot
	cmd.Env = append(os.Environ(), "PWD="+cmd.Dir, "GOROOT="+s.goroot)
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return s
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		if path, file, ok := strings.Cut(scanner.Text(), "\t"); ok && file != "" {
			s.exports[path] = file
		}
	}
	return s
}

// newImporter returns a per-worker importer backed by the resolved exports.
func (s *stdExports) newImporter() types.Importer {
	inner := importer.ForCompiler(token.NewFileSet(), "gc", func(path string) (io.ReadCloser, error) {
		file, ok := s.exports[path]
		if !ok {
			return nil, fmt.Errorf("can't find import: %q", path)
		}
		return os.Open(file)
	}).(types.ImporterFrom)
	return &stdImporter{std: s, inner: inner}
}

type stdImporter struct {
	std   *stdExports
	inner types.ImporterFrom
}

func (i *stdImporter) Import(path string) (*types.Package, error) {
	return i.ImportFrom(path, "", 0)
}

// ImportFrom canonicalizes path the way go/build does for the default
// importer: files inside GOROOT/src see GOROOT's vendored packages.
func (i *stdImporter) ImportFrom(path, dir string, mode types.ImportMode) (*types.Package, error) {
	if _, ok := i.std.exports[path]; !ok {
		if vendored := i.std.vendorPath(path, dir); vendored != "" {
			if _, found := i.std.exports[vendored]; found {
				path = vendored
			}
		}
	}
	return i.inner.ImportFrom(path, dir, mode)
}

// vendorPath is the GOROOT-vendored import path for files in GOROOT: the
// "cmd" module vendors under src/cmd/vendor, the "std" module under
// src/vendor. Files elsewhere never see GOROOT's vendored packages.
func (s *stdExports) vendorPath(path, dir string) string {
	if dir == "" || s.goroot == "" {
		return ""
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	abs += string(filepath.Separator)
	src := filepath.Join(s.goroot, "src") + string(filepath.Separator)
	switch {
	case strings.HasPrefix(abs, src+"cmd"+string(filepath.Separator)):
		return "cmd/vendor/" + path
	case strings.HasPrefix(abs, src):
		return "vendor/" + path
	}
	return ""
}

// typeCheckImports parses files cheaply to find the ones that will be
// type-checked (see hasBlankCallAssign) and returns the union of their
// imports.
func typeCheckImports(paths []string) []string {
	probe := &stdExports{goroot: filepath.Clean(build.Default.GOROOT)}
	perFile := shared.ParallelMap(paths, func() struct{} { return struct{}{} }, func(_ struct{}, p string) ([]string, bool) {
		data, err := os.ReadFile(p)
		if err != nil || shared.IsGeneratedSource(data) {
			return nil, false
		}
		imports := importsIfTypeChecked(p, data)
		for _, imp := range imports {
			if v := probe.vendorPath(imp, filepath.Dir(p)); v != "" {
				imports = append(imports, v)
			}
		}
		return imports, true
	})
	seen := map[string]bool{}
	for _, imports := range perFile {
		for _, imp := range imports {
			seen[imp] = true
		}
	}
	out := make([]string, 0, len(seen))
	for imp := range seen {
		out = append(out, imp)
	}
	sort.Strings(out)
	return out
}

func importsIfTypeChecked(path string, data []byte) []string {
	f, err := parser.ParseFile(token.NewFileSet(), path, data, parser.SkipObjectResolution)
	if err != nil || !hasBlankCallAssign(f) {
		return nil
	}
	return importPaths(f)
}

func importPaths(f *ast.File) []string {
	out := make([]string, 0, len(f.Imports))
	for _, spec := range f.Imports {
		if p, err := strconv.Unquote(spec.Path.Value); err == nil {
			out = append(out, p)
		}
	}
	return out
}
