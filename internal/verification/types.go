package verification

type Command struct {
	Name       string `json:"name"`
	Script     string `json:"script,omitempty"`
	Command    string `json:"command"`
	Required   bool   `json:"required"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
	Type       string `json:"type,omitempty"`
	Exists     bool   `json:"exists"`
}

type Result struct {
	WorkDir         string    `json:"workDir"`
	PackageManager  string    `json:"packageManager"`
	Commands        []Command `json:"commands"`
	HasConfig       bool      `json:"hasConfig"`
	HasConvex       bool      `json:"hasConvex"`
	HasDocker       bool      `json:"hasDocker"`
	HasCI           bool      `json:"hasCI"`
	HasGolangCILint bool      `json:"hasGolangCILint,omitempty"`
	HasMakefile     bool      `json:"hasMakefile,omitempty"`
}
