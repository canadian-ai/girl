package verifier

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CommandCheck struct {
	Name       string `json:"name"`
	Script     string `json:"script"`
	Command    string `json:"command"`
	Required   bool   `json:"required"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
	Exists     bool   `json:"exists"`
}

type VerifyResult struct {
	WorkDir        string         `json:"workDir"`
	PackageManager string         `json:"packageManager"`
	Commands       []CommandCheck `json:"commands"`
	HasConfig      bool           `json:"hasConfig"`
	HasConvex      bool           `json:"hasConvex"`
	HasDocker      bool           `json:"hasDocker"`
	HasCI          bool           `json:"hasCI"`
	HasGolangCILint bool          `json:"hasGolangCILint,omitempty"`
	HasMakefile    bool           `json:"hasMakefile,omitempty"`
}

func NewVerifier() *Verifier {
	return &Verifier{}
}

type Verifier struct{}

func (v *Verifier) Verify(path string) (*VerifyResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", path, err)
	}

	if !info.IsDir() {
		path = filepath.Dir(path)
	}

	pm := v.detectPackageManager(path)
	scripts := v.detectScripts(path)

	result := &VerifyResult{
		WorkDir:        path,
		PackageManager: pm,
		Commands:       scripts,
	}

	if pathExists(filepath.Join(path, "tsconfig.json")) {
		result.HasConfig = true
	}
	if pathExists(filepath.Join(path, "convex")) {
		result.HasConvex = true
	}
	if pathExists(filepath.Join(path, "Dockerfile")) {
		result.HasDocker = true
	}
	if pathExists(filepath.Join(path, ".github/workflows")) {
		result.HasCI = true
	}
	if pathExists(filepath.Join(path, ".golangci.yml")) || pathExists(filepath.Join(path, ".golangci.yaml")) {
		result.HasGolangCILint = true
	}
	result.HasMakefile = pathExists(filepath.Join(path, "Makefile"))

	goCmds := v.detectGoCommands(path, pm)
	result.Commands = append(result.Commands, goCmds...)

	optionalCmds := v.detectOptionalCommands(path)
	result.Commands = append(result.Commands, optionalCmds...)

	return result, nil
}

func (v *Verifier) detectPackageManager(path string) string {
	lockfiles := []struct {
		name    string
		manager string
	}{
		{name: "bun.lock", manager: "bun"},
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
	return "unknown"
}

func pathExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info != nil
}

func (v *Verifier) confidenceFor(pm string) string {
	switch pm {
	case "bun", "pnpm", "yarn", "npm", "go":
		return "high"
	default:
		return "medium"
	}
}

func (v *Verifier) detectScripts(path string) []CommandCheck {
	pkgPath := filepath.Join(path, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return []CommandCheck{}
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return []CommandCheck{}
	}

	pm := v.detectPackageManager(path)
	runner := "npm run"
	source := "binding-default"
	confidence := v.confidenceFor(pm)
	switch pm {
	case "bun":
		runner = "bun run"
		source = "lockfile"
		confidence = "high"
	case "pnpm":
		runner = "pnpm"
		source = "lockfile"
		confidence = "high"
	case "yarn":
		runner = "yarn"
		source = "lockfile"
		confidence = "high"
	}
	if pm == "npm" {
		source = "lockfile"
		confidence = "high"
	}

	wantedScripts := []string{"typecheck", "lint", "test", "build", "format"}
	var checks []CommandCheck

	for _, name := range wantedScripts {
		if _, exists := pkg.Scripts[name]; exists {
			required := name == "build" || name == "typecheck"
			checks = append(checks, CommandCheck{
				Name:       name,
				Script:     name,
				Command:    fmt.Sprintf("%s %s", runner, name),
				Required:   required,
				Source:     source,
				Confidence: confidence,
				Exists:     true,
			})
		}
	}

	return checks
}

func (v *Verifier) detectGoCommands(path string, pm string) []CommandCheck {
	if pm != "go" {
		return nil
	}
	return []CommandCheck{
		{Name: "Go build", Command: "go build ./...", Required: true, Source: "binding-default", Confidence: "high", Exists: true},
		{Name: "Go vet", Command: "go vet ./...", Required: true, Source: "binding-default", Confidence: "high", Exists: true},
		{Name: "Go test", Command: "go test ./...", Required: true, Source: "binding-default", Confidence: "high", Exists: true},
	}
}

func (v *Verifier) detectOptionalCommands(path string) []CommandCheck {
	var cmds []CommandCheck
	makefile := filepath.Join(path, "Makefile")
	if data, err := os.ReadFile(makefile); err == nil {
		if strings.Contains(string(data), "test:") {
			cmds = append(cmds, CommandCheck{
				Name: "make test", Command: "make test",
				Required: false, Source: "Makefile", Confidence: "medium", Exists: true,
			})
		}
	}
	if pathExists(filepath.Join(path, ".golangci.yml")) || pathExists(filepath.Join(path, ".golangci.yaml")) {
		cmds = append(cmds, CommandCheck{
			Name: "golangci-lint", Command: "golangci-lint run",
			Required: false, Source: "config-file", Confidence: "high", Exists: true,
		})
	}
	return cmds
}

func (v *Verifier) RunCommand(cmdStr string, workDir string) (string, error) {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	var cmd *exec.Cmd
	if len(parts) == 1 {
		cmd = exec.Command(parts[0])
	} else {
		cmd = exec.Command(parts[0], parts[1:]...)
	}

	if workDir != "" {
		cmd.Dir = workDir
	}

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (v *Verifier) ToJSON(result *VerifyResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
