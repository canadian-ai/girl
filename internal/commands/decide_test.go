package commands

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/canadian-ai/girl/internal/extensions/decisions"
)

func TestParseTriageTargetsFromPlanAndSonar(t *testing.T) {
	plan := []byte(`{"diagnostics":[
		{"id":"diag_001","code":"react.large-component","severity":"medium","file":"src/App.tsx","line":3,"message":"big"},
		{"id":"diag_002","code":"react.complex-conditionals","severity":"low","file":"src/App.tsx","line":38,"message":"branches"},
		{"id":"diag_003","code":"x.unsafe","severity":"high","file":"../outside.ts","line":1,"message":"traversal"}],
		"unplanned":[{"diagnosticId":"diag_002"},{"diagnosticId":"diag_003"},{"diagnosticId":"diag_404"}]}`)
	got, err := parseTriageTargets(plan, false, false)
	if err != nil {
		t.Fatal(err)
	}
	// Only the unplanned finding with a safe path is sent; mapped findings,
	// unsafe paths and dangling references are not.
	if len(got) != 1 || got[0].Kind != "unplanned" || got[0].Finding.RuleID != "react.complex-conditionals" || got[0].Finding.Fingerprint == "" {
		t.Fatalf("plan targets = %+v", got)
	}

	// --scope all adds mapped findings too, still excluding unsafe paths.
	all, err := parseTriageTargets(plan, true, false)
	if err != nil || len(all) != 2 || all[0].Kind != "planned" || all[0].Signals["findings_in_file"] != 2 {
		t.Fatalf("all targets = %+v err=%v", all, err)
	}

	sonar := []byte(`{"findings":[],"actions":[{"id":"a1","finding":{"provider":"sonarqube","externalId":"K1","ruleId":"python:S3776","file":"pkg/a.py","line":4,"severity":"high"},"recipe":"py.simplify-branches","files":["pkg/a.py"],"changedLines":80,"risk":"medium"}],
		"unmapped":[{"finding":{"provider":"sonarqube","externalId":"K2","ruleId":"python:S5332","file":"pkg/b.py","line":9,"severity":"medium","provenance":{"category":"security"}},"reason":"no recipe"}]}`)
	got, err = parseTriageTargets(sonar, false, false)
	if err != nil || len(got) != 1 || got[0].Kind != "unmapped" || got[0].Finding.Provenance["category"] != "security" {
		t.Fatalf("sonar targets = %+v err=%v", got, err)
	}
	got, err = parseTriageTargets(sonar, false, true)
	if err != nil || len(got) != 2 || got[1].Kind != "action" || got[1].Action == nil {
		t.Fatalf("with actions = %+v err=%v", got, err)
	}

	if _, err := parseTriageTargets([]byte(`{"something":"else"}`), false, false); err == nil {
		t.Fatal("unrecognised input accepted")
	}
}

// stubProvider answers every question with fixed values.
type stubProvider struct{ calls int }

func (s *stubProvider) Name() string { return "stub" }

func (s *stubProvider) Decide(_ context.Context, req decisions.Request) (*decisions.Result, error) {
	s.calls++
	p := func(v float64) *float64 { return &v }
	answers := map[string]decisions.Answer{}
	for key, q := range req.Questions {
		switch q.Type {
		case decisions.TypeNoul:
			v := 0.1
			if key == "needs_review" {
				v = 0.9
			} else if key == "real_issue" {
				v = 0.9
			}
			answers[key] = decisions.Answer{Type: q.Type, Noul: p(v)}
		case decisions.TypeChoice:
			answers[key] = decisions.Answer{Type: q.Type, Choice: decisions.ManualRecipe, Confidence: 0.9}
		case decisions.TypeScore:
			answers[key] = decisions.Answer{Type: q.Type, Score: p(0.5), Confidence: 0.7, Probabilities: map[string]float64{"2": 0.1}}
		}
	}
	return &decisions.Result{Provider: "stub", Model: "stub-1", InputHash: "h", Answers: answers}, nil
}

const samplePlan = `{"diagnostics":[
	{"id":"diag_001","code":"react.large-component","severity":"high","file":"src/App.tsx","line":3,"message":"big"},
	{"id":"diag_002","code":"react.complex-conditionals","severity":"low","file":"src/App.tsx","line":38,"message":"branches"}],
	"steps":[{"id":"step_001","recipe":"react.split-large-component","requires":["diag_001"]}],
	"unplanned":[{"diagnosticId":"diag_002"}]}`

func TestTriageDocumentShadowLedgerAndRanking(t *testing.T) {
	ledger := &decisions.Ledger{Path: filepath.Join(t.TempDir(), "ledger.jsonl")}
	settings := &decisionSettings{Mode: decisions.ModeShadow, MaxFindings: 25, Consistency: true, Thresholds: decisions.DefaultThresholds()}
	stub := &stubProvider{}
	report, err := triageDocument(context.Background(), stub, "example.invalid", "stub-1", []byte(samplePlan), settings,
		triageOptions{Scope: "all", Signals: true, Ledger: ledger})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Entries) != 2 || report.Mode != decisions.ModeShadow {
		t.Fatalf("report = %+v", report)
	}
	// Ranked: the high-severity finding first.
	if report.Entries[0].Triage.Code != "react.large-component" || report.Entries[0].Triage.Priority <= report.Entries[1].Triage.Priority {
		t.Fatalf("not ranked by priority: %+v", report.Entries)
	}
	for _, e := range report.Entries {
		if len(e.Triage.Effects) != 0 || len(e.Triage.WouldEffects) == 0 {
			t.Fatalf("shadow mode had an effect: %+v", e.Triage)
		}
	}
	entries, err := ledger.Read()
	if err != nil || len(entries) != 3 || entries[0].Mode != decisions.ModeShadow || entries[0].Signals["findings_in_file"] != 2 {
		t.Fatalf("ledger = %+v err=%v", entries, err)
	}
	// GIRL already had a recipe for the planned finding; the stub answered
	// "manual", so the disagreement is recorded as free ground truth.
	planned := report.Entries[0]
	if planned.AgreesWithRegistry == nil || *planned.AgreesWithRegistry {
		t.Fatalf("registry agreement not recorded: %+v", planned)
	}
	cases := decisions.Cases(entries)
	if !cases[0].Labeled || cases[0].Outcome.RecipeCorrect == nil || *cases[0].Outcome.RecipeCorrect {
		t.Fatalf("agreement outcome missing from ledger: %+v", cases[0])
	}
	if report.Entries[1].AgreesWithRegistry != nil {
		t.Fatal("unplanned finding has no registry recipe to agree with")
	}
	// The cost guard holds findings back rather than sending them.
	settings.MaxFindings = 1
	stub.calls = 0
	capped, _ := triageDocument(context.Background(), stub, "h", "m", []byte(samplePlan), settings, triageOptions{Scope: "all"})
	if stub.calls != 1 || capped.Skipped != 1 {
		t.Fatalf("max findings not enforced: calls=%d skipped=%d", stub.calls, capped.Skipped)
	}
}

func TestDecisionSettingsFromConfig(t *testing.T) {
	dir := t.TempDir()
	// No config: shadow by default.
	s, err := resolveDecisionSettings(nil, dir)
	if err != nil || s.Mode != decisions.ModeShadow || !s.Consistency || s.Thresholds != decisions.DefaultThresholds() {
		t.Fatalf("defaults = %+v err=%v", s, err)
	}
	writeConfig(t, dir, "decisions:\n  mode: enforce\n  model: jev-1.14\n  max_findings: 7\n  consistency: false\n  min_recipe_confidence: 0.9\n  review_probability: 0.4\n")
	s, err = resolveDecisionSettings(nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Mode != decisions.ModeEnforce || s.Model != "jev-1.14" || s.MaxFindings != 7 || s.Consistency ||
		s.Thresholds.MinRecipeConfidence != 0.9 || s.Thresholds.ReviewProbability != 0.4 || s.Thresholds.HighRiskProbability != 0.3 {
		t.Fatalf("settings = %+v", s)
	}
	// Invalid values are reported, and an invalid mode never becomes enforce.
	writeConfig(t, dir, "decisions:\n  mode: yolo\n  review_probability: 7\n  surprise: 1\n")
	s, err = resolveDecisionSettings(nil, dir)
	if err != nil || s.Mode != decisions.ModeShadow || len(s.Problems) != 3 {
		t.Fatalf("invalid config: %+v err=%v", s, err)
	}
}

func TestReceiptDecisionsReadsTriageAndGuardOutput(t *testing.T) {
	dir := t.TempDir()
	triage := writeFixture(t, dir, "triage.json", `{"entries":[
		{"triage":{"fingerprint":"abc","code":"py.x","model":"jev-1.14.0","inputHash":"h1","mode":"shadow","effects":[],"wouldEffects":["escalate-to-review"]}},
		{"triage":{"fingerprint":"def","abstained":true}}]}`)
	guard := writeFixture(t, dir, "guard.json", `{"fingerprint":"g1","stepId":"step_001","verdict":"review","mode":"enforce","model":"jev-1.14.0","ruleReasons":["touches 9 files (budget 3)"],"modelReasons":[]}`)
	recs, err := receiptDecisions(triage)
	if err != nil || len(recs) != 1 || recs[0].Subject != "triage" || recs[0].WouldEffects[0] != "escalate-to-review" {
		t.Fatalf("triage records = %+v err=%v", recs, err)
	}
	recs, err = receiptDecisions(guard)
	if err != nil || len(recs) != 1 || recs[0].Subject != "guard" || recs[0].Verdict != "review" || len(recs[0].Effects) != 1 {
		t.Fatalf("guard records = %+v err=%v", recs, err)
	}
	if _, err := receiptDecisions(writeFixture(t, dir, "empty.json", `{}`)); err == nil {
		t.Fatal("empty decisions file accepted")
	}
}

func TestServeDecideEndpointIsOffUnlessEnabled(t *testing.T) {
	root := t.TempDir()
	srv, err := newAnalysisServer(root, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"plan":` + samplePlan + `,"scope":"all"}`
	post := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/decide/triage", strings.NewReader(body)))
		return rec
	}
	if rec := post(); rec.Code != http.StatusNotFound {
		t.Fatalf("decision endpoint reachable without --decisions: %d", rec.Code)
	}
	stub := &stubProvider{}
	srv.decide = &decideBackend{provider: stub, host: "example.invalid", model: "stub-1",
		settings: &decisionSettings{Mode: decisions.ModeShadow, MaxFindings: 25, Thresholds: decisions.DefaultThresholds()},
		ledger:   &decisions.Ledger{Path: filepath.Join(root, "ledger.jsonl")}}
	rec := post()
	if rec.Code != http.StatusOK || stub.calls != 2 || !strings.Contains(rec.Body.String(), `"mode":"shadow"`) {
		t.Fatalf("enabled endpoint: %d %s", rec.Code, rec.Body.String())
	}
}
