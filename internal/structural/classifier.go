package structural

import (
	"math"
	"path/filepath"
	"strings"

	"github.com/canadian-ai/girl/internal/diffstats"
)

func Classify(diff *diffstats.DiffStats) *Classification {
	result := &Classification{}

	for _, f := range diff.Files {
		if f.IsBinary {
			continue
		}
		added := f.AddedLines
		if added == 0 {
			continue
		}

		bucket := classifyByPath(f)
		if bucket == "generated" {
			result.Added.Generated += added
			continue
		}

		if bucket == "logic" || bucket == "test" {
			pct := ephemeralPercent(f.Path)
			eph := int(math.Round(float64(added) * pct))
			if eph > added {
				eph = added
			}
			rem := added - eph
			result.Added.EphemeralSupport += eph
			switch bucket {
			case "logic":
				result.Added.Logic += rem
			case "test":
				result.Added.Test += rem
			}
		} else {
			switch bucket {
			case "config_data":
				result.Added.ConfigData += added
			case "config_structural":
				result.Added.ConfigStructural += added
			}
		}
	}

	result.Ratios = computeRatios(result.Added)
	result.Cohesion = computeCohesion(diff)
	return result
}

type bucketLabel string

const (
	bucketLogic           bucketLabel = "logic"
	bucketTest            bucketLabel = "test"
	bucketConfigData      bucketLabel = "config_data"
	bucketConfigStructural bucketLabel = "config_structural"
	bucketGenerated       bucketLabel = "generated"
)

func classifyByPath(f diffstats.FileStat) bucketLabel {
	if f.IsGenerated || f.IsLockfile {
		return bucketGenerated
	}

	path := f.Path
	base := filepath.Base(path)

	if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, "_test.ts") || strings.HasSuffix(base, "_test.tsx") {
		return bucketTest
	}
	if strings.Contains(base, ".spec.") || strings.Contains(base, ".test.") {
		return bucketTest
	}
	if strings.Contains(path, "/test/") || strings.HasPrefix(path, "test/") {
		return bucketTest
	}
	if strings.Contains(path, "/__tests__/") || strings.HasPrefix(path, "__tests__/") {
		return bucketTest
	}

	// Check config_structural patterns FIRST (before generic config_data checks)
	if strings.HasPrefix(base, "tsconfig") ||
		strings.HasPrefix(base, ".babelrc") ||
		base == ".browserslistrc" ||
		strings.HasPrefix(base, ".eslintrc") ||
		strings.HasPrefix(base, ".prettierrc") ||
		base == "Makefile" ||
		base == "Dockerfile" ||
		strings.Contains(path, ".github/") ||
		base == ".gitlab-ci.yml" ||
		base == "go.mod" || base == "go.sum" ||
		base == "Gemfile" || base == "Gemfile.lock" ||
		base == "Package.resolved" ||
		base == "Cargo.toml" {
		return bucketConfigStructural
	}

	if strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".scss") || strings.HasSuffix(path, ".less") {
		return bucketConfigData
	}
	if strings.HasSuffix(path, ".json") {
		return bucketConfigData
	}

	if strings.HasPrefix(path, "vendor/") ||
		strings.HasPrefix(path, "node_modules/") ||
		strings.HasPrefix(path, "gen/") ||
		strings.HasSuffix(base, ".pb.go") ||
		strings.HasSuffix(base, ".pb.ts") ||
		strings.HasSuffix(base, ".pb.swift") ||
		base == "yarn.lock" || base == "package-lock.json" || base == "Cargo.lock" {
		return bucketGenerated
	}

	return bucketLogic
}

func ephemeralPercent(path string) float64 {
	ext := filepath.Ext(path)
	if ext == ".go" {
		return 0.10
	}
	return 0.05
}

func computeRatios(b BucketLineCounts) StructuralRatios {
	r := StructuralRatios{}

	denomSO := float64(b.Logic + b.ReusableSupport + b.EphemeralSupport + b.ConfigStructural)
	if denomSO != 0 {
		r.StructuralOverhead = float64(b.EphemeralSupport+b.ConfigStructural) / denomSO
	}

	denomTL := float64(b.Logic + b.ReusableSupport)
	if denomTL != 0 {
		r.TestToLogic = float64(b.Test) / denomTL
	} else if b.Test > 0 {
		r.TestToLogic = math.Inf(1)
	}

	denomPS := float64(b.ReusableSupport + b.EphemeralSupport)
	if denomPS != 0 {
		r.ProductiveScaffold = float64(b.ReusableSupport) / denomPS
	}

	return r
}

func computeCohesion(diff *diffstats.DiffStats) CohesionResult {
	var files []string
	for _, f := range diff.Files {
		if f.IsGenerated || f.IsLockfile || f.IsBinary {
			continue
		}
		files = append(files, f.Path)
	}

	if len(files) < 2 {
		return CohesionResult{}
	}

	tokens := make([][]string, len(files))
	for i, p := range files {
		tokens[i] = pathTokens(p)
	}

	var totalDist float64
	var pairCount int
	for i := 0; i < len(files); i++ {
		for j := i + 1; j < len(files); j++ {
			jacc := jaccard(tokens[i], tokens[j])
			totalDist += 1.0 - jacc
			pairCount++
		}
	}

	variance := totalDist / float64(pairCount)

	var clusters [][]string
	if variance > 0.6 {
		clusters = clusterFiles(files, tokens, 0.3)
	}

	return CohesionResult{
		Variance:         variance,
		SuggestedClusters: clusters,
	}
}

func pathTokens(path string) []string {
	var tokens []string
	parts := strings.Split(path, "/")
	for _, part := range parts {
		sub := strings.Split(part, ".")
		for _, s := range sub {
			if s != "" {
				tokens = append(tokens, s)
			}
		}
	}
	return tokens
}

func jaccard(a, b []string) float64 {
	setA := make(map[string]struct{}, len(a))
	setB := make(map[string]struct{}, len(b))
	for _, t := range a {
		setA[t] = struct{}{}
	}
	for _, t := range b {
		setB[t] = struct{}{}
	}

	intersection := 0
	for t := range setA {
		if _, ok := setB[t]; ok {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func clusterFiles(files []string, tokens [][]string, threshold float64) [][]string {
	n := len(files)
	parent := make([]int, n)
	for i := 0; i < n; i++ {
		parent[i] = i
	}

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if jaccard(tokens[i], tokens[j]) >= threshold {
				union(i, j)
			}
		}
	}

	rootToMembers := make(map[int][]int)
	for i := 0; i < n; i++ {
		r := find(i)
		rootToMembers[r] = append(rootToMembers[r], i)
	}

	var clusters [][]string
	for _, members := range rootToMembers {
		if len(members) < 2 {
			continue
		}
		cluster := make([]string, len(members))
		for k, idx := range members {
			cluster[k] = files[idx]
		}
		clusters = append(clusters, cluster)
	}

	return clusters
}
