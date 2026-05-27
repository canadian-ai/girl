package grp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMarshalPlan(t *testing.T) {
	p := &Plan{
		SpecVersion: "0.1",
		ID:          "grp_test",
		Type:        "dev.refactor.plan",
		Source:      "github.com/canadian-ai/girl",
		Subject:     ".",
		Language:    "go",
		Goal:        "test",
		Risk:        SeverityLow,
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	js := string(data)
	if !strings.Contains(js, `"specversion"`) {
		t.Errorf("JSON should contain \"specversion\", got: %s", js)
	}
	if strings.Contains(js, `"specVersion"`) {
		t.Errorf("JSON should not contain \"specVersion\", got: %s", js)
	}
	if strings.Contains(js, `"SpecVersion"`) {
		t.Errorf("JSON should not contain \"SpecVersion\", got: %s", js)
	}
}

func TestUnmarshalDiagnostic(t *testing.T) {
	js := `{
		"id": "diag_001",
		"code": "go.high-complexity",
		"severity": "high",
		"confidence": "high",
		"message": "Function handleRequest has cyclomatic complexity 22",
		"file": "internal/server/handler.go",
		"line": 42,
		"endLine": 89,
		"tags": ["complexity", "refactor"]
	}`

	var d Diagnostic
	if err := json.Unmarshal([]byte(js), &d); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if d.ID != "diag_001" {
		t.Errorf("ID = %q, want %q", d.ID, "diag_001")
	}
	if d.Code != "go.high-complexity" {
		t.Errorf("Code = %q, want %q", d.Code, "go.high-complexity")
	}
	if d.Severity != SeverityHigh {
		t.Errorf("Severity = %q, want %q", d.Severity, SeverityHigh)
	}
	if d.Confidence != ConfidenceHigh {
		t.Errorf("Confidence = %q, want %q", d.Confidence, ConfidenceHigh)
	}
	if d.Message != "Function handleRequest has cyclomatic complexity 22" {
		t.Errorf("Message = %q, want %q", d.Message, "Function handleRequest has cyclomatic complexity 22")
	}
	if d.File != "internal/server/handler.go" {
		t.Errorf("File = %q, want %q", d.File, "internal/server/handler.go")
	}
	if d.Line != 42 {
		t.Errorf("Line = %d, want %d", d.Line, 42)
	}
	if d.EndLine != 89 {
		t.Errorf("EndLine = %d, want %d", d.EndLine, 89)
	}
	if len(d.Tags) != 2 || d.Tags[0] != "complexity" || d.Tags[1] != "refactor" {
		t.Errorf("Tags = %v, want [complexity refactor]", d.Tags)
	}
}

func TestExtensionsRoundTrip(t *testing.T) {
	p := &Plan{
		SpecVersion: "0.1",
		ID:          "grp_ext_test",
		Type:        "dev.refactor.plan",
		Source:      "github.com/canadian-ai/girl",
		Subject:     ".",
		Language:    "go",
		Goal:        "ext test",
		Risk:        SeverityMedium,
		Extensions: map[string]interface{}{
			"go": map[string]interface{}{
				"analyzer": "go/parser",
			},
			"threshold": 10,
		},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var p2 Plan
	if err := json.Unmarshal(data, &p2); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if p2.ID != p.ID {
		t.Errorf("ID = %q, want %q", p2.ID, p.ID)
	}
	if p2.Goal != p.Goal {
		t.Errorf("Goal = %q, want %q", p2.Goal, p.Goal)
	}

	ext := p2.Extensions
	if ext == nil {
		t.Fatal("Extensions is nil after round trip")
	}
	goExt, ok := ext["go"]
	if !ok {
		t.Fatal("extensions[\"go\"] missing after round trip")
	}
	goMap, ok := goExt.(map[string]interface{})
	if !ok {
		t.Fatalf("extensions[\"go\"] type = %T, want map[string]interface{}", goExt)
	}
	if goMap["analyzer"] != "go/parser" {
		t.Errorf("extensions.go.analyzer = %v, want %q", goMap["analyzer"], "go/parser")
	}
	if ext["threshold"] != 10.0 {
		t.Errorf("extensions.threshold = %v, want %v", ext["threshold"], 10.0)
	}
}

func TestMarshalStep(t *testing.T) {
	s := Step{
		ID:     "step_001",
		Title:  "Test step",
		Action: "do something",
		Target: Target{
			File: "test.go",
		},
		Risk: SeverityLow,
		Verify: []Verification{
			{Command: "go test", Required: true, Source: "go", Confidence: "high"},
		},
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	js := string(data)
	if !strings.Contains(js, `"id"`) {
		t.Errorf("JSON should contain \"id\", got: %s", js)
	}
	if !strings.Contains(js, `"target"`) {
		t.Errorf("JSON should contain \"target\", got: %s", js)
	}
	if strings.Contains(js, `"ID"`) {
		t.Errorf("JSON should not contain \"ID\", got: %s", js)
	}
	if strings.Contains(js, `"Target"`) {
		t.Errorf("JSON should not contain \"Target\", got: %s", js)
	}
}

func TestMarshalSpan(t *testing.T) {
	s := Span{
		StartLine:   1,
		StartColumn: 5,
		EndLine:     10,
		EndColumn:   15,
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	js := string(data)
	if !strings.Contains(js, `"startLine"`) {
		t.Errorf("JSON should contain \"startLine\", got: %s", js)
	}
	if !strings.Contains(js, `"endLine"`) {
		t.Errorf("JSON should contain \"endLine\", got: %s", js)
	}
	if strings.Contains(js, `"StartLine"`) {
		t.Errorf("JSON should not contain \"StartLine\", got: %s", js)
	}
	if strings.Contains(js, `"StartColumn"`) {
		t.Errorf("JSON should not contain \"StartColumn\", got: %s", js)
	}
}
