package sonarqube

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLiveInstance runs only when a test SonarQube/SonarCloud instance is
// configured:
//
//	SONAR_TEST_URL, SONAR_TEST_PROJECT, SONAR_TOKEN [, SONAR_TEST_ORGANIZATION, SONAR_TEST_BRANCH]
//
// It is read-only and asserts only that the documented endpoints respond with
// shapes the adapter can parse.
func TestLiveInstance(t *testing.T) {
	url, project, token := os.Getenv("SONAR_TEST_URL"), os.Getenv("SONAR_TEST_PROJECT"), os.Getenv("SONAR_TOKEN")
	if url == "" || project == "" || token == "" {
		t.Skip("set SONAR_TEST_URL, SONAR_TEST_PROJECT and SONAR_TOKEN to run against a live instance")
	}
	c, err := NewClient(Config{BaseURL: url, Token: token, ProjectKey: project,
		Organization: os.Getenv("SONAR_TEST_ORGANIZATION"), Branch: os.Getenv("SONAR_TEST_BRANCH"), Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	st, err := c.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("server version %s", st.Version)
	if err := c.ValidateToken(ctx); err != nil {
		t.Fatal(err)
	}
	analysis, err := c.LatestAnalysis(ctx)
	if err != nil {
		t.Fatal(err)
	}
	set, err := c.OpenIssues(ctx)
	if err != nil {
		t.Fatal(err)
	}
	scan := ScanIdentity{Project: project}
	if analysis != nil {
		scan.AnalysisKey, scan.Revision = analysis.Key, analysis.Revision
	}
	findings, rejected := Normalize(set, scan)
	actions, unmapped := ToActions(findings)
	t.Logf("issues=%d findings=%d rejected=%d actions=%d unmapped=%d", len(set.Issues), len(findings), len(rejected), len(actions), len(unmapped))
}
