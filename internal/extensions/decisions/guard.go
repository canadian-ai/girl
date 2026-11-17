package decisions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/canadian-ai/girl/internal/redact"
)

// Guard verdicts. There is deliberately no "approved": the guard can object,
// it cannot bless. A change with no objection still needs verification.
const (
	GuardNoObjection = "no-objection"
	GuardReview      = "review"
)

// GuardStep is the planned step an agent was asked to carry out.
type GuardStep struct {
	ID     string   `json:"id"`
	Recipe string   `json:"recipe"`
	Action string   `json:"action"`
	Files  []string `json:"files"`
}

// GuardFile is one file in the change, as counted from the diff.
type GuardFile struct {
	Path      string `json:"path"`
	Added     int    `json:"added"`
	Deleted   int    `json:"deleted"`
	Binary    bool   `json:"binary,omitempty"`
	Generated bool   `json:"generated,omitempty"`
	Lockfile  bool   `json:"lockfile,omitempty"`
	Rename    bool   `json:"rename,omitempty"`
}

// GuardBudget bounds the change. Zero values use the defaults.
type GuardBudget struct {
	MaxFiles        int `json:"maxFiles"`
	MaxChangedLines int `json:"maxChangedLines"`
}

// GuardInput is everything the guard looks at.
type GuardInput struct {
	Step   GuardStep
	Files  []GuardFile
	Budget GuardBudget
	// Diff is the diff text. It is set only when the caller explicitly opted
	// in to sending it; it is sanitized and capped before leaving the
	// machine. Without it the model is not consulted at all.
	Diff string
}

// GuardThresholds turn model probabilities into objections.
type GuardThresholds struct {
	MinInScope        float64 // below this, the change looks out of scope
	MinMatchesRecipe  float64 // below this, the change does not look like the recipe
	MaxBehaviorChange float64 // at or above this, behaviour probably changed
}

func DefaultGuardThresholds() GuardThresholds {
	return GuardThresholds{MinInScope: 0.5, MinMatchesRecipe: 0.5, MaxBehaviorChange: 0.5}
}

// GuardResult is the guard's record for one change.
type GuardResult struct {
	Fingerprint string `json:"fingerprint"`
	StepID      string `json:"stepId"`
	Verdict     string `json:"verdict"`
	Mode        string `json:"mode"`
	// RuleReasons come from GIRL's own deterministic checks and always apply.
	RuleReasons []string `json:"ruleReasons"`
	// ModelReasons come from the decision model; they apply in enforce mode
	// and are recorded as WouldReasons in shadow mode.
	ModelReasons []string `json:"modelReasons,omitempty"`
	WouldReasons []string `json:"wouldReasons,omitempty"`

	Model          string  `json:"model,omitempty"`
	InputHash      string  `json:"inputHash,omitempty"`
	InScope        float64 `json:"inScope"`
	MatchesRecipe  float64 `json:"matchesRecipe"`
	BehaviorChange float64 `json:"behaviorChange"`
	DiffSent       bool    `json:"diffSent"`
	Abstained      bool    `json:"abstained,omitempty"`
	Reason         string  `json:"reason,omitempty"`
}

// MaxGuardDiffBytes caps opt-in diff text sent to a model.
const MaxGuardDiffBytes = 16 << 10

// IsTestPath reports whether a path looks like a test file.
func IsTestPath(p string) bool {
	base := strings.ToLower(path.Base(p))
	dir := "/" + strings.ToLower(path.Dir(p)) + "/"
	return strings.HasSuffix(base, "_test.go") || strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") ||
		strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") ||
		strings.Contains(dir, "/tests/") || strings.Contains(dir, "/test/") || strings.Contains(dir, "/__tests__/")
}

func isManifest(p string) bool {
	switch strings.ToLower(path.Base(p)) {
	case "package.json", "go.mod", "cargo.toml", "pyproject.toml", "requirements.txt", "setup.py", "setup.cfg",
		"dockerfile", "makefile", ".gitignore":
		return true
	}
	return strings.HasPrefix(p, ".github/") || strings.HasPrefix(p, ".girl/")
}

// ruleChecks are GIRL's own objections. They need no model and no network,
// and they hold in every mode.
func ruleChecks(in GuardInput) []string {
	budget := in.Budget
	if budget.MaxFiles <= 0 {
		budget.MaxFiles = 3
	}
	if budget.MaxChangedLines <= 0 {
		budget.MaxChangedLines = 200
	}
	planned := map[string]bool{}
	for _, f := range in.Step.Files {
		planned[f] = true
	}
	var reasons, outside []string
	total, testAdded, testDeleted := 0, 0, 0
	for _, f := range in.Files {
		total += f.Added + f.Deleted
		switch {
		case f.Lockfile:
			reasons = append(reasons, fmt.Sprintf("changes lockfile %s", f.Path))
		case f.Binary:
			reasons = append(reasons, fmt.Sprintf("changes binary file %s", f.Path))
		case f.Generated:
			reasons = append(reasons, fmt.Sprintf("changes generated file %s", f.Path))
		case isManifest(f.Path):
			reasons = append(reasons, fmt.Sprintf("changes build, dependency or policy file %s", f.Path))
		}
		if IsTestPath(f.Path) {
			testAdded += f.Added
			testDeleted += f.Deleted
		} else if len(planned) > 0 && !planned[f.Path] {
			outside = append(outside, f.Path)
		}
	}
	if len(outside) > 0 {
		sort.Strings(outside)
		shown := outside
		if len(shown) > 5 {
			shown = append(append([]string{}, shown[:5]...), fmt.Sprintf("and %d more", len(outside)-5))
		}
		reasons = append(reasons, fmt.Sprintf("touches files outside the planned step: %s", strings.Join(shown, ", ")))
	}
	if testDeleted > testAdded && testDeleted-testAdded >= 5 {
		reasons = append(reasons, fmt.Sprintf("removes more test code than it adds (-%d / +%d lines)", testDeleted, testAdded))
	}
	if len(in.Files) > budget.MaxFiles {
		reasons = append(reasons, fmt.Sprintf("touches %d files (budget %d)", len(in.Files), budget.MaxFiles))
	}
	if total > budget.MaxChangedLines {
		reasons = append(reasons, fmt.Sprintf("changes %d lines (budget %d)", total, budget.MaxChangedLines))
	}
	if len(in.Files) == 0 {
		reasons = append(reasons, "the diff is empty: nothing was changed")
	}
	return reasons
}

// GuardFingerprint identifies a change for the ledger: the step plus the
// shape of the diff.
func GuardFingerprint(in GuardInput) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\x00%s\x00", in.Step.ID, in.Step.Recipe)
	files := append([]GuardFile{}, in.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	for _, f := range files {
		fmt.Fprintf(h, "%s\x00%d\x00%d\x00", f.Path, f.Added, f.Deleted)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Guard checks an agent's change against the step it was asked to perform.
// p may be nil: the deterministic checks still run. Provider failures are an
// abstention; they never clear an objection and never raise one.
func Guard(ctx context.Context, p Provider, in GuardInput, th GuardThresholds, mode string) *GuardResult {
	if mode != ModeEnforce {
		mode = ModeShadow
	}
	r := &GuardResult{Fingerprint: GuardFingerprint(in), StepID: in.Step.ID, Mode: mode, RuleReasons: ruleChecks(in)}
	if r.RuleReasons == nil {
		r.RuleReasons = []string{}
	}
	finish := func() *GuardResult {
		r.Verdict = GuardNoObjection
		if len(r.RuleReasons) > 0 || len(r.ModelReasons) > 0 {
			r.Verdict = GuardReview
		}
		return r
	}
	if p == nil || len(in.Files) == 0 {
		r.Abstained, r.Reason = true, "decision model not consulted"
		return finish()
	}
	// File names and line counts alone do not let a model judge a change:
	// measured against the live model, its answers without the diff were
	// near chance. Rather than act on noise, the model is consulted only
	// when the caller opted in to sending the sanitized diff.
	if in.Diff == "" {
		r.Abstained, r.Reason = true, "decision model not consulted: it needs the diff text to judge a change (opt in with --send-diff); rule checks ran"
		return finish()
	}

	files := make([]map[string]any, 0, len(in.Files))
	for _, f := range in.Files {
		files = append(files, map[string]any{"path": f.Path, "added": f.Added, "deleted": f.Deleted, "test": IsTestPath(f.Path), "rename": f.Rename})
	}
	state := map[string]any{
		"planned_step": map[string]any{"recipe": in.Step.Recipe, "action": redact.String(in.Step.Action), "files": in.Step.Files},
		"change":       map[string]any{"files": files},
		"context":      "An automated agent was asked to perform the planned refactoring step. Judge whether the change it produced is that step and nothing more.",
	}
	diff := redact.String(in.Diff)
	if len(diff) > MaxGuardDiffBytes {
		diff = diff[:MaxGuardDiffBytes] + "\n... (truncated)"
	}
	state["change"].(map[string]any)["diff"] = diff
	r.DiffSent = true
	if findings := redact.ScanValue(state); len(findings) > 0 {
		r.Abstained, r.Reason, r.DiffSent = true, fmt.Sprintf("sanitization could not be established (%s); nothing was sent", findings[0].Detector), false
		return finish()
	}
	req := Request{State: state, Questions: map[string]Question{
		"in_scope":        {Type: TypeNoul, Instructions: "The change is limited to what the planned step describes and touches nothing unrelated."},
		"matches_recipe":  {Type: TypeNoul, Instructions: "The change is an instance of the planned refactoring recipe."},
		"behavior_change": {Type: TypeNoul, Instructions: "The change alters observable behaviour, beyond a behaviour-preserving refactor."},
	}}
	res, err := p.Decide(ctx, req)
	if err == nil {
		err = Validate(req, res)
	}
	if err != nil {
		r.Abstained, r.Reason = true, err.Error()
		return finish()
	}
	r.Model, r.InputHash = res.Model, res.InputHash
	r.InScope = *res.Answers["in_scope"].Noul
	r.MatchesRecipe = *res.Answers["matches_recipe"].Noul
	r.BehaviorChange = *res.Answers["behavior_change"].Noul
	var reasons []string
	if r.InScope < th.MinInScope {
		reasons = append(reasons, fmt.Sprintf("model judges the change out of scope for the step (in scope %.2f)", r.InScope))
	}
	if r.MatchesRecipe < th.MinMatchesRecipe {
		reasons = append(reasons, fmt.Sprintf("model judges the change is not the planned recipe (match %.2f)", r.MatchesRecipe))
	}
	if r.BehaviorChange >= th.MaxBehaviorChange {
		reasons = append(reasons, fmt.Sprintf("model judges behaviour may have changed (%.2f)", r.BehaviorChange))
	}
	if mode == ModeEnforce {
		r.ModelReasons = reasons
	} else {
		r.WouldReasons = reasons
	}
	return finish()
}
