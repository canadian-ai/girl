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
	{
		Code:   "agent.diff-too-large",
		Recipe: "agent.decompose-large-diff",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityHigh },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go test ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Decompose large diff in %s into smaller reviewable tasks", targetName(d))
		},
	},
	{
		Code:   "agent.too-many-files-touched",
		Recipe: "agent.decompose-large-diff",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityHigh },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go test ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Split multi-file change in %s into smaller focused tasks", targetName(d))
		},
	},
	{
		Code:   "agent.mixed-boundaries",
		Recipe: "agent.split-mixed-boundary",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityMedium },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go vet ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Separate concerns in %s: isolate schema, API, and UI changes", targetName(d))
		},
	},
	{
		Code:   "agent.unreviewable-plan",
		Recipe: "agent.extract-reviewable-task",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityHigh },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go test ./...", "go vet ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Break unreviewable plan in %s into bounded reviewable tasks", targetName(d))
		},
	},
	{
		Code:   "agent.parallelization-opportunity",
		Recipe: "agent.generate-parallel-task-packs",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityLow },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go test ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Create parallel task packs for independent changes in %s", targetName(d))
		},
	},
	{
		Code:   "agent.high-overhead",
		Recipe: "agent.extract-reusable-scaffold",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityMedium },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go test ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Refactor ephemeral scaffold in %s into reusable builders", targetName(d))
		},
	},
	{
		Code:   "agent.low-cohesion",
		Recipe: "agent.flatten-cohesion-variance",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityMedium },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go vet ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Consolidate %s changes into fewer concern boundaries", targetName(d))
		},
	},
	{
		Code:   "agent.test-to-code-imbalance",
		Recipe: "agent.extract-reusable-scaffold",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityMedium },
		Verify: func(d ir.Diagnostic) []string { return []string{"go test ./...", "go vet ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Extract test helpers from %s to reduce test-to-code ratio", targetName(d))
		},
	},
	{
		Code:   "agent.ceremonial-noise",
		Recipe: "agent.split-mixed-boundary",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityHigh },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go test ./...", "go vet ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Reduce ceremonial noise in %s by splitting concerns and consolidating scaffold", targetName(d))
		},
	},
	{
		Code:   "agent.productive-scaffold",
		Recipe: "agent.extract-reusable-scaffold",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityLow },
		Verify: func(d ir.Diagnostic) []string { return []string{"go build ./...", "go test ./..."} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Continue extracting reusable scaffold from %s", targetName(d))
		},
	},
	{
		Code:   "openrewrite.refactor-opportunity",
		Recipe: "openrewrite.export-yaml-recipe",
		Risk:   func(d ir.Diagnostic) ir.Severity { return d.Severity },
		Verify: func(d ir.Diagnostic) []string { return []string{"mvn rewrite:run", "gradle rewriteRun"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Export diagnostics from %s as OpenRewrite YAML recipe", targetName(d))
		},
	},
	{
		Code:   "openrewrite.export-yaml",
		Recipe: "openrewrite.export-yaml-recipe",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityLow },
		Verify: func(d ir.Diagnostic) []string { return []string{"mvn rewrite:run", "gradle rewriteRun"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Export GIRL diagnostics as OpenRewrite YAML recipe for %s", targetName(d))
		},
	},
	{
		Code:   "rtk.optimize-commands",
		Recipe: "rtk.pipe-through-proxy",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityLow },
		Verify: func(d ir.Diagnostic) []string { return []string{"rtk version"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Pipe GIRL commands through RTK for token-efficient output: rtk girl analyze/plan/pack %s", targetName(d))
		},
	},
	{
		Code:   "rtk.hook-configure",
		Recipe: "rtk.pipe-through-proxy",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityLow },
		Verify: func(d ir.Diagnostic) []string { return []string{"rtk version"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Configure RTK pre-tool hook for transparent command rewriting in %s", targetName(d))
		},
	},
	{
		Code:   "gritql.pattern-available",
		Recipe: "gritql.generate-pattern",
		Risk:   func(d ir.Diagnostic) ir.Severity { return d.Severity },
		Verify: func(d ir.Diagnostic) []string { return []string{"grit check"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Generate GritQL pattern from %s diagnostic for automated rewriting", targetName(d))
		},
	},
	{
		Code:   "gritql.generate-pattern",
		Recipe: "gritql.generate-pattern",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityLow },
		Verify: func(d ir.Diagnostic) []string { return []string{"grit check"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Generate GritQL transformation pattern for %s", targetName(d))
		},
	},
	{
		Code:   "gritql.apply-transform",
		Recipe: "gritql.generate-pattern",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityMedium },
		Verify: func(d ir.Diagnostic) []string { return []string{"grit apply", "grit check"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Apply GritQL transformation to %s: grit apply generated-pattern.grit", targetName(d))
		},
	},
	{
		Code:   "rust-lsp.long-function",
		Recipe: "rust-lsp.export-diagnostics",
		Risk:   func(d ir.Diagnostic) ir.Severity { return d.Severity },
		Verify: func(d ir.Diagnostic) []string { return []string{"cargo check"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Export long function %s as LSP diagnostic for rust-analyzer", targetName(d))
		},
	},
	{
		Code:   "rust-lsp.high-complexity",
		Recipe: "rust-lsp.export-diagnostics",
		Risk:   func(d ir.Diagnostic) ir.Severity { return d.Severity },
		Verify: func(d ir.Diagnostic) []string { return []string{"cargo check", "cargo clippy"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Export high complexity in %s as LSP diagnostic", targetName(d))
		},
	},
	{
		Code:   "rust-lsp.deep-nesting",
		Recipe: "rust-lsp.export-diagnostics",
		Risk:   func(d ir.Diagnostic) ir.Severity { return d.Severity },
		Verify: func(d ir.Diagnostic) []string { return []string{"cargo check"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Export deep nesting in %s as LSP diagnostic", targetName(d))
		},
	},
	{
		Code:   "rust-lsp.large-file",
		Recipe: "rust-lsp.export-diagnostics",
		Risk:   func(d ir.Diagnostic) ir.Severity { return d.Severity },
		Verify: func(d ir.Diagnostic) []string { return []string{"cargo check"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Export large file %s as LSP diagnostic", targetName(d))
		},
	},
	{
		Code:   "rust-lsp.export-diagnostics",
		Recipe: "rust-lsp.export-diagnostics",
		Risk:   func(d ir.Diagnostic) ir.Severity { return ir.SeverityLow },
		Verify: func(d ir.Diagnostic) []string { return []string{"cargo check"} },
		Action: func(d ir.Diagnostic) string {
			return fmt.Sprintf("Export GIRL diagnostics for %s in LSP format", targetName(d))
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
