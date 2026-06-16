package commands

import (
	"strings"
	"testing"

	"github.com/canadian-ai/girl/internal/diffstats"
	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/reviewability"
)

func TestReviewStructuralMultiLayerDiff(t *testing.T) {
	input := `diff --git a/db/schema.sql b/db/schema.sql
index a..b 100644
--- a/db/schema.sql
+++ b/db/schema.sql
@@ -1 +1,31 @@
+CREATE TABLE users (id int);
+CREATE TABLE posts (id int);
diff --git a/internal/server/handler.go b/internal/server/handler.go
index a..c 100644
--- a/internal/server/handler.go
+++ b/internal/server/handler.go
@@ -1 +1,21 @@
+package server
+func HandleGet() string { return "ok" }
diff --git a/web/ui/components/Table.tsx b/web/ui/components/Table.tsx
index a..d 100644
--- a/web/ui/components/Table.tsx
+++ b/web/ui/components/Table.tsx
@@ -1 +1,21 @@
+export function Table() { return <div />; }
diff --git a/web/ui/components/Table.test.tsx b/web/ui/components/Table.test.tsx
new file mode 100644
index 0..e
--- /dev/null
+++ b/web/ui/components/Table.test.tsx
@@ -0,0 +1,15 @@
+import { render } from '@testing-library/react';
+import { Table } from './Table';
+test('renders', () => { render(<Table />); });
`

	stats, err := diffstats.ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	budget := ir.ReviewabilityBudget{
		MaxDiffLines:    1500,
		MaxTouchedFiles: 12,
		MaxRisk:         ir.SeverityMedium,
	}
	r := reviewability.Evaluate(stats, budget)
	if r == nil {
		t.Fatal("expected reviewability result")
	}
	if r.Structural == nil {
		t.Fatal("expected structural classification to be non-nil")
	}
	if r.Structural.Added.Logic <= 0 {
		t.Errorf("expected logic lines > 0, got %d", r.Structural.Added.Logic)
	}
	if r.Structural.Added.Test <= 0 {
		t.Errorf("expected test lines > 0, got %d", r.Structural.Added.Test)
	}
	if r.Structural.Cohesion.Variance <= 0 {
		t.Errorf("expected cohesion variance > 0 for multi-layer diff, got %.2f", r.Structural.Cohesion.Variance)
	}
	if len(r.Diagnostics) == 0 {
		t.Fatal("expected structural diagnostics")
	}
	hasLowCohesion := false
	hasMixedBoundaries := false
	for _, d := range r.Diagnostics {
		switch d.Code {
		case "agent.low-cohesion":
			hasLowCohesion = true
		case "agent.mixed-boundaries":
			hasMixedBoundaries = true
		}
	}
	if !hasLowCohesion {
		t.Error("expected agent.low-cohesion diagnostic for multi-layer diff")
	}
	if !hasMixedBoundaries {
		t.Error("expected agent.mixed-boundaries diagnostic for multi-layer diff")
	}
}

func TestReviewStructuralEmptyDiff(t *testing.T) {
	stats, err := diffstats.ParseDiff(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}

	budget := ir.ReviewabilityBudget{
		MaxDiffLines:    1500,
		MaxTouchedFiles: 12,
		MaxRisk:         ir.SeverityMedium,
	}
	r := reviewability.Evaluate(stats, budget)
	if r == nil {
		t.Fatal("expected reviewability result")
	}
	if r.Structural == nil {
		t.Fatal("expected structural classification for empty diff")
	}
	if r.Structural.Added.Logic != 0 {
		t.Errorf("expected empty structural, got Logic=%d", r.Structural.Added.Logic)
	}
	if r.Structural.Ratios.StructuralOverhead != 0 {
		t.Errorf("expected zero overhead for empty diff, got %.2f", r.Structural.Ratios.StructuralOverhead)
	}
}
