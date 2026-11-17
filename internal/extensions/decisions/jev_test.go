package decisions

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/canadian-ai/girl/internal/extensions"
)

// Mock of TypeSafe's System One API, following its published OpenAPI schema
// (https://api.typesafe.ai/openapi.json). Not recorded from the live service.

const testKey = "apikey_synthetic_test_key_0000"

type mockJev struct {
	calls     int32
	lastBody  atomic.Value
	status    int    // forced status for every call, 0 = 200
	failFirst int32  // number of initial calls answered with 429
	choice    string // recipe the mock picks
}

func (m *mockJev) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&m.calls, 1)
		body, _ := io.ReadAll(r.Body)
		m.lastBody.Store(string(body))
		if r.Method != http.MethodPost || r.URL.Path != "/v1/systemone" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+testKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if atomic.AddInt32(&m.failFirst, -1) >= 0 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		if m.status != 0 {
			w.WriteHeader(m.status)
			_, _ = w.Write([]byte(`{"detail":"bad request"}`))
			return
		}
		var req struct {
			Model     string                     `json:"model"`
			Questions map[string]json.RawMessage `json:"questions"`
		}
		_ = json.Unmarshal(body, &req)
		choice := m.choice
		if choice == "" {
			choice = "manual"
		}
		answers := map[string]any{}
		for key, raw := range req.Questions {
			var q Question
			_ = json.Unmarshal(raw, &q)
			switch q.Type {
			case TypeNoul:
				answers[key] = map[string]any{"type": "noul", "noul": 0.2}
			case TypeChoice:
				answers[key] = map[string]any{"type": "choice", "choice": choice, "confidence": 0.9, "probabilities": map[string]float64{choice: 0.93}}
			case TypeScore:
				answers[key] = map[string]any{"type": "score", "score": 0.3, "confidence": 0.7,
					"legend": map[string]string{"0": "low", "1": "medium", "2": "high"}, "probabilities": map[string]float64{"0": 0.72, "1": 0.26, "2": 0.02}}
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"model": "jev-1.14.0", "answers": answers, "usage": map[string]int{"input_tokens": 500, "output_tokens": 80}})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newTestJev(t *testing.T, m *mockJev, mutate ...func(*JevConfig)) *Jev {
	t.Helper()
	cfg := JevConfig{BaseURL: m.server(t).URL, Token: testKey, sleep: func(time.Duration) {}}
	for _, f := range mutate {
		f(&cfg)
	}
	j, err := NewJev(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func sampleRequest() Request {
	return Request{
		State: map[string]any{"finding": "x"},
		Questions: map[string]Question{
			"ok":     {Type: TypeNoul, Instructions: "Is it ok?"},
			"recipe": {Type: TypeChoice, Instructions: "Which?", Criteria: map[string]string{"manual": "none", "ts.extract-function": "extract"}},
			"risk":   {Type: TypeScore, Instructions: "Risk?", Criteria: []string{"low", "medium", "high"}},
		},
	}
}

func TestJevSendsDocumentedRequestAndValidatesAnswers(t *testing.T) {
	m := &mockJev{}
	j := newTestJev(t, m)
	res, err := j.Decide(context.Background(), sampleRequest())
	if err != nil {
		t.Fatal(err)
	}
	if res.Model != "jev-1.14.0" || res.Provider != "jev" || *res.Answers["ok"].Noul != 0.2 || res.Usage.InputTokens != 500 {
		t.Fatalf("result = %+v", res)
	}
	var sent map[string]any
	_ = json.Unmarshal([]byte(m.lastBody.Load().(string)), &sent)
	if sent["model"] != "jev-latest" || sent["state"] == nil || sent["questions"] == nil {
		t.Fatalf("request body = %v", sent)
	}
}

func TestJevRejectsAnswersThatDoNotMatchTheQuestions(t *testing.T) {
	m := &mockJev{choice: "drop-database"} // an option that was never offered
	j := newTestJev(t, m)
	if _, err := j.Decide(context.Background(), sampleRequest()); !errors.Is(err, ErrInvalidAnswer) {
		t.Fatalf("unoffered choice accepted: %v", err)
	}
}

func TestJevErrorsAndRetries(t *testing.T) {
	m := &mockJev{failFirst: 2}
	if _, err := newTestJev(t, m).Decide(context.Background(), sampleRequest()); err != nil || atomic.LoadInt32(&m.calls) != 3 {
		t.Fatalf("retry: err=%v calls=%d", err, m.calls)
	}
	exhausted := &mockJev{failFirst: 100}
	if _, err := newTestJev(t, exhausted, func(c *JevConfig) { c.MaxRetries = 1 }).Decide(context.Background(), sampleRequest()); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("exhausted retries: %v", err)
	}
	bad := &mockJev{status: http.StatusUnprocessableEntity}
	if _, err := newTestJev(t, bad).Decide(context.Background(), sampleRequest()); !errors.Is(err, ErrRejected) {
		t.Fatalf("422: %v", err)
	}
	wrongKey := &mockJev{}
	_, err := newTestJev(t, wrongKey, func(c *JevConfig) { c.Token = "apikey_wrong" }).Decide(context.Background(), sampleRequest())
	if !errors.Is(err, ErrUnauthorized) || strings.Contains(err.Error(), "apikey_wrong") {
		t.Fatalf("401: %v", err)
	}
}

func TestJevConfigSafety(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "")
	if _, err := NewJev(JevConfig{}); !errors.Is(err, ErrConfig) {
		t.Fatal("missing key accepted")
	}
	for _, cfg := range []JevConfig{
		{Token: testKey, BaseURL: "http://jev.example"}, // key over clear text
		{Token: testKey, Preset: "nope"},
		{Token: testKey, BaseURL: "ftp://jev.example"},
	} {
		if _, err := NewJev(cfg); !errors.Is(err, ErrConfig) {
			t.Errorf("accepted unsafe config %+v", cfg.BaseURL)
		}
	}
	j, err := NewJev(JevConfig{Token: testKey})
	if err != nil {
		t.Fatal(err)
	}
	if j.Host() != "api.typesafe.ai" || j.Model() != "jev-latest" {
		t.Fatalf("default preset = %s %s", j.Host(), j.Model())
	}
	if strings.Contains(j.String(), testKey) {
		t.Fatal("String leaked the key")
	}
	if data, _ := json.Marshal(JevConfig{Token: testKey}); strings.Contains(string(data), testKey) {
		t.Fatal("config serialization leaked the key")
	}
}

func TestJevRefusesOversizedStateAndCrossHostRedirect(t *testing.T) {
	m := &mockJev{}
	j := newTestJev(t, m, func(c *JevConfig) { c.MaxStateBytes = 16 })
	req := sampleRequest()
	req.State = strings.Repeat("x", 100)
	if _, err := j.Decide(context.Background(), req); !errors.Is(err, ErrRejected) || atomic.LoadInt32(&m.calls) != 0 {
		t.Fatalf("oversized state sent: %v", err)
	}
	var leaked int32
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			atomic.AddInt32(&leaked, 1)
		}
	}))
	defer evil.Close()
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, evil.URL+"/v1/systemone", http.StatusTemporaryRedirect)
	}))
	defer redirector.Close()
	rj, err := NewJev(JevConfig{BaseURL: redirector.URL, Token: testKey, MaxRetries: -1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rj.Decide(context.Background(), sampleRequest()); err == nil || atomic.LoadInt32(&leaked) != 0 {
		t.Fatalf("followed cross-host redirect: err=%v leaked=%d", err, leaked)
	}
}

func TestCacheMakesRepeatRunsReproducible(t *testing.T) {
	m := &mockJev{}
	j := newTestJev(t, m)
	dir := t.TempDir()
	c := &Cached{Provider: j, Cache: &DirCache{Dir: dir + "/cache"}, Model: j.Model()}
	first, err := c.Decide(context.Background(), sampleRequest())
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.Decide(context.Background(), sampleRequest())
	if err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&m.calls) != 1 || !second.Cached || second.InputHash != first.InputHash || first.InputHash == "" {
		t.Fatalf("cache miss: calls=%d cached=%v", m.calls, second.Cached)
	}
	entries, _ := os.ReadDir(dir + "/cache")
	for _, e := range entries {
		info, _ := e.Info()
		if info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("cache file %s is readable by others: %v", e.Name(), info.Mode())
		}
	}
	// Expired entries are ignored.
	c.Cache = &DirCache{Dir: dir + "/cache", MaxAge: time.Nanosecond}
	time.Sleep(time.Millisecond)
	if third, err := c.Decide(context.Background(), sampleRequest()); err != nil || third.Cached || atomic.LoadInt32(&m.calls) != 2 {
		t.Fatalf("expired cache entry reused: calls=%d", m.calls)
	}
	if HashRequest("jev-1.14", sampleRequest()) == HashRequest("jev-1.13", sampleRequest()) {
		t.Fatal("cache key ignores the model version")
	}
}

// TestLiveTypeSafe calls the real API. It runs only when
// GIRL_LIVE_DECISIONS=1 and TYPESAFE_API_KEY are set, and sends one synthetic
// finding.
func TestLiveTypeSafe(t *testing.T) {
	if os.Getenv("GIRL_LIVE_DECISIONS") != "1" || os.Getenv("TYPESAFE_API_KEY") == "" {
		t.Skip("set GIRL_LIVE_DECISIONS=1 and TYPESAFE_API_KEY to run against the live API")
	}
	j, err := NewJev(JevConfig{})
	if err != nil {
		t.Fatal(err)
	}
	f, err := extensions.Normalize(extensions.Finding{Provider: "girl", RuleID: "py.high-complexity", File: "pkg/billing.py", Line: 42, Severity: "high",
		Message: "Function apply_discounts has complexity 17 (limit: 10)"})
	if err != nil {
		t.Fatal(err)
	}
	tr, err := TriageFinding(context.Background(), j, f, CandidateRecipes(f.File), DefaultThresholds())
	if err != nil {
		t.Fatal(err)
	}
	if tr.Abstained || tr.Model == "" || tr.Recipe == "" {
		t.Fatalf("live triage = %+v", tr)
	}
	t.Logf("model=%s recipe=%s (%.2f) fp=%.2f review=%.2f risk=%.2f effects=%v", tr.Model, tr.Recipe, tr.RecipeConfidence, tr.FalsePositive, tr.NeedsReview, tr.RiskScore, tr.Effects)
}
