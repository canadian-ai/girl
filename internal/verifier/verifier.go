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
	Name     string `json:"name"`
	Script   string `json:"script"`
	Command  string `json:"command"`
	Exists   bool   `json:"exists"`
}

type VerifyResult struct {
	WorkDir        string         `json:"workDir"`
	PackageManager string         `json:"packageManager"`
	Commands       []CommandCheck `json:"commands"`
	HasConfig      bool           `json:"hasConfig"`
	HasConvex      bool           `json:"hasConvex"`
	HasDocker      bool           `json:"hasDocker"`
	HasCI          bool           `json:"hasCI"`
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

	if _, err := os.Stat(filepath.Join(path, "tsconfig.json")); err == nil {
		result.HasConfig = true
	}
	if _, err := os.Stat(filepath.Join(path, "convex")); err == nil {
		result.HasConvex = true
	}
	if _, err := os.Stat(filepath.Join(path, "Dockerfile")); err == nil {
		result.HasDocker = true
	}
	if _, err := os.Stat(filepath.Join(path, ".github/workflows")); err == nil {
		result.HasCI = true
	}

	return result, nil
}

func (v *Verifier) detectPackageManager(path string) string {
	if _, err := os.Stat(filepath.Join(path, "bun.lock")); err == nil {
		return "bun"
	}
	if _, err := os.Stat(filepath.Join(path, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(path, "yarn.lock")); err == nil {
		return "yarn"
	}
	if _, err := os.Stat(filepath.Join(path, "package-lock.json")); err == nil {
		return "npm"
	}
	if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
		return "go"
	}
	return "unknown"
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
	switch pm {
	case "bun":
		runner = "bun run"
	case "pnpm":
		runner = "pnpm"
	}

	wantedScripts := []string{"lint", "typecheck", "test", "build"}
	var checks []CommandCheck

	for _, name := range wantedScripts {
		if _, exists := pkg.Scripts[name]; exists {
			checks = append(checks, CommandCheck{
				Name:    name,
				Script:  name,
				Command: fmt.Sprintf("%s %s", runner, name),
				Exists:  true,
			})
		}
	}

	return checks
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
