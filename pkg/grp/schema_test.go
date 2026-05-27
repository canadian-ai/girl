package grp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaFilesAreValidJSON(t *testing.T) {
	schemaFiles := []string{
		"grp-diagnostic.v0.1.schema.json",
		"grp-step.v0.1.schema.json",
		"grp-verification.v0.1.schema.json",
		"grp-plan.v0.1.schema.json",
	}

	for _, sf := range schemaFiles {
		t.Run(sf, func(t *testing.T) {
			path := filepath.Join("..", "..", "schemas", sf)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", path, err)
			}
			var v interface{}
			if err := json.Unmarshal(data, &v); err != nil {
				t.Fatalf("invalid JSON in %s: %v", sf, err)
			}
		})
	}
}

func TestPlanSchemaHasProperties(t *testing.T) {
	path := filepath.Join("..", "..", "schemas", "grp-plan.v0.1.schema.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read schema: %v", err)
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("schema missing 'properties' key or not an object")
	}

	required := []string{"specversion", "id", "type", "source", "subject", "language", "goal", "risk", "diagnostics", "steps", "verification"}
	for _, f := range required {
		if _, ok := props[f]; !ok {
			t.Errorf("plan schema missing required property %q in 'properties'", f)
		}
	}
}

func TestPlanRoundTrip(t *testing.T) {
	p := Plan{
		SpecVersion: "0.1",
		ID:          "grp_roundtrip",
		Type:        "dev.refactor.plan",
		Source:      "github.com/canadian-ai/girl",
		Subject:     ".",
		Language:    "go",
		Goal:        "round trip test",
		Risk:        SeverityLow,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal plan: %v", err)
	}

	expectedKeys := []string{"specversion", "id", "type", "source", "subject", "language", "goal", "risk", "diagnostics", "steps", "verification"}
	for _, k := range expectedKeys {
		if _, ok := result[k]; !ok {
			t.Errorf("round-tripped plan missing key %q", k)
		}
	}
}
