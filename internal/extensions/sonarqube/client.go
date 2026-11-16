// Package sonarqube is a read-only SonarQube / SonarCloud provider for GIRL's
// governed extension contracts.
//
// STATUS: implemented against the documented SonarQube Web API (10.x / 2025.x
// and SonarCloud) and tested only against a mock server. It has not been
// exercised against a live instance; see docs/extensions/sonarqube.md.
//
// Endpoints used (all GET, all read-only):
//
//	/api/system/status            server reachability and version (no auth)
//	/api/authentication/validate  token validity
//	/api/issues/search            open issues, and specific issues by key
//	/api/project_analyses/search  latest analysis and its SCM revision
//	/api/ce/component             pending / failed background analysis tasks
//
// The adapter never writes to SonarQube (no transitions, no comments) and no
// provider finding can authorize a code change: findings become proposals that
// must pass extensions.Evaluate under an explicit policy.
package sonarqube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Config selects the server, project and scope. Token is never logged,
// serialized or included in errors.
type Config struct {
	BaseURL      string
	Token        string `json:"-"`
	Organization string // SonarCloud only
	ProjectKey   string
	Branch       string
	PullRequest  string
	// AuthScheme is "bearer" (SonarQube 10+, SonarCloud) or "basic"
	// (legacy: token as username, empty password). Default bearer.
	AuthScheme string
	Timeout    time.Duration
	MaxRetries int
	// PageSize for issue search; SonarQube caps it at 500.
	PageSize int
	// AllowInsecureHTTP permits plain http for non-loopback hosts. Off by
	// default so a token is never sent in clear text by accident.
	AllowInsecureHTTP bool
	HTTPClient        *http.Client
	// sleep is replaceable in tests.
	sleep func(time.Duration)
}

// Client is a read-only SonarQube Web API client.
type Client struct {
	cfg  Config
	base *url.URL
	http *http.Client
}

// Error categories make connectivity and authorization failures explicit.
var (
	ErrConfig        = errors.New("sonarqube: invalid configuration")
	ErrUnreachable   = errors.New("sonarqube: server unreachable")
	ErrUnauthorized  = errors.New("sonarqube: authentication failed (check SONAR_TOKEN)")
	ErrForbidden     = errors.New("sonarqube: token lacks Browse permission on the project")
	ErrNotFound      = errors.New("sonarqube: project, branch or resource not found")
	ErrRateLimited   = errors.New("sonarqube: rate limited")
	ErrServer        = errors.New("sonarqube: server error")
	ErrPagingCeiling = errors.New("sonarqube: result set exceeds the 10,000 issue search ceiling; narrow the query")
)

// searchCeiling is the hard limit of /api/issues/search (p * ps <= 10000).
const searchCeiling = 10000

func NewClient(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" || cfg.ProjectKey == "" {
		return nil, fmt.Errorf("%w: base URL and project key are required", ErrConfig)
	}
	u, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, fmt.Errorf("%w: base URL must be http(s)://host", ErrConfig)
	}
	if u.Scheme == "http" && !cfg.AllowInsecureHTTP && !isLoopback(u.Hostname()) {
		return nil, fmt.Errorf("%w: refusing to send a token over plain http to %s", ErrConfig, u.Hostname())
	}
	if cfg.Branch != "" && cfg.PullRequest != "" {
		return nil, fmt.Errorf("%w: choose a branch or a pull request, not both", ErrConfig)
	}
	switch cfg.AuthScheme {
	case "":
		cfg.AuthScheme = "bearer"
	case "bearer", "basic":
	default:
		return nil, fmt.Errorf("%w: unknown auth scheme %q", ErrConfig, cfg.AuthScheme)
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	} else if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.PageSize <= 0 || cfg.PageSize > 500 {
		cfg.PageSize = 500
	}
	if cfg.sleep == nil {
		cfg.sleep = time.Sleep
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: cfg.Timeout}
	}
	// Never forward credentials across a redirect to another host.
	base := *hc
	base.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 && req.URL.Host != via[0].URL.Host {
			return fmt.Errorf("%w: refusing cross-host redirect", ErrConfig)
		}
		return nil
	}
	return &Client{cfg: cfg, base: u, http: &base}, nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// String never includes the token.
func (c *Client) String() string {
	return fmt.Sprintf("sonarqube(%s, project=%s)", c.base.Host, c.cfg.ProjectKey)
}

func (c *Client) scopeParams(q url.Values) {
	if c.cfg.Branch != "" {
		q.Set("branch", c.cfg.Branch)
	}
	if c.cfg.PullRequest != "" {
		q.Set("pullRequest", c.cfg.PullRequest)
	}
	if c.cfg.Organization != "" {
		q.Set("organization", c.cfg.Organization)
	}
}

// get performs a GET with retry on 429/502/503/504 and transport errors,
// honoring Retry-After. Errors never include the token or response bodies
// beyond SonarQube's own error messages.
func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	u := *c.base
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawQuery = q.Encode()
	var lastErr error
	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(math.Pow(2, float64(attempt-1))) * 500 * time.Millisecond
			if ra, ok := lastErr.(retryAfter); ok && ra.after > 0 {
				wait = ra.after
			}
			c.cfg.sleep(wait)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "girl-sonarqube-provider")
		if c.cfg.Token != "" {
			if c.cfg.AuthScheme == "basic" {
				req.SetBasicAuth(c.cfg.Token, "")
			} else {
				req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
			}
		}
		resp, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastErr = fmt.Errorf("%w: %s", ErrUnreachable, redactURLError(err))
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("%w: reading response: %v", ErrUnreachable, readErr)
			continue
		}
		switch {
		case resp.StatusCode == http.StatusOK:
			if out == nil {
				return nil
			}
			if err := json.Unmarshal(body, out); err != nil {
				return fmt.Errorf("%w: invalid JSON from %s: %v", ErrServer, path, err)
			}
			return nil
		case resp.StatusCode == http.StatusUnauthorized:
			return ErrUnauthorized
		case resp.StatusCode == http.StatusForbidden:
			return ErrForbidden
		case resp.StatusCode == http.StatusNotFound:
			return fmt.Errorf("%w: %s", ErrNotFound, apiMessage(body))
		case resp.StatusCode == http.StatusBadRequest:
			return fmt.Errorf("%w: %s", ErrConfig, apiMessage(body))
		case resp.StatusCode == http.StatusTooManyRequests:
			lastErr = retryAfter{error: ErrRateLimited, after: parseRetryAfter(resp.Header.Get("Retry-After"))}
		case resp.StatusCode == http.StatusBadGateway, resp.StatusCode == http.StatusServiceUnavailable, resp.StatusCode == http.StatusGatewayTimeout:
			lastErr = retryAfter{error: fmt.Errorf("%w: HTTP %d", ErrServer, resp.StatusCode), after: parseRetryAfter(resp.Header.Get("Retry-After"))}
		default:
			return fmt.Errorf("%w: HTTP %d: %s", ErrServer, resp.StatusCode, apiMessage(body))
		}
	}
	if ra, ok := lastErr.(retryAfter); ok {
		return ra.error
	}
	return lastErr
}

type retryAfter struct {
	error
	after time.Duration
}

func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		if secs > 60 {
			secs = 60
		}
		return time.Duration(secs) * time.Second
	}
	return 0
}

func apiMessage(body []byte) string {
	var e struct {
		Errors []struct {
			Msg string `json:"msg"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &e) == nil && len(e.Errors) > 0 {
		var msgs []string
		for _, m := range e.Errors {
			msgs = append(msgs, m.Msg)
		}
		return strings.Join(msgs, "; ")
	}
	return "no error detail"
}

// redactURLError strips query strings from transport errors.
func redactURLError(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) {
		if u, perr := url.Parse(ue.URL); perr == nil {
			u.RawQuery = ""
			u.User = nil
			return fmt.Sprintf("%s %s: %v", ue.Op, u.String(), ue.Err)
		}
	}
	return err.Error()
}

// ServerStatus is /api/system/status.
type ServerStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Status checks reachability. It fails unless the server reports UP.
func (c *Client) Status(ctx context.Context) (*ServerStatus, error) {
	var s ServerStatus
	if err := c.get(ctx, "/api/system/status", url.Values{}, &s); err != nil {
		return nil, err
	}
	if s.Status != "UP" {
		return &s, fmt.Errorf("%w: server status is %q", ErrUnreachable, s.Status)
	}
	return &s, nil
}

// ValidateToken checks the token with /api/authentication/validate.
func (c *Client) ValidateToken(ctx context.Context) error {
	var v struct {
		Valid bool `json:"valid"`
	}
	if err := c.get(ctx, "/api/authentication/validate", url.Values{}, &v); err != nil {
		return err
	}
	if !v.Valid {
		return ErrUnauthorized
	}
	return nil
}
