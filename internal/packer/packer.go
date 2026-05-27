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
	PlanID      string
	PrivacyMode string
}

func (p *Packer) Pack(req PackRequest) (*ir.ContextPack, error) {
	budget := p.MaxTokens
	if req.TokenBudget > 0 {
		budget = req.TokenBudget
	}

	diagCounts, codeCounts, fileDiagCounts := countDiagnostics(req.Diagnostics)
	topCodes := topDiagnosticCodes(codeCounts, 5)
	pack := newContextPack(req, budget, diagCounts, topCodes)

	p.addFileSummaries(pack, req.Files, fileDiagCounts)

	tier := snippetTier(budget)
	var remaining int
	if tier == 0 {
		remaining = p.addDiagnosticSnippets(pack, req.Diagnostics, budget)
		if remaining > 0 {
			remaining = p.addComponentSnippets(pack, req.Files, remaining)
		}
	} else {
		remaining = p.addComponentSnippets(pack, req.Files, budget)
		remaining = p.addDiagnosticSnippets(pack, req.Diagnostics, remaining)
	}
	pack.TokenEstimate = budget - remaining

	pack.Risks = collectRisks(req.Diagnostics, req.Steps)
	pack.Verification = p.detectAvailableVerification()

	privacy := req.PrivacyMode
	if privacy == "" {
		privacy = "private"
	}
	p.applyPrivacy(privacy, pack, req.Files)

	return pack, nil
}

func snippetTier(budget int) int {
	switch {
	case budget <= 4000:
		return 0
	case budget <= 8000:
		return 1
	case budget <= 16000:
		return 2
	default:
		return 3
	}
}

func countDiagnostics(diags []ir.Diagnostic) (map[string]int, map[string]int, map[string]int) {
	diagCounts := map[string]int{}
	codeCounts := map[string]int{}
	fileDiagCounts := map[string]int{}
	for _, d := range diags {
		diagCounts[string(d.Severity)]++
		codeCounts[d.Code]++
		fileDiagCounts[d.File]++
	}
	return diagCounts, codeCounts, fileDiagCounts
}

func topDiagnosticCodes(codeCounts map[string]int, limit int) []string {
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
		if i >= limit {
			break
		}
		topCodes = append(topCodes, cf.code)
	}
	return topCodes
}

func newContextPack(req PackRequest, budget int, diagCounts map[string]int, topCodes []string) *ir.ContextPack {
	return &ir.ContextPack{
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
}

func (p *Packer) addFileSummaries(pack *ir.ContextPack, files []*ir.FileIR, fileDiagCounts map[string]int) {
	for _, f := range files {
		rel := packRelPath(f.Path)
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
}

func (p *Packer) addComponentSnippets(pack *ir.ContextPack, files []*ir.FileIR, budget int) int {
	remaining := budget
	for _, f := range files {
		if remaining <= 0 {
			break
		}
		rel := packRelPath(f.Path)
		for _, c := range f.Components {
			if remaining <= 0 {
				break
			}
			snippet := p.createSnippet(f.Path, rel, c, remaining)
			pack.SelectedSnippets = append(pack.SelectedSnippets, *snippet)
			remaining -= snippet.Tokens
		}
	}
	return remaining
}

func (p *Packer) addDiagnosticSnippets(pack *ir.ContextPack, diags []ir.Diagnostic, remaining int) int {
	for _, d := range diags {
		if remaining <= 0 {
			break
		}
		if d.Component != "" || d.File == "" {
			continue
		}
		rel := packRelPath(d.File)
		snippet := p.createDiagnosticSnippet(d.File, rel, d, remaining)
		pack.SelectedSnippets = append(pack.SelectedSnippets, *snippet)
		remaining -= snippet.Tokens
	}
	return remaining
}

func collectRisks(diags []ir.Diagnostic, steps []ir.GrpStep) []string {
	riskSet := map[string]bool{}
	for _, d := range diags {
		if d.Severity == ir.SeverityHigh {
			riskSet[d.Message] = true
		}
	}
	for _, s := range steps {
		if s.Risk == ir.SeverityHigh {
			riskSet[s.Action] = true
		}
	}
	risks := []string{}
	for r := range riskSet {
		risks = append(risks, r)
	}
	return risks
}

func packRelPath(path string) string {
	rel, err := filepath.Rel(".", path)
	if err != nil {
		return rel
	}
	return rel
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
		if statExists(c) {
			cmds = append(cmds, "npm run typecheck", "npm run lint", "npm test", "npm run build")
			break
		}
	}

	goChecks := []string{"go.mod"}
	for _, c := range goChecks {
		if statExists(c) {
			cmds = append(cmds, "go build ./...", "go vet ./...", "go test ./...")
			break
		}
	}

	return cmds
}

func statExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info != nil
}

func (p *Packer) applyPrivacy(mode string, pack *ir.ContextPack, files []*ir.FileIR) {
	switch mode {
	case "private":
		return
	case "redacted":
		homeDir, _ := os.UserHomeDir()
		for i, f := range pack.Files {
			pack.Files[i] = redactPath(f, homeDir)
		}
		for i, s := range pack.Summaries {
			pack.Summaries[i].Path = redactPath(s.Path, homeDir)
		}
		for i, sn := range pack.SelectedSnippets {
			pack.SelectedSnippets[i].File = redactPath(sn.File, homeDir)
		}
	case "public":
		for i, f := range pack.Files {
			pack.Files[i] = sanitizePublicPath(f)
		}
		for i, s := range pack.Summaries {
			pack.Summaries[i].Path = sanitizePublicPath(s.Path)
		}
		for i, sn := range pack.SelectedSnippets {
			pack.SelectedSnippets[i].File = sanitizePublicPath(sn.File)
		}
	}
}

func redactPath(path, homeDir string) string {
	if filepath.IsAbs(path) {
		parts := strings.Split(path, string(filepath.Separator))
		if len(parts) > 2 {
			return filepath.Join("<redacted>", parts[len(parts)-2], parts[len(parts)-1])
		}
		return filepath.Join("<redacted>", parts[len(parts)-1])
	}
	if homeDir != "" && strings.HasPrefix(path, homeDir) {
		return strings.Replace(path, homeDir, "~", 1)
	}
	return path
}

func sanitizePublicPath(path string) string {
	cleaned := filepath.Clean(path)
	parts := strings.Split(cleaned, string(filepath.Separator))
	var filtered []string
	for _, p := range parts {
		if strings.Contains(p, "private") || strings.Contains(p, "secret") || strings.Contains(p, "internal") {
			filtered = append(filtered, "synthetic")
		} else {
			filtered = append(filtered, p)
		}
	}
	return strings.Join(filtered, string(filepath.Separator))
}

func (p *Packer) GrpContextPack(pack *ir.ContextPack, planID string) *ir.GrpContextPack {
	return pack.ToGrpContextPack(planID)
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
