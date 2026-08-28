package complexity

const SchemaVersion = "1"

// FunctionMetric is one function-like unit measured independently. Nested
// functions do not contribute to their parent's complexity.
type FunctionMetric struct {
	ID                 string `json:"id"`
	File               string `json:"file"`
	Symbol             string `json:"symbol"`
	Kind               string `json:"kind"`
	Language           string `json:"language"`
	StartLine          int    `json:"startLine"`
	EndLine            int    `json:"endLine"`
	Lines              int    `json:"lines"`
	Complexity         int    `json:"complexity"`
	DecisionPoints     int    `json:"decisionPoints"`
	OverThreshold      bool   `json:"overThreshold"`
	BaselineComplexity int    `json:"baselineComplexity,omitempty"`
	Delta              int    `json:"delta,omitempty"`
	Change             string `json:"change,omitempty"`
}

type Summary struct {
	Files         int     `json:"files"`
	Functions     int     `json:"functions"`
	Total         int     `json:"total"`
	Average       float64 `json:"average"`
	Maximum       int     `json:"maximum"`
	OverThreshold int     `json:"overThreshold"`
}

type ParseError struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

type Change struct {
	ID         string `json:"id"`
	File       string `json:"file"`
	Symbol     string `json:"symbol"`
	Before     int    `json:"before"`
	After      int    `json:"after"`
	Delta      int    `json:"delta"`
	Status     string `json:"status"`
	Regression bool   `json:"regression"`
}

type Comparison struct {
	BaselineFunctions int      `json:"baselineFunctions"`
	Increased         int      `json:"increased"`
	Decreased         int      `json:"decreased"`
	Added             int      `json:"added"`
	Removed           int      `json:"removed"`
	Regressions       int      `json:"regressions"`
	NetComplexity     int      `json:"netComplexity"`
	Changes           []Change `json:"changes"`
}

type Report struct {
	SchemaVersion string           `json:"schemaVersion"`
	Metric        string           `json:"metric"`
	Root          string           `json:"root"`
	Threshold     int              `json:"threshold"`
	Summary       Summary          `json:"summary"`
	Functions     []FunctionMetric `json:"functions"`
	ParseErrors   []ParseError     `json:"parseErrors,omitempty"`
	Comparison    *Comparison      `json:"comparison,omitempty"`
}

type Options struct {
	Language  string
	Threshold int
	Exclude   []string
}
