package decisions

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/canadian-ai/girl/internal/extensions"
	"github.com/canadian-ai/girl/internal/ir"
	"github.com/canadian-ai/girl/internal/recipes"
	"github.com/canadian-ai/girl/internal/redact"
)

// ManualRecipe is always offered: the model may always say "no recipe fits".
const ManualRecipe = "manual"

// Thresholds turn probabilities into effects. A value that does not clear its
// threshold has no effect at all: GIRL falls back to its deterministic path.
type Thresholds struct {
	// MinRecipeConfidence is the confidence needed to suggest a recipe.
	MinRecipeConfidence float64
	// ReviewProbability is P(needs review) at which an allowed action is
	// escalated to human review.
	ReviewProbability float64
	// HighRiskProbability is P(risk = high) at which an allowed action is
	// escalated to human review.
	HighRiskProbability float64
	// FalsePositiveProbability is P(false positive) at which a finding is
	// deprioritized. It is never suppressed and never removed from a gate.
	FalsePositiveProbability float64
}

func DefaultThresholds() Thresholds {
	return Thresholds{MinRecipeConfidence: 0.8, ReviewProbability: 0.5, HighRiskProbability: 0.3, FalsePositiveProbability: 0.8}
}

// RecipeOption is a recipe the model may choose.
type RecipeOption struct {
	ID          string
	Description string
}

// Modes. Shadow is the default: answers are recorded and reported but change
// nothing, so a model can be measured on real work before it is trusted.
const (
	ModeOff     = "off"
	ModeShadow  = "shadow"
	ModeEnforce = "enforce"
)

// MaxInconsistency is how far a question and its negation may disagree
// (|p + q - 1|) before that signal is discarded.
const MaxInconsistency = 0.35

// TriageConfig controls one triage call.
type TriageConfig struct {
	Thresholds Thresholds
	// Mode is ModeShadow (default) or ModeEnforce.
	Mode string
	// Signals are extra facts about the finding (metrics, churn, test
	// presence). They pass the sanitizer with the rest of the state.
	Signals map[string]any
	// Consistency asks each yes/no question together with its negation and
	// discards signals whose two answers disagree.
	Consistency bool
}

// Triage is the decision record for one finding.
type Triage struct {
	Fingerprint string `json:"fingerprint"`
	Code        string `json:"code"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Provider    string `json:"provider,omitempty"`
	Model       string `json:"model,omitempty"`
	InputHash   string `json:"inputHash,omitempty"`
	Cached      bool   `json:"cached,omitempty"`

	FalsePositive    float64            `json:"falsePositive"`
	NeedsReview      float64            `json:"needsReview"`
	Recipe           string             `json:"recipe,omitempty"`
	RecipeConfidence float64            `json:"recipeConfidence"`
	RiskScore        float64            `json:"riskScore"`
	RiskLevels       map[string]float64 `json:"riskProbabilities,omitempty"`

	// Mode is the mode the triage ran in.
	Mode string `json:"mode"`
	// Effects lists what the decision changes. Empty means no effect. In
	// shadow mode it is always empty and WouldEffects shows what enforce
	// mode would have done.
	Effects      []string `json:"effects"`
	WouldEffects []string `json:"wouldEffects,omitempty"`
	// Inconsistent lists signals discarded because the model contradicted
	// itself on a question and its negation.
	Inconsistent []string `json:"inconsistent,omitempty"`
	// Priority ranks findings: higher means more worth doing first (severe,
	// likely real, low risk). It never removes a finding.
	Priority float64 `json:"priority"`
	// NumericSignals are the numeric extra facts sent, kept for the ledger.
	NumericSignals map[string]float64 `json:"signals,omitempty"`
	// Abstained is true when the model was not consulted or its answer was
	// discarded; Reason says why.
	Abstained bool   `json:"abstained,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// Effects a triage can have. All of them are stricter, never looser.
const (
	EffectEscalateReview = "escalate-to-review"
	EffectDeprioritize   = "deprioritize"
	EffectSuggestRecipe  = "suggest-recipe"
)

// riskLevels are the score criteria, lowest first.
var riskLevels = []string{"low: a mechanical change, tests cover it", "medium: behaviour could change in edge cases", "high: likely to change behaviour or needs domain knowledge"}

// LanguageOf maps a file to a GRP binding prefix.
func LanguageOf(file string) string {
	switch strings.ToLower(path.Ext(file)) {
	case ".py":
		return "py"
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
		return "ts"
	}
	return ""
}

// CandidateRecipes lists registered refactoring recipes for the file's
// language. Repair recipes are excluded: parser failures are never triaged.
func CandidateRecipes(file string) []RecipeOption {
	lang := LanguageOf(file)
	if lang == "" {
		return nil
	}
	prefixes := []string{lang + "."}
	if ext := strings.ToLower(path.Ext(file)); ext == ".tsx" || ext == ".jsx" {
		prefixes = append(prefixes, "react.")
	}
	seen := map[string]bool{}
	var out []RecipeOption
	for _, r := range recipes.Registered() {
		if seen[r.Recipe] || strings.HasSuffix(r.Recipe, ".repair-syntax") {
			continue
		}
		for _, p := range prefixes {
			if strings.HasPrefix(r.Recipe, p) {
				seen[r.Recipe] = true
				name := strings.ReplaceAll(strings.TrimPrefix(r.Recipe, p), "-", " ")
				// Describe the recipe by what it does only. Naming the
				// diagnostic it is registered for would hand the model the
				// answer and make agreement with the registry meaningless.
				out = append(out, RecipeOption{ID: r.Recipe, Description: name})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// FindingFromDiagnostic adapts a GIRL diagnostic for triage.
func FindingFromDiagnostic(d ir.Diagnostic) extensions.Finding {
	f := extensions.Finding{Provider: "girl", RuleID: d.Code, File: d.File, Line: d.Line, Severity: string(d.Severity), Message: d.Message}
	if n, err := extensions.Normalize(f); err == nil {
		return n
	}
	return f
}

// buildRequest builds the questions for one finding. Only finding metadata is
// sent, never source code, and it passes the shared sanitizer first.
func buildRequest(f extensions.Finding, options []RecipeOption, cfg TriageConfig) (Request, error) {
	criteria := map[string]string{ManualRecipe: "No listed recipe fits; a human should handle it"}
	for _, o := range options {
		criteria[o.ID] = o.Description
	}
	state := map[string]any{
		"finding": map[string]any{
			"source":   f.Provider,
			"rule":     f.RuleID,
			"severity": f.Severity,
			"file":     f.File,
			"line":     f.Line,
			"message":  redact.String(f.Message),
		},
		"context": "Static-analysis finding in a software repository. Answers decide triage only; they never approve a change.",
	}
	if len(cfg.Signals) > 0 {
		signals := map[string]any{}
		for k, v := range cfg.Signals {
			if sv, ok := v.(string); ok {
				v = redact.String(sv)
			}
			signals[k] = v
		}
		state["signals"] = signals
	}
	if findings := redact.ScanValue(state); len(findings) > 0 {
		return Request{}, fmt.Errorf("finding %s: sanitization could not be established (%s); not sent", f.Fingerprint, findings[0].Detector)
	}
	questions := map[string]Question{
		"false_positive": {Type: TypeNoul, Instructions: "This finding is a false positive: the code is acceptable as written and needs no change."},
		"recipe":         {Type: TypeChoice, Instructions: "Which listed refactoring recipe best addresses this finding? Choose manual if none clearly fits.", Criteria: criteria},
		"needs_review":   {Type: TypeNoul, Instructions: "An automated refactor for this finding should be reviewed by a human before it is applied."},
		"risk":           {Type: TypeScore, Instructions: "How risky would an automated refactor for this finding be?", Criteria: riskLevels},
	}
	if cfg.Consistency {
		// Negated companions: a model that answers both a statement and its
		// opposite with "yes" is not telling us anything.
		questions["real_issue"] = Question{Type: TypeNoul, Instructions: "This finding is a real issue: the code should be changed to address it."}
		questions["safe_unreviewed"] = Question{Type: TypeNoul, Instructions: "An automated refactor for this finding is safe to apply without any human review."}
	}
	return Request{State: state, Questions: questions}, nil
}

var severityWeight = map[string]float64{"critical": 4, "high": 3, "medium": 2, "low": 1, "info": 0.5}

// priority is severity x P(real) x a risk discount, rounded for stable output.
func priority(severity string, falsePositive, riskScore float64) float64 {
	w, ok := severityWeight[severity]
	if !ok {
		w = 1
	}
	v := w * (1 - falsePositive) * (1 - 0.25*riskScore)
	return float64(int(v*1000+0.5)) / 1000
}

// TriageFinding triages with thresholds in enforce mode and no extras. It is
// the minimal entry point; commands use TriageFindingWith.
func TriageFinding(ctx context.Context, p Provider, f extensions.Finding, options []RecipeOption, th Thresholds) (*Triage, error) {
	return TriageFindingWith(ctx, p, f, options, TriageConfig{Thresholds: th, Mode: ModeEnforce})
}

// TriageFindingWith asks the provider about one finding and derives effects.
// Errors from the provider are returned; the caller should treat them as an
// abstention (no effect), never as a pass.
func TriageFindingWith(ctx context.Context, p Provider, f extensions.Finding, options []RecipeOption, cfg TriageConfig) (*Triage, error) {
	if cfg.Mode != ModeEnforce {
		cfg.Mode = ModeShadow
	}
	th := cfg.Thresholds
	t := &Triage{Fingerprint: f.Fingerprint, Code: f.RuleID, File: f.File, Line: f.Line, Mode: cfg.Mode, Effects: []string{}}
	for k, v := range cfg.Signals {
		if n, ok := toFloat(v); ok {
			if t.NumericSignals == nil {
				t.NumericSignals = map[string]float64{}
			}
			t.NumericSignals[k] = n
		}
	}
	req, err := buildRequest(f, options, cfg)
	if err != nil {
		t.Abstained, t.Reason = true, err.Error()
		return t, nil
	}
	res, err := p.Decide(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := Validate(req, res); err != nil {
		t.Abstained, t.Reason = true, err.Error()
		return t, nil
	}
	t.Provider, t.Model, t.InputHash, t.Cached = res.Provider, res.Model, res.InputHash, res.Cached
	t.FalsePositive = *res.Answers["false_positive"].Noul
	t.NeedsReview = *res.Answers["needs_review"].Noul
	rec := res.Answers["recipe"]
	t.Recipe, t.RecipeConfidence = rec.Choice, rec.Confidence
	risk := res.Answers["risk"]
	t.RiskScore, t.RiskLevels = *risk.Score, risk.Probabilities

	fpTrusted, reviewTrusted := true, true
	if cfg.Consistency {
		if d := *res.Answers["real_issue"].Noul + t.FalsePositive - 1; d > MaxInconsistency || d < -MaxInconsistency {
			fpTrusted = false
			t.Inconsistent = append(t.Inconsistent, "false_positive")
		}
		if d := *res.Answers["safe_unreviewed"].Noul + t.NeedsReview - 1; d > MaxInconsistency || d < -MaxInconsistency {
			reviewTrusted = false
			t.Inconsistent = append(t.Inconsistent, "needs_review")
		}
	}

	var effects []string
	// An untrusted review signal escalates: when the model contradicts itself
	// about safety, the stricter reading wins.
	if !reviewTrusted || t.NeedsReview >= th.ReviewProbability || t.RiskLevels["2"] >= th.HighRiskProbability {
		effects = append(effects, EffectEscalateReview)
	}
	if fpTrusted && t.FalsePositive >= th.FalsePositiveProbability {
		effects = append(effects, EffectDeprioritize)
	}
	if t.Recipe != ManualRecipe && t.RecipeConfidence >= th.MinRecipeConfidence && offered(options, t.Recipe) {
		effects = append(effects, EffectSuggestRecipe)
	}
	fpForRank := t.FalsePositive
	if !fpTrusted {
		fpForRank = 0.5
	}
	t.Priority = priority(f.Severity, fpForRank, t.RiskScore)
	if cfg.Mode == ModeEnforce {
		if effects != nil {
			t.Effects = effects
		}
	} else {
		t.WouldEffects = effects
	}
	return t, nil
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case bool:
		if n {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func offered(options []RecipeOption, id string) bool {
	for _, o := range options {
		if o.ID == id {
			return true
		}
	}
	return false
}

func (t *Triage) has(effect string) bool {
	for _, e := range t.Effects {
		if e == effect {
			return true
		}
	}
	return false
}

var strictness = map[string]int{extensions.Allow: 0, extensions.Review: 1, extensions.Deny: 2}

// Combine applies a triage to GIRL's own policy decision. The result is never
// less strict than base: deny stays deny, review stays review, and allow can
// only move to review. This is the only way a decision model influences
// whether a change may proceed.
func Combine(base extensions.Decision, t *Triage) extensions.Decision {
	if t == nil || t.Abstained || base.Status != extensions.Allow || !t.has(EffectEscalateReview) {
		return base
	}
	return extensions.Decision{
		Status: extensions.Review,
		Reasons: append(append([]string{}, base.Reasons...),
			fmt.Sprintf("decision model %s requests human review (needs review %.2f, high risk %.2f)", t.Model, t.NeedsReview, t.RiskLevels["2"])),
	}
}

// Stricter reports whether a is at least as strict as b.
func Stricter(a, b extensions.Decision) bool { return strictness[a.Status] >= strictness[b.Status] }

// DefaultChangedLines is the budget requested by a suggested action.
const DefaultChangedLines = 80

// SuggestedAction turns a confident recipe choice into an action proposal.
// It carries no authority: it must still pass extensions.Evaluate under an
// explicit policy, and then Combine with the same triage.
func SuggestedAction(f extensions.Finding, t *Triage) (*extensions.Action, bool) {
	if t == nil || t.Abstained || !t.has(EffectSuggestRecipe) || f.Fingerprint == "" {
		return nil, false
	}
	risk := "low"
	switch {
	case t.RiskScore >= 1.5:
		risk = "high"
	case t.RiskScore >= 0.5:
		risk = "medium"
	}
	return &extensions.Action{
		ID:           "jev_" + f.Fingerprint[:16],
		Finding:      f,
		Recipe:       t.Recipe,
		Files:        []string{f.File},
		ChangedLines: DefaultChangedLines,
		Risk:         risk,
		Category:     f.Provenance["category"],
	}, true
}
