package planner

import (
	"fmt"
	"strings"
	"time"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/recipes"
)

type Planner struct {
	engine *recipes.RecipeEngine
}

func NewPlanner() *Planner {
	return &Planner{
		engine: recipes.NewEngine(),
	}
}

type PlanRequest struct {
	Target      string
	Goal        string
	Recipe      string
	Diagnostics []ir.Diagnostic
	Files       []*ir.FileIR
	Lang        string
}

func (p *Planner) GeneratePlan(req PlanRequest) *ir.GrpPlan {
	diags := req.Diagnostics
	if diags == nil {
		diags = []ir.Diagnostic{}
	}
	plan := &ir.GrpPlan{
		PlanID:       fmt.Sprintf("grp_%d", time.Now().Unix()),
		Goal:         req.Goal,
		Target:       req.Target,
		Diagnostics:  diags,
		Steps:        []ir.GrpStep{},
		Verification: p.detectVerification(req.Target, req.Lang),
	}

	if req.Goal == "" {
		plan.Goal = p.inferGoal(req.Diagnostics, req.Target)
	}

	if req.Recipe != "" {
		p.applySpecificRecipe(plan, req)
	} else {
		p.generateStepsFromDiagnostics(plan, req)
	}

	plan.Risk = p.computeRisk(plan.Steps)
	plan.FileCount = len(req.Files)

	totalTokens := 0
	for _, s := range plan.Steps {
		totalTokens += len(s.Action) / 3
	}
	plan.TokenEstimate = totalTokens

	p.assignStepIDs(plan)

	return plan
}

func (p *Planner) inferGoal(diags []ir.Diagnostic, target string) string {
	if len(diags) == 0 {
		return "Improve code quality"
	}

	hasLargeFile := false
	hasLongFn := false
	hasComplexity := false
	hasNesting := false
	hasErrors := false
	for _, d := range diags {
		switch d.Code {
		case "go.large-file":
			hasLargeFile = true
		case "go.long-function":
			hasLongFn = true
		case "go.high-complexity":
			hasComplexity = true
		case "go.deep-nesting":
			hasNesting = true
		case "go.ignored-error":
			hasErrors = true
		}
	}

	parts := []string{}
	if hasLargeFile {
		parts = append(parts, "split large files")
	}
	if hasLongFn || hasComplexity || hasNesting {
		parts = append(parts, "simplify complex functions")
	}
	if hasErrors {
		parts = append(parts, "handle ignored errors")
	}

	if len(parts) == 0 {
		componentRelated := false
		hookRelated := false
		for _, d := range diags {
			if strings.Contains(d.Code, "component") {
				componentRelated = true
			}
			if strings.Contains(d.Code, "hook") {
				hookRelated = true
			}
		}
		if componentRelated {
			parts = append(parts, "reduce component size")
		}
		if hookRelated {
			parts = append(parts, "extract custom hooks")
		}
	}

	if len(parts) == 0 {
		parts = append(parts, "improve code structure")
	}

	return fmt.Sprintf("Refactor %s: %s", target, strings.Join(parts, " and "))
}

func (p *Planner) applySpecificRecipe(plan *ir.GrpPlan, req PlanRequest) {
	for _, d := range req.Diagnostics {
		step := p.engine.GenerateStep(req.Recipe, d)
		if step.ID != "" {
			plan.Steps = append(plan.Steps, step)
		}
	}
	if len(plan.Steps) == 0 {
		step := ir.GrpStep{
			ID:     "step_apply",
			Recipe: req.Recipe,
			Action: fmt.Sprintf("Apply recipe %s to %s", req.Recipe, req.Target),
			File:   req.Target,
			Risk:   ir.SeverityMedium,
			Verify: []string{"typecheck", "tests"},
		}
		plan.Steps = append(plan.Steps, step)
	}
}

func (p *Planner) generateStepsFromDiagnostics(plan *ir.GrpPlan, req PlanRequest) {
	for _, diag := range req.Diagnostics {
		var step ir.GrpStep

		switch diag.Code {
		case "react.large-component":
			step = ir.GrpStep{
				Recipe: "react.split-large-component",
				Action: fmt.Sprintf("Split %s (%s) into smaller focused components", diag.Component, diag.File),
				File:   diag.File,
				Risk:   diag.Severity,
				Verify: []string{"typecheck", "tests"},
			}
		case "react.repeated-jsx":
			step = ir.GrpStep{
				Recipe: "react.extract-repeated-jsx",
				Action: fmt.Sprintf("Extract repeated JSX in %s into a reusable component", diag.Component),
				File:   diag.File,
				Risk:   ir.SeverityLow,
				Verify: []string{"typecheck", "lint"},
			}
		case "react.too-many-hooks":
			step = ir.GrpStep{
				Recipe: "react.extract-custom-hook",
				Action: fmt.Sprintf("Extract hooks from %s into custom hooks", diag.Component),
				File:   diag.File,
				Risk:   ir.SeverityLow,
				Verify: []string{"typecheck", "tests"},
			}
		case "react.too-many-state-vars":
			step = ir.GrpStep{
				Recipe: "react.reduce-state-vars",
				Action: fmt.Sprintf("Consolidate state variables in %s using useReducer", diag.Component),
				File:   diag.File,
				Risk:   ir.SeverityMedium,
				Verify: []string{"typecheck", "tests"},
			}
		case "react.mixed-responsibilities":
			step = ir.GrpStep{
				Recipe: "react.split-large-component",
				Action: fmt.Sprintf("Separate concerns in %s by extracting non-UI logic", diag.Component),
				File:   diag.File,
				Risk:   ir.SeverityMedium,
				Verify: []string{"typecheck", "tests"},
			}
		case "react.too-many-effects":
			step = ir.GrpStep{
				Recipe: "react.consolidate-effects",
				Action: fmt.Sprintf("Consolidate useEffect calls in %s", diag.Component),
				File:   diag.File,
				Risk:   ir.SeverityMedium,
				Verify: []string{"typecheck", "tests"},
			}
		case "react.hardcoded-data":
			step = ir.GrpStep{
				Recipe: "react.extract-constants",
				Action: fmt.Sprintf("Move hardcoded data from %s to external constants file", diag.Component),
				File:   diag.File,
				Risk:   ir.SeverityLow,
				Verify: []string{"typecheck"},
			}
		case "react.missing-prop-types":
			step = ir.GrpStep{
				Recipe: "react.add-prop-types",
				Action: fmt.Sprintf("Add TypeScript props interface to %s", diag.Component),
				File:   diag.File,
				Risk:   ir.SeverityLow,
				Verify: []string{"typecheck"},
			}
		case "go.long-function":
			step = ir.GrpStep{
				Recipe: "go.extract-function",
				Action: fmt.Sprintf("Extract smaller functions from %s", extractTarget(diag)),
				File:   diag.File,
				Risk:   diag.Severity,
				Verify: []string{"go build ./...", "go vet ./...", "go test ./..."},
			}
		case "go.high-complexity":
			step = ir.GrpStep{
				Recipe: "go.simplify-branches",
				Action: fmt.Sprintf("Simplify branching logic in %s with guard clauses and early returns", extractTarget(diag)),
				File:   diag.File,
				Risk:   diag.Severity,
				Verify: []string{"go build ./...", "go vet ./...", "go test ./..."},
			}
		case "go.deep-nesting":
			step = ir.GrpStep{
				Recipe: "go.flatten-nesting",
				Action: fmt.Sprintf("Reduce nesting depth in %s by extracting helper functions", extractTarget(diag)),
				File:   diag.File,
				Risk:   diag.Severity,
				Verify: []string{"go build ./...", "go vet ./...", "go test ./..."},
			}
		case "go.large-file":
			step = ir.GrpStep{
				Recipe: "go.split-file",
				Action: fmt.Sprintf("Split %s into multiple files by responsibility", diag.File),
				File:   diag.File,
				Risk:   diag.Severity,
				Verify: []string{"go build ./...", "go test ./..."},
			}
		case "go.ignored-error":
			step = ir.GrpStep{
				Recipe: "go.handle-error",
				Action: fmt.Sprintf("Handle ignored errors in %s", extractTarget(diag)),
				File:   diag.File,
				Risk:   ir.SeverityMedium,
				Verify: []string{"go vet ./...", "go build ./..."},
			}
		case "go.too-many-params":
			step = ir.GrpStep{
				Recipe: "go.extract-options-struct",
				Action: fmt.Sprintf("Group parameters in %s into an options struct", extractTarget(diag)),
				File:   diag.File,
				Risk:   ir.SeverityLow,
				Verify: []string{"go build ./...", "go test ./..."},
			}
		default:
			continue
		}

		step.ID = fmt.Sprintf("step_%s", diag.Code)
		plan.Steps = append(plan.Steps, step)
	}
}

func extractTarget(d ir.Diagnostic) string {
	msg := d.Message
	if strings.HasPrefix(msg, "Function ") {
		rest := msg[len("Function "):]
		for _, sep := range []string{" in ", " has ", " ignores ", " "} {
			if idx := strings.Index(rest, sep); idx >= 0 {
				return rest[:idx]
			}
		}
		return rest
	}
	if strings.HasPrefix(msg, "File ") {
		if idx := strings.Index(msg, "\" is "); idx >= 0 {
			return msg[5:idx]
		}
	}
	return d.File
}

func (p *Planner) computeRisk(steps []ir.GrpStep) ir.Severity {
	hasHigh := false
	hasMedium := false
	for _, s := range steps {
		if s.Risk == ir.SeverityHigh {
			hasHigh = true
		}
		if s.Risk == ir.SeverityMedium {
			hasMedium = true
		}
	}
	if hasHigh {
		return ir.SeverityHigh
	}
	if hasMedium {
		return ir.SeverityMedium
	}
	return ir.SeverityLow
}

func (p *Planner) assignStepIDs(plan *ir.GrpPlan) {
	for i, s := range plan.Steps {
		if s.ID == "" {
			plan.Steps[i].ID = fmt.Sprintf("step_%d", i+1)
		}
	}
}

func (p *Planner) detectVerification(target string, lang string) []string {
	if lang == "go" {
		return []string{
			"go build ./...",
			"go vet ./...",
			"go test ./...",
		}
	}
	return []string{
		"npm run typecheck",
		"npm run lint",
		"npm test",
		"npm run build",
	}
}
