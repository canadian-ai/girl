package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/urfave/cli/v2"
)

func newTestServer(t *testing.T, root string, maxConcurrent int) *httptest.Server {
	t.Helper()
	app := &cli.App{Name: "girl", Version: "test", Commands: []*cli.Command{NextCommand()}}
	s, err := newAnalysisServer(root, maxConcurrent, app)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.routes())
	t.Cleanup(ts.Close)
	return ts
}

func postJSON(t *testing.T, url string, body any) (*http.Response, map[string]any) {
	t.Helper()
	data, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func TestServe_NextOnTSXFixtures(t *testing.T) {
	ts := newTestServer(t, filepath.Join(repoRootForTest(t), "testdata"), 2)

	resp, body := postJSON(t, ts.URL+"/v1/next", map[string]any{"path": "real", "limit": 5})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %v", resp.StatusCode, body)
	}
	if body["schema"] != NextSchema {
		t.Fatalf("schema = %v", body["schema"])
	}
	items := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("want one item per real-world component, got %d", len(items))
	}
	for _, raw := range items {
		it := raw.(map[string]any)
		if it["line"] == nil || it["excerpt"] == "" {
			t.Errorf("component item lacks a location/excerpt: %v", it["id"])
		}
	}
	if !strings.Contains(resp.Header.Get("Server-Timing"), "analysis;dur=") {
		t.Errorf("missing Server-Timing header")
	}
	if resp.Header.Get("X-Girl-Files") != "2" {
		t.Errorf("X-Girl-Files = %q", resp.Header.Get("X-Girl-Files"))
	}
}

func TestServe_ConfinesRequestsToRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	secret := filepath.Join(parent, "secret")
	for _, d := range []string{root, secret} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(secret, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	ts := newTestServer(t, root, 1)

	for _, path := range []string{"../secret", secret, "escape", "/etc"} {
		resp, body := postJSON(t, ts.URL+"/v1/next", map[string]any{"path": path})
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("path %q: status %d (%v), want 403", path, resp.StatusCode, body)
		}
	}
	resp, _ := postJSON(t, ts.URL+"/v1/next", map[string]any{"path": "missing"})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing path: status %d, want 404", resp.StatusCode)
	}
}

func TestServe_RejectsMalformedRequests(t *testing.T) {
	ts := newTestServer(t, t.TempDir(), 1)
	resp, _ := postJSON(t, ts.URL+"/v1/next", map[string]any{"unknown": true})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown field: status %d, want 400", resp.StatusCode)
	}
	get, err := http.Get(ts.URL + "/v1/next")
	if err != nil {
		t.Fatal(err)
	}
	get.Body.Close()
	if get.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET /v1/next: status %d, want 405", get.StatusCode)
	}
}

func TestServe_QueuesBeyondMaxConcurrent(t *testing.T) {
	ts := newTestServer(t, filepath.Join(repoRootForTest(t), "testdata"), 1)
	var wg sync.WaitGroup
	timings := make([]string, 4)
	for i := range timings {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, _ := postJSON(t, ts.URL+"/v1/benchmark", map[string]any{"path": "real"})
			if resp.StatusCode != http.StatusOK {
				t.Errorf("status %d", resp.StatusCode)
			}
			timings[i] = resp.Header.Get("Server-Timing")
		}(i)
	}
	wg.Wait()
	queued := 0
	for _, timing := range timings {
		var q float64
		if _, err := fmt.Sscanf(timing, "queue;dur=%f", &q); err == nil && q > 0 {
			queued++
		}
	}
	if queued == 0 {
		t.Errorf("with max-concurrent 1, concurrent requests should report queue time: %v", timings)
	}
}

func TestServe_Healthz(t *testing.T) {
	ts := newTestServer(t, t.TempDir(), 3)
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["ok"] != true || body["maxConcurrent"] != float64(3) {
		t.Fatalf("healthz = %v", body)
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
