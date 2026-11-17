// Package decisions connects GIRL to hosted decision models (typed,
// probabilistic answers rather than generated text), starting with
// TypeSafe's Jev.
//
// The one rule: a decision may only make GIRL stricter. It can escalate a
// change to human review, lower its priority, or choose among recipes GIRL
// already registers. It can never turn a denial into an approval, mark
// anything verified, skip verification or widen a policy. Low-confidence or
// invalid answers are discarded and GIRL behaves exactly as it would without
// a model.
package decisions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Question types, matching TypeSafe's System One API.
const (
	TypeNoul   = "noul"   // yes/no; answer is the probability of yes
	TypeChoice = "choice" // one key from criteria
	TypeScore  = "score"  // ordered levels, lowest first
)

// Question is one typed question about the request state.
type Question struct {
	Type         string `json:"type"`
	Instructions any    `json:"instructions,omitempty"`
	// Criteria is map[string]string for choice and []string for score.
	Criteria any `json:"criteria,omitempty"`
}

// Request is the provider-neutral decision request.
type Request struct {
	State     any                 `json:"state"`
	Questions map[string]Question `json:"questions"`
}

// Answer is one typed answer.
type Answer struct {
	Type          string             `json:"type"`
	Noul          *float64           `json:"noul,omitempty"`
	Choice        string             `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"`
	Confidence    float64            `json:"confidence,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Legend        map[string]string  `json:"legend,omitempty"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Result is a validated provider response plus audit metadata.
type Result struct {
	Provider  string            `json:"provider"`
	Model     string            `json:"model"`
	Answers   map[string]Answer `json:"answers"`
	Usage     Usage             `json:"usage"`
	InputHash string            `json:"inputHash"`
	Cached    bool              `json:"cached,omitempty"`
}

// Provider answers decision requests.
type Provider interface {
	Name() string
	Decide(ctx context.Context, req Request) (*Result, error)
}

// ErrInvalidAnswer means a response did not match the questions asked. Such a
// response is discarded, never partially trusted.
var ErrInvalidAnswer = errors.New("decisions: provider answer does not match the request")

// HashRequest is a stable digest of the request and model, used for caching
// and for recording exactly what a decision was based on.
func HashRequest(model string, req Request) string {
	// encoding/json sorts map keys, so the encoding is canonical.
	data, _ := json.Marshal(struct {
		Model string  `json:"model"`
		Req   Request `json:"req"`
	}{model, req})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func inUnit(p float64) bool { return p >= 0 && p <= 1 }

// Validate checks that every question has an answer of the right type with
// values in range, and that a choice is one of the offered keys.
func Validate(req Request, res *Result) error {
	if res == nil {
		return fmt.Errorf("%w: empty result", ErrInvalidAnswer)
	}
	for key, q := range req.Questions {
		a, ok := res.Answers[key]
		if !ok || a.Type != q.Type {
			return fmt.Errorf("%w: question %q missing or wrong type", ErrInvalidAnswer, key)
		}
		switch q.Type {
		case TypeNoul:
			if a.Noul == nil || !inUnit(*a.Noul) {
				return fmt.Errorf("%w: %q has no valid probability", ErrInvalidAnswer, key)
			}
		case TypeChoice:
			opts, _ := q.Criteria.(map[string]string)
			if _, ok := opts[a.Choice]; !ok || !inUnit(a.Confidence) {
				return fmt.Errorf("%w: %q chose an option that was not offered", ErrInvalidAnswer, key)
			}
		case TypeScore:
			levels, _ := q.Criteria.([]string)
			if a.Score == nil || *a.Score < 0 || *a.Score > float64(len(levels)-1) || !inUnit(a.Confidence) {
				return fmt.Errorf("%w: %q has no valid score", ErrInvalidAnswer, key)
			}
		default:
			return fmt.Errorf("%w: unsupported question type %q", ErrInvalidAnswer, q.Type)
		}
		for _, p := range a.Probabilities {
			if !inUnit(p) {
				return fmt.Errorf("%w: %q has an out-of-range probability", ErrInvalidAnswer, key)
			}
		}
	}
	return nil
}

// Cache stores results by input hash so repeated runs are reproducible and
// do not re-send the same content.
type Cache interface {
	Get(hash string) (*Result, bool)
	Put(hash string, res *Result) error
}

// DirCache is a file cache in a private directory (0700 / 0600). Entries
// older than MaxAge (default 30 days) are ignored, so answers cached under a
// floating alias such as jev-latest refresh after the provider promotes a new
// model. Pin a model version for strict reproducibility.
type DirCache struct {
	Dir    string
	MaxAge time.Duration
}

// DefaultCache returns a cache under the user cache directory, outside any
// repository so cached answers are never committed.
func DefaultCache() (*DirCache, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	return &DirCache{Dir: filepath.Join(base, "girl", "decisions")}, nil
}

func (c *DirCache) path(hash string) string { return filepath.Join(c.Dir, hash+".json") }

func (c *DirCache) Get(hash string) (*Result, bool) {
	maxAge := c.MaxAge
	if maxAge <= 0 {
		maxAge = 30 * 24 * time.Hour
	}
	if info, err := os.Stat(c.path(hash)); err != nil || time.Since(info.ModTime()) > maxAge {
		return nil, false
	}
	data, err := os.ReadFile(c.path(hash))
	if err != nil {
		return nil, false
	}
	var res Result
	if json.Unmarshal(data, &res) != nil || res.InputHash != hash {
		return nil, false
	}
	res.Cached = true
	return &res, true
}

func (c *DirCache) Put(hash string, res *Result) error {
	if err := os.MkdirAll(c.Dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(res)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(c.Dir, ".tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	tmp.Close()
	return os.Rename(tmp.Name(), c.path(hash))
}

// Cached wraps a provider with a cache. Only validated results are stored.
type Cached struct {
	Provider Provider
	Cache    Cache
	Model    string
}

func (c *Cached) Name() string { return c.Provider.Name() }

func (c *Cached) Decide(ctx context.Context, req Request) (*Result, error) {
	hash := HashRequest(c.Model, req)
	if c.Cache != nil {
		if res, ok := c.Cache.Get(hash); ok && Validate(req, res) == nil {
			return res, nil
		}
	}
	res, err := c.Provider.Decide(ctx, req)
	if err != nil {
		return nil, err
	}
	res.InputHash = hash
	if c.Cache != nil {
		_ = c.Cache.Put(hash, res)
	}
	return res, nil
}
