package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/canadian-ai/girl/internal/extensions/sonarqube"
	"github.com/urfave/cli/v2"
)

// SonarCommand exposes the read-only SonarQube provider. The token is read
// from SONAR_TOKEN (never from a flag, so it stays out of shell history and
// process listings).
func SonarCommand() *cli.Command {
	common := []cli.Flag{
		&cli.StringFlag{Name: "url", EnvVars: []string{"SONAR_HOST_URL"}, Usage: "SonarQube/SonarCloud base URL"},
		&cli.StringFlag{Name: "project", EnvVars: []string{"SONAR_PROJECT_KEY"}, Usage: "Project key"},
		&cli.StringFlag{Name: "organization", EnvVars: []string{"SONAR_ORGANIZATION"}, Usage: "Organization (SonarCloud)"},
		&cli.StringFlag{Name: "branch", Usage: "Branch to read"},
		&cli.StringFlag{Name: "pull-request", Usage: "Pull request ID to read"},
		&cli.StringFlag{Name: "auth", Value: "bearer", Usage: "Auth scheme: bearer (SonarQube 10+, SonarCloud) or basic (legacy)"},
		&cli.DurationFlag{Name: "timeout", Value: 30 * time.Second, Usage: "Per-request timeout"},
	}
	client := func(c *cli.Context) (*sonarqube.Client, error) {
		return sonarqube.NewClient(sonarqube.Config{
			BaseURL:      c.String("url"),
			Token:        os.Getenv("SONAR_TOKEN"),
			Organization: c.String("organization"),
			ProjectKey:   c.String("project"),
			Branch:       c.String("branch"),
			PullRequest:  c.String("pull-request"),
			AuthScheme:   c.String("auth"),
			Timeout:      c.Duration("timeout"),
		})
	}
	return &cli.Command{
		Name:  "sonar",
		Usage: "Read-only SonarQube provider: import findings, verify remediation (experimental)",
		Subcommands: []*cli.Command{
			{
				Name:  "status",
				Usage: "Check server reachability and token validity",
				Flags: common,
				Action: func(c *cli.Context) error {
					cl, err := client(c)
					if err != nil {
						return err
					}
					st, err := cl.Status(c.Context)
					if err != nil {
						return err
					}
					if err := cl.ValidateToken(c.Context); err != nil {
						return err
					}
					fmt.Printf("SonarQube %s is UP; token valid\n", st.Version)
					return nil
				},
			},
			{
				Name:  "issues",
				Usage: "Import open issues as governed findings and bounded GRP action proposals",
				Flags: common,
				Action: func(c *cli.Context) error {
					cl, err := client(c)
					if err != nil {
						return err
					}
					ctx := c.Context
					if ctx == nil {
						ctx = context.Background()
					}
					st, err := cl.Status(ctx)
					if err != nil {
						return err
					}
					analysis, err := cl.LatestAnalysis(ctx)
					if err != nil {
						return err
					}
					scan := sonarqube.ScanIdentity{ServerVersion: st.Version, Project: c.String("project"), Branch: c.String("branch"), PullRequest: c.String("pull-request")}
					if analysis != nil {
						scan.AnalysisKey, scan.AnalysisDate, scan.Revision = analysis.Key, analysis.Date, analysis.Revision
					}
					set, err := cl.OpenIssues(ctx)
					if err != nil {
						return err
					}
					findings, rejected := sonarqube.Normalize(set, scan)
					actions, unmapped := sonarqube.ToActions(findings)
					printJSON(map[string]any{
						"specversion": "girl.io/extensions/sonarqube/v0",
						"scan":        scan,
						"findings":    findings,
						"actions":     actions,
						"unmapped":    unmapped,
						"rejected":    rejected,
						"note":        "Actions are proposals. Evaluate them against an explicit policy; a provider finding never authorizes a change.",
					})
					return nil
				},
			},
			{
				Name:  "verify",
				Usage: "Verify remediation against a fresh provider scan of the fixing revision",
				Flags: append(common,
					&cli.StringFlag{Name: "issues", Required: true, Usage: "Comma-separated issue keys the fix claims to resolve"},
					&cli.StringFlag{Name: "revision", Required: true, Usage: "Commit SHA containing the fix"},
				),
				Action: func(c *cli.Context) error {
					cl, err := client(c)
					if err != nil {
						return err
					}
					var keys []string
					for _, k := range strings.Split(c.String("issues"), ",") {
						if k = strings.TrimSpace(k); k != "" {
							keys = append(keys, k)
						}
					}
					r, err := cl.VerifyRemediation(c.Context, sonarqube.VerifyRequest{IssueKeys: keys, Revision: c.String("revision")})
					if err != nil {
						return err
					}
					printJSON(r)
					if r.Status != sonarqube.ReceiptVerified {
						return cli.Exit(fmt.Sprintf("remediation %s", r.Status), 2)
					}
					return nil
				},
			},
		},
	}
}
