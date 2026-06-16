package commands

import (
	"strings"
	"testing"

	"github.com/canadian-ai/girl/internal/decomposer"
	"github.com/canadian-ai/girl/internal/diffstats"
	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/reviewability"
)

func TestReviewDecomposePipelineSmallDiff(t *testing.T) {
	input := `diff --git a/main.go b/main.go
index abc..def 100644
--- a/main.go
+++ b/main.go
@@ -1,5 +1,7 @@
 package main

+import "fmt"
+
 func main() {
-	println("hello")
+	fmt.Println("hello world")
 }
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
	if r.Result.Status != "pass" {
		t.Errorf("expected pass for small diff, got %s", r.Result.Status)
	}
	if len(r.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics for small diff, got %d", len(r.Diagnostics))
	}

	decomp := decomposer.Decompose(&decomposer.DecomposeRequest{
		DiffStats: stats,
	})
	if decomp == nil {
		t.Fatal("expected decomposition")
	}
	if len(decomp.Tasks) != 1 {
		t.Errorf("expected 1 task for single Go file, got %d", len(decomp.Tasks))
	}
	if len(decomp.Tasks) > 0 {
		task := decomp.Tasks[0]
		if !strings.HasPrefix(task.ID, "task_") {
			t.Errorf("task ID should start with task_, got %q", task.ID)
		}
		if len(task.AllowedFiles) != 1 || task.AllowedFiles[0] != "main.go" {
			t.Errorf("expected allowed file main.go, got %v", task.AllowedFiles)
		}
		if !task.Parallelizable {
			t.Errorf("single task should be parallelizable")
		}
	}
}

func TestReviewDecomposePipelineLargeDiff(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("diff --git a/main.go b/main.go\n")
	builder.WriteString("--- a/main.go\n+++ b/main.go\n")
	builder.WriteString("@@ -1,1 +1,2001 @@\n")
	for i := 0; i < 2000; i++ {
		builder.WriteString("+line ")
		builder.WriteByte(byte('0' + i%10))
		builder.WriteByte('\n')
	}
	builder.WriteString("diff --git a/schema.sql b/schema.sql\n")
	builder.WriteString("--- a/schema.sql\n+++ b/schema.sql\n")
	builder.WriteString("@@ -1,1 +1,51 @@\n")
	for i := 0; i < 50; i++ {
		builder.WriteString("+CREATE TABLE ")
		builder.WriteByte(byte('a' + i%26))
		builder.WriteString(" (id int);\n")
	}
	builder.WriteString("diff --git a/README.md b/README.md\n")
	builder.WriteString("--- a/README.md\n+++ b/README.md\n")
	builder.WriteString("@@ -1,1 +1,6 @@\n+## Docs\n+new docs\n")

	stats, err := diffstats.ParseDiff(strings.NewReader(builder.String()))
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
	if r.Result.Status != "fail" {
		t.Errorf("expected fail for large diff, got %s", r.Result.Status)
	}
	if r.Result.Recommendation != "decompose" {
		t.Errorf("expected decompose recommendation, got %s", r.Result.Recommendation)
	}
	if len(r.Diagnostics) == 0 {
		t.Fatal("expected diagnostics for large diff")
	}
	hasDiffTooLarge := false
	hasUnreviewable := false
	hasParallelization := false
	for _, d := range r.Diagnostics {
		switch d.Code {
		case "agent.diff-too-large":
			hasDiffTooLarge = true
		case "agent.unreviewable-plan":
			hasUnreviewable = true
		case "agent.parallelization-opportunity":
			hasParallelization = true
		}
	}
	if !hasDiffTooLarge {
		t.Error("expected agent.diff-too-large diagnostic")
	}
	if !hasUnreviewable {
		t.Error("expected agent.unreviewable-plan diagnostic")
	}
	if !hasParallelization {
		t.Error("expected agent.parallelization-opportunity diagnostic for multi-file diff")
	}

	decomp := decomposer.Decompose(&decomposer.DecomposeRequest{
		DiffStats: stats,
	})
	if decomp == nil {
		t.Fatal("expected decomposition")
	}
	if len(decomp.Tasks) < 2 {
		t.Errorf("expected at least 2 tasks for multi-file diff, got %d", len(decomp.Tasks))
	}
	hasSchema := false
	hasGo := false
	hasDocs := false
	for _, task := range decomp.Tasks {
		hasSchema = hasSchema || task.Goal == "Update database schema"
		hasGo = hasGo || task.Goal == "Implement Go logic"
		hasDocs = hasDocs || task.Goal == "Update documentation"
	}
	if !hasSchema {
		t.Error("expected schema task")
	}
	if !hasGo {
		t.Error("expected Go task")
	}
	if !hasDocs {
		t.Error("expected documentation task")
	}
}

func TestReviewDecomposePipelineExactBudget(t *testing.T) {
	input := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1,1501 @@
`
	for i := 0; i < 1500; i++ {
		input += "+line " + string(rune('0'+i%10)) + "\n"
	}

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
	if r.Result.Status != "pass" {
		t.Errorf("expected pass for exact budget match, got %s", r.Result.Status)
	}
}

func TestReviewDecomposePipelineBudgetOverride(t *testing.T) {
	input := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1,3001 @@
`
	for i := 0; i < 3000; i++ {
		input += "+line " + string(rune('0'+i%10)) + "\n"
	}

	stats, err := diffstats.ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	budget := ir.ReviewabilityBudget{
		MaxDiffLines:    5000,
		MaxTouchedFiles: 12,
		MaxRisk:         ir.SeverityMedium,
	}
	r := reviewability.Evaluate(stats, budget)
	if r.Result.Status != "pass" {
		t.Errorf("expected pass with relaxed budget, got %s", r.Result.Status)
	}
}

func TestReviewDecomposePipelineNoFiles(t *testing.T) {
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
	if r.Result.Status != "pass" {
		t.Errorf("expected pass for empty diff, got %s", r.Result.Status)
	}

	decomp := decomposer.Decompose(&decomposer.DecomposeRequest{
		DiffStats: stats,
	})
	if len(decomp.Tasks) != 0 {
		t.Errorf("expected 0 tasks for empty diff, got %d", len(decomp.Tasks))
	}
}

func TestReviewDecomposePipelineCustomBudgetFails(t *testing.T) {
	input := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1 +1,601 @@
`
	for i := 0; i < 600; i++ {
		input += "+line\n"
	}

	stats, err := diffstats.ParseDiff(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	budget := ir.ReviewabilityBudget{
		MaxDiffLines:    50,
		MaxTouchedFiles: 12,
		MaxRisk:         ir.SeverityMedium,
	}
	r := reviewability.Evaluate(stats, budget)
	if r.Result.Status != "fail" {
		t.Errorf("expected fail with tight budget, got %s", r.Result.Status)
	}
}
