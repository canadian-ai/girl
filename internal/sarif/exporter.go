package sarif

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/canadian-ai/girl/internal/ir"
)

type sarifLog struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []run  `json:"runs"`
}

type run struct {
	Tool       tool       `json:"tool"`
	Results    []result   `json:"results"`
	ColumnKind string     `json:"columnKind"`
	Properties properties `json:"properties"`
}

type tool struct {
	Driver driver `json:"driver"`
}

type driver struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	InformationURI string `json:"informationUri"`
	Rules          []rule `json:"rules"`
}

type rule struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	ShortDescription     description    `json:"shortDescription"`
	FullDescription      description    `json:"fullDescription"`
	DefaultConfiguration defaultConfig  `json:"defaultConfiguration"`
	Properties           ruleProperties `json:"properties"`
}

type description struct {
	Text string `json:"text"`
}

type defaultConfig struct {
	Level string `json:"level"`
}

type ruleProperties struct {
	Severity string `json:"severity"`
}

type result struct {
	RuleID    string      `json:"ruleId"`
	RuleIndex int         `json:"ruleIndex"`
	Message   description `json:"message"`
	Level     string      `json:"level"`
	Locations []location  `json:"locations"`
}

type location struct {
	PhysicalLocation physicalLocation `json:"physicalLocation"`
}

type physicalLocation struct {
	ArtifactLocation artifactLocation `json:"artifactLocation"`
	Region           region           `json:"region"`
}

type artifactLocation struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId"`
}

type region struct {
	StartLine   int `json:"startLine"`
	EndLine     int `json:"endLine"`
	StartColumn int `json:"startColumn,omitempty"`
	EndColumn   int `json:"endColumn,omitempty"`
}

type properties struct {
	DiagnosticCount int `json:"diagnosticCount"`
	ErrorCount      int `json:"errorCount"`
	WarningCount    int `json:"warningCount"`
	NoteCount       int `json:"noteCount"`
}

func firstSentence(s string) string {
	if idx := strings.Index(s, ". "); idx != -1 {
		return s[:idx+1]
	}
	if idx := strings.Index(s, ".\n"); idx != -1 {
		return s[:idx+1]
	}
	if strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}

func levelFromSeverity(sev ir.Severity) string {
	switch sev {
	case ir.SeverityHigh:
		return "error"
	case ir.SeverityMedium:
		return "warning"
	default:
		return "note"
	}
}

func getStartLine(d ir.Diagnostic) int {
	if d.Span != nil {
		return d.Span.StartLine
	}
	return d.Line
}

func getEndLine(d ir.Diagnostic) int {
	if d.Span != nil {
		return d.Span.EndLine
	}
	if d.EndLine > 0 {
		return d.EndLine
	}
	return d.Line
}

func getStartCol(d ir.Diagnostic) int {
	if d.Span != nil && d.Span.StartCol > 0 {
		return d.Span.StartCol
	}
	return 0
}

func getEndCol(d ir.Diagnostic) int {
	if d.Span != nil && d.Span.EndCol > 0 {
		return d.Span.EndCol
	}
	return 0
}

func includeCols(d ir.Diagnostic) bool {
	return getStartCol(d) > 0
}

func fullDescription(d ir.Diagnostic) string {
	if d.Suggestion != "" {
		return d.Message + "\n\nSuggestion: " + d.Suggestion
	}
	return d.Message
}

func ruleName(d ir.Diagnostic) string {
	if d.Symbol != "" {
		return d.Symbol
	}
	if d.Component != "" {
		return d.Component
	}
	return d.Code
}

func ExportDiagnostics(diags []ir.Diagnostic, toolName, toolVersion string) (string, error) {
	rules, ruleSeen, counts := buildRules(diags)
	results := buildResults(diags, ruleSeen)
	log := buildLog(toolName, toolVersion, rules, results, counts)

	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal sarif: %w", err)
	}

	return string(data), nil
}

func buildRules(diags []ir.Diagnostic) ([]rule, map[string]int, properties) {
	ruleSeen := make(map[string]int)
	rules := []rule{}
	counts := properties{DiagnosticCount: len(diags)}

	for _, d := range diags {
		addSeverityCount(&counts, d.Severity)
		if _, ok := ruleSeen[d.Code]; ok {
			continue
		}
		ruleSeen[d.Code] = len(rules)
		rules = append(rules, newRule(d))
	}

	return rules, ruleSeen, counts
}

func addSeverityCount(counts *properties, sev ir.Severity) {
	switch sev {
	case ir.SeverityHigh:
		counts.ErrorCount++
	case ir.SeverityMedium:
		counts.WarningCount++
	default:
		counts.NoteCount++
	}
}

func newRule(d ir.Diagnostic) rule {
	return rule{
		ID:   d.Code,
		Name: ruleName(d),
		ShortDescription: description{
			Text: firstSentence(d.Message),
		},
		FullDescription: description{
			Text: fullDescription(d),
		},
		DefaultConfiguration: defaultConfig{
			Level: levelFromSeverity(d.Severity),
		},
		Properties: ruleProperties{
			Severity: string(d.Severity),
		},
	}
}

func buildResults(diags []ir.Diagnostic, ruleSeen map[string]int) []result {
	results := make([]result, len(diags))
	for i, d := range diags {
		results[i] = newResult(d, ruleSeen[d.Code])
	}
	return results
}

func newResult(d ir.Diagnostic, ruleIndex int) result {
	return result{
		RuleID:    d.Code,
		RuleIndex: ruleIndex,
		Message:   description{Text: d.Message},
		Level:     levelFromSeverity(d.Severity),
		Locations: []location{{
			PhysicalLocation: physicalLocation{
				ArtifactLocation: artifactLocation{URI: d.File, URIBaseID: "%SRCROOT%"},
				Region:           diagnosticRegion(d),
			},
		}},
	}
}

func diagnosticRegion(d ir.Diagnostic) region {
	r := region{StartLine: getStartLine(d), EndLine: getEndLine(d)}
	if includeCols(d) {
		r.StartColumn = getStartCol(d)
		r.EndColumn = getEndCol(d)
	}
	return r
}

func buildLog(toolName, toolVersion string, rules []rule, results []result, counts properties) sarifLog {
	return sarifLog{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []run{
			{
				Tool: tool{
					Driver: driver{
						Name:           toolName,
						Version:        toolVersion,
						InformationURI: "https://github.com/canadian-ai/girl",
						Rules:          rules,
					},
				},
				Results:    results,
				ColumnKind: "utf16CodeUnits",
				Properties: counts,
			},
		},
	}
}
