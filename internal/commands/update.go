package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

const (
	updateLatestURL = "https://api.github.com/repos/canadian-ai/girl/releases/latest"
	updateModule    = "github.com/canadian-ai/girl/cmd/girl"
)

type releaseInfo struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type updateStatus struct {
	Current       string `json:"current"`
	Latest        string `json:"latest"`
	UpdateNeeded  bool   `json:"updateNeeded"`
	InstallMethod string `json:"installMethod,omitempty"`
	Installed     string `json:"installed,omitempty"`
	Error         string `json:"error,omitempty"`
}

// semverCompare compares numeric X.Y.Z versions. Returns -1/0/1 when a is
// less than / equal to / greater than b. Tags like "v0.1.18" are normalized.
func semverCompare(a, b string) int {
	ai, err := versionParts(a)
	if err != nil {
		return 0
	}
	bi, err := versionParts(b)
	if err != nil {
		return 0
	}
	for i := 0; i < 3; i++ {
		if ai[i] < bi[i] {
			return -1
		}
		if ai[i] > bi[i] {
			return 1
		}
	}
	return 0
}

func versionParts(v string) ([3]int, error) {
	var parts [3]int
	trimmed := strings.TrimPrefix(strings.TrimSpace(v), "v")
	fields := strings.SplitN(trimmed, ".", 3)
	for i := range fields {
		if i >= len(parts) {
			break
		}
		n, err := strconv.Atoi(fields[i])
		if err != nil {
			return parts, fmt.Errorf("invalid version %q: %w", v, err)
		}
		parts[i] = n
	}
	return parts, nil
}

// pickReleaseAsset selects a usable binary asset for the current platform.
// The release workflow attaches a linux/amd64 "girl" binary; other platforms
// match by GOOS/GOARCH suffixes and otherwise fall back to `go install`.
func pickReleaseAsset(assets []releaseAsset, goos, goarch string) *releaseAsset {
	wanted := []string{
		fmt.Sprintf("girl-%s-%s", goos, goarch),
		fmt.Sprintf("girl_%s_%s", goos, goarch),
		fmt.Sprintf("girl-%s", goos),
		fmt.Sprintf("girl_%s", goos),
	}
	if goos == "linux" && goarch == "amd64" {
		wanted = append(wanted, "girl")
	}
	for _, byName := range wanted {
		for i := range assets {
			if assets[i].Name == byName && assets[i].BrowserDownloadURL != "" {
				return &assets[i]
			}
		}
	}
	return nil
}

func fetchLatestRelease(client *http.Client, url string) (*releaseInfo, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("update: build release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "girl-update/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("update: reach GitHub releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("update: GitHub returns %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var release releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("update: decode latest release: %w", err)
	}
	if release.TagName == "" {
		return nil, fmt.Errorf("update: release response has no tag_name")
	}
	return &release, nil
}

func currentBinary() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("update: resolve current binary: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = resolved
	}
	return path, nil
}

// installData writes bytes next to the running binary, verifies the result
// boots, then atomically replaces the current binary.
func installData(binary string, data []byte) error {
	dir := filepath.Dir(binary)
	tmp, err := os.CreateTemp(dir, ".girl-update-*")
	if err != nil {
		return fmt.Errorf("update: create temp binary: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("update: write temp binary: %w", err)
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return fmt.Errorf("update: chmod temp binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("update: close temp binary: %w", err)
	}

	if err := verifyBinary(tmpPath); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, binary); err != nil {
		return fmt.Errorf("update: replace current binary: %w", err)
	}
	return nil
}

func verifyBinaryImpl(path string) error {
	out, err := exec.Command(path, "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("update: downloaded binary failed to run: %w", err)
	}
	if !strings.Contains(string(out), "girl") {
		return fmt.Errorf("update: downloaded binary is not a girl binary")
	}
	return nil
}

// verifyBinary is swappable in tests so checksum paths can be exercised
// without exec'ing fixture bytes.
var verifyBinary = verifyBinaryImpl

func downloadAsset(client *http.Client, url, expectedSHA256, binary string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("update: download release asset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update: asset download returns %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("update: read release asset: %w", err)
	}

	if expectedSHA256 != "" {
		sum := sha256.Sum256(data)
		got := hex.EncodeToString(sum[:])
		if !strings.EqualFold(got, expectedSHA256) {
			return fmt.Errorf("update: checksum mismatch (have %s, want %s)", got, expectedSHA256)
		}
	}

	return installData(binary, data)
}

// fetchChecksums reads the SHA256SUMS asset from the release (best effort).
// Missing or unparseable checksums are not fatal: the binary is still
// verified by running it before install.
func fetchChecksums(client *http.Client, release *releaseInfo) map[string]string {
	for i := range release.Assets {
		if release.Assets[i].Name != "SHA256SUMS" || release.Assets[i].BrowserDownloadURL == "" {
			continue
		}
		resp, err := client.Get(release.Assets[i].BrowserDownloadURL)
		if err != nil {
			return nil
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			return nil
		}
		return parseChecksums(string(body))
	}
	return nil
}

func parseChecksums(content string) map[string]string {
	sums := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[1], "*")
		sums[name] = strings.ToLower(fields[0])
	}
	return sums
}

func installViaGo(tag, binary string) error {
	tmpDir, err := os.MkdirTemp("", "girl-gobin-*")
	if err != nil {
		return fmt.Errorf("update: create gobin temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("go", "install", updateModule+"@"+tag)
	cmd.Env = append(os.Environ(), "GOBIN="+tmpDir, "GOFLAGS=-trimpath")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("update: go install %s@%s failed: %v: %s", updateModule, tag, err, strings.TrimSpace(string(out)))
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "girl"))
	if err != nil {
		return fmt.Errorf("update: locate installed girl binary: %w", err)
	}
	return installData(binary, data)
}

func UpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Check for and install the latest GIRL release from GitHub",
		Description: "Reads the latest vX.Y.Z release from the GitHub Releases API, downloads the " +
			"platform binary asset (falling back to `go install`), verifies it, and atomically replaces " +
			"the running binary. Use --check to only report whether an update is available.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "check",
				Usage: "Only report whether an update is available (exit 2 if one is)",
			},
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Usage:   "Output machine-readable JSON",
			},
		},
		Action: func(c *cli.Context) error {
			status := updateStatus{Current: Version}
			emit := func() error {
				if c.Bool("json") {
					data, _ := json.MarshalIndent(status, "", "  ")
					fmt.Println(string(data))
					return nil
				}
				if status.Error != "" {
					return cli.Exit(status.Error, 1)
				}
				if status.UpdateNeeded {
					fmt.Printf("GIRL update available: %s -> %s\n", status.Current, status.Latest)
				} else {
					fmt.Printf("GIRL is up to date (%s)\n", status.Current)
				}
				if status.Installed != "" {
					fmt.Printf("Installed: %s (%s)\n", status.Installed, status.InstallMethod)
				}
				return nil
			}

			client := &http.Client{Timeout: 20 * time.Second}
			release, err := fetchLatestRelease(client, updateLatestURL)
			if err != nil {
				status.Error = err.Error()
				if c.Bool("check") {
					_ = emit()
					return cli.Exit(err.Error(), 1)
				}
				return cli.Exit(status.Error, 1)
			}

			tag := strings.TrimPrefix(release.TagName, "v")
			status.Latest = tag
			status.UpdateNeeded = semverCompare(tag, Version) > 0

			if !status.UpdateNeeded {
				if c.Bool("check") {
					_ = emit()
					return nil
				}
				_ = emit()
				return nil
			}

			if c.Bool("check") {
				_ = emit()
				return cli.Exit("", 2)
			}

			binary, err := currentBinary()
			if err != nil {
				status.Error = err.Error()
				_ = emit()
				return cli.Exit(err.Error(), 1)
			}

			// Fast path: download the platform release asset, verify its
			// checksum (when published) and that it actually runs. If anything
			// fails — e.g. the runner-built binary needs a newer glibc than the
			// host provides — fall back to building from source with the local
			// toolchain, which always matches the host.
			if asset := pickReleaseAsset(release.Assets, runtime.GOOS, runtime.GOARCH); asset != nil {
				checksums := fetchChecksums(client, release)
				expected := checksums[asset.Name]
				if installErr := downloadAsset(client, asset.BrowserDownloadURL, expected, binary); installErr == nil {
					status.InstallMethod = "release-asset"
					status.Installed = tag
					_ = emit()
					return nil
				} else {
					note := fmt.Sprintf("update: asset path failed (%v); falling back to source build", installErr)
					fmt.Fprintln(os.Stderr, note)
				}
			}

			if err := installViaGo(tag, binary); err != nil {
				status.Error = err.Error()
				_ = emit()
				return cli.Exit(err.Error(), 1)
			}

			status.InstallMethod = "go-install"
			status.Installed = tag
			_ = emit()
			return nil
		},
	}
}
