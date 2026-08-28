package ir

type ComponentKind string

const (
	ComponentKindFunction ComponentKind = "function"
	ComponentKindArrow    ComponentKind = "arrow"
	ComponentKindClass    ComponentKind = "class"
)

type ComponentIR struct {
	Name             string           `json:"name"`
	FilePath         string           `json:"filePath"`
	Kind             ComponentKind    `json:"kind"`
	StartLine        int              `json:"startLine"`
	EndLine          int              `json:"endLine"`
	Lines            int              `json:"lines"`
	Hooks            []HookIR         `json:"hooks"`
	JSXBlocks        []JSXBlockIR     `json:"jsxBlocks"`
	Props            []PropIR         `json:"props"`
	StateVars        []StateVarIR     `json:"stateVars"`
	Effects          []EffectIR       `json:"effects"`
	EventHandlers    []EventHandlerIR `json:"eventHandlers"`
	Imports          []ImportIR       `json:"imports"`
	Exports          []ExportIR       `json:"exports"`
	ChildComponents  []string         `json:"childComponents"`
	HasKeyDown       bool             `json:"hasKeyDown"`
	HasAnalytics     bool             `json:"hasAnalytics"`
	ConditionalCount int              `json:"conditionalCount"`
	LoopCount        int              `json:"loopCount"`
}

type HookIR struct {
	Name      string   `json:"name"`
	Line      int      `json:"line"`
	Args      []string `json:"args"`
	DepsCount int      `json:"depsCount"`
}

type JSXBlockIR struct {
	Element     string `json:"element"`
	Line        int    `json:"line"`
	ChildCount  int    `json:"childCount"`
	PropCount   int    `json:"propCount"`
	ContentHash string `json:"contentHash"`
	Source      string `json:"-"`
}

type PropIR struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Line     int    `json:"line"`
}

type StateVarIR struct {
	Name       string `json:"name"`
	Line       int    `json:"line"`
	HasUpdater bool   `json:"hasUpdater"`
}

type EffectIR struct {
	Name      string `json:"name"`
	Line      int    `json:"line"`
	DepsCount int    `json:"depsCount"`
	IsAsync   bool   `json:"isAsync"`
	HasReturn bool   `json:"hasReturn"`
}

type EventHandlerIR struct {
	Name   string `json:"name"`
	Line   int    `json:"line"`
	Target string `json:"target"`
}

type ImportIR struct {
	Source  string   `json:"source"`
	Names   []string `json:"names"`
	Default string   `json:"default"`
}

type ExportIR struct {
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

type FileIR struct {
	Path       string        `json:"path"`
	Language   string        `json:"language"`
	Lines      int           `json:"lines"`
	Components []ComponentIR `json:"components"`
	Hooks      []HookIR      `json:"hooks"`
	Imports    []ImportIR    `json:"imports"`
}

type AnalyzerResult struct {
	Files       []*FileIR    `json:"files"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type Diagnostic struct {
	Code       string            `json:"code"`
	Severity   Severity          `json:"severity"`
	Confidence string            `json:"confidence,omitempty"`
	Message    string            `json:"message"`
	File       string            `json:"file"`
	Line       int               `json:"line"`
	Component  string            `json:"component,omitempty"`
	Suggestion string            `json:"suggestion,omitempty"`
	Kind       NodeKind          `json:"kind,omitempty"`
	Symbol     string            `json:"symbol,omitempty"`
	EndLine    int               `json:"endLine,omitempty"`
	Package    string            `json:"package,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Span       *Span             `json:"span,omitempty"`
	Related    []RelatedInfo     `json:"related,omitempty"`
	Fixes      []Fix             `json:"fixes,omitempty"`
}

type NodeKind string

const (
	NodeKindFunction  NodeKind = "function"
	NodeKindFile      NodeKind = "file"
	NodeKindComponent NodeKind = "component"
	NodeKindHook      NodeKind = "hook"
	NodeKindReference NodeKind = "reference"
	NodeKindState     NodeKind = "state"
)

type Span struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
	StartCol  int `json:"startCol,omitempty"`
	EndCol    int `json:"endCol,omitempty"`
}

type RelatedInfo struct {
	Message string `json:"message"`
	Span    Span   `json:"span"`
}

type Fix struct {
	Title string `json:"title"`
	Kind  string `json:"kind"`
	Span  Span   `json:"span"`
	Text  string `json:"text,omitempty"`
}

func (d Diagnostic) DiagnosticTarget() string {
	if d.Symbol != "" {
		return d.Symbol
	}
	if d.Component != "" {
		return d.Component
	}
	return d.File
}

type GrpStep struct {
	ID              string   `json:"id"`
	Recipe          string   `json:"recipe"`
	Action          string   `json:"action"`
	File            string   `json:"file"`
	Risk            Severity `json:"risk"`
	Verify          []string `json:"verify"`
	SourceDiagIndex int      `json:"-"`
}

type ReviewabilityBudget struct {
	MaxDiffLines    int      `json:"maxDiffLines,omitempty"`
	MaxTouchedFiles int      `json:"maxTouchedFiles,omitempty"`
	MaxRisk         Severity `json:"maxRisk,omitempty"`
}

type ReviewabilityObserved struct {
	AddedLines   int `json:"addedLines,omitempty"`
	DeletedLines int `json:"deletedLines,omitempty"`
	ChangedLines int `json:"changedLines,omitempty"`
	ChangedFiles int `json:"changedFiles,omitempty"`
	LargestDelta int `json:"largestFileDelta,omitempty"`
}

type ReviewabilityResult struct {
	Status         string                 `json:"status"`
	Budget         *ReviewabilityBudget   `json:"budget,omitempty"`
	Observed       *ReviewabilityObserved `json:"observed,omitempty"`
	Recommendation string                 `json:"recommendation,omitempty"`
	Reason         string                 `json:"reason,omitempty"`
}

type DecompositionTask struct {
	ID             string   `json:"id"`
	Goal           string   `json:"goal"`
	AllowedFiles   []string `json:"allowedFiles,omitempty"`
	ForbiddenFiles []string `json:"forbiddenFiles,omitempty"`
	MaxDiffLines   int      `json:"maxDiffLines,omitempty"`
	Parallelizable bool     `json:"parallelizable"`
	DependsOn      []string `json:"dependsOn,omitempty"`
	Verification   []string `json:"verification,omitempty"`
}

type Decomposition struct {
	Strategy   string              `json:"strategy"`
	ParentPlan string              `json:"parentPlan,omitempty"`
	Tasks      []DecompositionTask `json:"tasks"`
}

type GrpPlan struct {
	PlanID        string               `json:"planId"`
	Goal          string               `json:"goal"`
	Risk          Severity             `json:"risk"`
	Target        string               `json:"target"`
	Language      string               `json:"language,omitempty"`
	TokenEstimate int                  `json:"tokenEstimate"`
	FileCount     int                  `json:"fileCount"`
	Diagnostics   []Diagnostic         `json:"diagnostics"`
	Steps         []GrpStep            `json:"steps"`
	Verification  []string             `json:"verification"`
	Reviewability *ReviewabilityResult `json:"reviewability,omitempty"`
	Decomposition *Decomposition       `json:"decomposition,omitempty"`
}
