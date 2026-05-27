package grp

type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type Confidence string

const (
	ConfidenceLow    Confidence = "low"
	ConfidenceMedium Confidence = "medium"
	ConfidenceHigh   Confidence = "high"
)

type Span struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
	EndLine     int `json:"endLine"`
	EndColumn   int `json:"endColumn,omitempty"`
}

type Symbol struct {
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
}

type Target struct {
	File   string `json:"file"`
	Symbol string `json:"symbol,omitempty"`
	Kind   string `json:"kind,omitempty"`
}

type Execution struct {
	Mode string `json:"mode,omitempty"`
}

type Verification struct {
	Command    string `json:"command"`
	Required   bool   `json:"required"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
	Type       string `json:"type,omitempty"`
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

type Diagnostic struct {
	ID         string            `json:"id"`
	Code       string            `json:"code"`
	Severity   Severity          `json:"severity"`
	Confidence Confidence        `json:"confidence"`
	Message    string            `json:"message"`
	File       string            `json:"file"`
	Span       *Span             `json:"span,omitempty"`
	Line       int               `json:"line,omitempty"`
	EndLine    int               `json:"endLine,omitempty"`
	Symbol     *Symbol           `json:"symbol,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Related    []RelatedInfo     `json:"related,omitempty"`
	Fixes      []Fix             `json:"fixes,omitempty"`
}

type Step struct {
	ID        string         `json:"id"`
	Recipe    string         `json:"recipe,omitempty"`
	Title     string         `json:"title"`
	Action    string         `json:"action"`
	Target    Target         `json:"target"`
	Risk      Severity       `json:"risk"`
	Requires  []string       `json:"requires,omitempty"`
	Verify    []Verification `json:"verify,omitempty"`
	Execution *Execution     `json:"execution,omitempty"`
}

type Plan struct {
	SpecVersion        string                 `json:"specversion"`
	ID                 string                 `json:"id"`
	Type               string                 `json:"type"`
	Source             string                 `json:"source"`
	Subject            string                 `json:"subject"`
	Language           string                 `json:"language"`
	Goal               string                 `json:"goal"`
	Risk               Severity               `json:"risk"`
	Diagnostics        []Diagnostic           `json:"diagnostics"`
	Steps              []Step                 `json:"steps"`
	Verification       []Verification         `json:"verification"`
	Time               string                 `json:"time,omitempty"`
	Repository         string                 `json:"repository,omitempty"`
	Commit             string                 `json:"commit,omitempty"`
	Tool               string                 `json:"tool,omitempty"`
	Extensions         map[string]interface{} `json:"extensions,omitempty"`
	RequiredExtensions []string               `json:"requiredExtensions,omitempty"`
	Context            map[string]interface{} `json:"context,omitempty"`
	Artifacts          []string               `json:"artifacts,omitempty"`
}
