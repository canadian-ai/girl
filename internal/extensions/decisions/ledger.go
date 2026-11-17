package decisions

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Ledger entry kinds.
const (
	KindDecision = "decision"
	KindOutcome  = "outcome"
)

// Outcome is what actually happened to a finding after a decision. Every
// field is optional; several outcome entries for one finding are merged, the
// latest value of each field winning.
type Outcome struct {
	// FalsePositive: a human confirmed the finding was (not) a false positive.
	FalsePositive *bool `json:"falsePositive,omitempty"`
	// RecipeCorrect: the suggested recipe was (not) the right one.
	RecipeCorrect *bool `json:"recipeCorrect,omitempty"`
	// NeededReview: in hindsight the change did (not) need human review.
	NeededReview *bool `json:"neededReview,omitempty"`
	// VerificationPassed: the applied change passed (failed) verification.
	VerificationPassed *bool `json:"verificationPassed,omitempty"`
	// Reverted: the applied change was later reverted.
	Reverted *bool  `json:"reverted,omitempty"`
	Source   string `json:"source,omitempty"`
	Note     string `json:"note,omitempty"`
}

// LedgerEntry is one line of the append-only decision ledger.
type LedgerEntry struct {
	Kind        string `json:"kind"`
	Time        string `json:"time"`
	Fingerprint string `json:"fingerprint"`

	// Decision fields.
	Subject          string             `json:"subject,omitempty"` // "triage" or "guard"
	Code             string             `json:"code,omitempty"`
	File             string             `json:"file,omitempty"`
	Line             int                `json:"line,omitempty"`
	Severity         string             `json:"severity,omitempty"`
	Model            string             `json:"model,omitempty"`
	InputHash        string             `json:"inputHash,omitempty"`
	Mode             string             `json:"mode,omitempty"`
	FalsePositive    float64            `json:"falsePositiveP,omitempty"`
	NeedsReview      float64            `json:"needsReviewP,omitempty"`
	Recipe           string             `json:"recipe,omitempty"`
	RecipeConfidence float64            `json:"recipeConfidence,omitempty"`
	RiskScore        float64            `json:"riskScore,omitempty"`
	HighRisk         float64            `json:"highRiskP,omitempty"`
	Effects          []string           `json:"effects,omitempty"`
	Signals          map[string]float64 `json:"signals,omitempty"`

	Outcome *Outcome `json:"outcome,omitempty"`
}

// Ledger is an append-only JSONL record of decisions and their outcomes. It
// is what lets thresholds be measured instead of guessed.
type Ledger struct{ Path string }

// DefaultLedgerPath returns a per-repository ledger under the user config
// directory, outside the repository so it is never committed.
func DefaultLedgerPath(repoRoot string) (string, error) {
	if p := os.Getenv("GIRL_DECISION_LEDGER"); p != "" {
		return p, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	sum := sha256.Sum256([]byte(abs))
	return filepath.Join(base, "girl", "ledger", hex.EncodeToString(sum[:8])+".jsonl"), nil
}

// Append writes one entry. The file is private (0600 in a 0700 directory).
func (l *Ledger) Append(e LedgerEntry) error {
	if l == nil || l.Path == "" {
		return nil
	}
	if e.Fingerprint == "" {
		return fmt.Errorf("ledger: entry has no fingerprint")
	}
	if e.Time == "" {
		e.Time = time.Now().UTC().Format(time.RFC3339)
	}
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	return err
}

// Read returns all entries in order. A missing ledger is empty, not an
// error; a corrupt line is an error (the ledger is evidence).
func (l *Ledger) Read() ([]LedgerEntry, error) {
	f, err := os.Open(l.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []LedgerEntry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	for n := 1; sc.Scan(); n++ {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var e LedgerEntry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("ledger %s line %d: %w", filepath.Base(l.Path), n, err)
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

// DecisionEntry converts a triage into a ledger entry.
func DecisionEntry(t *Triage, severity string) LedgerEntry {
	effects := t.Effects
	if t.Mode == ModeShadow {
		effects = t.WouldEffects
	}
	return LedgerEntry{
		Kind: KindDecision, Subject: "triage", Fingerprint: t.Fingerprint, Code: t.Code, File: t.File, Line: t.Line, Severity: severity,
		Model: t.Model, InputHash: t.InputHash, Mode: t.Mode,
		FalsePositive: t.FalsePositive, NeedsReview: t.NeedsReview, Recipe: t.Recipe, RecipeConfidence: t.RecipeConfidence,
		RiskScore: t.RiskScore, HighRisk: t.RiskLevels["2"], Effects: effects, Signals: t.NumericSignals,
	}
}

// Case is the latest decision for a finding joined with its merged outcomes.
type Case struct {
	Decision LedgerEntry
	Outcome  Outcome
	Labeled  bool
}

func mergeOutcome(dst *Outcome, src *Outcome) {
	if src.FalsePositive != nil {
		dst.FalsePositive = src.FalsePositive
	}
	if src.RecipeCorrect != nil {
		dst.RecipeCorrect = src.RecipeCorrect
	}
	if src.NeededReview != nil {
		dst.NeededReview = src.NeededReview
	}
	if src.VerificationPassed != nil {
		dst.VerificationPassed = src.VerificationPassed
	}
	if src.Reverted != nil {
		dst.Reverted = src.Reverted
	}
}

// Cases joins decisions with outcomes by fingerprint, in first-seen order.
// Outcomes recorded before any decision for that finding are ignored.
func Cases(entries []LedgerEntry) []Case {
	index := map[string]int{}
	var cases []Case
	for _, e := range entries {
		switch e.Kind {
		case KindDecision:
			if e.Subject != "" && e.Subject != "triage" {
				continue
			}
			if i, ok := index[e.Fingerprint]; ok {
				cases[i].Decision = e
			} else {
				index[e.Fingerprint] = len(cases)
				cases = append(cases, Case{Decision: e})
			}
		case KindOutcome:
			if i, ok := index[e.Fingerprint]; ok && e.Outcome != nil {
				mergeOutcome(&cases[i].Outcome, e.Outcome)
				cases[i].Labeled = true
			}
		}
	}
	return cases
}
