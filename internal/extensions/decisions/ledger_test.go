package decisions

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func b(v bool) *bool { return &v }

func TestLedgerRoundTripAndJoin(t *testing.T) {
	l := &Ledger{Path: filepath.Join(t.TempDir(), "nested", "ledger.jsonl")}
	if entries, err := l.Read(); err != nil || entries != nil {
		t.Fatalf("missing ledger should be empty: %v %v", entries, err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(l.Append(LedgerEntry{Kind: KindOutcome, Fingerprint: "orphan", Outcome: &Outcome{Reverted: b(true)}}))
	must(l.Append(LedgerEntry{Kind: KindDecision, Fingerprint: "a", Code: "py.x", FalsePositive: 0.2}))
	must(l.Append(LedgerEntry{Kind: KindDecision, Fingerprint: "b", Code: "py.y"}))
	must(l.Append(LedgerEntry{Kind: KindDecision, Fingerprint: "a", Code: "py.x", FalsePositive: 0.9})) // re-triaged: latest wins
	must(l.Append(LedgerEntry{Kind: KindOutcome, Fingerprint: "a", Outcome: &Outcome{FalsePositive: b(true)}}))
	must(l.Append(LedgerEntry{Kind: KindOutcome, Fingerprint: "a", Outcome: &Outcome{VerificationPassed: b(false)}}))
	must(l.Append(LedgerEntry{Kind: KindDecision, Subject: "guard", Fingerprint: "g"}))
	if err := l.Append(LedgerEntry{Kind: KindDecision}); err == nil {
		t.Fatal("entry without fingerprint accepted")
	}
	info, _ := os.Stat(l.Path)
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("ledger readable by others: %v", info.Mode())
	}
	entries, err := l.Read()
	must(err)
	cases := Cases(entries)
	if len(cases) != 2 {
		t.Fatalf("cases = %+v", cases)
	}
	a := cases[0]
	if a.Decision.FalsePositive != 0.9 || !a.Labeled || a.Outcome.FalsePositive == nil || !*a.Outcome.FalsePositive || *a.Outcome.VerificationPassed {
		t.Fatalf("merged case = %+v", a)
	}
	if cases[1].Labeled {
		t.Fatal("unlabeled case marked labeled")
	}
	// A corrupt ledger is an error, not silently partial evidence.
	must(os.WriteFile(l.Path, []byte("{not json\n"), 0o600))
	if _, err := l.Read(); err == nil {
		t.Fatal("corrupt ledger accepted")
	}
}

func labeledCases(n int, p func(i int) float64, label func(i int) bool) []Case {
	var cases []Case
	for i := 0; i < n; i++ {
		cases = append(cases, Case{Labeled: true,
			Decision: LedgerEntry{Fingerprint: fmt.Sprint(i), FalsePositive: p(i)},
			Outcome:  Outcome{FalsePositive: b(label(i))}})
	}
	return cases
}

func TestEvaluateNeverSuggestsFromThinData(t *testing.T) {
	r := Evaluate(labeledCases(10, func(int) float64 { return 0.95 }, func(int) bool { return true }), DefaultThresholds())
	fp := r.Signals[0]
	if fp.Suggested != nil || fp.Samples != 10 || fp.Precision != 1 {
		t.Fatalf("suggested a threshold from 10 samples: %+v", fp)
	}
	empty := Evaluate(nil, DefaultThresholds())
	if empty.Labeled != 0 || empty.Signals[0].Suggested != nil {
		t.Fatalf("empty eval = %+v", empty)
	}
}

func TestEvaluateMeasuresCalibrationAndSuggestsThreshold(t *testing.T) {
	// 60 cases: scores of 0.9 are right, scores of 0.6 are wrong half the
	// time, scores of 0.2 are never positive.
	cases := labeledCases(60,
		func(i int) float64 { return []float64{0.9, 0.6, 0.2}[i%3] },
		func(i int) bool { return i%3 == 0 || (i%3 == 1 && i%2 == 0) })
	th := DefaultThresholds()
	th.FalsePositiveProbability = 0.5
	fp := Evaluate(cases, th).Signals[0]
	if fp.Samples != 60 || fp.Flagged != 40 || fp.Precision >= 0.9 {
		t.Fatalf("at 0.5 precision should be poor: %+v", fp)
	}
	if fp.Suggested == nil || *fp.Suggested < 0.65 || *fp.Suggested > 0.9 {
		t.Fatalf("expected a suggestion between the 0.6 and 0.9 groups: %+v", fp.Suggested)
	}
	if len(fp.Buckets) != 3 {
		t.Fatalf("calibration buckets = %+v", fp.Buckets)
	}

	// A model that is confidently wrong gets no suggestion at all.
	bad := Evaluate(labeledCases(60, func(int) float64 { return 0.95 }, func(i int) bool { return i%2 == 0 }), DefaultThresholds()).Signals[0]
	if bad.Suggested != nil {
		t.Fatalf("suggested a threshold for an unreliable signal: %+v", bad)
	}
}

func TestNeededReviewLabelIsDerivedFromOutcomes(t *testing.T) {
	for _, tc := range []struct {
		o        Outcome
		want, ok bool
	}{
		{Outcome{Reverted: b(true)}, true, true},
		{Outcome{VerificationPassed: b(false)}, true, true},
		{Outcome{VerificationPassed: b(true)}, false, true},
		{Outcome{VerificationPassed: b(true), Reverted: b(true)}, true, true},
		{Outcome{NeededReview: b(false), Reverted: b(true)}, false, true}, // explicit label wins
		{Outcome{FalsePositive: b(true)}, false, false},
	} {
		if got, ok := neededReview(tc.o); got != tc.want || ok != tc.ok {
			t.Errorf("%+v: got %v/%v", tc.o, got, ok)
		}
	}
}
