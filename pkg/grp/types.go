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

type Reviewability struct {
	Status         string                `json:"status"`
	Budget         ReviewabilityBudget   `json:"budget,omitempty"`
	Observed       ReviewabilityObserved `json:"observed,omitempty"`
	Recommendation string                `json:"recommendation,omitempty"`
	Reason         string                `json:"reason,omitempty"`
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

// GarbageClass classifies why a software graph node is a collection candidate.
type GarbageClass string

const (
	GarbageUnreachable           GarbageClass = "unreachable"
	GarbageDuplicate             GarbageClass = "duplicate"
	GarbageObsolete              GarbageClass = "obsolete"
	GarbageRedundant             GarbageClass = "redundant"
	GarbageDeadAPI               GarbageClass = "dead-api"
	GarbageDeadPolicy            GarbageClass = "dead-policy"
	GarbageDeadSchemaField       GarbageClass = "dead-schema-field"
	GarbageDeadDependencyAdapter GarbageClass = "dead-dependency-adapter"
)

// Step action prefixes for the GRP Core reduction/collection lifecycle.
// A collect step is never safe on its own: it must reference migration steps
// and carry a verification gate (see ValidatePlan).
const (
	ActionCanonicalize = "grp.reduction.canonicalize"
	ActionMigrate      = "grp.reduction.migrate"
	ActionCollect      = "grp.reduction.collect"
)

// NodeIdentity is the semantic identity of a software graph node. GRP Core
// only defines the shape; bindings and graph providers (e.g. a CAI Lang/TEMPLE
// graph) supply canonical capability IDs without GIRL depending on them. IDs are
// opaque, non-empty, provider-supplied stable strings.
type NodeIdentity struct {
	ID     string `json:"id"`             // semantic node / capability ID, e.g. "cap.booking" or "booking.create"
	Kind   string `json:"kind,omitempty"` // capability, api, schemaField, policy, dependencyAdapter
	Symbol string `json:"symbol,omitempty"`
	File   string `json:"file,omitempty"` // repo-relative source file
}

// Identity returns the semantic node identity of a reduction node.
func (n ReductionNode) Identity() NodeIdentity {
	return NodeIdentity{ID: n.ID, Kind: n.Kind, Symbol: n.Symbol, File: n.File}
}

// ReferenceEdge is source-grounded reachability/reference evidence linking two
// semantic nodes. From and To refer to NodeIdentity IDs within a Reduction.
type ReferenceEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind,omitempty"` // call, import, type, duplicate-of, superseded-by
	File string `json:"file,omitempty"` // repo-relative file where the reference was observed
	Line int    `json:"line,omitempty"`
}

// ReductionNode is a semantic node with its garbage classification, canonical
// target, and reference evidence. The ID/Kind/Symbol/File fields together form
// the semantic node identity (see NodeIdentity).
type ReductionNode struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind,omitempty"`
	GarbageClass GarbageClass      `json:"garbageClass,omitempty"`
	CanonicalID  string            `json:"canonicalID,omitempty"` // canonical target capability ID
	Reachable    bool              `json:"reachable,omitempty"`
	RefCount     int               `json:"refCount,omitempty"`
	Symbol       string            `json:"symbol,omitempty"`
	File         string            `json:"file,omitempty"`
	References   []ReferenceEdge   `json:"references,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ReductionBlock is compression metadata for collapsing a repeated subgraph
// into a standard block with explicit inputs and outputs.
type ReductionBlock struct {
	ID           string   `json:"id"`
	CapabilityID string   `json:"capabilityId,omitempty"` // owning canonical capability
	Standard     bool     `json:"standard,omitempty"`     // collapses repeated subgraphs into one standard block
	Inputs       []string `json:"inputs,omitempty"`
	Outputs      []string `json:"outputs,omitempty"`
	Nodes        []string `json:"nodes,omitempty"` // semantic node IDs in this block
}

// Reduction carries the software graph reduction/collection metadata for a
// plan. Plans without reduction metadata remain fully backward compatible.
type Reduction struct {
	Nodes  []ReductionNode  `json:"nodes,omitempty"`
	Blocks []ReductionBlock `json:"blocks,omitempty"`
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
	Reviewability      *Reviewability         `json:"reviewability,omitempty"`
	Decomposition      *Decomposition         `json:"decomposition,omitempty"`
	Reduction          *Reduction             `json:"reduction,omitempty"`
}
