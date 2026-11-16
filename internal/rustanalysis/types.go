package rustanalysis

import (
	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/shared"
)

type RustFile struct {
	Path      string
	Package   string
	Lines     int
	Functions []RustFunction
	// SyntaxError is set when tree-sitter had to recover from invalid source.
	SyntaxError *ir.SyntaxError
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
	// Skip excludes paths (project exclusions) before any file is read.
	Skip shared.PathFilter
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
