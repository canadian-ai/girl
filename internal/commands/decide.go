package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/canadian-ai/girl/internal/extensions"
	"github.com/canadian-ai/girl/internal/extensions/decisions"
	"github.com/urfave/cli/v2"
)

// DecideCommand groups the decision-model commands. They are opt-in and
// advisory: decisions can only make GIRL stricter, and run in shadow mode
// (recorded, no effect) unless enforce mode is chosen.
func DecideCommand() *cli.Command {
	modelFlags := []cli.Flag{
		&cli.StringFlag{Name: "mode", Usage: "shadow (record only) or enforce; default from .girl/config.yaml, else shadow"},
		&cli.StringFlag{Name: "provider", Usage: "Endpoint preset: " + strings.Join(decisions.Presets(), ", ") + " (default typesafe)"},
		&cli.StringFlag{Name: "model", Usage: "Model name or alias; pin a version for reproducibility"},
		&cli.BoolFlag{Name: "no-cache", Usage: "Do not read or write the local decision cache"},
		&cli.StringFlag{Name: "ledger", Usage: "Decision ledger path (default: per-repository file in the user config directory)"},
		&cli.BoolFlag{Name: "no-ledger", Usage: "Do not record this run in the decision ledger"},
		&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "text", Usage: "Output format: text or json"},
	}
	return &cli.Command{
		Name:  "decide",
		Usage: "Triage findings and guard agent changes with a decision model (experimental, opt-in)",
		Subcommands: []*cli.Command{
			{
				Name:      "triage",
				Usage:     "Triage findings from a GRP plan or `girl sonar issues` output (sends metadata, never source)",
				ArgsUsage: "<file|->",
				Flags: append([]cli.Flag{
					&cli.StringFlag{Name: "scope", Value: "unplanned", Usage: "unplanned (findings GIRL could not map) or all (every finding, ranked by priority)"},
					&cli.StringFlag{Name: "policy", Usage: "Governed extension policy JSON; when set, actions are evaluated and combined"},
					&cli.IntFlag{Name: "max-findings", Usage: "Maximum findings to send (cost guard; default 25)"},
					&cli.StringFlag{Name: "min-confidence", Usage: "Confidence needed to suggest a recipe"},
					&cli.BoolFlag{Name: "include-actions", Usage: "Also triage already-mapped SonarQube actions (can only escalate them)"},
					&cli.BoolFlag{Name: "no-consistency", Usage: "Skip the negated-question consistency check"},
					&cli.BoolFlag{Name: "no-signals", Usage: "Send only the finding itself, without repository signals"},
				}, modelFlags...),
				Action: runDecideTriage,
			},
			{
				Name:  "guard",
				Usage: "Check an agent's diff against the plan step it was asked to perform",
				Flags: append([]cli.Flag{
					&cli.StringFlag{Name: "plan", Required: true, Usage: "GRP plan JSON"},
					&cli.StringFlag{Name: "step", Required: true, Usage: "Step ID the change implements"},
					&cli.StringFlag{Name: "diff-file", Usage: "Path to unified diff file"},
					&cli.BoolFlag{Name: "stdin", Usage: "Read diff from stdin"},
					&cli.IntFlag{Name: "max-files", Value: 3, Usage: "File budget for one step"},
					&cli.IntFlag{Name: "max-changed-lines", Value: 200, Usage: "Changed-line budget for one step"},
					&cli.BoolFlag{Name: "send-diff", Usage: "Consult the decision model, sending sanitized diff text (off by default: rule checks only, nothing is sent)"},
					&cli.BoolFlag{Name: "no-model", Usage: "Run only GIRL's deterministic checks; nothing is sent"},
					&cli.BoolFlag{Name: "fail-on-review", Usage: "Exit 2 when the verdict is review"},
				}, modelFlags...),
				Action: runDecideGuard,
			},
			{
				Name:      "outcome",
				Usage:     "Record what actually happened to a finding, so decisions can be measured",
				ArgsUsage: "<fingerprint>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "false-positive", Usage: "yes or no: the finding was a false positive"},
					&cli.StringFlag{Name: "recipe-correct", Usage: "yes or no: the suggested recipe was right"},
					&cli.StringFlag{Name: "needed-review", Usage: "yes or no: the change needed human review"},
					&cli.StringFlag{Name: "verification", Usage: "pass or fail: result of verifying the applied change"},
					&cli.StringFlag{Name: "reverted", Usage: "yes or no: the change was later reverted"},
					&cli.StringFlag{Name: "note", Usage: "Free-text note"},
					&cli.StringFlag{Name: "source", Value: "human", Usage: "Who recorded this (human, ci, agent)"},
					&cli.StringFlag{Name: "ledger", Usage: "Decision ledger path"},
				},
				Action: runDecideOutcome,
			},
			{
				Name:  "eval",
				Usage: "Replay the ledger: how often was the model right at each confidence level?",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "ledger", Usage: "Decision ledger path"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "text", Usage: "Output format: text or json"},
				},
				Action: runDecideEval,
			},
			{
				Name:  "export",
				Usage: "Export the ledger as a table (decision features plus outcome labels) for tabular models",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "ledger", Usage: "Decision ledger path"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Value: "csv", Usage: "Output format: csv or json"},
				},
				Action: runDecideExport,
			},
		},
	}
}

// decisionSettings is flags over config over defaults.
type decisionSettings struct {
	Mode        string
	Provider    string
	Model       string
	MaxFindings int
	Consistency bool
	Thresholds  decisions.Thresholds
	Problems    []string
}

// flagGiven reports whether a flag was passed, including after positional
// arguments (urfave/cli stops parsing flags at the first positional).
func flagGiven(c *cli.Context, name string) bool {
	if c.IsSet(name) {
		return true
	}
	for _, arg := range os.Args {
		if arg == "--"+name || strings.HasPrefix(arg, "--"+name+"=") {
			return true
		}
	}
	return false
}

func resolveDecisionSettings(c *cli.Context, root string) (*decisionSettings, error) {
	s := &decisionSettings{Mode: decisions.ModeShadow, Provider: "typesafe", MaxFindings: 25, Consistency: true, Thresholds: decisions.DefaultThresholds()}
	dir := findProjectConfigDir(root)
	if dir != "" {
		cfg, err := loadGirlProjectConfig(dir)
		if err != nil {
			return nil, fmt.Errorf("load GIRL config: %w", err)
		}
		d := cfg.Decisions
		s.Problems = cfg.Problems
		if d.Mode != "" {
			s.Mode = d.Mode
		}
		if d.Provider != "" {
			s.Provider = d.Provider
		}
		s.Model = d.Model
		if d.MaxFindings > 0 {
			s.MaxFindings = d.MaxFindings
		}
		if d.Consistency != nil {
			s.Consistency = *d.Consistency
		}
		if d.MinRecipeConfidence > 0 {
			s.Thresholds.MinRecipeConfidence = d.MinRecipeConfidence
		}
		if d.ReviewProbability > 0 {
			s.Thresholds.ReviewProbability = d.ReviewProbability
		}
		if d.HighRiskProbability > 0 {
			s.Thresholds.HighRiskProbability = d.HighRiskProbability
		}
		if d.FalsePositiveProbability > 0 {
			s.Thresholds.FalsePositiveProbability = d.FalsePositiveProbability
		}
	}
	if c == nil {
		return s, nil
	}
	if m := stringFlag(c, "mode"); m != "" {
		if m != decisions.ModeShadow && m != decisions.ModeEnforce {
			return nil, fmt.Errorf("--mode must be shadow or enforce")
		}
		s.Mode = m
	} else if s.Mode == decisions.ModeOff {
		return nil, fmt.Errorf("decisions are disabled (decisions.mode: off in .girl/config.yaml)")
	}
	if v := stringFlag(c, "provider"); v != "" {
		s.Provider = v
	}
	if v := stringFlag(c, "model"); v != "" {
		s.Model = v
	}
	if flagGiven(c, "max-findings") {
		s.MaxFindings = intFlag(c, "max-findings")
	}
	if flagGiven(c, "min-confidence") {
		v, err := strconv.ParseFloat(stringFlag(c, "min-confidence"), 64)
		if err != nil || v < 0 || v > 1 {
			return nil, fmt.Errorf("--min-confidence must be between 0 and 1")
		}
		s.Thresholds.MinRecipeConfidence = v
	}
	if boolFlag(c, "no-consistency") {
		s.Consistency = false
	}
	return s, nil
}

func openLedger(c *cli.Context) (*decisions.Ledger, error) {
	if boolFlag(c, "no-ledger") {
		return nil, nil
	}
	path := stringFlag(c, "ledger")
	if path == "" {
		root := "."
		if r, err := gitRepoRoot("."); err == nil {
			root = r
		}
		p, err := decisions.DefaultLedgerPath(root)
		if err != nil {
			return nil, err
		}
		path = p
	}
	return &decisions.Ledger{Path: path}, nil
}

func newDecisionProvider(s *decisionSettings, noCache bool) (decisions.Provider, *decisions.Jev, error) {
	jev, err := decisions.NewJev(decisions.JevConfig{Preset: s.Provider, Model: s.Model})
	if err != nil {
		return nil, nil, err
	}
	var provider decisions.Provider = jev
	if !noCache {
		if cache, err := decisions.DefaultCache(); err == nil {
			provider = &decisions.Cached{Provider: jev, Cache: cache, Model: jev.Model()}
		}
	}
	return provider, jev, nil
}

type triageTarget struct {
	Kind    string // "unplanned", "planned", "unmapped" or "action"
	Finding extensions.Finding
	Action  *extensions.Action
	Signals map[string]any
	// PlannedRecipe is the recipe GIRL's own registry chose for this finding
	// ("" when it has none). It is free ground truth for the model's choice.
	PlannedRecipe string
}

type triageEntry struct {
	Kind   string            `json:"kind"`
	Triage *decisions.Triage `json:"triage"`
	// AgreesWithRegistry compares the model's recipe with the one GIRL's
	// registry already assigned; nil when GIRL had no recipe.
	AgreesWithRegistry *bool                `json:"agreesWithRegistry,omitempty"`
	Action             *extensions.Action   `json:"action,omitempty"`
	PolicyDecision     *extensions.Decision `json:"policyDecision,omitempty"`
	CombinedOutcome    *extensions.Decision `json:"combinedDecision,omitempty"`
}

type triageReport struct {
	SpecVersion string        `json:"specversion"`
	Provider    string        `json:"provider"`
	Host        string        `json:"host"`
	Model       string        `json:"model"`
	Mode        string        `json:"mode"`
	Scope       string        `json:"scope"`
	Entries     []triageEntry `json:"entries"`
	Skipped     int           `json:"skipped"`
	Abstained   int           `json:"abstained"`
	Ledger      string        `json:"ledger,omitempty"`
	Note        string        `json:"note"`
}

type triageOptions struct {
	Scope          string
	IncludeActions bool
	Signals        bool
	RepoRoot       string
	Policy         *extensions.Policy
	Ledger         *decisions.Ledger
}

const triageNote = "Advisory. Decisions can only escalate to review, deprioritize, or suggest a registered recipe; every suggested action must still pass policy evaluation. In shadow mode nothing takes effect."

// triageDocument runs triage over a plan or sonar document. It is shared by
// the CLI and the serve endpoint.
func triageDocument(ctx context.Context, provider decisions.Provider, host, model string, data []byte, s *decisionSettings, o triageOptions) (*triageReport, error) {
	targets, err := parseTriageTargets(data, o.Scope == "all", o.IncludeActions)
	if err != nil {
		return nil, err
	}
	report := &triageReport{SpecVersion: "girl.io/decisions/v0", Provider: provider.Name(), Host: host, Model: model, Mode: s.Mode, Scope: o.Scope,
		Entries: []triageEntry{}, Note: triageNote}
	if o.Ledger != nil {
		report.Ledger = o.Ledger.Path
	}
	if s.MaxFindings >= 0 && len(targets) > s.MaxFindings {
		report.Skipped = len(targets) - s.MaxFindings
		targets = targets[:s.MaxFindings]
	}
	for _, tg := range targets {
		signals := tg.Signals
		if !o.Signals {
			signals = nil
		} else if o.RepoRoot != "" {
			addRepoSignals(signals, o.RepoRoot, tg.Finding.File)
		}
		cfg := decisions.TriageConfig{Thresholds: s.Thresholds, Mode: s.Mode, Signals: signals, Consistency: s.Consistency}
		entry := triageEntry{Kind: tg.Kind}
		t, err := decisions.TriageFindingWith(ctx, provider, tg.Finding, decisions.CandidateRecipes(tg.Finding.File), cfg)
		if err != nil {
			if errors.Is(err, decisions.ErrUnauthorized) || errors.Is(err, decisions.ErrForbidden) || errors.Is(err, decisions.ErrConfig) {
				return nil, err
			}
			// Any other failure is an abstention: no effect, never a pass.
			t = &decisions.Triage{Fingerprint: tg.Finding.Fingerprint, Code: tg.Finding.RuleID, File: tg.Finding.File, Line: tg.Finding.Line,
				Mode: s.Mode, Effects: []string{}, Abstained: true, Reason: err.Error()}
		}
		if t.Abstained {
			report.Abstained++
		} else if o.Ledger != nil {
			if err := o.Ledger.Append(decisions.DecisionEntry(t, tg.Finding.Severity)); err != nil {
				return nil, fmt.Errorf("write decision ledger: %w", err)
			}
		}
		entry.Triage = t
		if !t.Abstained && tg.PlannedRecipe != "" && t.Recipe != "" {
			agrees := t.Recipe == tg.PlannedRecipe
			entry.AgreesWithRegistry = &agrees
			if o.Ledger != nil {
				// Recorded as an outcome so `girl decide eval` can measure
				// recipe confidence without waiting for human labels.
				if err := o.Ledger.Append(decisions.LedgerEntry{Kind: decisions.KindOutcome, Fingerprint: t.Fingerprint,
					Outcome: &decisions.Outcome{RecipeCorrect: &agrees, Source: "girl-registry"}}); err != nil {
					return nil, fmt.Errorf("write decision ledger: %w", err)
				}
			}
		}
		action := tg.Action
		if action == nil {
			if a, ok := decisions.SuggestedAction(tg.Finding, t); ok {
				action = a
			}
		}
		entry.Action = action
		if action != nil && o.Policy != nil {
			base := extensions.Evaluate(*action, *o.Policy)
			combined := decisions.Combine(base, t)
			entry.PolicyDecision, entry.CombinedOutcome = &base, &combined
		}
		report.Entries = append(report.Entries, entry)
	}
	// Rank: most worth doing first. Abstentions keep their place at the end.
	sort.SliceStable(report.Entries, func(i, j int) bool {
		a, b := report.Entries[i].Triage, report.Entries[j].Triage
		if a.Abstained != b.Abstained {
			return !a.Abstained
		}
		return a.Priority > b.Priority
	})
	return report, nil
}

func runDecideTriage(c *cli.Context) error {
	src := c.Args().First()
	if src == "" {
		return fmt.Errorf("usage: girl decide triage <file|->")
	}
	scope := stringFlag(c, "scope")
	if scope != "unplanned" && scope != "all" {
		return fmt.Errorf("--scope must be unplanned or all")
	}
	data, err := readInput(src)
	if err != nil {
		return err
	}
	s, err := resolveDecisionSettings(c, ".")
	if err != nil {
		return err
	}
	var policy *extensions.Policy
	if p := stringFlag(c, "policy"); p != "" {
		raw, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read policy: %w", err)
		}
		policy = &extensions.Policy{}
		if err := json.Unmarshal(raw, policy); err != nil {
			return fmt.Errorf("parse policy: %w", err)
		}
	}
	provider, jev, err := newDecisionProvider(s, boolFlag(c, "no-cache"))
	if err != nil {
		return err
	}
	ledger, err := openLedger(c)
	if err != nil {
		return err
	}
	repoRoot, _ := gitRepoRoot(".")
	fmt.Fprintf(os.Stderr, "Triage with %s via %s (%s mode): finding metadata only, never source code; cached answers are reused.\n", jev.Model(), jev.Host(), s.Mode)
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	report, err := triageDocument(ctx, provider, jev.Host(), jev.Model(), data, s,
		triageOptions{Scope: scope, IncludeActions: boolFlag(c, "include-actions"), Signals: !boolFlag(c, "no-signals"), RepoRoot: repoRoot, Policy: policy, Ledger: ledger})
	if err != nil {
		return err
	}
	if stringFlag(c, "output", "o") == "json" {
		printJSON(report)
		return nil
	}
	printTriageText(report)
	return nil
}

func readInput(src string) ([]byte, error) {
	if src == "-" {
		return io.ReadAll(io.LimitReader(os.Stdin, 32<<20))
	}
	return os.ReadFile(src)
}

type planDiagnostic struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
	Span     *struct {
		StartLine int `json:"startLine"`
		EndLine   int `json:"endLine"`
	} `json:"span"`
	Symbol *struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	} `json:"symbol"`
}

// diagnosticSignals are facts GIRL already computed about a finding. No
// source text is included.
func diagnosticSignals(d planDiagnostic, all []planDiagnostic) map[string]any {
	sig := map[string]any{"is_test_file": decisions.IsTestPath(d.File)}
	if d.Span != nil && d.Span.EndLine >= d.Span.StartLine && d.Span.StartLine > 0 {
		sig["span_lines"] = d.Span.EndLine - d.Span.StartLine + 1
	}
	if d.Symbol != nil && d.Symbol.Kind != "" {
		sig["symbol_kind"] = d.Symbol.Kind
	}
	inFile := 0
	seen := map[string]bool{}
	var others []string
	for _, o := range all {
		if o.File != d.File {
			continue
		}
		inFile++
		if o.Code != d.Code && !seen[o.Code] {
			seen[o.Code] = true
			others = append(others, o.Code)
		}
	}
	sig["findings_in_file"] = inFile
	if len(others) > 0 {
		sort.Strings(others)
		if len(others) > 8 {
			others = others[:8]
		}
		sig["other_findings_in_file"] = strings.Join(others, ", ")
	}
	return sig
}

// addRepoSignals adds cheap, local repository facts: recent churn and
// whether a test file sits beside the source file.
func addRepoSignals(sig map[string]any, repoRoot, file string) {
	if sig == nil {
		return
	}
	if _, reason := safeScopedFile(repoRoot, file); reason != "" {
		return
	}
	if out, err := gitOutput(repoRoot, "log", "--since=90.days", "--format=%h", "--", file); err == nil {
		sig["commits_last_90_days"] = len(strings.Fields(string(out)))
	}
	sig["has_test_file"] = hasSiblingTest(repoRoot, file)
}

func hasSiblingTest(repoRoot, file string) bool {
	ext := ""
	if i := strings.LastIndex(file, "."); i >= 0 {
		ext = file[i:]
	}
	stem := strings.TrimSuffix(file, ext)
	dir, base := "", file
	if i := strings.LastIndex(file, "/"); i >= 0 {
		dir, base = file[:i+1], file[i+1:]
	}
	baseStem := strings.TrimSuffix(base, ext)
	candidates := []string{stem + "_test" + ext, stem + ".test" + ext, stem + ".spec" + ext,
		dir + "test_" + base, dir + "tests/test_" + base, dir + "__tests__/" + baseStem + ".test" + ext, "tests/test_" + base}
	for _, cand := range candidates {
		if _, reason := safeScopedFile(repoRoot, cand); reason == "" {
			return true
		}
	}
	return false
}

// parseTriageTargets accepts a GRP plan (diagnostics + unplanned) or the
// output of `girl sonar issues` (findings + actions + unmapped).
func parseTriageTargets(data []byte, all, includeActions bool) ([]triageTarget, error) {
	var doc struct {
		Diagnostics []planDiagnostic `json:"diagnostics"`
		Unplanned   []struct {
			DiagnosticID string `json:"diagnosticId"`
		} `json:"unplanned"`
		Steps []struct {
			Recipe   string   `json:"recipe"`
			Requires []string `json:"requires"`
		} `json:"steps"`
		Actions  []extensions.Action `json:"actions"`
		Unmapped []struct {
			Finding extensions.Finding `json:"finding"`
		} `json:"unmapped"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}
	if len(doc.Diagnostics) == 0 && len(doc.Unmapped) == 0 && len(doc.Actions) == 0 {
		return nil, fmt.Errorf("input has no diagnostics, unmapped findings or actions; pass a GRP plan (girl plan -o grp-json) or girl sonar issues output")
	}
	var out []triageTarget
	unplanned := map[string]bool{}
	for _, u := range doc.Unplanned {
		unplanned[u.DiagnosticID] = true
	}
	plannedRecipe := map[string]string{}
	for _, st := range doc.Steps {
		for _, id := range st.Requires {
			if _, dup := plannedRecipe[id]; !dup {
				plannedRecipe[id] = st.Recipe
			}
		}
	}
	for _, d := range doc.Diagnostics {
		kind := "planned"
		if unplanned[d.ID] {
			kind = "unplanned"
		} else if !all {
			continue
		}
		// Parser failures are repaired, never triaged.
		if strings.HasSuffix(d.Code, ".syntax-error") || strings.HasSuffix(d.Code, ".analysis-error") {
			continue
		}
		sev := d.Severity
		if sev != "low" && sev != "medium" && sev != "high" {
			sev = "medium"
		}
		f, err := extensions.Normalize(extensions.Finding{Provider: "girl", ExternalID: d.ID, RuleID: d.Code, File: d.File, Line: max(d.Line, 1), Severity: sev, Message: d.Message})
		if err != nil {
			continue // unsafe or non-canonical path: never sent
		}
		out = append(out, triageTarget{Kind: kind, Finding: f, Signals: diagnosticSignals(d, doc.Diagnostics), PlannedRecipe: plannedRecipe[d.ID]})
	}
	for _, u := range doc.Unmapped {
		if f, err := extensions.Normalize(u.Finding); err == nil {
			f.Provenance = u.Finding.Provenance
			out = append(out, triageTarget{Kind: "unmapped", Finding: f, Signals: map[string]any{"is_test_file": decisions.IsTestPath(f.File)}})
		}
	}
	if includeActions {
		for i := range doc.Actions {
			a := doc.Actions[i]
			if f, err := extensions.Normalize(a.Finding); err == nil {
				f.Provenance = a.Finding.Provenance
				a.Finding = f
				out = append(out, triageTarget{Kind: "action", Finding: f, Action: &a, Signals: map[string]any{"is_test_file": decisions.IsTestPath(f.File)}})
			}
		}
	}
	return out, nil
}

func printTriageText(r *triageReport) {
	fmt.Printf("Decision triage — %s via %s — %s mode\n\n", r.Model, r.Host, r.Mode)
	if len(r.Entries) == 0 {
		fmt.Println("Nothing to triage: every finding already has a recipe (use --scope all to rank them).")
	}
	for _, e := range r.Entries {
		t := e.Triage
		fmt.Printf("%s:%d %s [%s]\n", t.File, t.Line, t.Code, e.Kind)
		if t.Abstained {
			fmt.Printf("  abstained: %s\n", t.Reason)
			continue
		}
		fmt.Printf("  priority %.2f  recipe %s (%.2f)  false positive %.2f  needs review %.2f  risk %.2f/2\n",
			t.Priority, t.Recipe, t.RecipeConfidence, t.FalsePositive, t.NeedsReview, t.RiskScore)
		if len(t.Inconsistent) > 0 {
			fmt.Printf("  inconsistent answers discarded: %s\n", strings.Join(t.Inconsistent, ", "))
		}
		switch {
		case t.Mode == decisions.ModeShadow && len(t.WouldEffects) > 0:
			fmt.Printf("  would: %s (shadow mode: no effect)\n", strings.Join(t.WouldEffects, ", "))
		case len(t.Effects) > 0:
			fmt.Printf("  effects: %s\n", strings.Join(t.Effects, ", "))
		default:
			fmt.Println("  effects: none")
		}
		if e.AgreesWithRegistry != nil {
			if *e.AgreesWithRegistry {
				fmt.Println("  agrees with GIRL's own recipe for this finding")
			} else {
				fmt.Println("  DISAGREES with GIRL's own recipe for this finding (GIRL's recipe stands)")
			}
		}
		if e.CombinedOutcome != nil {
			fmt.Printf("  policy: %s -> %s\n", e.PolicyDecision.Status, e.CombinedOutcome.Status)
		}
		fmt.Printf("  id: %s\n", t.Fingerprint)
	}
	if r.Skipped > 0 {
		fmt.Printf("\n%d finding(s) not sent (raise --max-findings).\n", r.Skipped)
	}
	fmt.Printf("\n%s\n", r.Note)
	if r.Ledger != "" {
		fmt.Println("Record what happens next with `girl decide outcome <id> ...`, then measure with `girl decide eval`.")
	}
}
