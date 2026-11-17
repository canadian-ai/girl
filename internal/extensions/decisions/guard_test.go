package decisions

import (
	"context"
	"strings"
	"testing"
)

func guardInput(files ...GuardFile) GuardInput {
	return GuardInput{Step: GuardStep{ID: "step_001", Recipe: "py.extract-function", Action: "Split apply_discounts into helpers", Files: []string{"pkg/billing.py"}}, Files: files}
}

func TestGuardRuleChecksNeedNoModel(t *testing.T) {
	cases := []struct {
		name  string
		files []GuardFile
		want  string // substring of a rule reason, "" for none
	}{
		{"in scope", []GuardFile{{Path: "pkg/billing.py", Added: 30, Deleted: 25}}, ""},
		{"with its test", []GuardFile{{Path: "pkg/billing.py", Added: 30, Deleted: 25}, {Path: "pkg/test_billing.py", Added: 12}}, ""},
		{"other file", []GuardFile{{Path: "pkg/billing.py", Added: 3}, {Path: "pkg/auth.py", Added: 9}}, "outside the planned step"},
		{"manifest", []GuardFile{{Path: "pkg/billing.py", Added: 3}, {Path: "pyproject.toml", Added: 1}}, "build, dependency or policy file"},
		{"gate policy", []GuardFile{{Path: ".girl/config.yaml", Added: 1, Deleted: 1}}, "build, dependency or policy file"},
		{"lockfile", []GuardFile{{Path: "pkg/billing.py", Added: 3}, {Path: "poetry.lock", Added: 40, Lockfile: true}}, "lockfile"},
		{"tests removed", []GuardFile{{Path: "pkg/billing.py", Added: 3}, {Path: "pkg/test_billing.py", Deleted: 40}}, "removes more test code"},
		{"too large", []GuardFile{{Path: "pkg/billing.py", Added: 300, Deleted: 10}}, "budget 200"},
		{"empty", nil, "diff is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := Guard(context.Background(), nil, guardInput(tc.files...), DefaultGuardThresholds(), ModeEnforce)
			joined := strings.Join(r.RuleReasons, "; ")
			if tc.want == "" {
				if r.Verdict != GuardNoObjection || len(r.RuleReasons) != 0 {
					t.Fatalf("unexpected objection: %v", r.RuleReasons)
				}
				return
			}
			if r.Verdict != GuardReview || !strings.Contains(joined, tc.want) {
				t.Fatalf("verdict=%s reasons=%v", r.Verdict, r.RuleReasons)
			}
		})
	}
}

func TestGuardModelCanOnlyAddObjections(t *testing.T) {
	ok := guardInput(GuardFile{Path: "pkg/billing.py", Added: 30, Deleted: 25})
	ok.Diff = "+def helper():\n+    return 1\n"
	bad := guardInput(GuardFile{Path: "pkg/billing.py", Added: 3}, GuardFile{Path: "pkg/auth.py", Added: 9})
	bad.Diff = ok.Diff
	happy := &fakeProvider{inScope: 0.95, matches: 0.95, behavior: 0.05}
	unhappy := &fakeProvider{inScope: 0.2, matches: 0.3, behavior: 0.8}

	// A glowing model answer never clears GIRL's own objection.
	if r := Guard(context.Background(), happy, bad, DefaultGuardThresholds(), ModeEnforce); r.Verdict != GuardReview {
		t.Fatalf("model cleared a rule objection: %+v", r)
	}
	// In enforce mode the model can object to an otherwise clean change.
	r := Guard(context.Background(), unhappy, ok, DefaultGuardThresholds(), ModeEnforce)
	if r.Verdict != GuardReview || len(r.ModelReasons) != 3 {
		t.Fatalf("enforce = %+v", r)
	}
	// In shadow mode the same answers are recorded but change nothing.
	r = Guard(context.Background(), unhappy, ok, DefaultGuardThresholds(), ModeShadow)
	if r.Verdict != GuardNoObjection || len(r.WouldReasons) != 3 || len(r.ModelReasons) != 0 {
		t.Fatalf("shadow = %+v", r)
	}
	// A provider failure neither clears nor raises anything.
	r = Guard(context.Background(), &fakeProvider{err: ErrUnavailable}, ok, DefaultGuardThresholds(), ModeEnforce)
	if r.Verdict != GuardNoObjection || !r.Abstained {
		t.Fatalf("provider error changed the verdict: %+v", r)
	}
	if r := Guard(context.Background(), &fakeProvider{err: ErrUnavailable}, bad, DefaultGuardThresholds(), ModeEnforce); r.Verdict != GuardReview {
		t.Fatalf("provider error cleared a rule objection: %+v", r)
	}
}

func TestGuardSendsDiffOnlyWhenAskedAndSanitized(t *testing.T) {
	secret := strings.Join([]string{"ghp", "_", "SYNTHETICabcdefghijklmnop0123456789AB"}, "")
	in := guardInput(GuardFile{Path: "pkg/billing.py", Added: 2, Deleted: 1})
	p := &fakeProvider{inScope: 0.9, matches: 0.9}
	// Without opt-in nothing is sent at all, even with alarming answers queued.
	silent := &fakeProvider{inScope: 0, matches: 0, behavior: 1}
	r := Guard(context.Background(), silent, in, DefaultGuardThresholds(), ModeEnforce)
	if silent.calls != 0 || r.DiffSent || !r.Abstained || r.Verdict != GuardNoObjection {
		t.Fatalf("model consulted without the diff: %+v", r)
	}
	in.Diff = "+TOKEN = \"" + secret + "\"\n+def helper():\n+    return 1\n"
	r = Guard(context.Background(), p, in, DefaultGuardThresholds(), ModeShadow)
	diff, _ := p.lastState.(map[string]any)["change"].(map[string]any)["diff"].(string)
	if !r.DiffSent || strings.Contains(diff, secret) || !strings.Contains(diff, "def helper") {
		t.Fatalf("diff not sanitized or not sent: %q", diff)
	}
	if GuardFingerprint(in) == GuardFingerprint(guardInput(GuardFile{Path: "pkg/billing.py", Added: 9})) {
		t.Fatal("guard fingerprint ignores the diff shape")
	}
}
