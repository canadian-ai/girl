package sonarqube

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func verifyFixture(issues []map[string]any) *mockServer {
	return &mockServer{
		issues:   issues,
		analyses: []map[string]any{{"key": "AN2", "date": "2026-11-15T10:00:00+0000", "revision": "abc1234def5678"}},
		current:  map[string]any{"id": "T1", "status": "SUCCESS", "branch": "main", "analysisId": "AN2"},
	}
}

func TestVerifyRemediationSuccessfulRescan(t *testing.T) {
	m := verifyFixture([]map[string]any{issue("K1", "python:S3776", "CLOSED", "FIXED", 3), {"key": "K2", "rule": "python:S107", "component": "proj:src/app.py", "issueStatus": "FIXED", "status": "CLOSED"}})
	c := newTestClient(t, m)
	r, err := c.VerifyRemediation(context.Background(), VerifyRequest{IssueKeys: []string{"K1", "K2"}, Revision: "abc1234def5678"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != ReceiptVerified || r.Scan.AnalysisKey != "AN2" || r.Scan.ServerVersion == "" {
		t.Fatalf("receipt = %+v", r)
	}
}

func TestVerifyRemediationNeverVerifiesWithoutEvidence(t *testing.T) {
	fixed := []map[string]any{issue("K1", "python:S3776", "CLOSED", "FIXED", 3)}
	cases := []struct {
		name   string
		setup  func(*mockServer)
		keys   []string
		want   string
		reason string
	}{
		{"stale scan", func(m *mockServer) { m.analyses[0]["revision"] = "0000000old" }, []string{"K1"}, ReceiptUnverified, "stale"},
		{"pending scan", func(m *mockServer) {
			m.queue = []map[string]any{{"id": "T9", "status": "PENDING", "branch": "main"}}
		}, []string{"K1"}, ReceiptUnverified, "pending"},
		{"failed scan", func(m *mockServer) { m.current["status"] = "FAILED" }, []string{"K1"}, ReceiptUnverified, "FAILED"},
		{"no analysis", func(m *mockServer) { m.analyses = nil }, []string{"K1"}, ReceiptUnverified, "no analysis"},
		{"no revision recorded", func(m *mockServer) { m.analyses[0]["revision"] = "" }, []string{"K1"}, ReceiptUnverified, "no SCM revision"},
		{"missing issue", func(m *mockServer) {}, []string{"K1", "GONE"}, ReceiptUnverified, "provably"},
		{"still open", func(m *mockServer) { m.issues[0]["status"] = "OPEN"; m.issues[0]["resolution"] = "" }, []string{"K1"}, ReceiptFailed, "still open"},
		{"dismissed not fixed", func(m *mockServer) { m.issues[0]["resolution"] = "WONTFIX"; m.issues[0]["status"] = "RESOLVED" }, []string{"K1"}, ReceiptUnverified, "provably"},
		{"accepted not fixed", func(m *mockServer) { m.issues[0]["issueStatus"] = "ACCEPTED" }, []string{"K1"}, ReceiptUnverified, "provably"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var issues []map[string]any
			for _, is := range fixed {
				cp := map[string]any{}
				for k, v := range is {
					cp[k] = v
				}
				issues = append(issues, cp)
			}
			m := verifyFixture(issues)
			tc.setup(m)
			c := newTestClient(t, m)
			r, err := c.VerifyRemediation(context.Background(), VerifyRequest{IssueKeys: tc.keys, Revision: "abc1234def5678"})
			if err != nil {
				t.Fatal(err)
			}
			if r.Status != tc.want || !strings.Contains(strings.Join(r.Reasons, " "), tc.reason) {
				t.Fatalf("receipt = %+v", r)
			}
		})
	}
	c := newTestClient(t, verifyFixture(fixed))
	if _, err := c.VerifyRemediation(context.Background(), VerifyRequest{IssueKeys: []string{"K1"}}); !errors.Is(err, ErrConfig) {
		t.Fatal("verification without a revision must be rejected")
	}
}
