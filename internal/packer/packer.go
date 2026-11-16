package packer

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/redact"
	"github.com/canadian-ai/girl/internal/tokens"
	"github.com/canadian-ai/girl/internal/verification"
)

type Packer struct {
	MaxTokens int
	estimator tokens.Estimator
}

func NewPacker(maxTokens int, opts ...tokens.Estimator) *Packer {
	if maxTokens <= 0 {
		maxTokens = 12000
	}
	var est tokens.Estimator = tokens.NewHeuristicEstimator()
	if len(opts) > 0 && opts[0] != nil {
		est = opts[0]
	}
	return &Packer{MaxTokens: maxTokens, estimator: est}
}

type PackRequest struct {
	Goal        string
	Files       []*ir.FileIR
	Diagnostics []ir.Diagnostic
	Steps       []ir.GrpStep
	TokenBudget int
	PlanID      string
	TargetPath  string
	PrivacyMode string
	Reduction   *ir.Reduction
}

func (p *Packer) Pack(req PackRequest) (*ir.ContextPack, error) {
	budget := p.MaxTokens
	if req.TokenBudget > 0 {
		budget = req.TokenBudget
	}

	diagCounts, codeCounts, fileDiagCounts := countDiagnostics(req.Diagnostics)
	topCodes := topDiagnosticCodes(codeCounts, 5)
	pack := newContextPack(req, budget, diagCounts, topCodes)
	pack.Reduction = req.Reduction

	p.addFileSummaries(pack, req.Files, fileDiagCounts)

	tier := snippetTier(budget)
	var remaining int
	if tier == 0 {
		remaining = p.addDiagnosticSnippets(pack, req.Diagnostics, budget)
		if remaining > 0 {
			remaining = p.addComponentSnippets(pack, req.Files, remaining, req.Reduction)
		}
	} else {
		remaining = p.addComponentSnippets(pack, req.Files, budget, req.Reduction)
		remaining = p.addDiagnosticSnippets(pack, req.Diagnostics, remaining)
	}
	pack.TokenEstimate = budget - remaining

	pack.Risks = collectRisks(req.Diagnostics, req.Steps)
	pack.Verification = verification.Commands(req.TargetPath)

	privacy := req.PrivacyMode
	if privacy == "" {
		privacy = "private"
	}
	if err := p.applyPrivacy(privacy, pack, req.Files); err != nil {
		return nil, err
	}

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

// reducedNode is a node that collapses into a canonical capability block. When
// reduction metadata is present, the pack prefers the canonical block instead
// of dumping every duplicated implementation.
type reducedNode struct {
	ID          string
	CanonicalID string
	File        string
	Symbol      string
}

func indexReducedNodes(r *ir.Reduction) []reducedNode {
	if r == nil {
		return nil
	}
	var out []reducedNode
	for _, n := range r.Nodes {
		if n.CanonicalID != "" {
			out = append(out, reducedNode{ID: n.ID, CanonicalID: n.CanonicalID, File: n.File, Symbol: n.Symbol})
		}
	}
	return out
}

func matchesReducedNode(n reducedNode, f *ir.FileIR, c ir.ComponentIR) bool {
	if n.Symbol != c.Name {
		return false
	}
	return n.File == f.Path || n.File == packRelPath(f.Path)
}

func (p *Packer) addComponentSnippets(pack *ir.ContextPack, files []*ir.FileIR, budget int, reduction *ir.Reduction) int {
	remaining := budget
	reduced := indexReducedNodes(reduction)
	for _, f := range files {
		if remaining <= 0 {
			break
		}
		rel := packRelPath(f.Path)
		for _, c := range f.Components {
			if remaining <= 0 {
				break
			}
			if rn := firstReducedMatch(reduced, f, c); rn != nil {
				// Do not dump internals of a node that collapses into a canonical block.
				pack.SelectedSnippets = append(pack.SelectedSnippets, ir.Snippet{
					File:    rel,
					Content: fmt.Sprintf("// reduced: %s collapses into canonical %s", c.Name, rn.CanonicalID),
					Tokens:  0,
				})
				continue
			}
			snippet := p.createSnippet(f.Path, rel, c, remaining)
			pack.SelectedSnippets = append(pack.SelectedSnippets, *snippet)
			remaining -= snippet.Tokens
		}
	}
	return remaining
}

func firstReducedMatch(reduced []reducedNode, f *ir.FileIR, c ir.ComponentIR) *reducedNode {
	for i := range reduced {
		if matchesReducedNode(reduced[i], f, c) {
			return &reduced[i]
		}
	}
	return nil
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
	base := "."
	if filepath.IsAbs(path) {
		if wd, err := os.Getwd(); err == nil {
			base = wd
		}
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		// Never lose the path: the privacy layer needs it to recognise
		// sensitive files, and redactPath hides absolute prefixes.
		return path
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

	snipTokens := p.estimator.Estimate(content)

	if snipTokens > budget {
		maxChars := budget * 3
		if maxChars < 200 {
			maxChars = 200
		}
		content = content[:maxChars] + "\n// ... truncated to fit budget"
		snipTokens = budget
	}

	return &ir.Snippet{
		File:      relPath,
		StartLine: comp.StartLine,
		EndLine:   comp.EndLine,
		Content:   content,
		Tokens:    snipTokens,
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

	snipTokens := p.estimator.Estimate(content)
	if snipTokens > budget {
		maxChars := budget * 3
		if maxChars < 200 {
			maxChars = 200
		}
		content = content[:maxChars] + "\n// ... truncated"
		snipTokens = budget
	}

	return &ir.Snippet{
		File:      relPath,
		StartLine: startLine,
		EndLine:   endLine,
		Content:   content,
		Tokens:    snipTokens,
	}
}

// Privacy modes. The guarantees are deliberately modest and honest:
//
//   - private:  nothing is altered. For use on this machine only.
//   - redacted: every exported string (goal, paths, summaries, snippets,
//     diagnostics, steps, risks, verification, metadata) passes the shared
//     sanitizer; absolute and home paths are hidden; contents of sensitive
//     files (.env, keys, credentials) are omitted; any snippet on which a
//     detector still fires is omitted. Detection is heuristic, not perfect.
//   - public:   redacted, plus public path sanitization.
//
// Both sanitizing modes fail closed (an error, no pack) if any detector still
// fires anywhere in the output after sanitization.
const (
	PrivacyPrivate  = "private"
	PrivacyRedacted = "redacted"
	PrivacyPublic   = "public"
)

// ValidatePrivacyMode rejects unknown modes instead of silently exporting
// unsanitized content.
func ValidatePrivacyMode(mode string) error {
	switch mode {
	case "", PrivacyPrivate, PrivacyRedacted, PrivacyPublic:
		return nil
	}
	return fmt.Errorf("unknown privacy mode %q (expected private, redacted or public)", mode)
}

const omittedSnippet = "// omitted: content could not be safely sanitized for export"

func (p *Packer) applyPrivacy(mode string, pack *ir.ContextPack, files []*ir.FileIR) error {
	if err := ValidatePrivacyMode(mode); err != nil {
		return err
	}
	if mode == "" || mode == PrivacyPrivate {
		return nil
	}

	// Sensitive files never contribute content, whatever detectors say.
	for i, sn := range pack.SelectedSnippets {
		if redact.IsSensitiveFile(sn.File) {
			pack.SelectedSnippets[i].Content = omittedSnippet
		}
	}

	// Sanitize every exported string, then apply path transforms.
	redact.Value(pack)
	homeDir, _ := os.UserHomeDir()
	pathFn := func(path string) string {
		path = redactPath(path, homeDir)
		if mode == PrivacyPublic {
			path = sanitizePublicPath(path)
		}
		return path
	}
	for i := range pack.Files {
		pack.Files[i] = pathFn(pack.Files[i])
	}
	for i := range pack.Summaries {
		pack.Summaries[i].Path = pathFn(pack.Summaries[i].Path)
	}
	for i := range pack.SelectedSnippets {
		pack.SelectedSnippets[i].File = pathFn(pack.SelectedSnippets[i].File)
	}
	for i := range pack.Diagnostics {
		pack.Diagnostics[i].File = pathFn(pack.Diagnostics[i].File)
	}
	for i := range pack.Steps {
		pack.Steps[i].File = pathFn(pack.Steps[i].File)
	}

	// Independent verification: omit any snippet a detector still flags.
	for i, sn := range pack.SelectedSnippets {
		if len(redact.Scan(sn.Content)) > 0 {
			pack.SelectedSnippets[i].Content = omittedSnippet
		}
	}
	if findings := redact.ScanValue(pack); len(findings) > 0 {
		// Fail closed. Report field paths and detector names only, never values.
		var where []string
		for _, f := range findings {
			where = append(where, f.Path+" ("+f.Detector+")")
		}
		return fmt.Errorf("%s export refused: sanitization could not be established for %s", mode, strings.Join(where, ", "))
	}
	return nil
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
		replaced := p
		if strings.Contains(replaced, "private") || strings.Contains(replaced, "secret") {
			replaced = strings.NewReplacer("private", "synthetic", "secret", "synthetic").Replace(replaced)
		}
		filtered = append(filtered, replaced)
	}
	return strings.Join(filtered, string(filepath.Separator))
}

// redactContent applies the shared export sanitizer.
func redactContent(s string) string {
	return redact.String(s)
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
