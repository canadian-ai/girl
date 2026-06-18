package proof

import (
	"testing"

	"github.com/canadian-ai/girl/internal/ir"
)

func TestHealthScoreAndStatus(t *testing.T) {
	tests := []struct {
		high, medium, low, want int
		status                  string
	}{
		{0, 0, 0, 100, "Excellent"},
		{1, 1, 1, 88, "Good"},
		{3, 12, 26, 14, "High risk"},
		{20, 20, 20, 0, "High risk"},
	}
	for _, tt := range tests {
		got := HealthScore(tt.high, tt.medium, tt.low)
		if got != tt.want {
			t.Fatalf("HealthScore(%d,%d,%d)=%d, want %d", tt.high, tt.medium, tt.low, got, tt.want)
		}
		if Status(got) != tt.status {
			t.Fatalf("Status(%d)=%q, want %q", got, Status(got), tt.status)
		}
	}
}

func TestNewSummaryGroupsAndSorts(t *testing.T) {
	result := &ir.AnalyzerResult{
		Files: []*ir.FileIR{{Path: "b.go"}, {Path: "a.go"}},
		Diagnostics: []ir.Diagnostic{
			{Code: "go.z", Severity: ir.SeverityLow, File: "b.go"},
			{Code: "go.a", Severity: ir.SeverityHigh, File: "a.go"},
			{Code: "go.a", Severity: ir.SeverityMedium, File: "b.go"},
			{Code: "go.b", Severity: ir.SeverityLow, File: "a.go"},
		},
	}
	s := NewSummary(".", result, 2)
	if s.FilesScanned != 2 || s.Diagnostics != 4 || s.High != 1 || s.Medium != 1 || s.Low != 2 {
		t.Fatalf("unexpected summary: %#v", s)
	}
	if got := s.DiagnosticCodes; len(got) != 3 || got[0].Code != "go.a" || got[0].Count != 2 || got[1].Code != "go.b" || got[2].Code != "go.z" {
		t.Fatalf("codes not sorted deterministically: %#v", got)
	}
	if got := s.WorstFiles; len(got) != 2 || got[0].File != "a.go" || got[1].File != "b.go" {
		t.Fatalf("files not sorted deterministically: %#v", got)
	}
}
