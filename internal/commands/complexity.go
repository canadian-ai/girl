package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/canadian-ai/girl/internal/complexity"
	"github.com/urfave/cli/v2"
)

func ComplexityCommand() *cli.Command {
	return &cli.Command{
		Name:      "complexity",
		Usage:     "Measure and track cyclomatic complexity in TypeScript, JavaScript, and React",
		ArgsUsage: "<path>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "lang", Usage: "Source filter: auto, ts, tsx/react, js, jsx", Value: "auto"},
			&cli.IntFlag{Name: "max", Usage: "Per-function complexity threshold", Value: 10},
			&cli.IntFlag{Name: "top", Usage: "Functions shown in text/markdown output (0 shows all)", Value: 20},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text (default), json, markdown", Value: "text"},
			&cli.StringFlag{Name: "baseline", Usage: "Compare against a prior GIRL complexity JSON report"},
			&cli.StringFlag{Name: "write-baseline", Usage: "Write the current report as a reusable baseline"},
			&cli.StringFlag{Name: "fail-on", Usage: "Policy: never, threshold, or regression", Value: "never"},
			&cli.StringSliceFlag{Name: "exclude", Usage: "Additional directory names to exclude"},
		},
		Action: runComplexity,
	}
}

func runComplexity(c *cli.Context) error {
	path := commandPath(c)
	language := stringFlag(c, "lang")
	threshold := intFlag(c, "max")
	top := intFlag(c, "top")
	report, err := complexity.Analyze(path, complexity.Options{
		Language: language, Threshold: threshold, Exclude: stringSliceFlag(c, "exclude"),
	})
	if err != nil {
		return fmt.Errorf("complexity analysis failed: %w", err)
	}

	if baselinePath := stringFlag(c, "baseline"); baselinePath != "" {
		baseline, err := readComplexityReport(baselinePath)
		if err != nil {
			return fmt.Errorf("read complexity baseline: %w", err)
		}
		complexity.Compare(report, baseline)
	}

	switch stringFlag(c, "output", "o") {
	case "json":
		printJSON(report)
	case "markdown":
		fmt.Print(markdownComplexity(report, top))
	case "text":
		fmt.Print(textComplexity(report, top))
	default:
		return fmt.Errorf("unsupported output format %q", c.String("output"))
	}

	if baselinePath := stringFlag(c, "write-baseline"); baselinePath != "" {
		if err := writeComplexityReport(baselinePath, report); err != nil {
			return fmt.Errorf("write complexity baseline: %w", err)
		}
	}

	failed, err := complexityPolicyFailed(stringFlag(c, "fail-on"), report)
	if err != nil {
		return err
	}
	if failed {
		return cli.Exit("cyclomatic complexity policy failed", 2)
	}
	return nil
}

func readComplexityReport(path string) (*complexity.Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var report complexity.Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	if report.Metric != "cyclomatic-complexity" || report.SchemaVersion != complexity.SchemaVersion {
		return nil, fmt.Errorf("unsupported baseline schema %q for metric %q", report.SchemaVersion, report.Metric)
	}
	return &report, nil
}

func writeComplexityReport(path string, report *complexity.Report) error {
	baseline := *report
	baseline.Comparison = nil
	for i := range baseline.Functions {
		baseline.Functions[i].BaselineComplexity = 0
		baseline.Functions[i].Delta = 0
		baseline.Functions[i].Change = ""
	}
	data, err := json.MarshalIndent(&baseline, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func complexityPolicyFailed(policy string, report *complexity.Report) (bool, error) {
	switch strings.ToLower(policy) {
	case "never", "none", "":
		return false, nil
	case "threshold":
		return report.Summary.OverThreshold > 0, nil
	case "regression", "increase":
		if report.Comparison == nil {
			return false, fmt.Errorf("--fail-on regression requires --baseline")
		}
		return report.Comparison.Regressions > 0, nil
	default:
		return false, fmt.Errorf("unsupported --fail-on policy %q", policy)
	}
}

func sortedComplexityMetrics(report *complexity.Report) []complexity.FunctionMetric {
	metrics := append([]complexity.FunctionMetric(nil), report.Functions...)
	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].Complexity != metrics[j].Complexity {
			return metrics[i].Complexity > metrics[j].Complexity
		}
		return metrics[i].ID < metrics[j].ID
	})
	return metrics
}

func limitedMetrics(report *complexity.Report, top int) []complexity.FunctionMetric {
	metrics := sortedComplexityMetrics(report)
	if top > 0 && len(metrics) > top {
		return metrics[:top]
	}
	return metrics
}

func textComplexity(report *complexity.Report, top int) string {
	var out strings.Builder
	fmt.Fprintf(&out, "Cyclomatic complexity — TypeScript / JavaScript / React\n\n")
	fmt.Fprintf(&out, "%d functions across %d files | average %.2f | maximum %d | over %d: %d\n",
		report.Summary.Functions, report.Summary.Files, report.Summary.Average,
		report.Summary.Maximum, report.Threshold, report.Summary.OverThreshold)
	if report.Comparison != nil {
		fmt.Fprintf(&out, "Baseline: %d increased, %d decreased, %d added, %d removed | regressions: %d\n",
			report.Comparison.Increased, report.Comparison.Decreased, report.Comparison.Added,
			report.Comparison.Removed, report.Comparison.Regressions)
	}
	if len(report.ParseErrors) > 0 {
		fmt.Fprintf(&out, "Warning: %d file(s) could not be parsed.\n", len(report.ParseErrors))
	}
	out.WriteString("\nCC   Delta  Kind             Function                              Location\n")
	out.WriteString("---- ------ ---------------- ------------------------------------- ------------------------------\n")
	for _, metric := range limitedMetrics(report, top) {
		delta := "-"
		if metric.Change == "increased" || metric.Change == "decreased" {
			delta = fmt.Sprintf("%+d", metric.Delta)
		} else if metric.Change == "added" {
			delta = "new"
		}
		fmt.Fprintf(&out, "%-4d %-6s %-16s %-37s %s:%d\n",
			metric.Complexity, delta, metric.Kind, truncate(metric.Symbol, 37), metric.File, metric.StartLine)
	}
	return out.String()
}

func markdownComplexity(report *complexity.Report, top int) string {
	var out strings.Builder
	out.WriteString("# GIRL Cyclomatic Complexity Report\n\n")
	fmt.Fprintf(&out, "- **Functions:** %d\n- **Files:** %d\n- **Average:** %.2f\n- **Maximum:** %d\n- **Over threshold (%d):** %d\n",
		report.Summary.Functions, report.Summary.Files, report.Summary.Average,
		report.Summary.Maximum, report.Threshold, report.Summary.OverThreshold)
	if report.Comparison != nil {
		fmt.Fprintf(&out, "- **Regressions:** %d\n", report.Comparison.Regressions)
	}
	out.WriteString("\n| CC | Delta | Kind | Function | Location |\n|---:|---:|---|---|---|\n")
	for _, metric := range limitedMetrics(report, top) {
		delta := "—"
		if metric.Change == "increased" || metric.Change == "decreased" {
			delta = fmt.Sprintf("%+d", metric.Delta)
		} else if metric.Change == "added" {
			delta = "new"
		}
		fmt.Fprintf(&out, "| %d | %s | %s | `%s` | `%s:%d` |\n",
			metric.Complexity, delta, metric.Kind, metric.Symbol, metric.File, metric.StartLine)
	}
	return out.String()
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	if max <= 1 {
		return value[:max]
	}
	return value[:max-1] + "…"
}
