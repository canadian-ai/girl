package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/pkg/grp"
)

func TestContractJSONMatchesV1Bytes(t *testing.T) {
	values := map[string]any{
		"analysis-nil":   &ir.AnalyzerResult{},
		"analysis-empty": &ir.AnalyzerResult{Files: []*ir.FileIR{}, Diagnostics: []ir.Diagnostic{}},
		"grp":            &grp.Plan{},
		"legacy-plan":    &ir.GrpPlan{},
		"semantics": struct {
			NilSlice []string          `json:"nilSlice"`
			NilMap   map[string]string `json:"nilMap"`
			Zero     int               `json:"zero,omitempty"`
			False    bool              `json:"false,omitempty"`
			Text     string            `json:"text"`
			Number   uint64            `json:"number"`
			Metadata map[string]any    `json:"metadata"`
		}{Text: "<agent>&\u2028\u2029", Number: 9007199254740993, Metadata: map[string]any{"z": 2, "a": 1}},
	}
	for name, value := range values {
		t.Run(name, func(t *testing.T) {
			want, err := json.MarshalIndent(value, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got, err := marshalContractJSON(value)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("JSON bytes changed\ngot: %s\nwant: %s", got, want)
			}
			path := filepath.Join(t.TempDir(), "output.json")
			if err := writeJSONFile(path, value); err != nil {
				t.Fatal(err)
			}
			written, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(written, want) {
				t.Fatal("persisted JSON bytes changed")
			}
		})
	}
}

func TestPlanJSONMatchesV1Bytes(t *testing.T) {
	cases := []struct {
		name  string
		value any
		write func(string) error
	}{
		{"grp", &grp.Plan{}, func(path string) error { return writeGRPPlanFile(path, &grp.Plan{}) }},
		{"legacy", &ir.GrpPlan{}, func(path string) error { return writePlanFile(path, &ir.GrpPlan{}) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := t.TempDir()
			if err := tc.write(path); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(path, ".grp", "plan.json"))
			if err != nil {
				t.Fatal(err)
			}
			want, err := json.MarshalIndent(tc.value, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("plan bytes changed\ngot: %s\nwant: %s", got, want)
			}
		})
	}
}
