package commands

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/canadian-ai/girl/internal/extensions/decisions"
	"github.com/urfave/cli/v2"
)

func runDecideGuard(c *cli.Context) error {
	planData, err := os.ReadFile(stringFlag(c, "plan"))
	if err != nil {
		return fmt.Errorf("read plan: %w", err)
	}
	var plan struct {
		Steps []struct {
			ID     string `json:"id"`
			Recipe string `json:"recipe"`
			Action string `json:"action"`
			Target struct {
				File string `json:"file"`
			} `json:"target"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(planData, &plan); err != nil {
		return fmt.Errorf("parse plan: %w", err)
	}
	var step *decisions.GuardStep
	for _, s := range plan.Steps {
		if s.ID == stringFlag(c, "step") {
			step = &decisions.GuardStep{ID: s.ID, Recipe: s.Recipe, Action: s.Action}
			if s.Target.File != "" {
				step.Files = []string{s.Target.File}
			}
		}
	}
	if step == nil {
		return fmt.Errorf("step %q not found in plan", stringFlag(c, "step"))
	}
	in, err := readDiffFromFlags(c)
	if err != nil {
		return err
	}
	input := decisions.GuardInput{Step: *step, Budget: decisions.GuardBudget{MaxFiles: intFlag(c, "max-files"), MaxChangedLines: intFlag(c, "max-changed-lines")}}
	for _, f := range in.Stats.Files {
		input.Files = append(input.Files, decisions.GuardFile{Path: f.Path, Added: f.AddedLines, Deleted: f.DeletedLines,
			Binary: f.IsBinary, Generated: f.IsGenerated, Lockfile: f.IsLockfile, Rename: f.IsRename})
	}
	if boolFlag(c, "send-diff") {
		input.Diff = string(in.Raw)
	}

	mode := decisions.ModeShadow
	var provider decisions.Provider
	// The model is consulted only with --send-diff: without the diff text
	// its answers are not reliable enough to record, let alone act on.
	if !boolFlag(c, "no-model") && input.Diff != "" {
		s, err := resolveDecisionSettings(c, ".")
		if err != nil {
			return err
		}
		mode = s.Mode
		p, jev, err := newDecisionProvider(s, boolFlag(c, "no-cache"))
		if err != nil {
			return err
		}
		provider = p
		fmt.Fprintf(os.Stderr, "Guard with %s via %s (%s mode): sending file names, line counts and sanitized diff text.\n", jev.Model(), jev.Host(), mode)
	}
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	res := decisions.Guard(ctx, provider, input, decisions.DefaultGuardThresholds(), mode)
	if ledger, err := openLedger(c); err == nil && ledger != nil && !res.Abstained {
		reasons := res.ModelReasons
		if mode == decisions.ModeShadow {
			reasons = res.WouldReasons
		}
		_ = ledger.Append(decisions.LedgerEntry{Kind: decisions.KindDecision, Subject: "guard", Fingerprint: res.Fingerprint, Code: step.Recipe,
			File: strings.Join(step.Files, ","), Model: res.Model, InputHash: res.InputHash, Mode: mode, Effects: reasons,
			Signals: map[string]float64{"in_scope": res.InScope, "matches_recipe": res.MatchesRecipe, "behavior_change": res.BehaviorChange}})
	}
	if stringFlag(c, "output", "o") == "json" {
		printJSON(res)
	} else {
		fmt.Printf("Guard — step %s — %s (%s mode)\n", res.StepID, strings.ToUpper(res.Verdict), res.Mode)
		for _, r := range res.RuleReasons {
			fmt.Printf("  ✗ %s\n", r)
		}
		for _, r := range res.ModelReasons {
			fmt.Printf("  ✗ %s\n", r)
		}
		for _, r := range res.WouldReasons {
			fmt.Printf("  ~ %s (shadow mode: recorded, no effect)\n", r)
		}
		if res.Abstained {
			fmt.Printf("  model: %s\n", res.Reason)
		} else {
			fmt.Printf("  model: in scope %.2f, matches recipe %.2f, behaviour change %.2f\n", res.InScope, res.MatchesRecipe, res.BehaviorChange)
		}
		fmt.Println("No objection is not approval: the change must still pass verification.")
	}
	if boolFlag(c, "fail-on-review") && res.Verdict == decisions.GuardReview {
		return cli.Exit("guard: change needs review", 2)
	}
	return nil
}

func yesNo(name, v string, yes, no string) (*bool, error) {
	switch strings.ToLower(v) {
	case "":
		return nil, nil
	case yes, "true":
		t := true
		return &t, nil
	case no, "false":
		f := false
		return &f, nil
	}
	return nil, fmt.Errorf("--%s must be %s or %s", name, yes, no)
}

func runDecideOutcome(c *cli.Context) error {
	fp := c.Args().First()
	if fp == "" {
		return fmt.Errorf("usage: girl decide outcome <fingerprint> [--false-positive yes|no] [--verification pass|fail] ...")
	}
	ledger, err := openLedger(c)
	if err != nil {
		return err
	}
	entries, err := ledger.Read()
	if err != nil {
		return err
	}
	// Accept a unique prefix so IDs can be copied from text output.
	match := ""
	for _, e := range entries {
		if e.Kind == decisions.KindDecision && strings.HasPrefix(e.Fingerprint, fp) {
			if match != "" && match != e.Fingerprint {
				return fmt.Errorf("fingerprint prefix %q is ambiguous", fp)
			}
			match = e.Fingerprint
		}
	}
	if match == "" {
		return fmt.Errorf("no recorded decision matches %q; outcomes can only be recorded for findings that were triaged", fp)
	}
	o := &decisions.Outcome{Source: stringFlag(c, "source"), Note: stringFlag(c, "note")}
	if o.FalsePositive, err = yesNo("false-positive", stringFlag(c, "false-positive"), "yes", "no"); err != nil {
		return err
	}
	if o.RecipeCorrect, err = yesNo("recipe-correct", stringFlag(c, "recipe-correct"), "yes", "no"); err != nil {
		return err
	}
	if o.NeededReview, err = yesNo("needed-review", stringFlag(c, "needed-review"), "yes", "no"); err != nil {
		return err
	}
	if o.VerificationPassed, err = yesNo("verification", stringFlag(c, "verification"), "pass", "fail"); err != nil {
		return err
	}
	if o.Reverted, err = yesNo("reverted", stringFlag(c, "reverted"), "yes", "no"); err != nil {
		return err
	}
	if o.FalsePositive == nil && o.RecipeCorrect == nil && o.NeededReview == nil && o.VerificationPassed == nil && o.Reverted == nil {
		return fmt.Errorf("nothing to record: pass at least one of --false-positive, --recipe-correct, --needed-review, --verification, --reverted")
	}
	if err := ledger.Append(decisions.LedgerEntry{Kind: decisions.KindOutcome, Fingerprint: match, Outcome: o}); err != nil {
		return err
	}
	fmt.Printf("Recorded outcome for %s\n", match[:16])
	return nil
}

func loadCases(c *cli.Context) ([]decisions.Case, error) {
	ledger, err := openLedger(c)
	if err != nil {
		return nil, err
	}
	entries, err := ledger.Read()
	if err != nil {
		return nil, err
	}
	return decisions.Cases(entries), nil
}

func runDecideEval(c *cli.Context) error {
	cases, err := loadCases(c)
	if err != nil {
		return err
	}
	s, err := resolveDecisionSettings(nil, ".")
	if err != nil {
		return err
	}
	report := decisions.Evaluate(cases, s.Thresholds)
	if stringFlag(c, "output", "o") == "json" {
		printJSON(report)
		return nil
	}
	fmt.Printf("Decision evaluation — %d decisions, %d with recorded outcomes\n\n", report.Decisions, report.Labeled)
	for _, sig := range report.Signals {
		fmt.Printf("%s (threshold %.2f): %d labeled, %d positive\n", sig.Signal, sig.Threshold, sig.Samples, sig.Positives)
		if sig.Samples > 0 {
			fmt.Printf("  flagged %d, precision %.2f, recall %.2f, Brier %.3f\n", sig.Flagged, sig.Precision, sig.Recall, sig.Brier)
			for _, b := range sig.Buckets {
				fmt.Printf("    predicted %.1f-%.1f: n=%d, observed %.2f\n", b.Low, b.High, b.Count, b.Observed)
			}
		}
		fmt.Printf("  %s\n", sig.Verdict)
	}
	fmt.Printf("\n%s\n", report.Note)
	return nil
}

func labelCell(b *bool) string {
	if b == nil {
		return ""
	}
	if *b {
		return "1"
	}
	return "0"
}

func runDecideExport(c *cli.Context) error {
	cases, err := loadCases(c)
	if err != nil {
		return err
	}
	signalKeys := map[string]bool{}
	for _, cs := range cases {
		for k := range cs.Decision.Signals {
			signalKeys[k] = true
		}
	}
	var sigCols []string
	for k := range signalKeys {
		sigCols = append(sigCols, k)
	}
	sort.Strings(sigCols)
	header := []string{"fingerprint", "code", "language", "severity", "model", "false_positive_p", "needs_review_p", "recipe", "recipe_confidence", "risk_score", "high_risk_p"}
	for _, k := range sigCols {
		header = append(header, "signal_"+k)
	}
	header = append(header, "label_false_positive", "label_recipe_correct", "label_needed_review", "label_verification_passed", "label_reverted")
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	var rows [][]string
	for _, cs := range cases {
		d := cs.Decision
		row := []string{d.Fingerprint, d.Code, decisions.LanguageOf(d.File), d.Severity, d.Model, f(d.FalsePositive), f(d.NeedsReview), d.Recipe, f(d.RecipeConfidence), f(d.RiskScore), f(d.HighRisk)}
		for _, k := range sigCols {
			if v, ok := d.Signals[k]; ok {
				row = append(row, f(v))
			} else {
				row = append(row, "")
			}
		}
		o := cs.Outcome
		row = append(row, labelCell(o.FalsePositive), labelCell(o.RecipeCorrect), labelCell(o.NeededReview), labelCell(o.VerificationPassed), labelCell(o.Reverted))
		rows = append(rows, row)
	}
	if stringFlag(c, "output", "o") == "json" {
		out := make([]map[string]string, 0, len(rows))
		for _, r := range rows {
			m := map[string]string{}
			for i, h := range header {
				m[h] = r[i]
			}
			out = append(out, m)
		}
		printJSON(out)
		return nil
	}
	w := csv.NewWriter(os.Stdout)
	_ = w.Write(header)
	_ = w.WriteAll(rows)
	w.Flush()
	return w.Error()
}
