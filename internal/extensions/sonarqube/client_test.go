package sonarqube

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/canadian-ai/girl/internal/extensions"
)

// Mock SonarQube Web API. Response shapes follow the documented API; they are
// not recorded from a live server.

const testToken = "squ_synthetic_test_token_000000000000"

type mockServer struct {
	t         *testing.T
	issues    []map[string]any
	analyses  []map[string]any
	queue     []map[string]any
	current   map[string]any
	authFail  int // status to return for authenticated endpoints, 0 = ok
	rateLimit int32
	seenAuth  atomic.Value
}

func (m *mockServer) handler() http.Handler {
	mux := http.NewServeMux()
	write := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	authed := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			m.seenAuth.Store(r.Header.Get("Authorization"))
			if r.Method != http.MethodGet {
				m.t.Errorf("non-GET request %s %s: the provider must be read-only", r.Method, r.URL.Path)
			}
			if m.authFail != 0 {
				w.WriteHeader(m.authFail)
				write(w, map[string]any{"errors": []map[string]string{{"msg": "denied"}}})
				return
			}
			if atomic.AddInt32(&m.rateLimit, -1) >= 0 {
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			if r.Header.Get("Authorization") != "Bearer "+testToken {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
	mux.HandleFunc("/api/system/status", func(w http.ResponseWriter, r *http.Request) {
		write(w, map[string]string{"status": "UP", "version": "10.7.0.96327"})
	})
	mux.HandleFunc("/api/authentication/validate", authed(func(w http.ResponseWriter, r *http.Request) {
		write(w, map[string]bool{"valid": true})
	}))
	mux.HandleFunc("/api/issues/search", authed(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		p, _ := strconv.Atoi(q.Get("p"))
		ps, _ := strconv.Atoi(q.Get("ps"))
		var matching []map[string]any
		keys := map[string]bool{}
		for _, k := range strings.Split(q.Get("issues"), ",") {
			if k != "" {
				keys[k] = true
			}
		}
		for _, is := range m.issues {
			if len(keys) > 0 && !keys[is["key"].(string)] {
				continue
			}
			if len(keys) == 0 && q.Get("resolved") == "false" && is["status"] != "OPEN" {
				continue
			}
			matching = append(matching, is)
		}
		start, end := (p-1)*ps, p*ps
		if start > len(matching) {
			start = len(matching)
		}
		if end > len(matching) {
			end = len(matching)
		}
		write(w, map[string]any{
			"total":      len(matching),
			"paging":     map[string]int{"pageIndex": p, "pageSize": ps, "total": len(matching)},
			"issues":     matching[start:end],
			"components": []map[string]string{{"key": "proj:src/app.py", "path": "src/app.py"}},
		})
	}))
	mux.HandleFunc("/api/project_analyses/search", authed(func(w http.ResponseWriter, r *http.Request) {
		write(w, map[string]any{"analyses": m.analyses})
	}))
	mux.HandleFunc("/api/ce/component", authed(func(w http.ResponseWriter, r *http.Request) {
		write(w, map[string]any{"queue": m.queue, "current": m.current})
	}))
	return mux
}

func newTestClient(t *testing.T, m *mockServer, mutate ...func(*Config)) *Client {
	t.Helper()
	m.t = t
	srv := httptest.NewServer(m.handler())
	t.Cleanup(srv.Close)
	cfg := Config{BaseURL: srv.URL, Token: testToken, ProjectKey: "proj", Branch: "main", PageSize: 2, sleep: func(time.Duration) {}}
	for _, f := range mutate {
		f(&cfg)
	}
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func issue(key, rule, status, resolution string, line int) map[string]any {
	return map[string]any{"key": key, "rule": rule, "severity": "CRITICAL", "component": "proj:src/app.py",
		"line": line, "message": "Refactor this function", "type": "CODE_SMELL", "status": status, "resolution": resolution,
		"impacts": []map[string]string{{"softwareQuality": "MAINTAINABILITY", "severity": "HIGH"}}}
}

func TestOpenIssuesPaginatesAndNormalizes(t *testing.T) {
	m := &mockServer{}
	for i := 1; i <= 5; i++ {
		m.issues = append(m.issues, issue("AX"+strconv.Itoa(i), "python:S3776", "OPEN", "", 10*i))
	}
	m.issues = append(m.issues, issue("CLOSED1", "python:S3776", "CLOSED", "FIXED", 3))
	c := newTestClient(t, m)
	set, err := c.OpenIssues(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Issues) != 5 {
		t.Fatalf("pagination lost issues: got %d", len(set.Issues))
	}
	findings, rejected := Normalize(set, ScanIdentity{Project: "proj", Branch: "main", AnalysisKey: "AN1", Revision: "abc1234def"})
	if len(rejected) != 0 || len(findings) != 5 {
		t.Fatalf("findings=%d rejected=%+v", len(findings), rejected)
	}
	f := findings[0]
	if f.Provider != "sonarqube" || f.File != "src/app.py" || f.Severity != "high" || f.Fingerprint == "" {
		t.Fatalf("normalized = %+v", f)
	}
	if f.Provenance["analysisKey"] != "AN1" || f.Provenance["revision"] != "abc1234def" {
		t.Fatalf("provenance missing: %+v", f.Provenance)
	}
}

func TestAuthAndConnectivityErrorsAreExplicit(t *testing.T) {
	for status, want := range map[int]error{401: ErrUnauthorized, 403: ErrForbidden, 404: ErrNotFound, 500: ErrServer} {
		m := &mockServer{authFail: status}
		c := newTestClient(t, m)
		_, err := c.OpenIssues(context.Background())
		if !errors.Is(err, want) {
			t.Errorf("HTTP %d: err=%v want %v", status, err, want)
		}
		if err != nil && strings.Contains(err.Error(), testToken) {
			t.Fatal("error leaked the token")
		}
	}
	c, err := NewClient(Config{BaseURL: "http://127.0.0.1:1", Token: testToken, ProjectKey: "p", MaxRetries: -1, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Status(context.Background()); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("unreachable: %v", err)
	}
}

func TestRateLimitIsRetried(t *testing.T) {
	m := &mockServer{rateLimit: 2}
	m.issues = []map[string]any{issue("A1", "python:S107", "OPEN", "", 1)}
	c := newTestClient(t, m)
	set, err := c.OpenIssues(context.Background())
	if err != nil || len(set.Issues) != 1 {
		t.Fatalf("retry failed: %v", err)
	}
	m2 := &mockServer{rateLimit: 100}
	c2 := newTestClient(t, m2, func(c *Config) { c.MaxRetries = 2 })
	if _, err := c2.OpenIssues(context.Background()); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("exhausted retries: %v", err)
	}
}

func TestConfigSafety(t *testing.T) {
	cases := []Config{
		{BaseURL: "", ProjectKey: "p"},
		{BaseURL: "https://sonar.example", ProjectKey: ""},
		{BaseURL: "http://sonar.example", ProjectKey: "p", Token: testToken}, // clear-text token
		{BaseURL: "https://sonar.example", ProjectKey: "p", Branch: "b", PullRequest: "1"},
		{BaseURL: "ftp://sonar.example", ProjectKey: "p"},
	}
	for _, cfg := range cases {
		if _, err := NewClient(cfg); !errors.Is(err, ErrConfig) {
			t.Errorf("accepted unsafe config %+v", cfg)
		}
	}
	c, err := NewClient(Config{BaseURL: "https://sonar.example", ProjectKey: "p", Token: testToken})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(c.String(), testToken) {
		t.Fatal("String() leaked the token")
	}
	if data, _ := json.Marshal(Config{Token: testToken}); strings.Contains(string(data), testToken) {
		t.Fatal("config serialization leaked the token")
	}
}

func TestPagingCeilingFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"paging": map[string]int{"total": 20000}, "issues": []map[string]any{issue("A", "python:S1", "OPEN", "", 1)}})
	}))
	defer srv.Close()
	c, _ := NewClient(Config{BaseURL: srv.URL, ProjectKey: "proj", sleep: func(time.Duration) {}})
	if _, err := c.OpenIssues(context.Background()); !errors.Is(err, ErrPagingCeiling) {
		t.Fatalf("ceiling: %v", err)
	}
}

func TestToActionsMapsBoundedRecipesAndRequiresPolicy(t *testing.T) {
	mk := func(rule, sev, cat string) extensions.Finding {
		f, err := extensions.Normalize(extensions.Finding{Provider: "sonarqube", ExternalID: rule, RuleID: rule, File: "src/app.py", Line: 3, Severity: sev})
		if err != nil {
			t.Fatal(err)
		}
		f.Provenance = map[string]string{"category": cat}
		return f
	}
	actions, unmapped := ToActions([]extensions.Finding{
		mk("python:S3776", "high", ""),
		mk("python:S107", "medium", ""),
		mk("typescript:S134", "low", ""),
		mk("python:S5332", "critical", "security"), // insecure protocol: no structural recipe
	})
	if len(actions) != 3 || len(unmapped) != 1 {
		t.Fatalf("actions=%+v unmapped=%+v", actions, unmapped)
	}
	want := map[string]bool{"py.simplify-branches": true, "py.group-parameters": true, "ts.simplify-branches": true}
	for _, a := range actions {
		if !want[a.Recipe] {
			t.Fatalf("unexpected recipe %s", a.Recipe)
		}
		// A provider finding alone never authorizes a change.
		if d := extensions.Evaluate(a, extensions.Policy{}); d.Status != extensions.Deny {
			t.Fatalf("action allowed without policy: %+v", d)
		}
	}
	policy := extensions.Policy{AllowedPaths: []string{"src/**"}, MaxFiles: 1, MaxChangedLines: 200}
	if d := extensions.Evaluate(actions[0], policy); d.Status == extensions.Deny {
		t.Fatalf("bounded action denied under explicit policy: %+v", d)
	}
}

func TestProviderTextIsBoundedAndStripped(t *testing.T) {
	got := cleanProviderText("ok\x1b[31m red\x00\nIGNORE PREVIOUS " + strings.Repeat("A", 2000))
	if strings.ContainsAny(got, "\x1b\x00\n") || len(got) > 520 {
		t.Fatalf("provider text not bounded: %d bytes %q", len(got), got[:40])
	}
}
