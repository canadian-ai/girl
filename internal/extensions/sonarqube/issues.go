package sonarqube

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/canadian-ai/girl/internal/extensions"
)

// Issue is the subset of /api/issues/search used by GIRL. Both the legacy
// fields (severity, type, status, resolution) and the Clean Code / MQR fields
// introduced in SonarQube 10.2+ (impacts, issueStatus) are read; newer fields
// win when present.
type Issue struct {
	Key         string `json:"key"`
	Rule        string `json:"rule"`
	Severity    string `json:"severity"`
	Component   string `json:"component"`
	Project     string `json:"project"`
	Line        int    `json:"line"`
	Message     string `json:"message"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Resolution  string `json:"resolution"`
	IssueStatus string `json:"issueStatus"`
	UpdateDate  string `json:"updateDate"`
	TextRange   *struct {
		StartLine int `json:"startLine"`
		EndLine   int `json:"endLine"`
	} `json:"textRange"`
	Impacts []struct {
		SoftwareQuality string `json:"softwareQuality"`
		Severity        string `json:"severity"`
	} `json:"impacts"`
}

type component struct {
	Key  string `json:"key"`
	Path string `json:"path"`
}

type issuesPage struct {
	Total  int `json:"total"`
	Paging struct {
		PageIndex int `json:"pageIndex"`
		PageSize  int `json:"pageSize"`
		Total     int `json:"total"`
	} `json:"paging"`
	Issues     []Issue     `json:"issues"`
	Components []component `json:"components"`
}

// IssueSet is the result of a search, with the component->path index needed
// to map issues onto repository paths.
type IssueSet struct {
	Issues []Issue
	paths  map[string]string
}

// searchIssues pages through /api/issues/search until all results are read.
func (c *Client) searchIssues(ctx context.Context, q url.Values) (*IssueSet, error) {
	set := &IssueSet{paths: map[string]string{}}
	ps := c.cfg.PageSize
	for page := 1; ; page++ {
		if page*ps > searchCeiling && page > 1 {
			return nil, ErrPagingCeiling
		}
		pq := url.Values{}
		for k, v := range q {
			pq[k] = v
		}
		pq.Set("p", strconv.Itoa(page))
		pq.Set("ps", strconv.Itoa(ps))
		var res issuesPage
		if err := c.get(ctx, "/api/issues/search", pq, &res); err != nil {
			return nil, err
		}
		for _, comp := range res.Components {
			if comp.Path != "" {
				set.paths[comp.Key] = comp.Path
			}
		}
		set.Issues = append(set.Issues, res.Issues...)
		total := res.Paging.Total
		if total == 0 {
			total = res.Total
		}
		if total > searchCeiling {
			return nil, ErrPagingCeiling
		}
		if len(res.Issues) == 0 || len(set.Issues) >= total {
			return set, nil
		}
	}
}

// OpenIssues lists unresolved issues for the configured project and scope.
func (c *Client) OpenIssues(ctx context.Context) (*IssueSet, error) {
	q := url.Values{}
	q.Set("components", c.cfg.ProjectKey)
	q.Set("resolved", "false")
	q.Set("additionalFields", "_all")
	c.scopeParams(q)
	return c.searchIssues(ctx, q)
}

// IssuesByKey fetches specific issues, resolved or not.
func (c *Client) IssuesByKey(ctx context.Context, keys []string) (*IssueSet, error) {
	q := url.Values{}
	q.Set("issues", strings.Join(keys, ","))
	q.Set("components", c.cfg.ProjectKey)
	c.scopeParams(q)
	return c.searchIssues(ctx, q)
}

// Path returns the repository-relative path of an issue's component.
func (s *IssueSet) Path(is Issue) string {
	if p, ok := s.paths[is.Component]; ok {
		return p
	}
	// Component keys are "<projectKey>:<path>".
	if i := strings.Index(is.Component, ":"); i >= 0 {
		return is.Component[i+1:]
	}
	return ""
}

var legacySeverity = map[string]string{"BLOCKER": "critical", "CRITICAL": "high", "MAJOR": "medium", "MINOR": "low", "INFO": "info"}
var impactSeverity = map[string]string{"BLOCKER": "critical", "HIGH": "high", "MEDIUM": "medium", "LOW": "low", "INFO": "info"}
var severityRank = map[string]int{"info": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}

// NormalizedSeverity prefers the highest MQR impact, falling back to the
// legacy severity.
func NormalizedSeverity(is Issue) string {
	best := ""
	for _, imp := range is.Impacts {
		if s, ok := impactSeverity[strings.ToUpper(imp.Severity)]; ok && (best == "" || severityRank[s] > severityRank[best]) {
			best = s
		}
	}
	if best != "" {
		return best
	}
	if s, ok := legacySeverity[strings.ToUpper(is.Severity)]; ok {
		return s
	}
	return "medium"
}

// IsSecurity reports vulnerabilities, hotspots and SECURITY impacts.
func IsSecurity(is Issue) bool {
	if is.Type == "VULNERABILITY" || is.Type == "SECURITY_HOTSPOT" {
		return true
	}
	for _, imp := range is.Impacts {
		if strings.EqualFold(imp.SoftwareQuality, "SECURITY") {
			return true
		}
	}
	return false
}

// ScanIdentity ties findings to one authoritative provider analysis.
type ScanIdentity struct {
	ServerVersion string `json:"serverVersion,omitempty"`
	Project       string `json:"project"`
	Branch        string `json:"branch,omitempty"`
	PullRequest   string `json:"pullRequest,omitempty"`
	AnalysisKey   string `json:"analysisKey,omitempty"`
	AnalysisDate  string `json:"analysisDate,omitempty"`
	Revision      string `json:"revision,omitempty"`
}

// Rejected is an issue that could not be normalized (e.g. unsafe path).
type Rejected struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

// Normalize converts issues into governed Findings with provenance.
func Normalize(set *IssueSet, scan ScanIdentity) ([]extensions.Finding, []Rejected) {
	var findings []extensions.Finding
	var rejected []Rejected
	for _, is := range set.Issues {
		line := is.Line
		if line < 1 && is.TextRange != nil {
			line = is.TextRange.StartLine
		}
		fileLevel := line < 1
		if fileLevel {
			line = 1
		}
		f, err := extensions.Normalize(extensions.Finding{
			Provider:   "sonarqube",
			ExternalID: is.Key,
			RuleID:     is.Rule,
			File:       set.Path(is),
			Line:       line,
			Severity:   NormalizedSeverity(is),
			Message:    cleanProviderText(is.Message),
		})
		if err != nil {
			rejected = append(rejected, Rejected{Key: is.Key, Reason: err.Error()})
			continue
		}
		f.Provenance = map[string]string{
			"project":     scan.Project,
			"analysisKey": scan.AnalysisKey,
			"revision":    scan.Revision,
			"type":        is.Type,
		}
		if scan.Branch != "" {
			f.Provenance["branch"] = scan.Branch
		}
		if scan.PullRequest != "" {
			f.Provenance["pullRequest"] = scan.PullRequest
		}
		if fileLevel {
			f.Provenance["location"] = "file"
		}
		if IsSecurity(is) {
			f.Provenance["category"] = "security"
		}
		findings = append(findings, f)
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].Fingerprint < findings[j].Fingerprint })
	return findings, rejected
}

// cleanProviderText bounds free text that originates outside the repository.
// Issue messages are untrusted input that may end up in an agent's context:
// control characters are dropped and length is capped so a provider-side
// message cannot smuggle terminal escapes or an oversized prompt.
func cleanProviderText(s string) string {
	const max = 500
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' {
			r = ' '
		}
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) || r == 0x1b {
			continue
		}
		b.WriteRune(r)
		if b.Len() >= max {
			b.WriteString("…")
			break
		}
	}
	return b.String()
}

// ruleRecipes maps Sonar rule numbers (language-independent suffix) to GRP
// recipe stems. Only structural refactors with deterministic recipes are
// mapped; everything else is reported as unmapped for manual handling.
var ruleRecipes = map[string]string{
	"S3776": "simplify-branches", // cognitive complexity
	"S1541": "simplify-branches", // cyclomatic complexity
	"S134":  "flatten-nesting",   // nesting depth
	"S1066": "flatten-nesting",   // collapsible if
	"S138":  "extract-function",  // function too long
	"S107":  "params",            // too many parameters
	"S104":  "split",             // file too long
}

var langPrefix = map[string]string{"python": "py", "py": "py", "typescript": "ts", "ts": "ts", "javascript": "ts", "js": "ts", "go": "go", "rust": "rust"}

func recipeFor(rule string) (string, bool) {
	repo, num, ok := strings.Cut(rule, ":")
	if !ok {
		return "", false
	}
	lang, ok := langPrefix[strings.ToLower(repo)]
	if !ok {
		return "", false
	}
	stem, ok := ruleRecipes[num]
	if !ok {
		return "", false
	}
	switch stem {
	case "params":
		switch lang {
		case "py":
			return "py.group-parameters", true
		case "go":
			return "go.extract-options-struct", true
		case "rust":
			return "rust.extract-options-struct", true
		}
		return "", false
	case "split":
		switch lang {
		case "py":
			return "py.split-module", true
		case "go", "rust":
			return lang + ".split-file", true
		}
		return "", false
	case "flatten-nesting":
		if lang == "ts" {
			return "ts.simplify-branches", true
		}
	}
	return lang + "." + stem, true
}

// Unmapped is a finding with no bounded GRP recipe; it needs manual handling.
type Unmapped struct {
	Finding extensions.Finding `json:"finding"`
	Reason  string             `json:"reason"`
}

// DefaultChangedLines is the budget requested per proposed action. It is a
// request, not an authorization: extensions.Evaluate decides.
const DefaultChangedLines = 80

// ToActions maps findings to bounded GRP action proposals. A proposal carries
// no authority; the caller must Evaluate it against an explicit Policy.
func ToActions(findings []extensions.Finding) ([]extensions.Action, []Unmapped) {
	var actions []extensions.Action
	var unmapped []Unmapped
	for _, f := range findings {
		recipe, ok := recipeFor(f.RuleID)
		if !ok {
			unmapped = append(unmapped, Unmapped{Finding: f, Reason: fmt.Sprintf("no bounded GRP recipe for rule %s", f.RuleID)})
			continue
		}
		risk := "low"
		switch f.Severity {
		case "high":
			risk = "medium"
		case "critical":
			risk = "high"
		}
		actions = append(actions, extensions.Action{
			ID:           "sonar_" + f.Fingerprint[:16],
			Finding:      f,
			Recipe:       recipe,
			Files:        []string{f.File},
			ChangedLines: DefaultChangedLines,
			Risk:         risk,
			Category:     f.Provenance["category"],
		})
	}
	return actions, unmapped
}
