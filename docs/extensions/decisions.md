# Decision models (experimental)

`girl decide` asks a hosted decision model about findings and agent changes.
The first provider is TypeSafe's **Jev**, a "System One" model that returns
typed, probabilistic answers (yes/no, choice, score) instead of text.

It is built around one idea: **measure before trusting**. Every answer is
recorded, outcomes are recorded against it, and the model's effect on GIRL is
off until you turn it on.

## The rule: stricter only

A decision can only make GIRL stricter:

| A decision may | A decision may never |
|----------------|----------------------|
| escalate an allowed action to human review | turn a denial or review into an approval |
| deprioritize a likely false positive | suppress a finding or remove it from a gate |
| suggest one of GIRL's **registered** recipes | invent a recipe, mark anything verified, skip verification, widen a policy |
| object to an agent's change | approve an agent's change |

`decisions.Combine` enforces this for policy decisions and a test checks every
combination of answers against every policy outcome. Suggested actions carry
no authority: they must pass `extensions.Evaluate` under an explicit policy.
Answers that don't match the questions, provider errors and low-confidence
answers have **no effect**.

## Modes

| Mode | Behaviour |
|------|-----------|
| `shadow` (default) | Answers are recorded and shown as "would: …" but change nothing. |
| `enforce` | Answers take effect, within the stricter-only rule. |
| `off` | `girl decide` refuses to run. |

Set the mode in `.girl/config.yaml` or per run with `--mode`. An unrecognised
mode is treated as shadow, never as enforce.

```yaml
decisions:
  mode: shadow                    # off | shadow | enforce
  provider: typesafe              # typesafe | aimlapi | eden
  model: jev-latest               # pin e.g. jev-1.14 for reproducibility
  max_findings: 25                # cost guard per run
  consistency: true               # negated-question check
  min_recipe_confidence: 0.8
  review_probability: 0.5
  high_risk_probability: 0.3
  false_positive_probability: 0.8
```

Because `.girl/config.yaml` is a gate policy file, `girl check` lists it in
`policyFilesChanged` when a change edits it.

## Commands

```bash
export TYPESAFE_API_KEY=...       # from console.typesafe.ai; never passed as a flag

girl plan . -o grp-json > plan.json
girl decide triage plan.json                  # findings GIRL could not map
girl decide triage plan.json --scope all      # every finding, ranked by priority
girl decide guard --plan plan.json --step <id> --diff-file change.diff
girl decide outcome <id> --verification pass  # record what actually happened
girl decide eval                              # how often was the model right?
girl decide export > ledger.csv               # table for tabular models
```

### `triage`

Reads a GRP plan or `girl sonar issues` output. Per finding it asks:

| Question | Type | Effect when over threshold |
|----------|------|----------------------------|
| `false_positive` | yes/no | `deprioritize` |
| `recipe` | choice among registered recipes for the file's language, plus `manual` | `suggest-recipe` |
| `needs_review` | yes/no | `escalate-to-review` |
| `risk` | score: low / medium / high | P(high) also escalates |

- **Consistency check.** Each yes/no question is asked together with its
  negation. If the two answers contradict each other the signal is discarded;
  for the safety question the stricter reading wins (escalate).
- **Signals.** Alongside the finding GIRL sends facts it already has: span
  length, symbol kind, other findings in the same file, whether it is a test
  file, commits in the last 90 days, and whether a test file sits beside it.
  `--no-signals` sends the finding alone.
- **Priority.** Results are ranked by severity × P(real issue) × a risk
  discount. Ranking never removes a finding.
- **Free ground truth.** For findings GIRL already maps to a recipe, the
  model's choice is compared with GIRL's (`agreesWithRegistry`) and recorded
  as an outcome, so recipe confidence can be evaluated without human labels.
  GIRL's own recipe always stands.

With `--policy`, actions are evaluated and the result shows `policyDecision`
and `combinedDecision`.

### `guard`

Checks an agent's diff against the plan step it was asked to perform. There
is no "approved" verdict: only `no-objection` or `review`, and no objection
still requires verification. `--fail-on-review` exits 2 on `review`.

GIRL's own rules run first, need no model, send nothing, and hold in every
mode. They object when the change:

- touches files outside the planned step (a test file beside it is fine);
- changes a lockfile, binary, generated file, build or dependency manifest,
  CI workflow or `.girl/` policy file;
- removes more test code than it adds;
- exceeds the file or changed-line budget; or is empty.

The model is consulted **only with `--send-diff`**. File names and line counts
alone were measured against the live model and its answers were near chance,
so GIRL does not send them or act on them. With `--send-diff` the diff is
sanitized, capped at 16 KiB, and the model is asked whether the change is in
scope, matches the recipe, and alters behaviour. A model can add an objection;
it can never clear one of GIRL's.

### `outcome`, `eval`, `export`

`outcome` records what happened to a triaged finding: `--false-positive`,
`--recipe-correct`, `--needed-review`, `--verification pass|fail`,
`--reverted`. A unique ID prefix is enough.

`eval` replays the ledger and, per signal, reports precision and recall at the
current threshold, a Brier score and calibration buckets (predicted versus
observed). It suggests a threshold only with at least 30 labeled cases and
only if some threshold reaches 90% precision; otherwise it says to keep the
signal in shadow mode. Suggestions are never applied automatically.

`export` writes one row per finding: decision features, numeric signals and
outcome labels (empty when unknown). This is the table a tabular model such as
SAP RPT needs: labeled rows are context, unlabeled rows are what to predict.

## What is sent

`girl decide` is the only GIRL feature that sends data off the machine, and
only when you run it (or start `girl serve --decisions`). `girl check`,
`analyze`, `plan` and `next` never call a model.

| Command | Sent | Never sent |
|---------|------|------------|
| `triage` | rule, severity, repository-relative path, line, message, signals, candidate recipe names | source code, snippets, absolute paths |
| `guard` | nothing | — |
| `guard --send-diff` | step recipe and action, file names, line counts, sanitized diff text (≤ 16 KiB) | anything outside the diff |

Everything passes the shared sanitizer (`internal/redact`); if a detector
still fires, that request is not sent. Keys are sent only over HTTPS, never
across a cross-host redirect, and never appear in output or errors.

## Where records live

| Record | Location | Notes |
|--------|----------|-------|
| Answer cache | `~/.cache/girl/decisions` | keyed by model + exact request; 30-day expiry; `--no-cache` bypasses |
| Decision ledger | `~/.config/girl/ledger/<repo>.jsonl` | append-only; `--ledger` or `GIRL_DECISION_LEDGER` overrides; `--no-ledger` skips |
| Receipts | `girl receipt --decisions triage.json` | attaches model version, input hash, mode and effects to a change receipt |

All are outside the repository and private to the user (0700 / 0600).

## HTTP

`girl serve --decisions` adds `POST /v1/decide/triage` with body
`{"plan": <GRP plan>, "scope": "unplanned" | "all"}`. Without the flag the
route does not exist and the server never contacts a model.

## Providers

| `--provider` | Endpoint | Key variable | Default model |
|--------------|----------|--------------|---------------|
| `typesafe` (default) | `https://api.typesafe.ai/v1/systemone` | `TYPESAFE_API_KEY` | `jev-latest` |
| `aimlapi` | `https://api.aimlapi.com/v1/decisions` | `AIMLAPI_API_KEY` | `typesafe/jev` |
| `eden` | `https://api.edenai.run/v3/alpha/decisions` | `EDENAI_API_KEY` | `typesafe/jev-latest` |

The TypeSafe preset follows the published OpenAPI spec and was exercised
against the live API (`jev-1.14.0`). The AIML API and Eden AI presets use the
same request and response bodies per their documentation but have not been
called live.

## Measured so far

Small samples from GIRL's own examples, not a benchmark:

- **Recipe choice:** on 16 findings GIRL already maps, Jev independently chose
  GIRL's recipe 15 times (the candidate list does not reveal the mapping).
- **Guard with diff text:** a correct hook extraction scored 0.93 "matches
  recipe"; a change that added an unrelated network call and a dependency
  scored 0.01, with behaviour change 0.97.
- **Guard without diff text:** near chance, which is why it is not used.
- **`needs_review`** came back between 0.78 and 0.94 for every finding tried.
  A signal that says yes to everything discriminates nothing; this is exactly
  what shadow mode and `eval` exist to expose before it is enforced.

## Not yet

- A SAP RPT provider. `export` produces the table it needs; the client is not
  written because the API could not be verified from here.
- Automatic outcome capture from `girl check` and CI (outcomes are recorded
  with `girl decide outcome`, or automatically for registry agreement).
- Reading decision thresholds from the base ref in CI.
