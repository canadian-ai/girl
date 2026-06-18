package proof

// CodeCount summarizes diagnostics for a diagnostic code.
type CodeCount struct {
	Code  string `json:"code"`
	Count int    `json:"count"`
}

// FileCount summarizes diagnostics for a file.
type FileCount struct {
	File  string `json:"file"`
	Count int    `json:"count"`
}

// Summary contains deterministic aggregate analyzer output.
type Summary struct {
	Target          string      `json:"target"`
	FilesScanned    int         `json:"filesScanned"`
	Diagnostics     int         `json:"diagnostics"`
	High            int         `json:"high"`
	Medium          int         `json:"medium"`
	Low             int         `json:"low"`
	DiagnosticCodes []CodeCount `json:"diagnosticCodes"`
	WorstFiles      []FileCount `json:"worstFiles"`
}

// Report is the shareable proof report model used by benchmark and prove.
type Report struct {
	Kind            string   `json:"kind"`
	Summary         Summary  `json:"summary"`
	HealthScore     int      `json:"healthScore"`
	Status          string   `json:"status,omitempty"`
	TopImprovements []string `json:"topImprovements,omitempty"`
}
