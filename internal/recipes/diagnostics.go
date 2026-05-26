package recipes

import (
	"fmt"

	"github.com/canadian-ai/girl/internal/ir"
)

type DiagnosticRecipe struct {
	Code   string
	Recipe string
	Risk   func(ir.Diagnostic) ir.Severity
	Verify func(ir.Diagnostic) []string
	Action func(ir.Diagnostic) string
}

var registry []DiagnosticRecipe

func Register(r DiagnosticRecipe) {
	registry = append(registry, r)
}

func Registered() []DiagnosticRecipe {
	result := make([]DiagnosticRecipe, len(registry))
	copy(result, registry)
	return result
}

func StepForDiagnostic(diag ir.Diagnostic) ir.GrpStep {
	for _, r := range registry {
		if r.Code == diag.Code {
			return ir.GrpStep{
				Recipe: r.Recipe,
				Action: r.Action(diag),
				File:   diag.File,
				Risk:   r.Risk(diag),
				Verify: r.Verify(diag),
			}
		}
	}
	return ir.GrpStep{}
}

var builtInRecipes = []DiagnosticRecipe{
	{
		Code:   "react.large-component",
		Recipe: "react.split-large-component",
		Risk:   func(d ir.Diagnostic) ir.Severity { return d.Severity },
		Verify: func(d ir.Diagnostic) []string { return []string{"typecheck", "tests"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Split %s (%s) into smaller focused components", targetName(d), d.File)
		},
	},
	{
		Code:   "react.repeated-jsx",
		Recipe: "react.extract-repeated-jsx",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityLow },
		Verify: func(d ir.Diagnostic) []string { return []string{"typecheck", "lint"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Extract repeated JSX in %s into a reusable component", targetName(d))
		},
	},
	{
		Code:   "react.too-many-hooks",
		Recipe: "react.extract-custom-hook",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityLow },
		Verify: func(d ir.Diagnostic) []string { return []string{"typecheck", "tests"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Extract hooks from %s into custom hooks", targetName(d))
		},
	},
	{
		Code:   "react.too-many-state-vars",
		Recipe: "react.reduce-state-vars",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityMedium },
		Verify: func(d ir.Diagnostic) []string { return []string{"typecheck", "tests"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Consolidate state variables in %s using useReducer", targetName(d))
		},
	},
	{
		Code:   "react.mixed-responsibilities",
		Recipe: "react.split-large-component",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityMedium },
		Verify: func(d ir.Diagnostic) []string { return []string{"typecheck", "tests"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Separate concerns in %s by extracting non-UI logic", targetName(d))
		},
	},
	{
		Code:   "react.too-many-effects",
		Recipe: "react.consolidate-effects",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityMedium },
		Verify: func(d ir.Diagnostic) []string { return []string{"typecheck", "tests"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Consolidate useEffect calls in %s", targetName(d))
		},
	},
	{
		Code:   "react.hardcoded-data",
		Recipe: "react.extract-constants",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityLow },
		Verify: func(d ir.Diagnostic) []string { return []string{"typecheck"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Move hardcoded data from %s to external constants file", targetName(d))
		},
	},
	{
		Code:   "react.missing-prop-types",
		Recipe: "react.add-prop-types",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityLow },
		Verify: func(d ir.Diagnostic) []string { return []string{"typecheck"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Add TypeScript props interface to %s", targetName(d))
		},
	},
	{
		Code:   "go.long-function",
		Recipe: "go.extract-function",
		Risk:   func(d ir.Diagnostic) ir.Severity { return d.Severity },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go vet ./...", "go test ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Extract smaller functions from %s", targetName(d))
		},
	},
	{
		Code:   "go.high-complexity",
		Recipe: "go.simplify-branches",
		Risk:   func(d ir.Diagnostic) ir.Severity { return d.Severity },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go vet ./...", "go test ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Simplify branching logic in %s with guard clauses and early returns", targetName(d))
		},
	},
	{
		Code:   "go.deep-nesting",
		Recipe: "go.flatten-nesting",
		Risk:   func(d ir.Diagnostic) ir.Severity { return d.Severity },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go vet ./...", "go test ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Reduce nesting depth in %s by extracting helper functions", targetName(d))
		},
	},
	{
		Code:   "go.large-file",
		Recipe: "go.split-file",
		Risk:   func(d ir.Diagnostic) ir.Severity { return d.Severity },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go test ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Split %s into multiple files by responsibility", targetName(d))
		},
	},
	{
		Code:   "go.ignored-error",
		Recipe: "go.handle-error",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityMedium },
		Verify: func(d ir.Diagnostic) []string { return []string{"go vet ./...", "go build ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Handle ignored errors in %s", targetName(d))
		},
	},
	{
		Code:   "go.too-many-params",
		Recipe: "go.extract-options-struct",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityLow },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go test ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Group parameters in %s into an options struct", targetName(d))
		},
	},
}

func init() {
	for _, recipe := range builtInRecipes {
		Register(recipe)
	}
}

func targetName(d ir.Diagnostic) string {
	return d.DiagnosticTarget()
}
