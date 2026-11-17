package decisions

import (
	"fmt"
	"math"
)

// Evaluation settings. Below MinSamples labeled cases, no threshold is ever
// suggested: a handful of outcomes is not evidence.
const (
	MinSamples      = 30
	MinFlagged      = 5
	TargetPrecision = 0.9
)

// Bucket is one calibration bin: how often the event actually happened among
// cases the model scored in [Low, High).
type Bucket struct {
	Low      float64 `json:"low"`
	High     float64 `json:"high"`
	Count    int     `json:"count"`
	MeanP    float64 `json:"meanPredicted"`
	Observed float64 `json:"observedRate"`
}

// SignalEval measures one decision signal against recorded outcomes.
type SignalEval struct {
	Signal    string  `json:"signal"`
	Samples   int     `json:"samples"`
	Positives int     `json:"positives"`
	Threshold float64 `json:"threshold"`
	// Flagged counts cases at or above the threshold; Precision is the share
	// of those that were truly positive; Recall the share of positives caught.
	Flagged   int      `json:"flagged"`
	Precision float64  `json:"precision"`
	Recall    float64  `json:"recall"`
	Brier     float64  `json:"brier"`
	Buckets   []Bucket `json:"calibration,omitempty"`
	// Suggested is the lowest threshold reaching TargetPrecision with enough
	// support; nil when the data cannot support a recommendation.
	Suggested *float64 `json:"suggestedThreshold,omitempty"`
	Verdict   string   `json:"verdict"`
}

// EvalReport is the result of replaying the ledger.
type EvalReport struct {
	Decisions int          `json:"decisions"`
	Labeled   int          `json:"labeled"`
	Signals   []SignalEval `json:"signals"`
	Note      string       `json:"note"`
}

type sample struct {
	p     float64
	label bool
}

func evalSignal(name string, samples []sample, threshold float64) SignalEval {
	ev := SignalEval{Signal: name, Samples: len(samples), Threshold: threshold}
	if len(samples) == 0 {
		ev.Verdict = "no labeled outcomes yet"
		return ev
	}
	const bins = 5
	sums := make([]float64, bins)
	hits := make([]int, bins)
	counts := make([]int, bins)
	var brier float64
	tp := 0
	for _, s := range samples {
		y := 0.0
		if s.label {
			y = 1
			ev.Positives++
		}
		brier += (s.p - y) * (s.p - y)
		b := int(s.p * bins)
		if b >= bins {
			b = bins - 1
		}
		counts[b]++
		sums[b] += s.p
		if s.label {
			hits[b]++
		}
		if s.p >= threshold {
			ev.Flagged++
			if s.label {
				tp++
			}
		}
	}
	ev.Brier = round3(brier / float64(len(samples)))
	if ev.Flagged > 0 {
		ev.Precision = round3(float64(tp) / float64(ev.Flagged))
	}
	if ev.Positives > 0 {
		ev.Recall = round3(float64(tp) / float64(ev.Positives))
	}
	for b := 0; b < bins; b++ {
		if counts[b] == 0 {
			continue
		}
		ev.Buckets = append(ev.Buckets, Bucket{
			Low: float64(b) / bins, High: float64(b+1) / bins, Count: counts[b],
			MeanP: round3(sums[b] / float64(counts[b])), Observed: round3(float64(hits[b]) / float64(counts[b])),
		})
	}
	if len(samples) < MinSamples {
		ev.Verdict = fmt.Sprintf("insufficient data: %d of %d labeled outcomes needed before thresholds can be tuned", len(samples), MinSamples)
		return ev
	}
	// Step in whole hundredths: accumulating 0.05 drifts (0.5+0.05+0.05 is
	// just above 0.6) and would suggest a threshold the data does not support.
	for step := 50; step <= 95; step += 5 {
		t := float64(step) / 100
		flagged, hit := 0, 0
		for _, s := range samples {
			if s.p >= t {
				flagged++
				if s.label {
					hit++
				}
			}
		}
		if flagged >= MinFlagged && float64(hit)/float64(flagged) >= TargetPrecision {
			v := t
			ev.Suggested = &v
			break
		}
	}
	switch {
	case ev.Suggested == nil:
		ev.Verdict = fmt.Sprintf("no threshold reaches %.0f%% precision; keep this signal in shadow mode", TargetPrecision*100)
	case math.Abs(*ev.Suggested-threshold) < 0.001:
		ev.Verdict = "current threshold is supported by the data"
	default:
		ev.Verdict = fmt.Sprintf("data supports threshold %.2f (currently %.2f)", *ev.Suggested, threshold)
	}
	return ev
}

func round3(v float64) float64 { return math.Round(v*1000) / 1000 }

// neededReview derives the "needed review" label: an explicit label wins;
// otherwise a failed verification or a revert means review was needed, and a
// passed, unreverted change means it was not.
func neededReview(o Outcome) (bool, bool) {
	if o.NeededReview != nil {
		return *o.NeededReview, true
	}
	if (o.Reverted != nil && *o.Reverted) || (o.VerificationPassed != nil && !*o.VerificationPassed) {
		return true, true
	}
	if o.VerificationPassed != nil && *o.VerificationPassed {
		return false, true
	}
	return false, false
}

// Evaluate replays ledger cases against the thresholds in use.
func Evaluate(cases []Case, th Thresholds) *EvalReport {
	r := &EvalReport{Decisions: len(cases),
		Note: "Precision is measured only on findings with recorded outcomes. Suggestions need at least 30 labeled cases and are never applied automatically."}
	var fp, review, recipe []sample
	for _, c := range cases {
		if !c.Labeled {
			continue
		}
		r.Labeled++
		if c.Outcome.FalsePositive != nil {
			fp = append(fp, sample{c.Decision.FalsePositive, *c.Outcome.FalsePositive})
		}
		if label, ok := neededReview(c.Outcome); ok {
			p := math.Max(c.Decision.NeedsReview, c.Decision.HighRisk)
			review = append(review, sample{p, label})
		}
		if c.Outcome.RecipeCorrect != nil && c.Decision.Recipe != "" {
			recipe = append(recipe, sample{c.Decision.RecipeConfidence, *c.Outcome.RecipeCorrect})
		}
	}
	r.Signals = []SignalEval{
		evalSignal("false_positive", fp, th.FalsePositiveProbability),
		evalSignal("needs_review", review, th.ReviewProbability),
		evalSignal("recipe", recipe, th.MinRecipeConfidence),
	}
	return r
}
