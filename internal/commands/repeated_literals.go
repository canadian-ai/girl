package commands

import (
    "fmt"
    "regexp"
    "sort"
    "strings"

    "github.com/canadian-ai/girl/internal/ir"
)

// This is deliberately a conservative advisory check, not a general duplicate
// string detector. It looks for repeated literals with semantic use as a state,
// identity, or comparison operand. Literal syntax in TS/JS is richer than this
// line scanner; a future AST implementation may expand coverage.
var domainLiteralPattern = regexp.MustCompile(`"[^"\n]{3,80}"|'[^'\n]{3,80}'`)
var domainLiteralContext = regexp.MustCompile(`(?:===|!==|==|!=|\bcase\s+|\b(?:status|intent|id|role|eventType)\s*:\s*)$`)

type literalOccurrence struct {
    line int
    contextual bool
}

// scanRepeatedDomainLiterals reports repeated meaningful string values in a
// single TypeScript/JavaScript file. It is advisory: no automatic replacements.
func scanRepeatedDomainLiterals(path string, source []byte) []ir.Diagnostic {
    switch {
    case strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".tsx"),
        strings.HasSuffix(path, ".js"), strings.HasSuffix(path, ".jsx"):
    default:
        return nil
    }
    occurrences := map[string][]literalOccurrence{}
    for lineIndex, line := range strings.Split(string(source), "\n") {
        trimmed := strings.TrimSpace(line)
        if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") ||
            strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "export {") {
            continue
        }
        // Skip inline comment tails; do not interpret string-looking examples.
        if i := strings.Index(line, " //"); i >= 0 {
            line = line[:i]
        }
        for _, loc := range domainLiteralPattern.FindAllStringIndex(line, -1) {
            value := line[loc[0]+1 : loc[1]-1]
            if strings.Contains(value, "\\") || strings.ContainsAny(value, "/<> \t") {
                continue
            }
            prefix := line[:loc[0]]
            occurrences[value] = append(occurrences[value], literalOccurrence{
                line: lineIndex + 1,
                contextual: domainLiteralContext.MatchString(prefix),
            })
        }
    }

    keys := make([]string, 0, len(occurrences))
    for value := range occurrences {
        keys = append(keys, value)
    }
    sort.Strings(keys)
    var diagnostics []ir.Diagnostic
    for _, value := range keys {
        positions := occurrences[value]
        if len(positions) < 3 {
            continue
        }
        firstContextLine := 0
        for _, pos := range positions {
            if pos.contextual {
                firstContextLine = pos.line
                break
            }
        }
        if firstContextLine == 0 {
            continue
        }
        diagnostics = append(diagnostics, ir.Diagnostic{
            Code: "ts.repeated-domain-literal",
            Severity: ir.SeverityLow,
            File: path,
            Line: firstContextLine,
            Message: fmt.Sprintf("Domain literal %q is repeated %d times", value, len(positions)),
            Suggestion: "Consider a named shared constant or typed enum if these occurrences represent one concept; avoid extracting incidental UI or protocol strings.",
        })
    }
    return diagnostics
}
