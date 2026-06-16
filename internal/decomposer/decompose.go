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
	DiffStats         *diffstats.DiffStats
	PlanID            string
	SuggestedClusters [][]string // from structural analysis cohesion
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

	// If suggested clusters are provided, override with cluster-based decomposition
	if len(req.SuggestedClusters) > 0 {
		clusterTasks := buildClusterTasks(req.SuggestedClusters, req.DiffStats)
		if len(clusterTasks) > 0 {
			return &ir.Decomposition{
				Strategy:   "cluster-decomposition",
				ParentPlan: req.PlanID,
				Tasks:      clusterTasks,
			}
		}
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

func buildClusterTasks(clusters [][]string, diff *diffstats.DiffStats) []ir.DecompositionTask {
	if len(clusters) == 0 {
		return nil
	}

	fileToCluster := make(map[string]int)
	for ci, cluster := range clusters {
		for _, path := range cluster {
			fileToCluster[path] = ci
		}
	}

	// Count assigned vs unassigned files
	assignedCount := 0
	seen := make(map[string]bool)
	for _, f := range diff.Files {
		if _, ok := fileToCluster[f.Path]; ok {
			assignedCount++
		}
	}

	// Only use clusters if at least one real cluster (≥2 files) exists
	hasRealCluster := false
	for _, cluster := range clusters {
		if len(cluster) >= 2 {
			hasRealCluster = true
			break
		}
	}
	if !hasRealCluster {
		return nil
	}

	// Create one task per cluster
	var tasks []ir.DecompositionTask
	for ci, cluster := range clusters {
		if len(cluster) == 0 {
			continue
		}
		info := struct {
			files  []string
			total  int
			hasGo  bool
			hasTS  bool
			hasSQL bool
		}{files: cluster}

		for _, path := range cluster {
			seen[path] = true
			info.total += 10
			if strings.HasSuffix(path, ".go") {
				info.hasGo = true
			}
			if strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx") {
				info.hasTS = true
			}
			if strings.HasSuffix(path, ".sql") {
				info.hasSQL = true
			}
		}

		maxLines := info.total + info.total/2
		if maxLines < 200 {
			maxLines = 200
		}

		var verify []string
		if info.hasGo {
			verify = append(verify, "go build ./...", "go vet ./...", "go test ./...")
		}
		if info.hasTS {
			verify = append(verify, "tsc --noEmit")
		}
		if info.hasSQL {
			verify = append(verify, "go build ./...")
		}
		if len(verify) == 0 {
			verify = []string{"go build ./...", "go test ./..."}
		}

		goal := fmt.Sprintf("Implement %s layer changes", clusterFilesLabel(cluster))
		id := fmt.Sprintf("task_%03d_cluster_%s", ci+1, clusterFilesLabel(cluster))

		task := ir.DecompositionTask{
			ID:             id,
			Goal:           goal,
			AllowedFiles:   cluster,
			MaxDiffLines:   maxLines,
			Parallelizable: ci == 0,
			Verification:   verify,
		}
		if ci > 0 {
			task.DependsOn = []string{tasks[0].ID}
		}
		tasks = append(tasks, task)
	}

	// Add a remainder task for files not in any cluster
	var unassigned []string
	for _, f := range diff.Files {
		if !seen[f.Path] {
			unassigned = append(unassigned, f.Path)
		}
	}
	if len(unassigned) > 0 {
		var verify []string
		hasGo := false
		for _, path := range unassigned {
			if strings.HasSuffix(path, ".go") {
				hasGo = true
			}
		}
		if hasGo {
			verify = []string{"go build ./...", "go test ./..."}
		} else {
			verify = []string{"go build ./...", "go test ./..."}
		}

		remainderTask := ir.DecompositionTask{
			ID:           fmt.Sprintf("task_%03d_remainder", len(tasks)+1),
			Goal:         "Implement remaining changes",
			AllowedFiles: unassigned,
			MaxDiffLines: 400,
			Verification: verify,
		}
		// Remainder depends on the first cluster task
		if len(tasks) > 0 {
			remainderTask.DependsOn = []string{tasks[0].ID}
		} else {
			remainderTask.Parallelizable = true
		}
		tasks = append(tasks, remainderTask)
	}

	return tasks
}

func clusterFilesLabel(files []string) string {
	if len(files) == 0 {
		return "unknown"
	}
	// Find longest common directory prefix
	parts := make([][]string, len(files))
	for i, f := range files {
		parts[i] = strings.Split(f, "/")
	}
	// Find common prefix length
	maxDepth := len(parts[0])
	for _, p := range parts[1:] {
		if len(p) < maxDepth {
			maxDepth = len(p)
		}
	}
	common := 0
	for common < maxDepth {
		allMatch := true
		for _, p := range parts[1:] {
			if len(p) <= common || p[common] != parts[0][common] {
				allMatch = false
				break
			}
		}
		if !allMatch {
			break
		}
		common++
	}
	// Use last common dir component, or first file's name if no common dir
	if common >= 2 {
		return parts[0][common-1]
	}
	if common >= 1 {
		return parts[0][common-1]
	}
	// No common prefix — use first file's top-level dir
	return parts[0][0]
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
