package decisions

import (
	"context"
	"testing"

	"github.com/canadian-ai/girl/internal/extensions"
)

func TestShadowModeRecordsButHasNoEffect(t *testing.T) {
	f := pyFinding(t)
	p := &fakeProvider{recipe: "py.extract-function", conf: 0.95, review: 0.9}
	shadow, err := TriageFindingWith(context.Background(), p, f, CandidateRecipes(f.File), TriageConfig{Thresholds: DefaultThresholds()})
	if err != nil {
		t.Fatal(err)
	}
	if shadow.Mode != ModeShadow || len(shadow.Effects) != 0 || len(shadow.WouldEffects) != 2 {
		t.Fatalf("shadow = %+v", shadow)
	}
	// No effect means no effect: policy outcome unchanged, no action proposed.
	if got := Combine(extensions.Decision{Status: extensions.Allow}, shadow); got.Status != extensions.Allow {
		t.Fatalf("shadow triage changed a decision: %+v", got)
	}
	if _, ok := SuggestedAction(f, shadow); ok {
		t.Fatal("shadow triage proposed an action")
	}
	enforce, _ := TriageFindingWith(context.Background(), p, f, CandidateRecipes(f.File), TriageConfig{Thresholds: DefaultThresholds(), Mode: ModeEnforce})
	if len(enforce.Effects) != 2 || len(enforce.WouldEffects) != 0 {
		t.Fatalf("enforce = %+v", enforce)
	}
	// An unknown mode is treated as shadow, never as enforce.
	odd, _ := TriageFindingWith(context.Background(), p, f, CandidateRecipes(f.File), TriageConfig{Thresholds: DefaultThresholds(), Mode: "ENFORCE!"})
	if odd.Mode != ModeShadow || len(odd.Effects) != 0 {
		t.Fatalf("unknown mode enforced: %+v", odd)
	}
}

func TestConsistencyCheckDiscardsContradictions(t *testing.T) {
	f := pyFinding(t)
	v := func(x float64) *float64 { return &x }
	cfg := TriageConfig{Thresholds: DefaultThresholds(), Mode: ModeEnforce, Consistency: true}

	// Says "false positive 0.9" AND "real issue 0.9": discard, do not deprioritize.
	p := &fakeProvider{recipe: ManualRecipe, conf: 0.9, fp: 0.9, realIssue: v(0.9)}
	tr, err := TriageFindingWith(context.Background(), p, f, CandidateRecipes(f.File), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if tr.has(EffectDeprioritize) || len(tr.Inconsistent) != 1 || tr.Inconsistent[0] != "false_positive" {
		t.Fatalf("contradictory false-positive answer trusted: %+v", tr)
	}

	// Says "needs review 0.1" AND "safe unreviewed 0.1": the stricter reading wins.
	p = &fakeProvider{recipe: ManualRecipe, conf: 0.9, review: 0.1, safe: v(0.1)}
	tr, _ = TriageFindingWith(context.Background(), p, f, CandidateRecipes(f.File), cfg)
	if !tr.has(EffectEscalateReview) {
		t.Fatalf("contradiction about safety did not escalate: %+v", tr)
	}

	// Consistent answers behave normally.
	p = &fakeProvider{recipe: ManualRecipe, conf: 0.9, fp: 0.9}
	tr, _ = TriageFindingWith(context.Background(), p, f, CandidateRecipes(f.File), cfg)
	if !tr.has(EffectDeprioritize) || len(tr.Inconsistent) != 0 {
		t.Fatalf("consistent answer discarded: %+v", tr)
	}
}

func TestSignalsAreSentSanitizedAndPriorityRanks(t *testing.T) {
	f := pyFinding(t)
	p := &fakeProvider{recipe: ManualRecipe, conf: 0.9, fp: 0.1}
	cfg := TriageConfig{Thresholds: DefaultThresholds(), Signals: map[string]any{"span_lines": 120, "has_test_file": true, "symbol_kind": "function"}}
	tr, err := TriageFindingWith(context.Background(), p, f, nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	state := p.lastState.(map[string]any)
	if state["signals"].(map[string]any)["span_lines"] != 120 {
		t.Fatalf("signals not sent: %+v", state)
	}
	if tr.NumericSignals["span_lines"] != 120 || tr.NumericSignals["has_test_file"] != 1 {
		t.Fatalf("numeric signals not kept for the ledger: %+v", tr.NumericSignals)
	}
	likelyFP, _ := TriageFindingWith(context.Background(), &fakeProvider{recipe: ManualRecipe, conf: 0.9, fp: 0.9}, f, nil, cfg)
	high := f
	high.Severity = "high"
	severe, _ := TriageFindingWith(context.Background(), &fakeProvider{recipe: ManualRecipe, conf: 0.9, fp: 0.1}, high, nil, cfg)
	if !(severe.Priority > tr.Priority && tr.Priority > likelyFP.Priority) {
		t.Fatalf("priority order wrong: severe=%.2f normal=%.2f likelyFP=%.2f", severe.Priority, tr.Priority, likelyFP.Priority)
	}
}
