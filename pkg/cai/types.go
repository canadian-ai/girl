package cai

// SigilVersion is the version of the SIGIL spec
const SigilVersion = "0.1"

// SigilSpec is the spec identifier
const SigilSpec = "cai-agent-safety"

// SigilManifest represents the SIGIL (Safety Integrity Guardrails and Instrumentation Layer) manifest
type SigilManifest struct {
	Version  string          `json:"version"`
	Spec     string          `json:"spec"`
	Manifest SigilProperties `json:"manifest"`
}

type SigilProperties struct {
	Project   string      `json:"project"`
	Tenancy   string      `json:"tenancy"`
	RiskLevel string      `json:"riskLevel"`
	Gates     SigilGates  `json:"gates"`
}

type SigilGates struct {
	Preflight bool `json:"preflight"`
	Review    bool `json:"review"`
	Receipt   bool `json:"receipt"`
	Prove     bool `json:"prove"`
}

// TenancyConfig defines the deployment context and isolation boundaries
type TenancyConfig struct {
	Environment     string           `json:"environment"`
	Isolation       string           `json:"isolation"`
	DeploymentGates []DeploymentGate `json:"deploymentGates,omitempty"`
}

// DeploymentGate defines a deployment quality gate
type DeploymentGate struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// LaunchKit defines a launch kit with quality gates
type LaunchKit struct {
	Version string               `json:"version"`
	Gates   map[string]GateSpec  `json:"gates"`
	Receipt *ReceiptConfig       `json:"receipt,omitempty"`
}

type GateSpec struct {
	Required bool   `json:"required"`
	Command  string `json:"command"`
}

type ReceiptConfig struct {
	Enabled bool   `json:"enabled"`
	Format  string `json:"format"`
}

// PreflightCheck represents a preflight check result
type PreflightCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // pass, warn, fail
	Message string `json:"message,omitempty"`
}

// PreflightResult represents the full preflight check result
type PreflightResult struct {
	SpecVersion string          `json:"specversion"`
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Path        string          `json:"path"`
	Status      string          `json:"status"`
	Checks      []PreflightCheck `json:"checks"`
	Summary     struct {
		Total int `json:"total"`
		Pass  int `json:"pass"`
		Warn  int `json:"warn"`
		Fail  int `json:"fail"`
	} `json:"summary"`
	Timestamp string `json:"timestamp"`
}

// WorkOrder represents an agent-ready task specification
type WorkOrder struct {
	SpecVersion    string   `json:"specversion"`
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	Goal           string   `json:"goal"`
	Risk           string   `json:"risk"`
	AllowedFiles   []string `json:"allowedFiles,omitempty"`
	ForbiddenFiles []string `json:"forbiddenFiles,omitempty"`
	MaxDiffLines   int      `json:"maxDiffLines,omitempty"`
	Parallelizable bool     `json:"parallelizable"`
	DependsOn      []string `json:"dependsOn,omitempty"`
	Verification   []string `json:"verification,omitempty"`
	SourcePlanID   string   `json:"sourcePlanId,omitempty"`
	CreatedAt      string   `json:"createdAt"`
}

// AgentReceipt represents an agent-change receipt
type AgentReceipt struct {
	SpecVersion string            `json:"specversion"`
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	CreatedAt   string            `json:"createdAt"`
	Source      string            `json:"source,omitempty"`
	Diff        DiffSummary       `json:"diff"`
	PlanRef     string            `json:"planRef,omitempty"`
	Verify      interface{}       `json:"verify,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type DiffSummary struct {
	TotalLines   int         `json:"totalLines"`
	AddedLines   int         `json:"addedLines"`
	DeletedLines int         `json:"deletedLines"`
	FilesChanged int         `json:"filesChanged"`
	Files        []DiffFile  `json:"files,omitempty"`
	ContentHash  string      `json:"contentHash"`
}

type DiffFile struct {
	Path         string `json:"path"`
	LinesAdded   int    `json:"linesAdded"`
	LinesRemoved int    `json:"linesRemoved"`
}

// ProveAppResult represents the result of a prove-app readiness check
type ProveAppResult struct {
	SpecVersion string           `json:"specversion"`
	ID          string           `json:"id"`
	Type        string           `json:"type"`
	Path        string           `json:"path"`
	Status      string           `json:"status"`
	Checks      []ReadinessCheck `json:"checks"`
	Summary     struct {
		Total int `json:"total"`
		Pass  int `json:"pass"`
		Warn  int `json:"warn"`
		Fail  int `json:"fail"`
	} `json:"summary"`
	Timestamp string `json:"timestamp"`
}

type ReadinessCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	Required bool   `json:"required"`
}

// ValidateLaunchKit validates a launch kit configuration
func ValidateLaunchKit(lk *LaunchKit) []string {
	var errors []string
	if lk.Version == "" {
		errors = append(errors, "launch kit version is required")
	}
	if len(lk.Gates) == 0 {
		errors = append(errors, "at least one gate is required")
	}
	for name, gate := range lk.Gates {
		if gate.Command == "" {
			errors = append(errors, name+": command is required")
		}
	}
	return errors
}

// DefaultSigil returns a default SIGIL manifest
func DefaultSigil() *SigilManifest {
	return &SigilManifest{
		Version: SigilVersion,
		Spec:    SigilSpec,
		Manifest: SigilProperties{
			Project:   "",
			Tenancy:   "unknown",
			RiskLevel: "low",
			Gates: SigilGates{
				Preflight: true,
				Review:    true,
				Receipt:   true,
				Prove:     true,
			},
		},
	}
}
