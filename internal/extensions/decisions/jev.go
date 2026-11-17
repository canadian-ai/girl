package decisions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Endpoint presets. TypeSafe is the model's own API; AIML API and Eden AI
// resell it with the same request and response bodies on different paths.
var presets = map[string]struct {
	BaseURL, Path, Model, KeyEnv string
}{
	"typesafe": {"https://api.typesafe.ai", "/v1/systemone", "jev-latest", "TYPESAFE_API_KEY"},
	"aimlapi":  {"https://api.aimlapi.com", "/v1/decisions", "typesafe/jev", "AIMLAPI_API_KEY"},
	"eden":     {"https://api.edenai.run", "/v3/alpha/decisions", "typesafe/jev-latest", "EDENAI_API_KEY"},
}

// Presets lists the supported endpoint presets.
func Presets() []string { return []string{"typesafe", "aimlapi", "eden"} }

// KeyEnv returns the environment variable that holds the key for a preset.
func KeyEnv(preset string) string { return presets[preset].KeyEnv }

// JevConfig configures the Jev provider. Token is never serialized, logged or
// included in errors.
type JevConfig struct {
	Preset  string // typesafe (default), aimlapi or eden
	BaseURL string // overrides the preset
	Model   string // overrides the preset, e.g. "jev-1.14" to pin a version
	Token   string `json:"-"`
	Timeout time.Duration
	// MaxRetries for 429 and 5xx responses; default 2, negative disables.
	MaxRetries int
	// MaxStateBytes caps the serialized state; default 64 KiB. Larger states
	// are refused rather than truncated.
	MaxStateBytes     int
	AllowInsecureHTTP bool
	HTTPClient        *http.Client
	sleep             func(time.Duration)
}

// Jev is a Provider backed by TypeSafe's System One API.
type Jev struct {
	cfg      JevConfig
	endpoint *url.URL
	model    string
	http     *http.Client
}

var (
	ErrConfig       = errors.New("jev: invalid configuration")
	ErrUnauthorized = errors.New("jev: authentication failed (check the API key)")
	ErrForbidden    = errors.New("jev: key is not permitted to use this model")
	ErrRateLimited  = errors.New("jev: rate limited")
	ErrUnavailable  = errors.New("jev: service unavailable")
	ErrRejected     = errors.New("jev: request rejected")
)

// NewJev builds a provider. The token defaults to the preset's environment
// variable (TYPESAFE_API_KEY for the TypeSafe preset).
func NewJev(cfg JevConfig) (*Jev, error) {
	if cfg.Preset == "" {
		cfg.Preset = "typesafe"
	}
	p, ok := presets[cfg.Preset]
	if !ok {
		return nil, fmt.Errorf("%w: unknown preset %q", ErrConfig, cfg.Preset)
	}
	base := p.BaseURL
	if cfg.BaseURL != "" {
		base = cfg.BaseURL
	}
	u, err := url.Parse(strings.TrimRight(base, "/") + p.Path)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, fmt.Errorf("%w: base URL must be http(s)://host", ErrConfig)
	}
	if u.Scheme == "http" && !cfg.AllowInsecureHTTP && !isLoopback(u.Hostname()) {
		return nil, fmt.Errorf("%w: refusing to send a key over plain http to %s", ErrConfig, u.Hostname())
	}
	if cfg.Token == "" {
		cfg.Token = os.Getenv(p.KeyEnv)
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("%w: set %s", ErrConfig, p.KeyEnv)
	}
	model := p.Model
	if cfg.Model != "" {
		model = cfg.Model
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 2
	} else if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.MaxStateBytes <= 0 {
		cfg.MaxStateBytes = 64 << 10
	}
	if cfg.sleep == nil {
		cfg.sleep = time.Sleep
	}
	hc := http.Client{Timeout: cfg.Timeout}
	if cfg.HTTPClient != nil {
		hc = *cfg.HTTPClient
	}
	// Never forward the key across a redirect to another host.
	hc.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 && req.URL.Host != via[0].URL.Host {
			return fmt.Errorf("%w: refusing cross-host redirect", ErrConfig)
		}
		return nil
	}
	return &Jev{cfg: cfg, endpoint: u, model: model, http: &hc}, nil
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (j *Jev) Name() string { return "jev" }

// Host is where requests are sent; shown to users before anything is sent.
func (j *Jev) Host() string { return j.endpoint.Host }

// Model is the configured model name or alias.
func (j *Jev) Model() string { return j.model }

// String never includes the key.
func (j *Jev) String() string { return fmt.Sprintf("jev(%s, %s)", j.endpoint.Host, j.model) }

func (j *Jev) Decide(ctx context.Context, req Request) (*Result, error) {
	state, err := json.Marshal(req.State)
	if err != nil {
		return nil, fmt.Errorf("%w: state is not JSON-encodable: %v", ErrConfig, err)
	}
	if len(state) > j.cfg.MaxStateBytes {
		return nil, fmt.Errorf("%w: state is %d bytes (limit %d)", ErrRejected, len(state), j.cfg.MaxStateBytes)
	}
	body, err := json.Marshal(map[string]any{"model": j.model, "state": req.State, "questions": req.Questions})
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt <= j.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(attempt) * time.Second
			var ra retryAfter
			if errors.As(lastErr, &ra) && ra.after > 0 {
				wait = ra.after
			}
			j.cfg.sleep(wait)
		}
		res, err := j.post(ctx, body)
		if err == nil {
			res.Provider = "jev"
			if verr := Validate(req, res); verr != nil {
				return nil, verr
			}
			return res, nil
		}
		var ra retryAfter
		if !errors.As(err, &ra) {
			return nil, err
		}
		lastErr = err
	}
	var ra retryAfter
	if errors.As(lastErr, &ra) {
		return nil, ra.err
	}
	return nil, lastErr
}

type retryAfter struct {
	err   error
	after time.Duration
}

func (r retryAfter) Error() string { return r.err.Error() }
func (r retryAfter) Unwrap() error { return r.err }

func (j *Jev) post(ctx context.Context, body []byte) (*Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, j.endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+j.cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "girl-decisions")
	resp, err := j.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, retryAfter{err: fmt.Errorf("%w: %s", ErrUnavailable, redactURLError(err))}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, retryAfter{err: fmt.Errorf("%w: reading response: %v", ErrUnavailable, err)}
	}
	after := parseRetryAfter(resp.Header.Get("Retry-After"))
	switch {
	case resp.StatusCode == http.StatusOK:
		var res Result
		if err := json.Unmarshal(data, &res); err != nil {
			return nil, fmt.Errorf("%w: invalid JSON: %v", ErrInvalidAnswer, err)
		}
		return &res, nil
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case resp.StatusCode == http.StatusForbidden:
		return nil, ErrForbidden
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, retryAfter{err: ErrRateLimited, after: after}
	case resp.StatusCode >= 500:
		return nil, retryAfter{err: fmt.Errorf("%w: HTTP %d", ErrUnavailable, resp.StatusCode), after: after}
	default:
		return nil, fmt.Errorf("%w: HTTP %d: %s", ErrRejected, resp.StatusCode, apiDetail(data))
	}
}

func parseRetryAfter(v string) time.Duration {
	secs, err := strconv.Atoi(v)
	if err != nil || secs < 0 {
		return 0
	}
	if secs > 60 {
		secs = 60
	}
	return time.Duration(secs) * time.Second
}

// apiDetail extracts a short error message without echoing request content.
func apiDetail(data []byte) string {
	var e struct {
		Detail any    `json:"detail"`
		Error  any    `json:"error"`
		Msg    string `json:"message"`
	}
	if json.Unmarshal(data, &e) == nil {
		for _, v := range []any{e.Msg, e.Error, e.Detail} {
			if s, ok := v.(string); ok && s != "" {
				if len(s) > 200 {
					s = s[:200] + "…"
				}
				return s
			}
		}
	}
	return "no error detail"
}

func redactURLError(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) {
		if u, perr := url.Parse(ue.URL); perr == nil {
			u.RawQuery, u.User = "", nil
			return fmt.Sprintf("%s %s: %v", ue.Op, u.String(), ue.Err)
		}
	}
	return err.Error()
}
