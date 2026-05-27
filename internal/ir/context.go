package ir

type FileSummary struct {
	Path            string `json:"path"`
	Lines           int    `json:"lines"`
	ComponentCount  int    `json:"componentCount"`
	HookCount       int    `json:"hookCount"`
	Summary         string `json:"summary"`
}

type Snippet struct {
	File      string `json:"file"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
	Content   string `json:"content"`
	Tokens    int    `json:"tokens"`
}

type BudgetInfo struct {
	MaxTokens       int `json:"maxTokens"`
	EstimatedTokens int `json:"estimatedTokens"`
}

type ContextPack struct {
	Goal             string            `json:"goal"`
	TokenBudget      int               `json:"tokenBudget"`
	TokenEstimate    int               `json:"tokenEstimate"`
	Files            []string          `json:"files"`
	Summaries        []FileSummary     `json:"summaries"`
	SelectedSnippets []Snippet         `json:"selectedSnippets"`
	Diagnostics      []Diagnostic      `json:"diagnostics"`
	Steps            []GrpStep         `json:"steps"`
	Risks            []string          `json:"risks"`
	Verification     []string          `json:"verification"`
	DiagnosticCounts map[string]int    `json:"diagnosticCounts,omitempty"`
	TopCodes         []string          `json:"topCodes,omitempty"`
}

type GrpContextPack struct {
	SpecVersion  string        `json:"specversion"`
	Type         string        `json:"type"`
	PlanID       string        `json:"planId"`
	Budget       BudgetInfo    `json:"budget"`
	Goal         string        `json:"goal"`
	Diagnostics  []Diagnostic  `json:"diagnostics"`
	Steps        []GrpStep     `json:"steps"`
	Files        []string      `json:"files"`
	Snippets     []Snippet     `json:"snippets"`
	Verification []string      `json:"verification"`
}

func (p *ContextPack) ToGrpContextPack(planID string) *GrpContextPack {
	return &GrpContextPack{
		SpecVersion:  "0.1",
		Type:         "dev.refactor.context",
		PlanID:       planID,
		Budget:       BudgetInfo{MaxTokens: p.TokenBudget, EstimatedTokens: p.TokenEstimate},
		Goal:         p.Goal,
		Diagnostics:  p.Diagnostics,
		Steps:        p.Steps,
		Files:        p.Files,
		Snippets:     p.SelectedSnippets,
		Verification: p.Verification,
	}
}
