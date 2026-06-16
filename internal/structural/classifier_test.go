package structural

import (
	"math"
	"testing"

	"github.com/canadian-ai/girl/internal/diffstats"
)

func TestClassify_PureLogic(t *testing.T) {
	diff := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "internal/server/handler.go", AddedLines: 100},
		},
	}
	c := Classify(diff)

	if c.Added.Logic != 90 {
		t.Errorf("Logic = %d, want 90", c.Added.Logic)
	}
	if c.Added.Test != 0 {
		t.Errorf("Test = %d, want 0", c.Added.Test)
	}
	if c.Added.EphemeralSupport != 10 {
		t.Errorf("EphemeralSupport = %d, want 10", c.Added.EphemeralSupport)
	}
	if c.Added.ReusableSupport != 0 {
		t.Errorf("ReusableSupport = %d, want 0", c.Added.ReusableSupport)
	}
	if c.Added.ConfigData != 0 {
		t.Errorf("ConfigData = %d, want 0", c.Added.ConfigData)
	}
	if c.Added.ConfigStructural != 0 {
		t.Errorf("ConfigStructural = %d, want 0", c.Added.ConfigStructural)
	}
	if c.Added.Generated != 0 {
		t.Errorf("Generated = %d, want 0", c.Added.Generated)
	}

	if c.Ratios.StructuralOverhead != 0.1 {
		t.Errorf("StructuralOverhead = %f, want 0.1", c.Ratios.StructuralOverhead)
	}
	if c.Ratios.TestToLogic != 0 {
		t.Errorf("TestToLogic = %f, want 0", c.Ratios.TestToLogic)
	}
	if c.Ratios.ProductiveScaffold != 0 {
		t.Errorf("ProductiveScaffold = %f, want 0", c.Ratios.ProductiveScaffold)
	}

	if c.Cohesion.Variance != 0 {
		t.Errorf("Cohesion.Variance = %f, want 0", c.Cohesion.Variance)
	}
	if len(c.Cohesion.SuggestedClusters) != 0 {
		t.Errorf("expected no suggested clusters, got %d", len(c.Cohesion.SuggestedClusters))
	}
}

func TestClassify_MixedTestAndLogic(t *testing.T) {
	diff := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "internal/server/handler.go", AddedLines: 100},
			{Path: "internal/server/handler_test.go", AddedLines: 50},
		},
	}
	c := Classify(diff)

	if c.Added.Logic != 90 {
		t.Errorf("Logic = %d, want 90", c.Added.Logic)
	}
	if c.Added.Test != 45 {
		t.Errorf("Test = %d, want 45", c.Added.Test)
	}
	if c.Added.EphemeralSupport != 15 {
		t.Errorf("EphemeralSupport = %d, want 15", c.Added.EphemeralSupport)
	}

	expectedSO := 15.0 / 105.0
	if math.Abs(c.Ratios.StructuralOverhead-expectedSO) > 1e-9 {
		t.Errorf("StructuralOverhead = %f, want %f", c.Ratios.StructuralOverhead, expectedSO)
	}
	if c.Ratios.TestToLogic != 0.5 {
		t.Errorf("TestToLogic = %f, want 0.5", c.Ratios.TestToLogic)
	}

	if c.Cohesion.Variance != 0.4 {
		t.Errorf("Cohesion.Variance = %f, want 0.4", c.Cohesion.Variance)
	}
	if len(c.Cohesion.SuggestedClusters) != 0 {
		t.Errorf("expected no suggested clusters, got %d", len(c.Cohesion.SuggestedClusters))
	}
}

func TestClassify_MultiLayer(t *testing.T) {
	diff := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "internal/server/handler.go", AddedLines: 50},
			{Path: "internal/server/middleware.go", AddedLines: 30},
			{Path: "frontend/pages/home.tsx", AddedLines: 20},
		},
	}
	c := Classify(diff)

	if c.Added.Logic != 91 {
		t.Errorf("Logic = %d, want 91", c.Added.Logic)
	}
	if c.Added.EphemeralSupport != 9 {
		t.Errorf("EphemeralSupport = %d, want 9", c.Added.EphemeralSupport)
	}

	if math.Abs(c.Cohesion.Variance-0.8) > 1e-9 {
		t.Errorf("Cohesion.Variance = %f, want 0.8", c.Cohesion.Variance)
	}

	if len(c.Cohesion.SuggestedClusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(c.Cohesion.SuggestedClusters))
	}
	cluster := c.Cohesion.SuggestedClusters[0]
	if len(cluster) != 2 {
		t.Fatalf("expected cluster of size 2, got %d", len(cluster))
	}
	paths := make(map[string]bool)
	for _, p := range cluster {
		paths[p] = true
	}
	if !paths["internal/server/handler.go"] {
		t.Error("expected internal/server/handler.go in cluster")
	}
	if !paths["internal/server/middleware.go"] {
		t.Error("expected internal/server/middleware.go in cluster")
	}
}

func TestClassify_ConfigHeavy(t *testing.T) {
	diff := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "tsconfig.json", AddedLines: 10},
			{Path: ".eslintrc.json", AddedLines: 5},
			{Path: "styles.css", AddedLines: 20},
		},
	}
	c := Classify(diff)

	if c.Added.ConfigStructural != 15 {
		t.Errorf("ConfigStructural = %d, want 15", c.Added.ConfigStructural)
	}
	if c.Added.ConfigData != 20 {
		t.Errorf("ConfigData = %d, want 20", c.Added.ConfigData)
	}
	if c.Added.Logic != 0 {
		t.Errorf("Logic = %d, want 0", c.Added.Logic)
	}
	if c.Added.Test != 0 {
		t.Errorf("Test = %d, want 0", c.Added.Test)
	}
	if c.Added.EphemeralSupport != 0 {
		t.Errorf("EphemeralSupport = %d, want 0", c.Added.EphemeralSupport)
	}
	if c.Added.Generated != 0 {
		t.Errorf("Generated = %d, want 0", c.Added.Generated)
	}

	if c.Ratios.StructuralOverhead != 1.0 {
		t.Errorf("StructuralOverhead = %f, want 1.0", c.Ratios.StructuralOverhead)
	}
	if c.Ratios.TestToLogic != 0 {
		t.Errorf("TestToLogic = %f, want 0", c.Ratios.TestToLogic)
	}
}

func TestClassify_GeneratedFiles(t *testing.T) {
	diff := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "go.sum", AddedLines: 20, IsGenerated: true, IsLockfile: true},
			{Path: "node_modules/pkg/index.js", AddedLines: 50, IsGenerated: true},
			{Path: "cmd/girl/main.go", AddedLines: 30},
		},
	}
	c := Classify(diff)

	if c.Added.Generated != 70 {
		t.Errorf("Generated = %d, want 70", c.Added.Generated)
	}
	if c.Added.Logic != 27 {
		t.Errorf("Logic = %d, want 27", c.Added.Logic)
	}
	if c.Added.EphemeralSupport != 3 {
		t.Errorf("EphemeralSupport = %d, want 3", c.Added.EphemeralSupport)
	}

	if c.Cohesion.Variance != 0 {
		t.Errorf("Cohesion.Variance = %f, want 0 (only one non-excluded file)", c.Cohesion.Variance)
	}
}

func TestClassify_EmptyDiff(t *testing.T) {
	diff := &diffstats.DiffStats{}
	c := Classify(diff)

	if c.Added.Logic != 0 {
		t.Errorf("Logic = %d, want 0", c.Added.Logic)
	}
	if c.Added.Test != 0 {
		t.Errorf("Test = %d, want 0", c.Added.Test)
	}
	if c.Added.ReusableSupport != 0 {
		t.Errorf("ReusableSupport = %d, want 0", c.Added.ReusableSupport)
	}
	if c.Added.EphemeralSupport != 0 {
		t.Errorf("EphemeralSupport = %d, want 0", c.Added.EphemeralSupport)
	}
	if c.Added.ConfigData != 0 {
		t.Errorf("ConfigData = %d, want 0", c.Added.ConfigData)
	}
	if c.Added.ConfigStructural != 0 {
		t.Errorf("ConfigStructural = %d, want 0", c.Added.ConfigStructural)
	}
	if c.Added.Generated != 0 {
		t.Errorf("Generated = %d, want 0", c.Added.Generated)
	}

	if c.Ratios.StructuralOverhead != 0 {
		t.Errorf("StructuralOverhead = %f, want 0", c.Ratios.StructuralOverhead)
	}
	if c.Ratios.TestToLogic != 0 {
		t.Errorf("TestToLogic = %f, want 0", c.Ratios.TestToLogic)
	}
	if c.Ratios.ProductiveScaffold != 0 {
		t.Errorf("ProductiveScaffold = %f, want 0", c.Ratios.ProductiveScaffold)
	}

	if c.Cohesion.Variance != 0 {
		t.Errorf("Cohesion.Variance = %f, want 0", c.Cohesion.Variance)
	}
	if len(c.Cohesion.SuggestedClusters) != 0 {
		t.Errorf("expected no suggested clusters, got %d", len(c.Cohesion.SuggestedClusters))
	}
}
