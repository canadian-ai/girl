package goanalysis

type GoFile struct {
	Path      string
	Package   string
	Lines     int
	Functions []GoFunction
}

type GoFunction struct {
	Name        string
	Receiver    string
	StartLine   int
	EndLine     int
	Lines       int
	Params      int
	Returns     int
	Complexity  int
	MaxNesting  int
	HasErrors   bool
	IgnoredErrs int
}

type Config struct {
	MaxFunctionLines int
	MaxComplexity    int
	MaxNesting       int
	MaxFileLines     int
	MaxParams        int
	MinSwitchCases   int
}

func DefaultConfig() *Config {
	return &Config{
		MaxFunctionLines: 80,
		MaxComplexity:    10,
		MaxNesting:       4,
		MaxFileLines:     500,
		MaxParams:        5,
		MinSwitchCases:   8,
	}
}
