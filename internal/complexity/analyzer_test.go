package complexity

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSource(t *testing.T, name, source string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAnalyzeTypeScriptDecisionPoints(t *testing.T) {
	path := writeSource(t, "decisions.ts", `
function decide(flag: boolean) {
  if (flag) {}
  for (let i = 0; i < 2; i++) {}
  switch (flag) {
    case true: break;
    case false: break;
    default: break;
  }
  try {} catch (error) {}
  const selected = flag && first || second;
  return flag ? 1 : 0;
}
`)
	report, err := Analyze(path, Options{Threshold: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Functions) != 1 {
		t.Fatalf("got %d functions, want 1", len(report.Functions))
	}
	metric := report.Functions[0]
	if metric.Complexity != 9 {
		t.Errorf("complexity = %d, want 9", metric.Complexity)
	}
	if metric.DecisionPoints != 8 {
		t.Errorf("decision points = %d, want 8", metric.DecisionPoints)
	}
}

func TestAnalyzeReactAndNestedCallbackIndependently(t *testing.T) {
	path := writeSource(t, "Dashboard.tsx", `
type Props = { items?: Array<{ active: boolean }> };

const Dashboard = ({ items }: Props) => {
  if (!items) return null;
  return <div>{items.map(item => item.active ? <b/> : <i/>)}</div>;
};
`)
	report, err := Analyze(path, Options{Language: "react", Threshold: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Functions) != 2 {
		t.Fatalf("got %d functions, want component and callback", len(report.Functions))
	}
	component := report.Functions[0]
	callback := report.Functions[1]
	if component.Symbol != "Dashboard" || component.Kind != "react-component" {
		t.Fatalf("unexpected component metric: %#v", component)
	}
	if component.Complexity != 2 {
		t.Errorf("component complexity = %d, want 2; nested callback must be excluded", component.Complexity)
	}
	if callback.Symbol != "Dashboard/<callback#1>" || callback.Kind != "callback" {
		t.Fatalf("unexpected callback metric: %#v", callback)
	}
	if callback.Complexity != 2 {
		t.Errorf("callback complexity = %d, want 2", callback.Complexity)
	}
}

func TestAnalyzeMemoizedReactComponentUsesVariableName(t *testing.T) {
	path := writeSource(t, "Memo.tsx", `
const MemoPanel = React.memo(({ ready }: { ready: boolean }) => {
  return ready && <section/>;
});
`)
	report, err := Analyze(path, Options{Language: "react", Threshold: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Functions) != 1 {
		t.Fatalf("got %d functions, want 1", len(report.Functions))
	}
	metric := report.Functions[0]
	if metric.Symbol != "MemoPanel" || metric.Kind != "react-component" {
		t.Fatalf("unexpected memo component metric: %#v", metric)
	}
	if metric.Complexity != 2 {
		t.Errorf("complexity = %d, want 2", metric.Complexity)
	}
}

func TestAnalyzeAnonymousDefaultReactExport(t *testing.T) {
	path := writeSource(t, "Page.tsx", `
export default function({ ready }: { ready: boolean }) {
  return ready ? <main/> : null;
}
`)
	report, err := Analyze(path, Options{Language: "react", Threshold: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Functions) != 1 {
		t.Fatalf("got %d functions, want 1", len(report.Functions))
	}
	metric := report.Functions[0]
	if metric.Symbol != "<default-export>" || metric.Kind != "react-component" {
		t.Fatalf("unexpected default component metric: %#v", metric)
	}
}

func TestAnalyzeModernJavaScriptBranches(t *testing.T) {
	path := writeSource(t, "modern.js", `
function modern(options = {}) {
  const value = options?.primary ?? options?.fallback;
  enabled &&= value;
  return value;
}
`)
	report, err := Analyze(path, Options{Threshold: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Functions) != 1 {
		t.Fatalf("got %d functions, want 1", len(report.Functions))
	}
	// Base + default parameter + two optional-chain segments + ?? + &&=.
	if report.Functions[0].Complexity != 6 {
		t.Errorf("complexity = %d, want 6", report.Functions[0].Complexity)
	}
}

func TestCompareUsesRegressionRatchet(t *testing.T) {
	baseline := &Report{
		SchemaVersion: SchemaVersion,
		Metric:        "cyclomatic-complexity",
		Threshold:     10,
		Summary:       Summary{Total: 9},
		Functions: []FunctionMetric{
			{ID: "a.ts::stable", File: "a.ts", Symbol: "stable", Complexity: 4},
			{ID: "a.ts::improved", File: "a.ts", Symbol: "improved", Complexity: 5},
		},
	}
	current := &Report{
		SchemaVersion: SchemaVersion,
		Metric:        "cyclomatic-complexity",
		Threshold:     10,
		Summary:       Summary{Total: 22},
		Functions: []FunctionMetric{
			{ID: "a.ts::stable", File: "a.ts", Symbol: "stable", Complexity: 6},
			{ID: "a.ts::improved", File: "a.ts", Symbol: "improved", Complexity: 3},
			{ID: "a.ts::small", File: "a.ts", Symbol: "small", Complexity: 2},
			{ID: "a.ts::large", File: "a.ts", Symbol: "large", Complexity: 11},
		},
	}
	comparison := Compare(current, baseline)
	if comparison.Regressions != 2 {
		t.Errorf("regressions = %d, want increased existing + new over threshold", comparison.Regressions)
	}
	if comparison.Increased != 1 || comparison.Decreased != 1 || comparison.Added != 2 {
		t.Fatalf("unexpected comparison: %#v", comparison)
	}
}
