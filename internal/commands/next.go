package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/recipes"
	"github.com/canadian-ai/girl/internal/shared"
	"github.com/canadian-ai/girl/internal/verification"
	"github.com/urfave/cli/v2"
)

// NextSchema identifies the `girl next` JSON contract for agents.
const NextSchema = "girl.next/v1"

// NextResult is a bounded, ranked work queue. Large repositories produce
// thousands of diagnostics; agents need the few worth fixing now, with enough
// context to act and the commands that prove the fix.
type NextResult struct {
	Schema    string      `json:"schema"`
	Target    string      `json:"target"`
	Scope     string      `json:"scope"`
	Summary   NextSummary `json:"summary"`
	Items     []NextItem  `json:"items"`
	Remaining int         `json:"remaining"`
	Verify    []string    `json:"verify"`
	Then      []string    `json:"then"`
}

type NextSummary struct {
	Files       int `json:"files"`
	Diagnostics int `json:"diagnostics"`
	Locations   int `json:"locations"`
	High        int `json:"high"`
	Medium      int `json:"medium"`
	Low         int `json:"low"`
}

// NextItem is one location to change. Diagnostics on the same symbol are
// merged so an agent fixes a function once instead of once per rule.
type NextItem struct {
	Rank       int         `json:"rank"`
	ID         string      `json:"id"`
	Severity   ir.Severity `json:"severity"`
	File       string      `json:"file"`
	Line       int         `json:"line,omitempty"`
	EndLine    int         `json:"endLine,omitempty"`
	Symbol     string      `json:"symbol,omitempty"`
	Codes      []string    `json:"codes"`
	Problems   []string    `json:"problems"`
	Action     string      `json:"action"`
	Recipe     string      `json:"recipe,omitempty"`
	Suggestion string      `json:"suggestion,omitempty"`
	Excerpt    string      `json:"excerpt,omitempty"`
}

func NextCommand() *cli.Command {
	return &cli.Command{
		Name:      "next",
		Usage:     "Show the highest-value fixes as a bounded, agent-ready work queue",
		ArgsUsage: "[path]",
		Description: "Ranks findings, merges them per function/component, and returns the top items with a code\n" +
			"excerpt, a concrete action, and the repository's own verification commands. Built for agent\n" +
			"loops: `girl next -o json` -> fix -> `girl check --changed` -> repeat.",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "limit", Aliases: []string{"n"}, Value: 5, Usage: "Maximum work items to return"},
			&cli.BoolFlag{Name: "changed", Usage: "Only consider files changed against HEAD (or --base)"},
			&cli.StringFlag{Name: "base", Usage: "Git base ref for --changed"},
			&cli.StringFlag{Name: "min-severity", Value: "low", Usage: "Drop items below this severity: low, medium, high"},
			&cli.IntFlag{Name: "excerpt-lines", Value: 12, Usage: "Source lines to include per item (0 disables)"},
			&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "text", Usage: "Output format: text, json, markdown"},
		},
		Action: func(c *cli.Context) error {
			path := commandPath(c)
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("cannot access %s: %w", path, err)
			}
			base := stringFlag(c, "base")
			result, err := buildNext(path, nextOptions{
				Limit:        intFlag(c, "limit", "n"),
				Changed:      boolFlag(c, "changed") || base != "",
				Base:         base,
				MinSeverity:  ir.Severity(strings.ToLower(stringFlag(c, "min-severity"))),
				ExcerptLines: intFlag(c, "excerpt-lines"),
			})
			if err != nil {
				return err
			}
			switch stringFlag(c, "output", "o") {
			case "json":
				printJSON(result)
			case "markdown", "md":
				fmt.Print(renderNextMarkdown(result))
			default:
				fmt.Print(renderNextText(result))
			}
			return nil
		},
	}
}

type nextOptions struct {
	Limit        int
	Changed      bool
	Base         string
	MinSeverity  ir.Severity
	ExcerptLines int
}

func buildNext(path string, opts nextOptions) (*NextResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 5
	}
	scope := "all"
	var changed []string
	if opts.Changed {
		scope = "changed"
		files, err := gitChangedFiles(path, opts.Base)
		if err != nil {
			return nil, err
		}
		changed = files
	}

	analysis, err := analyzeScope(path, changed, opts.Changed)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Normalize paths first so actions, IDs, and files derived from a
	// diagnostic never carry machine-specific paths.
	diags := make([]ir.Diagnostic, len(analysis.Diagnostics))
	for i, d := range analysis.Diagnostics {
		d.File = shared.DisplayFile(path, d.File)
		diags[i] = d
	}
	items := groupNextItems(diags, severityRank(opts.MinSeverity))
	result := &NextResult{
		Schema:  NextSchema,
		Target:  shared.DisplayTarget(path),
		Scope:   scope,
		Summary: summarizeNext(analysis, len(items)),
		Items:   []NextItem{},
		Verify:  existingVerifyCommands(path),
	}
	for i, item := range items {
		if i >= opts.Limit {
			result.Remaining = len(items) - opts.Limit
			break
		}
		item.Rank = i + 1
		item.ID = nextItemID(item.File, item.Symbol, item.Line)
		if opts.ExcerptLines > 0 {
			item.Excerpt = sourceExcerpt(path, item, opts.ExcerptLines)
		}
		result.Items = append(result.Items, item)
	}
	result.Then = []string{"girl check --changed", "girl next --changed"}
	return result, nil
}

// analyzeScope analyzes a single file directly, or a directory the same way
// `girl check` does so both commands agree on findings in mixed repos.
func analyzeScope(path string, changed []string, changedOnly bool) (*ir.AnalyzerResult, error) {
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return analyzePath(path, resolveLang(path, "auto"))
	}
	return analyzeCheckScope(path, changed, changedOnly)
}

func groupNextItems(diags []ir.Diagnostic, minRank int) []NextItem {
	type group struct {
		item  NextItem
		score int
	}
	groups := map[string]*group{}
	var order []string
	for _, d := range diags {
		if severityRank(d.Severity) < minRank {
			continue
		}
		key := d.File + "::" + d.Symbol
		if d.Symbol == "" {
			key = fmt.Sprintf("%s::L%d", d.File, d.Line)
		}
		g, ok := groups[key]
		if !ok {
			step := recipes.StepForDiagnostic(d)
			g = &group{item: NextItem{
				Severity:   d.Severity,
				File:       d.File,
				Line:       d.Line,
				EndLine:    diagnosticEnd(d),
				Symbol:     d.Symbol,
				Action:     firstNonEmpty(step.Action, d.Suggestion, d.Message),
				Recipe:     step.Recipe,
				Suggestion: d.Suggestion,
			}}
			groups[key] = g
			order = append(order, key)
		} else if severityRank(d.Severity) > severityRank(g.item.Severity) {
			g.item.Severity = d.Severity
		}
		if !containsString(g.item.Codes, d.Code) {
			g.item.Codes = append(g.item.Codes, d.Code)
		}
		g.item.Problems = append(g.item.Problems, d.Message)
		g.score += severityWeight(d.Severity)
		if d.Line > 0 && (g.item.Line == 0 || d.Line < g.item.Line) {
			g.item.Line = d.Line
		}
		if end := diagnosticEnd(d); end > g.item.EndLine {
			g.item.EndLine = end
		}
	}

	all := make([]*group, 0, len(order))
	for _, k := range order {
		all = append(all, groups[k])
	}
	// Highest severity first, then locations hit by the most rules, then the
	// largest span (more code to simplify), then a stable file/line order.
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if ra, rb := severityRank(a.item.Severity), severityRank(b.item.Severity); ra != rb {
			return ra > rb
		}
		if a.score != b.score {
			return a.score > b.score
		}
		if sa, sb := a.item.EndLine-a.item.Line, b.item.EndLine-b.item.Line; sa != sb {
			return sa > sb
		}
		if a.item.File != b.item.File {
			return a.item.File < b.item.File
		}
		return a.item.Line < b.item.Line
	})
	items := make([]NextItem, len(all))
	for i, g := range all {
		items[i] = g.item
	}
	return items
}

func summarizeNext(analysis *ir.AnalyzerResult, locations int) NextSummary {
	s := NextSummary{Files: len(analysis.Files), Diagnostics: len(analysis.Diagnostics), Locations: locations}
	for _, d := range analysis.Diagnostics {
		switch d.Severity {
		case ir.SeverityHigh:
			s.High++
		case ir.SeverityMedium:
			s.Medium++
		default:
			s.Low++
		}
	}
	return s
}

// nextItemID is a stable handle for a location, e.g. "src/app.ts#render".
func nextItemID(file, symbol string, line int) string {
	if symbol == "" {
		symbol = fmt.Sprintf("L%d", line)
	}
	return filepath.ToSlash(file) + "#" + symbol
}

func diagnosticEnd(d ir.Diagnostic) int {
	if d.Span != nil && d.Span.EndLine > 0 {
		return d.Span.EndLine
	}
	if d.EndLine > 0 {
		return d.EndLine
	}
	return d.Line
}

func severityRank(s ir.Severity) int {
	switch s {
	case ir.SeverityHigh:
		return 3
	case ir.SeverityMedium:
		return 2
	case ir.SeverityLow:
		return 1
	default:
		return 0
	}
}

func severityWeight(s ir.Severity) int {
	return map[ir.Severity]int{ir.SeverityHigh: 9, ir.SeverityMedium: 3, ir.SeverityLow: 1}[s]
}

func existingVerifyCommands(path string) []string {
	detected, err := verification.Detect(path)
	if err != nil {
		return []string{}
	}
	cmds := []string{}
	for _, c := range detected.Commands {
		if c.Exists && c.Required {
			cmds = append(cmds, c.Command)
		}
	}
	return cmds
}

// sourceExcerpt returns up to maxLines of the item's span, numbered so an
// agent can anchor edits without a separate file read.
func sourceExcerpt(target string, item NextItem, maxLines int) string {
	if item.Line <= 0 {
		return ""
	}
	f, err := os.Open(resolveItemFile(target, item.File))
	if err != nil {
		return ""
	}
	defer f.Close()

	end := item.EndLine
	if end < item.Line || end-item.Line+1 > maxLines {
		end = item.Line + maxLines - 1
	}
	var b strings.Builder
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for n := 1; scanner.Scan() && n <= end; n++ {
		if n >= item.Line {
			fmt.Fprintf(&b, "%5d  %s\n", n, scanner.Text())
		}
	}
	if item.EndLine > end {
		fmt.Fprintf(&b, "  ...  (%d more lines to line %d)\n", item.EndLine-end, item.EndLine)
	}
	return b.String()
}

func resolveItemFile(target, file string) string {
	if _, err := os.Stat(file); err == nil {
		return file
	}
	root := target
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		root = filepath.Dir(target)
	}
	return filepath.Join(root, filepath.FromSlash(file))
}

func renderNextText(r *NextResult) string {
	var b strings.Builder
	if len(r.Items) == 0 {
		fmt.Fprintf(&b, "Nothing to fix in %s (%s scope, %d files scanned).\n", r.Target, r.Scope, r.Summary.Files)
		return b.String()
	}
	fmt.Fprintf(&b, "GIRL next: %d of %d locations (%d diagnostics, %d files, %s scope)\n\n",
		len(r.Items), r.Summary.Locations, r.Summary.Diagnostics, r.Summary.Files, r.Scope)
	for _, it := range r.Items {
		loc := it.File
		if it.Line > 0 {
			loc = fmt.Sprintf("%s:%d", it.File, it.Line)
		}
		fmt.Fprintf(&b, "%d. [%s] %s", it.Rank, strings.ToUpper(string(it.Severity)), loc)
		if it.Symbol != "" {
			fmt.Fprintf(&b, " (%s)", it.Symbol)
		}
		fmt.Fprintf(&b, "\n   do: %s\n", it.Action)
		for _, p := range it.Problems {
			fmt.Fprintf(&b, "   why: %s\n", p)
		}
		b.WriteString("\n")
	}
	if r.Remaining > 0 {
		fmt.Fprintf(&b, "%d more location(s); raise --limit or use --changed to focus.\n", r.Remaining)
	}
	if len(r.Verify) > 0 {
		fmt.Fprintf(&b, "Verify with: %s\n", strings.Join(r.Verify, " && "))
	}
	return b.String()
}

func renderNextMarkdown(r *NextResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# GIRL next: %s\n\n", r.Target)
	fmt.Fprintf(&b, "%d of %d locations · %d diagnostics · %d files · %s scope\n\n",
		len(r.Items), r.Summary.Locations, r.Summary.Diagnostics, r.Summary.Files, r.Scope)
	for _, it := range r.Items {
		fmt.Fprintf(&b, "## %d. `%s:%d` %s\n\n", it.Rank, it.File, it.Line, it.Symbol)
		fmt.Fprintf(&b, "- **Severity:** %s\n- **Codes:** `%s`\n- **Action:** %s\n", it.Severity, strings.Join(it.Codes, "`, `"), it.Action)
		for _, p := range it.Problems {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		if it.Excerpt != "" {
			fmt.Fprintf(&b, "\n```\n%s```\n", it.Excerpt)
		}
		b.WriteString("\n")
	}
	if len(r.Verify) > 0 {
		fmt.Fprintf(&b, "**Verify:** `%s`\n", strings.Join(r.Verify, "` · `"))
	}
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
