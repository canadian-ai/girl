package planner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/recipes"
	"github.com/canadian-ai/girl/internal/verification"
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
		PlanID:       "",
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
	parts := inferGoGoalParts(diags)
	if len(parts) == 0 {
		parts = inferReactGoalParts(diags)
	}
	if len(parts) == 0 {
		parts = append(parts, "improve code structure")
	}
	return fmt.Sprintf("Refactor %s: %s", target, strings.Join(parts, " and "))
}

func inferGoGoalParts(diags []ir.Diagnostic) []string {
	codes := diagnosticCodes(diags)
	parts := []string{}
	if codes["go.large-file"] {
		parts = append(parts, "split large files")
	}
	if codes["go.long-function"] || codes["go.high-complexity"] || codes["go.deep-nesting"] {
		parts = append(parts, "simplify complex functions")
	}
	if codes["go.ignored-error"] {
		parts = append(parts, "handle ignored errors")
	}
	return parts
}

func inferReactGoalParts(diags []ir.Diagnostic) []string {
	parts := []string{}
	if hasCodePart(diags, "component") {
		parts = append(parts, "reduce component size")
	}
	if hasCodePart(diags, "hook") {
		parts = append(parts, "extract custom hooks")
	}
	return parts
}

func diagnosticCodes(diags []ir.Diagnostic) map[string]bool {
	codes := map[string]bool{}
	for _, d := range diags {
		codes[d.Code] = true
	}
	return codes
}

func hasCodePart(diags []ir.Diagnostic, part string) bool {
	for _, d := range diags {
		if strings.Contains(d.Code, part) {
			return true
		}
	}
	return false
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
	for i, diag := range req.Diagnostics {
		step := recipes.StepForDiagnostic(diag)
		if step.Recipe == "" {
			continue
		}
		step.SourceDiagIndex = i
		plan.Steps = append(plan.Steps, step)
	}
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
	sort.SliceStable(plan.Steps, func(i, j int) bool {
		if plan.Steps[i].Risk != plan.Steps[j].Risk {
			sevOrder := map[ir.Severity]int{ir.SeverityHigh: 0, ir.SeverityMedium: 1, ir.SeverityLow: 2}
			return sevOrder[plan.Steps[i].Risk] < sevOrder[plan.Steps[j].Risk]
		}
		if plan.Steps[i].File != plan.Steps[j].File {
			return plan.Steps[i].File < plan.Steps[j].File
		}
		if plan.Steps[i].Recipe != plan.Steps[j].Recipe {
			return plan.Steps[i].Recipe < plan.Steps[j].Recipe
		}
		return plan.Steps[i].Action < plan.Steps[j].Action
	})

	for i, s := range plan.Steps {
		slug := slugTarget(s.Action)
		id := fmt.Sprintf("step_%03d_%s_%s", i+1, s.Recipe, slug)
		plan.Steps[i].ID = id
	}
}

func slugTarget(s string) string {
	var result strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case isSlugChar(r):
			result.WriteRune(r)
		case isSlugSeparator(r):
			if !strings.HasSuffix(result.String(), "-") {
				result.WriteRune('-')
			}
		}
	}
	slug := strings.TrimRight(result.String(), "-")
	if len(slug) > 40 {
		slug = slug[:40]
	}
	if slug == "" {
		slug = "target"
	}
	return slug
}

func isSlugChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

func isSlugSeparator(r rune) bool {
	return r == ' ' || r == '-' || r == '_'
}

func (p *Planner) detectVerification(target string, lang string) []string {
	return verification.Commands(target)
}
