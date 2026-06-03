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
	wantedScripts := []string{"typecheck", "lint", "test", "build", "format"}
	cmds := []Command{}
	for _, name := range wantedScripts {
		if _, exists := pkg.Scripts[name]; !exists {
			continue
		}
		cmds = append(cmds, Command{
			Name:       name,
			Script:     name,
			Command:    fmt.Sprintf("%s %s", runner, name),
			Required:   name == "build" || name == "typecheck",
			Source:     "package.json",
			Confidence: "high",
			Type:       "script",
			Exists:     true,
		})
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
