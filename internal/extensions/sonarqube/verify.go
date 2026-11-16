package sonarqube

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Remediation receipt states. Only "verified" means the provider proved the
// fix; every other state is unknown/unverified or failed.
const (
	ReceiptVerified   = "verified"
	ReceiptFailed     = "failed"     // at least one issue is still open on a fresh scan
	ReceiptUnverified = "unverified" // pending, stale, missing or failed scan; or issues not provably fixed
)

// Per-issue outcomes.
const (
	IssueFixed     = "fixed"     // closed by the provider with resolution FIXED
	IssueOpen      = "open"      // still open on the authoritative scan
	IssueDismissed = "dismissed" // accepted / false positive / won't fix: not a code fix
	IssueMissing   = "missing"   // not returned by the provider (purged, wrong branch, wrong key)
	IssueUnknown   = "unknown"   // returned with a state GIRL does not recognise
)

type Analysis struct {
	Key      string `json:"key"`
	Date     string `json:"date"`
	Revision string `json:"revision"`
}

type IssueOutcome struct {
	Key    string `json:"key"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// Receipt is an authoritative remediation record tied to one provider scan.
type Receipt struct {
	Provider string         `json:"provider"`
	Status   string         `json:"status"`
	Reasons  []string       `json:"reasons,omitempty"`
	Scan     ScanIdentity   `json:"scan"`
	Issues   []IssueOutcome `json:"issues"`
}

// VerifyRequest names the issues an approved fix claims to resolve and the
// revision that contains the fix.
type VerifyRequest struct {
	IssueKeys []string
	Revision  string
}

// LatestAnalysis returns the most recent analysis for the configured scope.
func (c *Client) LatestAnalysis(ctx context.Context) (*Analysis, error) {
	q := url.Values{}
	q.Set("project", c.cfg.ProjectKey)
	q.Set("ps", "1")
	c.scopeParams(q)
	var res struct {
		Analyses []Analysis `json:"analyses"`
	}
	if err := c.get(ctx, "/api/project_analyses/search", q, &res); err != nil {
		return nil, err
	}
	if len(res.Analyses) == 0 {
		return nil, nil
	}
	return &res.Analyses[0], nil
}

type ceTask struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	AnalysisID  string `json:"analysisId"`
	Branch      string `json:"branch"`
	PullRequest string `json:"pullRequest"`
}

// backgroundTasks returns queued tasks and the current (last finished) task.
func (c *Client) backgroundTasks(ctx context.Context) ([]ceTask, *ceTask, error) {
	q := url.Values{}
	q.Set("component", c.cfg.ProjectKey)
	if c.cfg.Organization != "" {
		q.Set("organization", c.cfg.Organization)
	}
	var res struct {
		Queue   []ceTask `json:"queue"`
		Current *ceTask  `json:"current"`
	}
	if err := c.get(ctx, "/api/ce/component", q, &res); err != nil {
		return nil, nil, err
	}
	return res.Queue, res.Current, nil
}

func (c *Client) taskInScope(t ceTask) bool {
	if c.cfg.PullRequest != "" {
		return t.PullRequest == c.cfg.PullRequest
	}
	if c.cfg.Branch != "" {
		return t.Branch == c.cfg.Branch
	}
	return t.PullRequest == "" // main branch tasks carry no PR
}

func revisionMatches(scanRev, want string) bool {
	if scanRev == "" || want == "" {
		return false
	}
	if len(want) < 7 {
		return false // too short to identify a commit unambiguously
	}
	return strings.HasPrefix(scanRev, want) || (len(scanRev) >= 7 && strings.HasPrefix(want, scanRev))
}

// VerifyRemediation checks, against a fresh authoritative scan of the fixing
// revision, whether the named issues are fixed. Missing evidence never becomes
// success: pending, stale, failed or absent scans yield "unverified".
func (c *Client) VerifyRemediation(ctx context.Context, req VerifyRequest) (*Receipt, error) {
	r := &Receipt{
		Provider: "sonarqube",
		Status:   ReceiptUnverified,
		Scan:     ScanIdentity{Project: c.cfg.ProjectKey, Branch: c.cfg.Branch, PullRequest: c.cfg.PullRequest},
		Issues:   []IssueOutcome{},
	}
	if len(req.IssueKeys) == 0 {
		return nil, fmt.Errorf("%w: at least one issue key is required", ErrConfig)
	}
	if req.Revision == "" {
		return nil, fmt.Errorf("%w: the fixing revision (commit SHA) is required", ErrConfig)
	}
	for _, k := range req.IssueKeys {
		r.Issues = append(r.Issues, IssueOutcome{Key: k, Status: IssueUnknown})
	}
	unverified := func(format string, args ...any) (*Receipt, error) {
		r.Status = ReceiptUnverified
		r.Reasons = append(r.Reasons, fmt.Sprintf(format, args...))
		return r, nil
	}

	if st, err := c.Status(ctx); err != nil {
		return nil, err
	} else {
		r.Scan.ServerVersion = st.Version
	}

	queue, current, err := c.backgroundTasks(ctx)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	for _, t := range queue {
		if c.taskInScope(t) {
			return unverified("analysis task %s is %s; wait for it to finish", t.ID, strings.ToLower(t.Status))
		}
	}
	if current != nil && c.taskInScope(*current) && current.Status != "SUCCESS" {
		return unverified("latest analysis task %s ended with status %s", current.ID, current.Status)
	}

	analysis, err := c.LatestAnalysis(ctx)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return unverified("no analysis found for the requested project/branch")
		}
		return nil, err
	}
	if analysis == nil {
		return unverified("no analysis found for the requested project/branch")
	}
	r.Scan.AnalysisKey, r.Scan.AnalysisDate, r.Scan.Revision = analysis.Key, analysis.Date, analysis.Revision
	if analysis.Revision == "" {
		return unverified("latest analysis %s has no SCM revision; configure the scanner to record it", analysis.Key)
	}
	if !revisionMatches(analysis.Revision, req.Revision) {
		return unverified("latest analysis is of revision %s, not the fixing revision %s (stale scan)", analysis.Revision, req.Revision)
	}

	set, err := c.IssuesByKey(ctx, req.IssueKeys)
	if err != nil {
		return nil, err
	}
	byKey := map[string]Issue{}
	for _, is := range set.Issues {
		byKey[is.Key] = is
	}
	anyOpen, allFixed := false, true
	for i, out := range r.Issues {
		is, ok := byKey[out.Key]
		if !ok {
			r.Issues[i] = IssueOutcome{Key: out.Key, Status: IssueMissing, Reason: "issue not returned for this project/branch"}
			allFixed = false
			continue
		}
		r.Issues[i] = classify(is)
		switch r.Issues[i].Status {
		case IssueFixed:
		case IssueOpen:
			anyOpen = true
			allFixed = false
		default:
			allFixed = false
		}
	}
	switch {
	case anyOpen:
		r.Status = ReceiptFailed
		r.Reasons = append(r.Reasons, "one or more issues are still open on the fixing revision's scan")
	case allFixed:
		r.Status = ReceiptVerified
	default:
		r.Status = ReceiptUnverified
		r.Reasons = append(r.Reasons, "not every issue is provably fixed by the provider")
	}
	return r, nil
}

func classify(is Issue) IssueOutcome {
	out := IssueOutcome{Key: is.Key}
	// SonarQube 10.4+ issueStatus.
	switch strings.ToUpper(is.IssueStatus) {
	case "FIXED":
		out.Status = IssueFixed
		return out
	case "OPEN", "CONFIRMED":
		out.Status = IssueOpen
		return out
	case "ACCEPTED", "FALSE_POSITIVE":
		out.Status, out.Reason = IssueDismissed, "dismissed in the provider ("+strings.ToLower(is.IssueStatus)+"), not fixed in code"
		return out
	}
	// Legacy status/resolution.
	switch strings.ToUpper(is.Status) {
	case "OPEN", "REOPENED", "CONFIRMED":
		out.Status = IssueOpen
	case "CLOSED", "RESOLVED":
		if strings.EqualFold(is.Resolution, "FIXED") {
			out.Status = IssueFixed
		} else if is.Resolution == "" {
			out.Status, out.Reason = IssueUnknown, "closed without a resolution"
		} else if is.Resolution == "REMOVED" {
			out.Status, out.Reason = IssueUnknown, "rule or file removed; not evidence of a code fix"
		} else {
			out.Status, out.Reason = IssueDismissed, "resolved as "+strings.ToLower(is.Resolution)+", not fixed in code"
		}
	default:
		out.Status, out.Reason = IssueUnknown, "unrecognised provider state "+is.Status
	}
	return out
}
