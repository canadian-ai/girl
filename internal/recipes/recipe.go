package recipes

import "github.com/canadian-ai/girl/internal/ir"

type RecipeMatch struct {
	RecipeID    string
	Description string
	Confidence  float64
	Diagnostic  ir.Diagnostic
}

type RecipeEngine struct {
	recipes []Recipe
}

type Recipe interface {
	ID() string
	Description() string
	Matches(comp *ir.ComponentIR) (*RecipeMatch, bool)
	GenerateStep(diag ir.Diagnostic) ir.GrpStep
}

func NewEngine(opts ...*Thresholds) *RecipeEngine {
	t := DefaultThresholds()
	if len(opts) > 0 && opts[0] != nil {
		t = opts[0]
	}
	return &RecipeEngine{
		recipes: []Recipe{
			&SplitLargeComponent{thresholds: t},
			&ExtractRepeatedJSX{thresholds: t},
			&ExtractCustomHook{thresholds: t},
			&ReduceStateVars{thresholds: t},
			&ConsolidateEffects{thresholds: t},
			&AddPropTypes{thresholds: t},
		},
	}
}

func (e *RecipeEngine) Match(comp *ir.ComponentIR) []RecipeMatch {
	var matches []RecipeMatch
	for _, r := range e.recipes {
		if m, ok := r.Matches(comp); ok {
			matches = append(matches, *m)
		}
	}
	return matches
}

func (e *RecipeEngine) GenerateStep(recipeID string, diag ir.Diagnostic) ir.GrpStep {
	for _, r := range e.recipes {
		if r.ID() == recipeID {
			return r.GenerateStep(diag)
		}
	}
	return ir.GrpStep{}
}

type SplitLargeComponent struct {
	thresholds *Thresholds
}

func (r *SplitLargeComponent) ID() string          { return "react.split-large-component" }
func (r *SplitLargeComponent) Description() string { return "Split a large component into smaller focused components" }

func (r *SplitLargeComponent) Matches(comp *ir.ComponentIR) (*RecipeMatch, bool) {
	if comp.Lines < r.thresholds.LargeComponentLines {
		return nil, false
	}
	return &RecipeMatch{
		RecipeID:    r.ID(),
		Description: r.Description(),
		Confidence:  float64(comp.Lines) / 500.0,
	}, true
}

func (r *SplitLargeComponent) GenerateStep(diag ir.Diagnostic) ir.GrpStep {
	return ir.GrpStep{
		Recipe: r.ID(),
		Action: "Split component into smaller components by responsibility boundary. Extract sub-sections into separate files.",
		File:   diag.File,
		Risk:   diag.Severity,
		Verify: []string{"typecheck", "tests"},
	}
}

type ExtractRepeatedJSX struct {
	thresholds *Thresholds
}

func (r *ExtractRepeatedJSX) ID() string          { return "react.extract-repeated-jsx" }
func (r *ExtractRepeatedJSX) Description() string { return "Extract repeated JSX blocks into a reusable component" }

func (r *ExtractRepeatedJSX) Matches(comp *ir.ComponentIR) (*RecipeMatch, bool) {
	counts := map[string]int{}
	for _, b := range comp.JSXBlocks {
		counts[b.Element]++
	}
	for _, count := range counts {
		if count >= r.thresholds.RepeatedJSXCount {
			return &RecipeMatch{
				RecipeID:    r.ID(),
				Description: r.Description(),
				Confidence:  0.9,
			}, true
		}
	}
	return nil, false
}

func (r *ExtractRepeatedJSX) GenerateStep(diag ir.Diagnostic) ir.GrpStep {
	return ir.GrpStep{
		Recipe: r.ID(),
		Action: "Extract repeated JSX into a reusable component. Identify differing props and create a clean interface.",
		File:   diag.File,
		Risk:   ir.SeverityLow,
		Verify: []string{"typecheck", "lint"},
	}
}

type ExtractCustomHook struct {
	thresholds *Thresholds
}

func (r *ExtractCustomHook) ID() string          { return "react.extract-custom-hook" }
func (r *ExtractCustomHook) Description() string { return "Extract related hook logic into a custom hook" }

func (r *ExtractCustomHook) Matches(comp *ir.ComponentIR) (*RecipeMatch, bool) {
	if len(comp.Hooks) > r.thresholds.MaxHooksPerComponent || (len(comp.Effects) > 1 && len(comp.StateVars) > 2) {
		return &RecipeMatch{
			RecipeID:    r.ID(),
			Description: r.Description(),
			Confidence:  0.75,
		}, true
	}
	if comp.HasKeyDown && len(comp.StateVars) > 0 {
		return &RecipeMatch{
			RecipeID:    r.ID(),
			Description: "Extract keyboard event handling into a custom useKeyDown hook",
			Confidence:  0.85,
		}, true
	}
	return nil, false
}

func (r *ExtractCustomHook) GenerateStep(diag ir.Diagnostic) ir.GrpStep {
	return ir.GrpStep{
		Recipe: r.ID(),
		Action: "Extract related state and effects into a custom hook. Move hook logic to a separate file.",
		File:   diag.File,
		Risk:   ir.SeverityLow,
		Verify: []string{"typecheck", "tests"},
	}
}

type ReduceStateVars struct {
	thresholds *Thresholds
}

func (r *ReduceStateVars) ID() string          { return "react.reduce-state-vars" }
func (r *ReduceStateVars) Description() string { return "Consolidate multiple state variables into a reducer or grouped state" }

func (r *ReduceStateVars) Matches(comp *ir.ComponentIR) (*RecipeMatch, bool) {
	if len(comp.StateVars) > r.thresholds.MaxStateVars {
		return &RecipeMatch{
			RecipeID:    r.ID(),
			Description: r.Description(),
			Confidence:  0.7,
		}, true
	}
	return nil, false
}

func (r *ReduceStateVars) GenerateStep(diag ir.Diagnostic) ir.GrpStep {
	return ir.GrpStep{
		Recipe: r.ID(),
		Action: "Consolidate related useState calls into useReducer or a single state object.",
		File:   diag.File,
		Risk:   ir.SeverityMedium,
		Verify: []string{"typecheck", "tests"},
	}
}

type ConsolidateEffects struct {
	thresholds *Thresholds
}

func (r *ConsolidateEffects) ID() string          { return "react.consolidate-effects" }
func (r *ConsolidateEffects) Description() string { return "Consolidate multiple useEffect calls" }

func (r *ConsolidateEffects) Matches(comp *ir.ComponentIR) (*RecipeMatch, bool) {
	if len(comp.Effects) > r.thresholds.MaxEffects {
		return &RecipeMatch{
			RecipeID:    r.ID(),
			Description: r.Description(),
			Confidence:  0.6,
		}, true
	}
	return nil, false
}

func (r *ConsolidateEffects) GenerateStep(diag ir.Diagnostic) ir.GrpStep {
	return ir.GrpStep{
		Recipe: r.ID(),
		Action: "Merge related useEffect calls or extract logic into custom hooks.",
		File:   diag.File,
		Risk:   ir.SeverityMedium,
		Verify: []string{"typecheck", "tests"},
	}
}

type AddPropTypes struct {
	thresholds *Thresholds
}

func (r *AddPropTypes) ID() string          { return "react.add-prop-types" }
func (r *AddPropTypes) Description() string { return "Add TypeScript interfaces for component props" }

func (r *AddPropTypes) Matches(comp *ir.ComponentIR) (*RecipeMatch, bool) {
	if len(comp.Props) == 0 {
		return nil, false
	}
	hasTypes := false
	for _, p := range comp.Props {
		if p.Type != "" {
			hasTypes = true
			break
		}
	}
	if !hasTypes && comp.Lines > r.thresholds.UntypedPropsMinLines {
		return &RecipeMatch{
			RecipeID:    r.ID(),
			Description: r.Description(),
			Confidence:  0.6,
		}, true
	}
	return nil, false
}

func (r *AddPropTypes) GenerateStep(diag ir.Diagnostic) ir.GrpStep {
	return ir.GrpStep{
		Recipe: r.ID(),
		Action: "Define TypeScript interface for component props with proper types for each prop.",
		File:   diag.File,
		Risk:   ir.SeverityLow,
		Verify: []string{"typecheck"},
	}
}
