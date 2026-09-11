package verification

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Detect(path string) (*Result, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", path, err)
	}
	if !info.IsDir() {
		path = filepath.Dir(path)
	}

	pm := detectPackageManager(path)
	result := &Result{
		WorkDir:         path,
		PackageManager:  pm,
		Commands:        []Command{},
		HasConfig:       pathExists(filepath.Join(path, "tsconfig.json")),
		HasConvex:       pathExists(filepath.Join(path, "convex")),
		HasDocker:       pathExists(filepath.Join(path, "Dockerfile")),
		HasCI:           pathExists(filepath.Join(path, ".github/workflows")),
		HasGolangCILint: pathExists(filepath.Join(path, ".golangci.yml")) || pathExists(filepath.Join(path, ".golangci.yaml")),
		HasMakefile:     pathExists(filepath.Join(path, "Makefile")),
	}

	result.Commands = append(result.Commands, detectPackageScripts(path, pm)...)
	result.Commands = append(result.Commands, detectGoCommands(path, pm)...)
	result.Commands = append(result.Commands, detectRustCommands(path, pm)...)
	result.Commands = append(result.Commands, detectOptionalCommands(path)...)
	return result, nil
}

func Commands(path string) []string {
	result, err := Detect(path)
	if err != nil {
		return nil
	}
	cmds := make([]string, 0, len(result.Commands))
	for _, cmd := range result.Commands {
		cmds = append(cmds, cmd.Command)
	}
	return cmds
}

func detectPackageManager(path string) string {
	lockfiles := []struct {
		name    string
		manager string
	}{
		{name: "bun.lock", manager: "bun"},
		{name: "bun.lockb", manager: "bun"},
		{name: "pnpm-lock.yaml", manager: "pnpm"},
		{name: "yarn.lock", manager: "yarn"},
		{name: "package-lock.json", manager: "npm"},
		{name: "go.mod", manager: "go"},
		{name: "Cargo.toml", manager: "cargo"},
	}
	for _, lockfile := range lockfiles {
		if pathExists(filepath.Join(path, lockfile.name)) {
			return lockfile.manager
		}
	}
	if pathExists(filepath.Join(path, "package.json")) {
		return "npm"
	}
	return "unknown"
}

type packageScriptSpec struct {
	Role     string
	Names    []string
	Required bool
	All      bool
}

func detectPackageScripts(path string, pm string) []Command {
	pkgPath := filepath.Join(path, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	runner := packageRunner(pm)
	specs := []packageScriptSpec{
		{Role: "typecheck", Names: []string{"typecheck", "type-check", "check:types", "types"}, Required: true},
		{Role: "lint", Names: []string{"lint"}},
		{Role: "test", Names: []string{"test"}},
		{Role: "build", Names: []string{"build:ci", "build"}, Required: true},
		{Role: "format", Names: []string{"format:check", "format"}},
		{Role: "architecture", Names: []string{"lint:copy-wrap", "provenance:check", "css:budget"}, All: true},
	}

	var cmds []Command
	for _, spec := range specs {
		for _, script := range spec.Names {
			if _, exists := pkg.Scripts[script]; !exists {
				continue
			}
			cmds = append(cmds, Command{
				Name:       script,
				Script:     script,
				Command:    fmt.Sprintf("%s %s", runner, script),
				Required:   spec.Required,
				Source:     "package.json",
				Confidence: "high",
				Type:       spec.Role,
				Exists:     true,
			})
			if !spec.All {
				break
			}
		}
	}
	return cmds
}

func packageRunner(pm string) string {
	switch pm {
	case "bun":
		return "bun run"
	case "pnpm":
		return "pnpm"
	case "yarn":
		return "yarn"
	default:
		return "npm run"
	}
}

func detectGoCommands(path string, pm string) []Command {
	if pm != "go" || !pathExists(filepath.Join(path, "go.mod")) {
		return nil
	}
	return []Command{
		{Name: "Go build", Command: "go build ./...", Required: true, Source: "go.mod", Confidence: "high", Type: "build", Exists: true},
		{Name: "Go vet", Command: "go vet ./...", Required: true, Source: "go.mod", Confidence: "high", Type: "lint", Exists: true},
		{Name: "Go test", Command: "go test ./...", Required: true, Source: "go.mod", Confidence: "high", Type: "test", Exists: true},
	}
}

func detectRustCommands(path string, pm string) []Command {
	if pm != "cargo" || !pathExists(filepath.Join(path, "Cargo.toml")) {
		return nil
	}
	return []Command{
		{Name: "Cargo build", Command: "cargo build", Required: true, Source: "Cargo.toml", Confidence: "high", Type: "build", Exists: true},
		{Name: "Cargo clippy", Command: "cargo clippy", Required: false, Source: "Cargo.toml", Confidence: "high", Type: "lint", Exists: true},
		{Name: "Cargo test", Command: "cargo test", Required: true, Source: "Cargo.toml", Confidence: "high", Type: "test", Exists: true},
	}
}

func detectOptionalCommands(path string) []Command {
	var cmds []Command
	if data, err := os.ReadFile(filepath.Join(path, "Makefile")); err == nil {
		if strings.Contains(string(data), "test:") {
			cmds = append(cmds, Command{Name: "make test", Command: "make test", Required: false, Source: "Makefile", Confidence: "high", Type: "test", Exists: true})
		}
	}
	if pathExists(filepath.Join(path, ".golangci.yml")) || pathExists(filepath.Join(path, ".golangci.yaml")) {
		cmds = append(cmds, Command{Name: "golangci-lint", Command: "golangci-lint run", Required: false, Source: "config-file", Confidence: "high", Type: "lint", Exists: true})
	}
	return cmds
}

func pathExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info != nil
}
