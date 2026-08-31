package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestSemverCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.1.18", "0.1.18", 0},
		{"0.1.18", "0.1.19", -1},
		{"0.1.19", "0.1.18", 1},
		{"v0.1.18", "0.1.19", -1},
		{"0.2.0", "0.1.99", 1},
		{"1.0.0", "0.9.9", 1},
		{"0.1.18", "garbage", 0},
	}
	for _, tc := range cases {
		if got := semverCompare(tc.a, tc.b); got != tc.want {
			t.Errorf("semverCompare(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestVersionParts(t *testing.T) {
	if p, err := versionParts("v0.1.18"); err != nil || p != [3]int{0, 1, 18} {
		t.Fatalf("versionParts(v0.1.18) = %v, %v", p, err)
	}
}

func TestPickReleaseAsset(t *testing.T) {
	assets := []releaseAsset{
		{Name: "girl-linux-arm64", BrowserDownloadURL: "https://x/girl-linux-arm64"},
		{Name: "girl", BrowserDownloadURL: "https://x/girl"},
		{Name: "notes.txt", BrowserDownloadURL: "https://x/notes"},
	}

	got := pickReleaseAsset(assets, "linux", "amd64")
	if got == nil || got.Name != "girl" {
		t.Fatalf("linux/amd64 should pick the plain 'girl' asset, got %+v", got)
	}

	got = pickReleaseAsset(assets, "linux", "arm64")
	if got == nil || got.Name != "girl-linux-arm64" {
		t.Fatalf("linux/arm64 should pick girl-linux-arm64, got %+v", got)
	}

	if got := pickReleaseAsset(assets, "darwin", "amd64"); got != nil {
		t.Fatalf("no darwin asset should match, got %+v", got)
	}

	if got := pickReleaseAsset([]releaseAsset{{Name: "girl"}}, "linux", "arm64"); got != nil {
		t.Fatalf("asset without download URL must not be selected, got %+v", got)
	}
}

func TestFetchLatestRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("expected a User-Agent header")
		}
		body := `{"tag_name":"v0.1.19","assets":[{"name":"girl","browser_download_url":"https://x/girl"}]}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer server.Close()

	release, err := fetchLatestRelease(&http.Client{}, server.URL)
	if err != nil {
		t.Fatalf("fetchLatestRelease: %v", err)
	}
	if release.TagName != "v0.1.19" {
		t.Fatalf("unexpected tag %q", release.TagName)
	}
	if len(release.Assets) != 1 || release.Assets[0].Name != "girl" {
		t.Fatalf("unexpected assets: %+v", release.Assets)
	}
}

func TestUpdateStatusJSON(t *testing.T) {
	status := updateStatus{Current: "0.1.18", Latest: "0.1.19", UpdateNeeded: true}
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded updateStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Latest != "0.1.19" {
		t.Fatalf("round-trip failed: %s", string(data))
	}
}

func TestCurrentBinary(t *testing.T) {
	path, err := currentBinary()
	if err != nil {
		t.Fatalf("currentBinary: %v", err)
	}
	if path == "" {
		t.Fatal("currentBinary returned empty path")
	}
}

var _ = runtime.GOOS
