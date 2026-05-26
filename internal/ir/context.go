package ir

type FileSummary struct {
	Path          string `json:"path"`
	Lines         int    `json:"lines"`
	ComponentCount int   `json:"componentCount"`
	HookCount     int    `json:"hookCount"`
	Summary       string `json:"summary"`
}

type Snippet struct {
	File      string `json:"file"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
	Content   string `json:"content"`
	Tokens    int    `json:"tokens"`
}

type ContextPack struct {
	Goal            string         `json:"goal"`
	TokenBudget     int            `json:"tokenBudget"`
	TokenEstimate   int            `json:"tokenEstimate"`
	Files           []string       `json:"files"`
	Summaries       []FileSummary  `json:"summaries"`
	SelectedSnippets []Snippet     `json:"selectedSnippets"`
	Diagnostics     []Diagnostic   `json:"diagnostics"`
	Steps           []GrpStep      `json:"steps"`
	Risks           []string       `json:"risks"`
	Verification    []string       `json:"verification"`
}
