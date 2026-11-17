package decisions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/canadian-ai/girl/internal/extensions"
)

// fakeProvider returns fixed answers for the triage and guard questions.
type fakeProvider struct {
	fp, review, highRisk, riskScore, conf float64
	recipe                                string
	// realIssue and safe answer the negated companions; nil means the
	// consistent complement (1 - p).
	realIssue, safe *float64
	// guard answers
	inScope, matches, behavior float64
	err                        error
	calls                      int32
	lastState                  any
}

func (f *fakeProvider) Name() string { return "fake" }

func (f *fakeProvider) Decide(_ context.Context, req Request) (*Result, error) {
	atomic.AddInt32(&f.calls, 1)
	f.lastState = req.State
	if f.err != nil {
		return nil, f.err
	}
	p := func(v float64) *float64 { return &v }
	or := func(v *float64, d float64) *float64 {
		if v != nil {
			return v
		}
		return &d
	}
	all := map[string]Answer{
		"false_positive":  {Type: TypeNoul, Noul: p(f.fp)},
		"needs_review":    {Type: TypeNoul, Noul: p(f.review)},
		"real_issue":      {Type: TypeNoul, Noul: or(f.realIssue, 1-f.fp)},
		"safe_unreviewed": {Type: TypeNoul, Noul: or(f.safe, 1-f.review)},
		"recipe":          {Type: TypeChoice, Choice: f.recipe, Confidence: f.conf},
		"risk":            {Type: TypeScore, Score: p(f.riskScore), Confidence: 0.7, Probabilities: map[string]float64{"0": 1 - f.highRisk, "2": f.highRisk}},
		"in_scope":        {Type: TypeNoul, Noul: p(f.inScope)},
		"matches_recipe":  {Type: TypeNoul, Noul: p(f.matches)},
		"behavior_change": {Type: TypeNoul, Noul: p(f.behavior)},
	}
	answers := map[string]Answer{}
	for key := range req.Questions {
		answers[key] = all[key]
	}
	return &Result{Provider: "fake", Model: "fake-1", InputHash: "hash", Answers: answers}, nil
}

func pyFinding(t *testing.T) extensions.Finding {
	t.Helper()
	f, err := extensions.Normalize(extensions.Finding{Provider: "sonarqube", ExternalID: "AX1", RuleID: "python:S5332", File: "pkg/api.py", Line: 9, Severity: "medium", Message: "Use https"})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestCandidateRecipesComeFromTheRegistry(t *testing.T) {
	py := CandidateRecipes("pkg/app.py")
	tsx := CandidateRecipes("src/App.tsx")
	if len(py) == 0 || len(tsx) == 0 || CandidateRecipes("README.md") != nil {
		t.Fatalf("py=%d tsx=%d", len(py), len(tsx))
	}
	hasReact := false
	for _, o := range append(py, tsx...) {
		if strings.HasSuffix(o.ID, ".repair-syntax") {
			t.Fatalf("parser repair must never be triaged: %s", o.ID)
		}
		hasReact = hasReact || strings.HasPrefix(o.ID, "react.")
	}
	for _, o := range py {
		if !strings.HasPrefix(o.ID, "py.") {
			t.Fatalf("python got %s", o.ID)
		}
	}
	if !hasReact {
		t.Fatal("tsx files should be offered react recipes")
	}
}

func TestTriageEffectsFollowThresholds(t *testing.T) {
	f := pyFinding(t)
	opts := CandidateRecipes(f.File)
	cases := []struct {
		name string
		p    fakeProvider
		want []string
	}{
		{"confident recipe", fakeProvider{recipe: "py.extract-function", conf: 0.9}, []string{EffectSuggestRecipe}},
		{"unsure recipe", fakeProvider{recipe: "py.extract-function", conf: 0.5}, nil},
		{"manual", fakeProvider{recipe: ManualRecipe, conf: 0.99}, nil},
		{"needs review", fakeProvider{recipe: ManualRecipe, conf: 0.9, review: 0.6}, []string{EffectEscalateReview}},
		{"high risk", fakeProvider{recipe: ManualRecipe, conf: 0.9, highRisk: 0.4}, []string{EffectEscalateReview}},
		{"false positive", fakeProvider{recipe: ManualRecipe, conf: 0.9, fp: 0.9}, []string{EffectDeprioritize}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr, err := TriageFinding(context.Background(), &tc.p, f, opts, DefaultThresholds())
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(tr.Effects, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("effects = %v, want %v", tr.Effects, tc.want)
			}
		})
	}
}

func TestInvalidOrFailedDecisionsHaveNoEffect(t *testing.T) {
	f := pyFinding(t)
	// A recipe that was not offered is an invalid answer: abstain.
	tr, err := TriageFinding(context.Background(), &fakeProvider{recipe: "rm.rf", conf: 1}, f, CandidateRecipes(f.File), DefaultThresholds())
	if err != nil || !tr.Abstained || len(tr.Effects) != 0 {
		t.Fatalf("invalid answer trusted: %+v err=%v", tr, err)
	}
	if _, ok := SuggestedAction(f, tr); ok {
		t.Fatal("abstained triage produced an action")
	}
	if _, err := TriageFinding(context.Background(), &fakeProvider{err: ErrUnavailable}, f, nil, DefaultThresholds()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("provider error swallowed: %v", err)
	}
}

// The core safety property: for every base decision and every possible
// triage, the combined decision is never less strict than GIRL's own.
func TestCombineIsNeverLooserThanPolicy(t *testing.T) {
	bases := []extensions.Decision{{Status: extensions.Allow}, {Status: extensions.Review}, {Status: extensions.Deny}}
	f := pyFinding(t)
	for _, review := range []float64{0, 0.49, 0.5, 1} {
		for _, high := range []float64{0, 0.29, 0.3, 1} {
			for _, fp := range []float64{0, 0.8, 1} {
				for _, conf := range []float64{0, 0.8, 1} {
					p := &fakeProvider{review: review, highRisk: high, fp: fp, conf: conf, recipe: "py.extract-function"}
					tr, err := TriageFinding(context.Background(), p, f, CandidateRecipes(f.File), DefaultThresholds())
					if err != nil {
						t.Fatal(err)
					}
					for _, b := range bases {
						got := Combine(b, tr)
						if !Stricter(got, b) {
							t.Fatalf("decision loosened %s -> %s (review=%.2f high=%.2f fp=%.2f)", b.Status, got.Status, review, high, fp)
						}
						if b.Status == extensions.Allow && tr.has(EffectEscalateReview) && got.Status != extensions.Review {
							t.Fatalf("escalation ignored: %+v", got)
						}
					}
				}
			}
		}
	}
	if got := Combine(extensions.Decision{Status: extensions.Allow}, nil); got.Status != extensions.Allow {
		t.Fatal("nil triage changed the decision")
	}
}

func TestSuggestedActionStillNeedsPolicy(t *testing.T) {
	f := pyFinding(t)
	tr, err := TriageFinding(context.Background(), &fakeProvider{recipe: "py.extract-function", conf: 0.95, riskScore: 1.6}, f, CandidateRecipes(f.File), DefaultThresholds())
	if err != nil {
		t.Fatal(err)
	}
	a, ok := SuggestedAction(f, tr)
	if !ok || a.Recipe != "py.extract-function" || a.Risk != "high" || a.Files[0] != f.File {
		t.Fatalf("action = %+v ok=%v", a, ok)
	}
	if d := extensions.Evaluate(*a, extensions.Policy{}); d.Status != extensions.Deny {
		t.Fatalf("a model suggestion was allowed without a policy: %+v", d)
	}
	policy := extensions.Policy{AllowedPaths: []string{"pkg/**"}, MaxFiles: 1, MaxChangedLines: 200}
	if d := Combine(extensions.Evaluate(*a, policy), tr); d.Status == extensions.Allow {
		t.Fatalf("high-risk suggestion skipped review: %+v", d)
	}
}

func TestOnlySanitizedMetadataIsSent(t *testing.T) {
	secret := strings.Join([]string{"ghp", "_", "SYNTHETICabcdefghijklmnop0123456789AB"}, "")
	f := pyFinding(t)
	f.Message = "hard-coded token " + secret + " in call"
	p := &fakeProvider{recipe: ManualRecipe, conf: 0.9}
	if _, err := TriageFinding(context.Background(), p, f, CandidateRecipes(f.File), DefaultThresholds()); err != nil {
		t.Fatal(err)
	}
	sent := strings.ToLower(strings.Join(flatten(p.lastState), " "))
	if strings.Contains(sent, strings.ToLower(secret)) {
		t.Fatal("secret sent to the decision provider")
	}
	if !strings.Contains(sent, "pkg/api.py") || !strings.Contains(sent, "python:s5332") {
		t.Fatalf("finding metadata missing: %s", sent)
	}
}

func flatten(v any) []string {
	switch x := v.(type) {
	case map[string]any:
		var out []string
		for _, e := range x {
			out = append(out, flatten(e)...)
		}
		return out
	default:
		return []string{fmt.Sprint(x)}
	}
}

func TestRecipeOptionsDoNotLeakTheRegistryMapping(t *testing.T) {
	for _, file := range []string{"a.py", "a.go", "a.rs", "a.tsx"} {
		for _, o := range CandidateRecipes(file) {
			if strings.Contains(o.Description, ".") || strings.Contains(o.Description, "registered") {
				t.Fatalf("%s description names a diagnostic code: %q", o.ID, o.Description)
			}
		}
	}
}
