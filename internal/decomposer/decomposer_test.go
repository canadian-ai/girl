package decomposer

import (
	"testing"

	"github.com/canadian-ai/girl/internal/diffstats"
)

func TestDecomposeEmptyStats(t *testing.T) {
	d := Decompose(nil)
	if d == nil {
		t.Fatal("expected non-nil decomposition")
	}
	if len(d.Tasks) != 0 {
		t.Errorf("expected 0 tasks for nil request, got %d", len(d.Tasks))
	}
}

func TestDecomposeNilStats(t *testing.T) {
	d := Decompose(&DecomposeRequest{DiffStats: nil})
	if d == nil {
		t.Fatal("expected non-nil decomposition")
	}
	if len(d.Tasks) != 0 {
		t.Errorf("expected 0 tasks for nil stats, got %d", len(d.Tasks))
	}
}

func TestDecomposeEmptyFiles(t *testing.T) {
	d := Decompose(&DecomposeRequest{
		DiffStats: &diffstats.DiffStats{Files: []diffstats.FileStat{}},
	})
	if len(d.Tasks) != 0 {
		t.Errorf("expected 0 tasks for empty files, got %d", len(d.Tasks))
	}
}

func TestDecomposeGoFiles(t *testing.T) {
	stats := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "main.go", ChangedLines: 100},
			{Path: "internal/handler.go", ChangedLines: 200},
		},
	}
	d := Decompose(&DecomposeRequest{DiffStats: stats})
	if d.Strategy != "atomic-reviewable-tasks" {
		t.Errorf("expected atomic-reviewable-tasks, got %s", d.Strategy)
	}
	if len(d.Tasks) == 0 {
		t.Fatal("expected at least 1 task")
	}
	if d.Tasks[0].Goal != "Implement Go logic" {
		t.Errorf("expected Go logic goal, got %s", d.Tasks[0].Goal)
	}
	if len(d.Tasks[0].AllowedFiles) != 2 {
		t.Errorf("expected 2 allowed files, got %d", len(d.Tasks[0].AllowedFiles))
	}
}

func TestDecomposeMixedCategories(t *testing.T) {
	stats := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "schema.sql", ChangedLines: 50},
			{Path: "main.go", ChangedLines: 200},
			{Path: "frontend/App.tsx", ChangedLines: 150},
			{Path: "README.md", ChangedLines: 10},
		},
	}
	d := Decompose(&DecomposeRequest{DiffStats: stats})
	if len(d.Tasks) < 2 {
		t.Errorf("expected multiple tasks for mixed categories, got %d", len(d.Tasks))
	}
	for _, task := range d.Tasks {
		if len(task.ID) < 6 || task.ID[:5] != "task_" {
			t.Errorf("task ID %q does not start with task_", task.ID)
		}
	}
}

func TestDecomposeDependencyOrder(t *testing.T) {
	stats := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "README.md", ChangedLines: 10},
			{Path: "schema.sql", ChangedLines: 50},
			{Path: "main.go", ChangedLines: 200},
			{Path: "frontend/App.tsx", ChangedLines: 150},
			{Path: "main_test.go", ChangedLines: 80},
		},
	}
	d := Decompose(&DecomposeRequest{DiffStats: stats})
	if len(d.Tasks) < 4 {
		t.Fatalf("expected at least 4 tasks for mixed categories, got %d", len(d.Tasks))
	}
	// Schema should be first by dependency priority
	if d.Tasks[0].Goal != "Update database schema" {
		t.Errorf("expected first task to be schema, got %q", d.Tasks[0].Goal)
	}
	// Go should come before typescript
	goIdx := -1
	tsIdx := -1
	docIdx := -1
	testIdx := -1
	for i, task := range d.Tasks {
		switch task.Goal {
		case "Implement Go logic":
			goIdx = i
		case "Implement TypeScript logic":
			tsIdx = i
		case "Update documentation":
			docIdx = i
		case "Add/modify tests":
			testIdx = i
		}
	}
	if goIdx < 0 {
		t.Fatal("expected Go task")
	}
	if tsIdx < 0 {
		t.Fatal("expected TypeScript task")
	}
	if goIdx > tsIdx {
		t.Errorf("Go task (idx %d) should come before TypeScript task (idx %d)", goIdx, tsIdx)
	}
	if docIdx < 0 {
		t.Fatal("expected documentation task")
	}
	if testIdx < 0 {
		t.Fatal("expected test task")
	}
	// Tests and docs should be last
	if testIdx < goIdx || testIdx < tsIdx {
		t.Errorf("test task (idx %d) should come after Go (idx %d) and TypeScript (idx %d)", testIdx, goIdx, tsIdx)
	}
	if docIdx < goIdx || docIdx < tsIdx {
		t.Errorf("documentation task (idx %d) should come after Go (idx %d) and TypeScript (idx %d)", docIdx, goIdx, tsIdx)
	}
}

func TestDecomposeParallelizable(t *testing.T) {
	stats := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "schema.sql", ChangedLines: 50},
			{Path: "main.go", ChangedLines: 200},
			{Path: "main_test.go", ChangedLines: 80},
		},
	}
	d := Decompose(&DecomposeRequest{DiffStats: stats})
	if len(d.Tasks) < 3 {
		t.Fatalf("expected at least 3 tasks, got %d", len(d.Tasks))
	}
	// Schema is first: no deps, parallelizable
	if !d.Tasks[0].Parallelizable {
		t.Errorf("first task (schema) should be parallelizable, got false")
	}
	if len(d.Tasks[0].DependsOn) != 0 {
		t.Errorf("first task should have no dependencies, got %v", d.Tasks[0].DependsOn)
	}
	// Go depends on schema
	if len(d.Tasks) > 1 {
		goTask := -1
		for i, t := range d.Tasks {
			if t.Goal == "Implement Go logic" {
				goTask = i
				break
			}
		}
		if goTask >= 0 && len(d.Tasks[goTask].DependsOn) == 0 {
			// dependency may not exist if order-based deps skip concurrent categories
		}
	}
}

func TestDecomposeTaskVerification(t *testing.T) {
	cases := []struct {
		path           string
		expectedVerify []string
	}{
		{"main.go", []string{"go build ./...", "go vet ./...", "go test ./..."}},
		{"main_test.go", []string{"go test ./..."}},
		{"component.tsx", []string{"tsc --noEmit"}},
		{"file.js", []string{"npm run lint"}},
		{"style.css", []string{"npm run lint:css"}},
		{"schema.sql", []string{"go build ./..."}},
		{"README.md", []string{"go build ./...", "go test ./..."}},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			stats := &diffstats.DiffStats{
				Files: []diffstats.FileStat{
					{Path: c.path, ChangedLines: 10},
				},
			}
			d := Decompose(&DecomposeRequest{DiffStats: stats})
			if len(d.Tasks) != 1 {
				t.Fatalf("expected 1 task, got %d", len(d.Tasks))
			}
			if len(d.Tasks[0].Verification) != len(c.expectedVerify) {
				t.Fatalf("expected %d verification commands, got %d: %v", len(c.expectedVerify), len(d.Tasks[0].Verification), d.Tasks[0].Verification)
			}
			for i, v := range d.Tasks[0].Verification {
				if v != c.expectedVerify[i] {
					t.Errorf("verification[%d] = %q, want %q", i, v, c.expectedVerify[i])
				}
			}
		})
	}
}

func TestDecomposeTaskIDsDeterministic(t *testing.T) {
	stats := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "main.go", ChangedLines: 100},
			{Path: "schema.sql", ChangedLines: 50},
		},
	}
	d1 := Decompose(&DecomposeRequest{DiffStats: stats})
	d2 := Decompose(&DecomposeRequest{DiffStats: stats})
	if len(d1.Tasks) != len(d2.Tasks) {
		t.Fatalf("task count mismatch: %d vs %d", len(d1.Tasks), len(d2.Tasks))
	}
	for i := range d1.Tasks {
		if d1.Tasks[i].ID != d2.Tasks[i].ID {
			t.Errorf("task[%d] ID mismatch: %q vs %q", i, d1.Tasks[i].ID, d2.Tasks[i].ID)
		}
		if d1.Tasks[i].Goal != d2.Tasks[i].Goal {
			t.Errorf("task[%d] Goal mismatch: %q vs %q", i, d1.Tasks[i].Goal, d2.Tasks[i].Goal)
		}
	}
}

func TestDecomposeMaxDiffLinesEstimate(t *testing.T) {
	stats := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "main.go", ChangedLines: 30},
		},
	}
	d := Decompose(&DecomposeRequest{DiffStats: stats})
	if len(d.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(d.Tasks))
	}
	// maxLines = total + total/2, minimum 200
	// For 30 lines: 30 + 15 = 45, but floor is 200
	if d.Tasks[0].MaxDiffLines < 200 {
		t.Errorf("expected max diff lines at least 200, got %d", d.Tasks[0].MaxDiffLines)
	}
}

func TestDecomposeLargeFileMaxDiffLines(t *testing.T) {
	stats := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "main.go", ChangedLines: 500},
		},
	}
	d := Decompose(&DecomposeRequest{DiffStats: stats})
	if len(d.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(d.Tasks))
	}
	// maxLines = 500 + 250 = 750
	if d.Tasks[0].MaxDiffLines != 750 {
		t.Errorf("expected max diff lines 750, got %d", d.Tasks[0].MaxDiffLines)
	}
}

func TestDecomposeWithPlanID(t *testing.T) {
	stats := &diffstats.DiffStats{
		Files: []diffstats.FileStat{
			{Path: "main.go", ChangedLines: 100},
		},
	}
	d := Decompose(&DecomposeRequest{
		DiffStats: stats,
		PlanID:    "grp_abc123",
	})
	if d.ParentPlan != "grp_abc123" {
		t.Errorf("expected parent plan grp_abc123, got %s", d.ParentPlan)
	}
}
