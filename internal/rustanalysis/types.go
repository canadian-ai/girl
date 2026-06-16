package rustanalysis

type RustFile struct {
	Path      string
	Package   string
	Lines     int
	Functions []RustFunction
}

type RustFunction struct {
	Name       string
	Receiver   string
	StartLine  int
	EndLine    int
	Lines      int
	Params     int
	Returns    int
	Complexity int
	MaxNesting int
	IsAsync    bool
	IsUnsafe   bool
	IsPub      bool
}

type Config struct {
	MaxFunctionLines int
	MaxComplexity    int
	MaxNesting       int
	MaxFileLines     int
	MaxParams        int
}

func DefaultConfig() *Config {
	return &Config{
		MaxFunctionLines: 80,
		MaxComplexity:    10,
		MaxNesting:       4,
		MaxFileLines:     500,
		MaxParams:        5,
	}
}
