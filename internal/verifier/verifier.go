package verifier

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/canadian-ai/girl/internal/verification"
)

type CommandCheck = verification.Command
type VerifyResult = verification.Result

func NewVerifier() *Verifier {
	return &Verifier{}
}

type Verifier struct{}

func (v *Verifier) Verify(path string) (*VerifyResult, error) {
	return verification.Detect(path)
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
