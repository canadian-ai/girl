package ir

import (
	"fmt"
	"strings"
)

// Tags that mark a diagnostic as a blocking analysis failure rather than an
// advisory refactoring finding. A blocking diagnostic means GIRL could not
// establish the facts it needs about a file, so a quality gate must not pass.
const (
	TagBlocking       = "blocking"
	TagSyntaxError    = "syntax-error"
	TagAnalysisError  = "analysis-error"
	TagRuntimeMissing = "runtime-missing"
)

// SyntaxError records where a parser rejected (or had to recover from) source.
type SyntaxError struct {
	Line    int    `json:"line"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
}

// IsBlocking reports whether d represents a parse/analysis failure that must
// fail the gate regardless of advisory severity policy.
func IsBlocking(d Diagnostic) bool {
	for _, t := range d.Tags {
		if t == TagBlocking {
			return true
		}
	}
	return strings.HasSuffix(d.Code, ".syntax-error") ||
		strings.HasSuffix(d.Code, ".parse-error") ||
		strings.HasSuffix(d.Code, ".analysis-error")
}

// TagParseRecovery marks an advisory finding from an error-recovering grammar.
const TagParseRecovery = "parse-recovery"

// ParseRecoveryDiagnostic reports that an error-recovering grammar
// (tree-sitter) could not cleanly parse part of a file. It is advisory and
// never blocking: such grammars are not the language's authoritative parser
// and reject some valid source (for example a bare "&" in JSX text). The
// authoritative signal for these languages is the project's own typecheck or
// build command in verification.
func ParseRecoveryDiagnostic(prefix, file string, se SyntaxError) Diagnostic {
	line := se.Line
	if line < 1 {
		line = 1
	}
	return Diagnostic{
		Code:       prefix + ".parse-recovery",
		Severity:   SeverityLow,
		Confidence: "low",
		Message:    fmt.Sprintf("Parser recovered from input it could not parse cleanly (%s); metrics for this file may be incomplete", se.Message),
		File:       file,
		Line:       line,
		EndLine:    line,
		Kind:       NodeKindFile,
		Suggestion: "Confirm the file compiles with the project's typecheck/build command; this grammar can reject valid source.",
		Tags:       []string{TagParseRecovery},
		Span:       &Span{StartLine: line, EndLine: line, StartCol: se.Column, EndCol: se.Column},
	}
}

// SyntaxErrorDiagnostic builds the canonical blocking diagnostic for a file
// the parser could not accept. prefix is the binding namespace (go, py, ...).
func SyntaxErrorDiagnostic(prefix, file string, se SyntaxError) Diagnostic {
	line := se.Line
	if line < 1 {
		line = 1
	}
	msg := se.Message
	if msg == "" {
		msg = "syntax error"
	}
	return Diagnostic{
		Code:       prefix + ".syntax-error",
		Severity:   SeverityHigh,
		Confidence: "high",
		Message:    fmt.Sprintf("Syntax error: %s", msg),
		File:       file,
		Line:       line,
		EndLine:    line,
		Kind:       NodeKindFile,
		Suggestion: "Repair the syntax error before any refactoring; GIRL cannot analyze this file.",
		Tags:       []string{TagBlocking, TagSyntaxError},
		Span:       &Span{StartLine: line, EndLine: line, StartCol: se.Column, EndCol: se.Column},
	}
}
