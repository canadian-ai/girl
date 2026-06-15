package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/parsertsx"
	"github.com/canadian-ai/girl/internal/visitor"
)

type Config struct {
	MaxComponentLines    int
	MinRepeatedJSX       int
	MaxHooksPerComponent int
	MaxStateVars         int
	MaxEffects           int
	MaxConditionals      int
	MaxLoops             int
	ExcludeDirs          []string
}

func DefaultConfig() *Config {
	return &Config{
		MaxComponentLines:    200,
		MinRepeatedJSX:       3,
		MaxHooksPerComponent: 5,
		MaxStateVars:         4,
		MaxEffects:           2,
		MaxConditionals:      5,
		MaxLoops:             3,
		ExcludeDirs:          []string{},
	}
}

type Analyzer struct {
	config *Config
	parser *parsertsx.Parser
}

func NewAnalyzer(config *Config) *Analyzer {
	if config == nil {
		config = DefaultConfig()
	}
	return &Analyzer{
		config: config,
		parser: parsertsx.New(),
	}
}

func (a *Analyzer) Analyze(path string) (*ir.AnalyzerResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path %s: %w", path, err)
	}

	var files []*ir.FileIR
	if info.IsDir() {
		files, err = a.parser.ParseDir(path, a.config.ExcludeDirs)
		if err != nil {
			return nil, err
		}
	} else {
		file, err := a.parser.ParseFile(path)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	pipeline := visitor.NewPipeline(visitor.NewResponsibilityVisitor())
	for _, f := range files {
		if err := pipeline.ProcessFile(f); err != nil {
			return nil, err
		}
	}

	diags := a.detectSmells(files)
	if diags == nil {
		diags = []ir.Diagnostic{}
	}

	return &ir.AnalyzerResult{
		Files:       files,
		Diagnostics: diags,
	}, nil
}

func (a *Analyzer) detectSmells(files []*ir.FileIR) []ir.Diagnostic {
	diags := []ir.Diagnostic{}

	for _, f := range files {
		diags = append(diags, a.detectLargeComponents(f)...)
		diags = append(diags, a.detectRepeatedJSX(f)...)
		diags = append(diags, a.detectTooManyHooks(f)...)
		diags = append(diags, a.detectTooManyStateVars(f)...)
		diags = append(diags, a.detectTooManyEffects(f)...)
		diags = append(diags, a.detectComplexConditionals(f)...)
		diags = append(diags, a.detectMixedResponsibilities(f)...)
		diags = append(diags, a.detectHardcodedData(f)...)
		diags = append(diags, a.detectMissingPropTypes(f)...)
	}

	sort.SliceStable(diags, func(i, j int) bool {
		if diags[i].Severity != diags[j].Severity {
			severityOrder := map[ir.Severity]int{
				ir.SeverityHigh:   0,
				ir.SeverityMedium: 1,
				ir.SeverityLow:    2,
			}
			return severityOrder[diags[i].Severity] < severityOrder[diags[j].Severity]
		}
		if diags[i].Code != diags[j].Code {
			return diags[i].Code < diags[j].Code
		}
		return diags[i].Message < diags[j].Message
	})

	return diags
}

func (a *Analyzer) detectLargeComponents(f *ir.FileIR) []ir.Diagnostic {
	var diags []ir.Diagnostic
	for _, c := range f.Components {
		if c.Lines > a.config.MaxComponentLines {
			severity := ir.SeverityLow
			if c.Lines > a.config.MaxComponentLines*2 {
				severity = ir.SeverityHigh
			} else if c.Lines > a.config.MaxComponentLines*3/2 {
				severity = ir.SeverityMedium
			}
			diags = append(diags, ir.Diagnostic{
				Code:       "react.large-component",
				Severity:   severity,
				Message:    fmt.Sprintf("Component %q is %d lines (limit: %d)", c.Name, c.Lines, a.config.MaxComponentLines),
				File:       relPath(f.Path),
				Line:       c.StartLine,
				Component:  c.Name,
				Suggestion: "Split this component into smaller focused components or extract logic into custom hooks.",
				Kind:       ir.NodeKindComponent,
				Symbol:     c.Name,
				EndLine:    c.EndLine,
				Span:       &ir.Span{StartLine: c.StartLine, EndLine: c.EndLine},
			})
		}
	}
	return diags
}

func (a *Analyzer) detectRepeatedJSX(f *ir.FileIR) []ir.Diagnostic {
	var diags []ir.Diagnostic
	for _, c := range f.Components {
		blockCount := make(map[string]int)
		for _, b := range c.JSXBlocks {
			blockCount[b.Element]++
		}
		for elem, count := range blockCount {
			if count >= a.config.MinRepeatedJSX {
				diags = append(diags, ir.Diagnostic{
					Code:       "react.repeated-jsx",
					Severity:   ir.SeverityLow,
					Message:    fmt.Sprintf("JSX element <%s> repeated %d times in component %q", elem, count, c.Name),
					File:       relPath(f.Path),
					Component:  c.Name,
					Suggestion: fmt.Sprintf("Extract repeated <%s> into a reusable sub-component.", elem),
					Kind:       ir.NodeKindComponent,
					Symbol:     c.Name,
				})
			}
		}
	}
	return diags
}

func (a *Analyzer) detectTooManyHooks(f *ir.FileIR) []ir.Diagnostic {
	var diags []ir.Diagnostic
	for _, c := range f.Components {
		if len(c.Hooks) > a.config.MaxHooksPerComponent {
			diags = append(diags, ir.Diagnostic{
				Code:       "react.too-many-hooks",
				Severity:   ir.SeverityLow,
				Message:    fmt.Sprintf("Component %q has %d hooks (limit: %d)", c.Name, len(c.Hooks), a.config.MaxHooksPerComponent),
				File:       relPath(f.Path),
				Component:  c.Name,
				Suggestion: "Extract related hooks into custom hooks using the Extract Custom Hook recipe.",
				Kind:       ir.NodeKindComponent,
				Symbol:     c.Name,
			})
		}
	}
	return diags
}

func (a *Analyzer) detectTooManyStateVars(f *ir.FileIR) []ir.Diagnostic {
	var diags []ir.Diagnostic
	for _, c := range f.Components {
		if len(c.StateVars) > a.config.MaxStateVars {
			diags = append(diags, ir.Diagnostic{
				Code:       "react.too-many-state-vars",
				Severity:   ir.SeverityMedium,
				Message:    fmt.Sprintf("Component %q has %d state variables (limit: %d)", c.Name, len(c.StateVars), a.config.MaxStateVars),
				File:       relPath(f.Path),
				Component:  c.Name,
				Suggestion: "Consider using useReducer or extracting related state into a custom hook.",
				Kind:       ir.NodeKindComponent,
				Symbol:     c.Name,
			})
		}
	}
	return diags
}

func (a *Analyzer) detectTooManyEffects(f *ir.FileIR) []ir.Diagnostic {
	var diags []ir.Diagnostic
	for _, c := range f.Components {
		if len(c.Effects) > a.config.MaxEffects {
			diags = append(diags, ir.Diagnostic{
				Code:       "react.too-many-effects",
				Severity:   ir.SeverityMedium,
				Message:    fmt.Sprintf("Component %q has %d useEffect calls (limit: %d)", c.Name, len(c.Effects), a.config.MaxEffects),
				File:       relPath(f.Path),
				Component:  c.Name,
				Suggestion: "Consider consolidating effects or moving side-effect logic into custom hooks.",
				Kind:       ir.NodeKindComponent,
				Symbol:     c.Name,
			})
		}
	}
	return diags
}

func (a *Analyzer) detectComplexConditionals(f *ir.FileIR) []ir.Diagnostic {
	var diags []ir.Diagnostic
	for _, c := range f.Components {
		if c.ConditionalCount > a.config.MaxConditionals {
			diags = append(diags, ir.Diagnostic{
				Code:       "react.complex-conditionals",
				Severity:   ir.SeverityLow,
				Message:    fmt.Sprintf("Component %q has %d conditional expressions (limit: %d)", c.Name, c.ConditionalCount, a.config.MaxConditionals),
				File:       relPath(f.Path),
				Component:  c.Name,
				Suggestion: "Extract conditional rendering into smaller components or use early returns.",
				Kind:       ir.NodeKindComponent,
				Symbol:     c.Name,
			})
		}
	}
	return diags
}

func (a *Analyzer) detectMixedResponsibilities(f *ir.FileIR) []ir.Diagnostic {
	var diags []ir.Diagnostic
	for _, c := range f.Components {
		respCount := 0
		if len(c.StateVars) > 2 {
			respCount++
		}
		if len(c.Effects) > 1 {
			respCount++
		}
		if c.HasKeyDown {
			respCount++
		}
		if c.HasAnalytics {
			respCount++
		}
		if len(c.Hooks) > 3 {
			respCount++
		}
		if respCount >= 3 {
			diags = append(diags, ir.Diagnostic{
				Code:       "react.mixed-responsibilities",
				Severity:   ir.SeverityMedium,
				Message:    fmt.Sprintf("Component %q has ~%d detected responsibilities", c.Name, respCount),
				File:       relPath(f.Path),
				Component:  c.Name,
				Suggestion: "Separate concerns by splitting this component. Responsibilities: state management, side effects, event handling, analytics.",
				Kind:       ir.NodeKindComponent,
				Symbol:     c.Name,
			})
		}
	}
	return diags
}

var hardcodedArrayRe = regexp.MustCompile(`const\s+\w+\s*[=:]\s*\[`)

func (a *Analyzer) detectHardcodedData(f *ir.FileIR) []ir.Diagnostic {
	var diags []ir.Diagnostic
	for _, c := range f.Components {
		body := getComponentBody(&c)
		matches := hardcodedArrayRe.FindAllString(body, -1)
		if len(matches) > 0 {
			diags = append(diags, ir.Diagnostic{
				Code:       "react.hardcoded-data",
				Severity:   ir.SeverityLow,
				Message:    fmt.Sprintf("Component %q contains %d hardcoded arrays", c.Name, len(matches)),
				File:       relPath(f.Path),
				Component:  c.Name,
				Suggestion: "Move hardcoded data outside the component or into a separate data file.",
				Kind:       ir.NodeKindComponent,
				Symbol:     c.Name,
			})
		}
	}
	return diags
}

func (a *Analyzer) detectMissingPropTypes(f *ir.FileIR) []ir.Diagnostic {
	var diags []ir.Diagnostic
	for _, c := range f.Components {
		if len(c.Props) == 0 {
			continue
		}
		typedCount := 0
		for _, p := range c.Props {
			if p.Type != "" {
				typedCount++
			}
		}
		if len(c.JSXBlocks) > 0 && typedCount == 0 {
			diags = append(diags, ir.Diagnostic{
				Code:       "react.missing-prop-types",
				Severity:   ir.SeverityLow,
				Message:    fmt.Sprintf("Component %q has %d props but no TypeScript types found", c.Name, len(c.Props)),
				File:       relPath(f.Path),
				Component:  c.Name,
				Suggestion: "Add TypeScript interfaces for props to improve type safety.",
				Kind:       ir.NodeKindComponent,
				Symbol:     c.Name,
			})
		}
	}
	return diags
}

func getComponentBody(c *ir.ComponentIR) string {
	return fmt.Sprintf("component %s (%d lines)", c.Name, c.Lines)
}

func relPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}
	if !strings.HasPrefix(rel, ".") && len(rel) > 0 && rel[0] != '/' {
		first := rune(rel[0])
		if unicode.IsUpper(first) || first == '~' {
			return rel
		}
		if strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, "..\\") {
			return rel
		}
	}
	return rel
}


