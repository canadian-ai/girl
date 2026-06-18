package proof

import (
	"fmt"
	"sort"
	"strings"

	"github.com/canadian-ai/girl/internal/ir"
)

var DefaultImprovements = []string{
	"Reduce high-complexity functions",
	"Split large files/components",
	"Flatten deeply nested logic",
	"Handle ignored errors explicitly",
}

// NewSummary builds deterministic aggregate counts from analyzer output.
func NewSummary(target string, result *ir.AnalyzerResult, topFiles int) Summary {
	if topFiles < 0 {
		topFiles = 0
	}
	s := Summary{Target: target}
	if result == nil {
		return s
	}
	s.FilesScanned = len(result.Files)
	s.Diagnostics = len(result.Diagnostics)
	codes := map[string]int{}
	files := map[string]int{}
	for _, d := range result.Diagnostics {
		switch d.Severity {
		case ir.SeverityHigh:
			s.High++
		case ir.SeverityMedium:
			s.Medium++
		case ir.SeverityLow:
			s.Low++
		}
		codes[d.Code]++
		files[d.File]++
	}
	s.DiagnosticCodes = sortedCodeCounts(codes, 0)
	s.WorstFiles = sortedFileCounts(files, topFiles)
	return s
}

func NewBenchmarkReport(target string, result *ir.AnalyzerResult, topFiles int) Report {
	return Report{Kind: "benchmark", Summary: NewSummary(target, result, topFiles)}
}

func NewProofReport(target string, result *ir.AnalyzerResult, topFiles int) Report {
	summary := NewSummary(target, result, topFiles)
	score := HealthScore(summary.High, summary.Medium, summary.Low)
	return Report{Kind: "proof", Summary: summary, HealthScore: score, Status: Status(score), TopImprovements: DefaultImprovements}
}

func sortedCodeCounts(counts map[string]int, limit int) []CodeCount {
	out := make([]CodeCount, 0, len(counts))
	for k, v := range counts {
		out = append(out, CodeCount{Code: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Code < out[j].Code
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func sortedFileCounts(counts map[string]int, limit int) []FileCount {
	out := make([]FileCount, 0, len(counts))
	for k, v := range counts {
		out = append(out, FileCount{File: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].File < out[j].File
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func TextBenchmark(r Report) string {
	s := r.Summary
	var b strings.Builder
	fmt.Fprintf(&b, "GIRL Benchmark\n==============\n\nTarget: %s\nFiles scanned: %d\nDiagnostics: %d\n\nSeverity\n--------\nHigh:   %d\nMedium: %d\nLow:    %d\n\nTop diagnostic codes\n--------------------\n", s.Target, s.FilesScanned, s.Diagnostics, s.High, s.Medium, s.Low)
	for _, c := range s.DiagnosticCodes {
		fmt.Fprintf(&b, "%-28s %d\n", c.Code, c.Count)
	}
	b.WriteString("\nWorst files\n-----------\n")
	for _, f := range s.WorstFiles {
		fmt.Fprintf(&b, "%-28s %d\n", f.File, f.Count)
	}
	return b.String()
}

func TextProof(r Report) string {
	s := r.Summary
	var b strings.Builder
	fmt.Fprintf(&b, "GIRL Proof Report\n=================\n\nTarget: %s\nHealth Score: %d/100 — %s\n\nFiles scanned: %d\nDiagnostics: %d\n\nHigh:   %d\nMedium: %d\nLow:    %d\n\nTop improvements\n----------------\n", s.Target, r.HealthScore, r.Status, s.FilesScanned, s.Diagnostics, s.High, s.Medium, s.Low)
	for i, item := range r.TopImprovements {
		fmt.Fprintf(&b, "%d. %s\n", i+1, item)
	}
	return b.String()
}

func MarkdownBenchmark(r Report) string {
	s := r.Summary
	var b strings.Builder
	fmt.Fprintf(&b, "# GIRL Benchmark\n\n**Target:** `%s`  \n**Files scanned:** %d  \n**Diagnostics:** %d\n\n## Severity\n\n| Severity | Count |\n|---|---:|\n| High | %d |\n| Medium | %d |\n| Low | %d |\n\n## Top diagnostic codes\n\n| Code | Count |\n|---|---:|\n", s.Target, s.FilesScanned, s.Diagnostics, s.High, s.Medium, s.Low)
	for _, c := range s.DiagnosticCodes {
		fmt.Fprintf(&b, "| `%s` | %d |\n", c.Code, c.Count)
	}
	b.WriteString("\n## Worst files\n\n| File | Count |\n|---|---:|\n")
	for _, f := range s.WorstFiles {
		fmt.Fprintf(&b, "| `%s` | %d |\n", f.File, f.Count)
	}
	return b.String()
}

func MarkdownProof(r Report) string {
	s := r.Summary
	var b strings.Builder
	fmt.Fprintf(&b, "# GIRL Proof Report\n\n**Target:** `%s`  \n**Health Score:** %d/100 — **%s**\n\n**Files scanned:** %d  \n**Diagnostics:** %d\n\n| Severity | Count |\n|---|---:|\n| High | %d |\n| Medium | %d |\n| Low | %d |\n\n## Top improvements\n\n", s.Target, r.HealthScore, r.Status, s.FilesScanned, s.Diagnostics, s.High, s.Medium, s.Low)
	for i, item := range r.TopImprovements {
		fmt.Fprintf(&b, "%d. %s\n", i+1, item)
	}
	return b.String()
}
