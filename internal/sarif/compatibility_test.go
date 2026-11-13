package sarif

import (
	"encoding/json"
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

func TestExportPreservesV1JSONContract(t *testing.T) {
	cases := map[string][]ir.Diagnostic{
		"nil":        nil,
		"empty":      {},
		"diagnostic": {{Code: "go.example", Message: "<agent>&\u2028", File: "example.go", Line: 1, Severity: ir.SeverityLow}},
	}
	for name, diags := range cases {
		t.Run(name, func(t *testing.T) {
			rules, seen, counts := buildRules(diags)
			log := buildLog("Girl", "test", rules, buildResults(diags, seen), counts)
			want, err := json.MarshalIndent(log, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got, err := ExportDiagnostics(diags, "Girl", "test")
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("SARIF bytes changed\ngot: %s\nwant: %s", got, want)
			}
		})
	}
}
