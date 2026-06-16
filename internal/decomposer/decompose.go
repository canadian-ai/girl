package decomposer

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/canadian-ai/girl/internal/diffstats"
	"github.com/canadian-ai/girl/internal/ir"
)

type DecomposeRequest struct {
	DiffStats *diffstats.DiffStats
	PlanID    string
}

func Decompose(req *DecomposeRequest) *ir.Decomposition {
	if req == nil || req.DiffStats == nil || len(req.DiffStats.Files) == 0 {
		return &ir.Decomposition{
			Strategy: "atomic-reviewable-tasks",
			Tasks:    []ir.DecompositionTask{},
		}
	}

	type fileGroup struct {
		category string
		files    []diffstats.FileStat
		total    int
	}
	groupMap := map[string]*fileGroup{}
	order := []string{}

	for _, f := range req.DiffStats.Files {
		cat := categoryForFile(f.Path)
		if cat == "" {
			cat = "other"
		}
		if _, ok := groupMap[cat]; !ok {
			groupMap[cat] = &fileGroup{category: cat}
			order = append(order, cat)
		}
		groupMap[cat].files = append(groupMap[cat].files, f)
		groupMap[cat].total += f.ChangedLines
	}

	depOrder := dependencyOrder(order)

	// Sort order by dependency priority so tasks emit in correct sequence.
	sort.SliceStable(order, func(i, j int) bool {
		return depOrder[order[i]] < depOrder[order[j]]
	})

	var tasks []ir.DecompositionTask
	for i, cat := range order {
		grp := groupMap[cat]
		if len(grp.files) == 0 {
			continue
		}

		goal := taskGoal(cat)
		allowedFiles := make([]string, len(grp.files))
		for j, f := range grp.files {
			allowedFiles[j] = f.Path
		}
		sort.Strings(allowedFiles)

		var deps []string
		if taskIdx, ok := depOrder[cat]; ok && taskIdx > 0 {
			for _, prevCat := range order[:taskIdx] {
				deps = append(deps, fmt.Sprintf("task_%03d_%s", indexOf(order, prevCat)+1, taskSlug(prevCat)))
			}
		}

		maxLines := grp.total + grp.total/2
		if maxLines < 200 {
			maxLines = 200
		}

		task := ir.DecompositionTask{
			ID:             fmt.Sprintf("task_%03d_%s", i+1, taskSlug(cat)),
			Goal:           goal,
			AllowedFiles:   allowedFiles,
			MaxDiffLines:   maxLines,
			Parallelizable: len(deps) == 0,
			DependsOn:      deps,
			Verification:   taskVerification(cat),
		}
		tasks = append(tasks, task)
	}

	return &ir.Decomposition{
		Strategy:   "atomic-reviewable-tasks",
		ParentPlan: req.PlanID,
		Tasks:      tasks,
	}
}

func categoryForFile(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))
	dir := filepath.Dir(path)

	switch {
	case ext == ".go" && !strings.HasSuffix(base, "_test.go"):
		return "go"
	case ext == ".go" && strings.HasSuffix(base, "_test.go"):
		return "test"
	case ext == ".ts" || ext == ".tsx":
		return "typescript"
	case ext == ".js" || ext == ".jsx":
		return "javascript"
	case ext == ".css" || ext == ".scss" || ext == ".less" || ext == ".sass":
		return "style"
	case ext == ".md":
		return "documentation"
	case ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".toml":
		return "config"
	case ext == ".sql":
		return "schema"
	case strings.Contains(dir, "migrations") || strings.Contains(dir, "migrate"):
		return "schema"
	case ext == ".proto":
		return "proto"
	case ext == ".py":
		return "python"
	case ext == ".rs":
		return "rust"
	default:
		return "other"
	}
}

func dependencyOrder(categories []string) map[string]int {
	priority := map[string]int{
		"schema":        0,
		"proto":         1,
		"config":        2,
		"go":            3,
		"typescript":    4,
		"javascript":    5,
		"python":        6,
		"rust":          7,
		"library":       8,
		"style":         9,
		"other":         10,
		"documentation": 11,
		"test":          12,
	}
	sorted := make([]string, len(categories))
	copy(sorted, categories)
	sort.SliceStable(sorted, func(i, j int) bool {
		pi := priority[sorted[i]]
		pj := priority[sorted[j]]
		if pi != pj {
			return pi < pj
		}
		return sorted[i] < sorted[j]
	})
	result := map[string]int{}
	for i, cat := range sorted {
		result[cat] = i
	}
	return result
}

func indexOf(slice []string, s string) int {
	for i, v := range slice {
		if v == s {
			return i
		}
	}
	return -1
}

func taskGoal(category string) string {
	switch category {
	case "go":
		return "Implement Go logic"
	case "test":
		return "Add/modify tests"
	case "typescript":
		return "Implement TypeScript logic"
	case "javascript":
		return "Implement JavaScript logic"
	case "style":
		return "Apply style changes"
	case "documentation":
		return "Update documentation"
	case "config":
		return "Update configuration"
	case "schema":
		return "Update database schema"
	case "proto":
		return "Update protobuf definitions"
	case "python":
		return "Implement Python logic"
	case "rust":
		return "Implement Rust logic"
	default:
		return "Implement remaining changes"
	}
}

func taskSlug(category string) string {
	slug := strings.NewReplacer(".", "-", "_", "-").Replace(category)
	if len(slug) > 30 {
		slug = slug[:30]
	}
	return slug
}

func taskVerification(category string) []string {
	switch category {
	case "go":
		return []string{"go build ./...", "go vet ./...", "go test ./..."}
	case "typescript":
		return []string{"tsc --noEmit"}
	case "javascript":
		return []string{"npm run lint"}
	case "test":
		return []string{"go test ./..."}
	case "style":
		return []string{"npm run lint:css"}
	case "schema":
		return []string{"go build ./..."}
	default:
		return []string{"go build ./...", "go test ./..."}
	}
}
