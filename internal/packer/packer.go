package packer

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/canadian-ai/girl/internal/ir"
)

type Packer struct {
	MaxTokens int
}

func NewPacker(maxTokens int) *Packer {
	if maxTokens <= 0 {
		maxTokens = 12000
	}
	return &Packer{MaxTokens: maxTokens}
}

type PackRequest struct {
	Goal        string
	Files       []*ir.FileIR
	Diagnostics []ir.Diagnostic
	Steps       []ir.GrpStep
	TokenBudget int
}

func (p *Packer) Pack(req PackRequest) (*ir.ContextPack, error) {
	budget := p.MaxTokens
	if req.TokenBudget > 0 {
		budget = req.TokenBudget
	}

	diagCounts := map[string]int{}
	codeCounts := map[string]int{}
	fileDiagCounts := map[string]int{}
	for _, d := range req.Diagnostics {
		diagCounts[string(d.Severity)]++
		codeCounts[d.Code]++
		fileDiagCounts[d.File]++
	}

	type codeFreq struct {
		code string
		freq int
	}
	var sorted []codeFreq
	for code, freq := range codeCounts {
		sorted = append(sorted, codeFreq{code, freq})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].freq > sorted[j].freq
	})
	var topCodes []string
	for i, cf := range sorted {
		if i >= 5 {
			break
		}
		topCodes = append(topCodes, cf.code)
	}

	pack := &ir.ContextPack{
		Goal:             req.Goal,
		TokenBudget:      budget,
		Files:            []string{},
		Summaries:        []ir.FileSummary{},
		Diagnostics:      req.Diagnostics,
		Steps:            req.Steps,
		Risks:            []string{},
		Verification:     []string{},
		DiagnosticCounts: diagCounts,
		TopCodes:         topCodes,
	}

	for _, f := range req.Files {
		rel, _ := filepath.Rel(".", f.Path)
		pack.Files = append(pack.Files, rel)

		summary := ir.FileSummary{
			Path:           rel,
			Lines:          f.Lines,
			ComponentCount: len(f.Components),
			HookCount:      len(f.Hooks),
			Summary:        summarizeFile(f, fileDiagCounts[f.Path]),
		}
		pack.Summaries = append(pack.Summaries, summary)
	}

	remaining := budget
	for _, f := range req.Files {
		if remaining <= 0 {
			break
		}
		rel, _ := filepath.Rel(".", f.Path)
		for _, c := range f.Components {
			if remaining <= 0 {
				break
			}
			snippet := p.createSnippet(f.Path, rel, c, remaining)
			pack.SelectedSnippets = append(pack.SelectedSnippets, *snippet)
			remaining -= snippet.Tokens
		}
	}

	for _, d := range req.Diagnostics {
		if remaining <= 0 {
			break
		}
		if d.Component != "" || d.File == "" {
			continue
		}
		rel, _ := filepath.Rel(".", d.File)
		snippet := p.createDiagnosticSnippet(d.File, rel, d, remaining)
		pack.SelectedSnippets = append(pack.SelectedSnippets, *snippet)
		remaining -= snippet.Tokens
	}

	pack.TokenEstimate = budget - remaining

	riskSet := map[string]bool{}
	for _, d := range req.Diagnostics {
		if d.Severity == ir.SeverityHigh {
			riskSet[d.Message] = true
		}
	}
	for _, s := range req.Steps {
		if s.Risk == ir.SeverityHigh {
			riskSet[s.Action] = true
		}
	}
	for r := range riskSet {
		pack.Risks = append(pack.Risks, r)
	}

	pack.Verification = p.detectAvailableVerification()

	return pack, nil
}

func (p *Packer) createSnippet(readPath, relPath string, comp ir.ComponentIR, budget int) *ir.Snippet {
	lines, err := readLines(readPath)
	if err != nil {
		return &ir.Snippet{
			File:      relPath,
			StartLine: comp.StartLine,
			EndLine:   comp.EndLine,
			Content:   fmt.Sprintf("// %s (%d lines)", comp.Name, comp.Lines),
			Tokens:    10,
		}
	}

	if comp.StartLine < 1 || comp.EndLine > len(lines) || comp.StartLine > comp.EndLine {
		return &ir.Snippet{
			File:    relPath,
			Content: fmt.Sprintf("// %s (%d lines)", comp.Name, comp.Lines),
			Tokens:  10,
		}
	}

	snippetLines := lines[comp.StartLine-1 : comp.EndLine]
	content := strings.Join(snippetLines, "\n")

	tokens := len(content) / 3

	if tokens > budget {
		maxChars := budget * 3
		if maxChars < 200 {
			maxChars = 200
		}
		content = content[:maxChars] + "\n// ... truncated to fit budget"
		tokens = budget
	}

	return &ir.Snippet{
		File:      relPath,
		StartLine: comp.StartLine,
		EndLine:   comp.EndLine,
		Content:   content,
		Tokens:    tokens,
	}
}

func (p *Packer) createDiagnosticSnippet(readPath, relPath string, d ir.Diagnostic, budget int) *ir.Snippet {
	var startLine, endLine int
	if d.Span != nil {
		startLine = d.Span.StartLine
		endLine = d.Span.EndLine
	} else if d.EndLine > 0 {
		startLine = d.Line
		endLine = d.EndLine
	} else {
		startLine = d.Line
		endLine = d.Line + 10
	}

	lines, err := readLines(readPath)
	if err != nil {
		return &ir.Snippet{
			File:    relPath,
			Content: fmt.Sprintf("// diagnostic: %s", d.Message),
			Tokens:  5,
		}
	}

	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > endLine {
		startLine = endLine
	}

	snippetLines := lines[startLine-1 : endLine]
	content := strings.Join(snippetLines, "\n")

	hash := sha256.Sum256([]byte(content))
	content = fmt.Sprintf("// contentHash: %x\n%s", hash[:8], content)

	tokens := len(content) / 3
	if tokens > budget {
		maxChars := budget * 3
		if maxChars < 200 {
			maxChars = 200
		}
		content = content[:maxChars] + "\n// ... truncated"
		tokens = budget
	}

	return &ir.Snippet{
		File:      relPath,
		StartLine: startLine,
		EndLine:   endLine,
		Content:   content,
		Tokens:    tokens,
	}
}

func (p *Packer) detectAvailableVerification() []string {
	var cmds []string
	checks := []string{"package.json"}
	for _, c := range checks {
		if _, err := os.Stat(c); err == nil {
			cmds = append(cmds, "npm run typecheck", "npm run lint", "npm test", "npm run build")
			break
		}
	}

	goChecks := []string{"go.mod"}
	for _, c := range goChecks {
		if _, err := os.Stat(c); err == nil {
			cmds = append(cmds, "go build ./...", "go vet ./...", "go test ./...")
			break
		}
	}

	return cmds
}

func summarizeFile(f *ir.FileIR, diagCount int) string {
	base := fmt.Sprintf("%d lines", f.Lines)
	if diagCount > 0 {
		base = fmt.Sprintf("%d lines, %d diagnostics", f.Lines, diagCount)
	}
	if len(f.Components) == 0 {
		return fmt.Sprintf("Module with %s, %d hooks, %d imports", base, len(f.Hooks), len(f.Imports))
	}
	names := make([]string, len(f.Components))
	for i, c := range f.Components {
		names[i] = c.Name
	}
	return fmt.Sprintf("Exports %s (%s, %d components, %d hooks)",
		strings.Join(names, ", "), base, len(f.Components), len(f.Hooks))
}

func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	if content == "" {
		return []string{}, nil
	}
	return strings.Split(content, "\n"), nil
}
