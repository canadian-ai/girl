package packer

import (
	"fmt"
	"os"
	"path/filepath"
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

	pack := &ir.ContextPack{
		Goal:        req.Goal,
		TokenBudget: budget,
		Files:       []string{},
		Summaries:   []ir.FileSummary{},
		Diagnostics: req.Diagnostics,
		Steps:       req.Steps,
		Risks:       []string{},
		Verification: []string{},
	}

	for _, f := range req.Files {
		rel, _ := filepath.Rel(".", f.Path)
		pack.Files = append(pack.Files, rel)

		summary := ir.FileSummary{
			Path:           rel,
			Lines:          f.Lines,
			ComponentCount: len(f.Components),
			HookCount:      len(f.Hooks),
			Summary:        summarizeFile(f),
		}
		pack.Summaries = append(pack.Summaries, summary)
	}

	remaining := budget
	for _, f := range req.Files {
		if remaining <= 0 {
			break
		}
		for _, c := range f.Components {
			if remaining <= 0 {
				break
			}
			snippet := p.createSnippet(f.Path, c, remaining)
			pack.SelectedSnippets = append(pack.SelectedSnippets, *snippet)
			remaining -= snippet.Tokens
		}
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

func (p *Packer) createSnippet(path string, comp ir.ComponentIR, budget int) *ir.Snippet {
	lines, err := readLines(path)
	if err != nil {
		return &ir.Snippet{
			File:      path,
			StartLine: comp.StartLine,
			EndLine:   comp.EndLine,
			Content:   fmt.Sprintf("// %s (%d lines)", comp.Name, comp.Lines),
			Tokens:    10,
		}
	}

	if comp.StartLine < 1 || comp.EndLine > len(lines) || comp.StartLine > comp.EndLine {
		return &ir.Snippet{
			File: path,
			Content: fmt.Sprintf("// %s (%d lines)", comp.Name, comp.Lines),
			Tokens: 10,
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
		File:      path,
		StartLine: comp.StartLine,
		EndLine:   comp.EndLine,
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

func summarizeFile(f *ir.FileIR) string {
	if len(f.Components) == 0 {
		return fmt.Sprintf("Module with %d lines, %d hooks, %d imports", f.Lines, len(f.Hooks), len(f.Imports))
	}
	names := make([]string, len(f.Components))
	for i, c := range f.Components {
		names[i] = c.Name
	}
	return fmt.Sprintf("Exports %s (%d lines total, %d components, %d hooks)",
		strings.Join(names, ", "), f.Lines, len(f.Components), len(f.Hooks))
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
