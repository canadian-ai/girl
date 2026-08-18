package ir

type FileSummary struct {
	Path           string `json:"path"`
	Lines          int    `json:"lines"`
	ComponentCount int    `json:"componentCount"`
	HookCount      int    `json:"hookCount"`
	Summary        string `json:"summary"`
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

// ReferenceEdge is source-grounded reference evidence between semantic nodes.
// Mirrors pkg/grp so the context pack can carry reduction metadata without an
// import cycle (pkg/grp already depends on internal/ir).
type ReferenceEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind,omitempty"`
	File string `json:"file,omitempty"`
	Line int    `json:"line,omitempty"`
}

type ReductionNode struct {
	ID           string          `json:"id"`
	Kind         string          `json:"kind,omitempty"`
	GarbageClass string          `json:"garbageClass,omitempty"`
	CanonicalID  string          `json:"canonicalID,omitempty"`
	Reachable    bool            `json:"reachable,omitempty"`
	RefCount     int             `json:"refCount,omitempty"`
	Symbol       string          `json:"symbol,omitempty"`
	File         string          `json:"file,omitempty"`
	References   []ReferenceEdge `json:"references,omitempty"`
}

type ReductionBlock struct {
	ID           string   `json:"id"`
	CapabilityID string   `json:"capabilityId,omitempty"`
	Standard     bool     `json:"standard,omitempty"`
	Inputs       []string `json:"inputs,omitempty"`
	Outputs      []string `json:"outputs,omitempty"`
	Nodes        []string `json:"nodes,omitempty"`
}

type Reduction struct {
	Nodes  []ReductionNode  `json:"nodes,omitempty"`
	Blocks []ReductionBlock `json:"blocks,omitempty"`
}

type ContextPack struct {
	Goal             string         `json:"goal"`
	TokenBudget      int            `json:"tokenBudget"`
	TokenEstimate    int            `json:"tokenEstimate"`
	Files            []string       `json:"files"`
	Summaries        []FileSummary  `json:"summaries"`
	SelectedSnippets []Snippet      `json:"selectedSnippets"`
	Diagnostics      []Diagnostic   `json:"diagnostics"`
	Steps            []GrpStep      `json:"steps"`
	Risks            []string       `json:"risks"`
	Verification     []string       `json:"verification"`
	DiagnosticCounts map[string]int `json:"diagnosticCounts,omitempty"`
	TopCodes         []string       `json:"topCodes,omitempty"`
	Reduction        *Reduction     `json:"reduction,omitempty"`
}

type GrpContextPack struct {
	SpecVersion  string       `json:"specversion"`
	Type         string       `json:"type"`
	PlanID       string       `json:"planId"`
	Budget       BudgetInfo   `json:"budget"`
	Goal         string       `json:"goal"`
	Diagnostics  []Diagnostic `json:"diagnostics"`
	Steps        []GrpStep    `json:"steps"`
	Files        []string     `json:"files"`
	Snippets     []Snippet    `json:"snippets"`
	Verification []string     `json:"verification"`
	Reduction    *Reduction   `json:"reduction,omitempty"`
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
		Reduction:    p.Reduction,
	}
}
