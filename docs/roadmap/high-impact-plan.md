# High-Impact Implementation Plan

This plan turns the current GIRL/GRP review into small, shippable slices. The
goal is to make GIRL better as an agent preprocessor: local analysis, compact
diagnostics, deterministic GRP plans, focused context packs, and repo-native
verification.

## Success Criteria

- Plans do not parse human diagnostic messages to find symbols or targets.
- Every GRP step has a unique, stable ID.
- Adding a new diagnostic or language does not grow the planner switch.
- Go context packs include the right function snippets, not just file metadata.
- Go diagnostics are useful enough to drive refactors with low false-positive
  noise.
- Auto-detection skips irrelevant directories and handles mixed repos safely.
- Public docs show synthetic examples only and do not expose local paths.

## Priority 1: Structured Diagnostics

Problem: planner logic currently extracts targets from `Diagnostic.Message`.
Messages are for humans; planner input should be structured.

Deliverables:

- Add structured fields to `ir.Diagnostic`:
  - `Kind`: `function`, `file`, `component`, `hook`, `reference`, etc.
  - `Symbol`: primary symbol name, such as `buildComponentFromBody`.
  - `EndLine`: optional end line for snippet selection.
  - `Package`: optional package/module name for Go and future languages.
  - `Metadata`: optional string map for analyzer-specific details.
- Populate these fields in Go diagnostics.
- Populate compatible fields in React diagnostics where available.
- Replace `extractTarget` message parsing with structured target selection.
- Keep `Message` as display-only text.

Implementation order:

1. Extend the IR struct.
2. Update all diagnostic constructors.
3. Add a `diagnosticTarget(d ir.Diagnostic) string` helper that prefers
   `Symbol`, then `Component`, then `File`.
4. Remove message parsing from planner code.
5. Add tests for Go and React target selection.

Verification:

- `go test ./...`
- `./girl plan . --lang go --output markdown`
- Confirm step actions include real function/file targets.

Risk: low. Mostly additive struct fields and planner cleanup.

## Priority 2: Stable Unique GRP Step IDs

Problem: multiple diagnostics with the same code can produce duplicate step IDs,
for example `step_go.high-complexity`.

Deliverables:

- Step IDs should include a stable ordinal or short target slug.
- IDs should be deterministic for the same diagnostic ordering.
- IDs should avoid leaking absolute paths.

Recommended format:

```txt
step_<ordinal>_<diagnostic-code>_<target-slug>
```

Example:

```txt
step_001_go.high-complexity_buildComponentFromBody
```

Implementation order:

1. Sort diagnostics deterministically by severity, file, line, code, symbol.
2. Generate step IDs after all steps are assembled.
3. Slugify symbol/component/file basename only.
4. Add tests for duplicate-code diagnostics.

Verification:

- `go test ./...`
- Generate a plan for GIRL itself and confirm no duplicate step IDs.

Risk: low. Changes plan stability and should improve downstream agent execution.

## Priority 3: Recipe Registry

Problem: `generateStepsFromDiagnostics` is becoming a large language-specific
switch. It will get harder to maintain as GIRL adds languages and more recipes.

Deliverables:

- Add a recipe registry keyed by diagnostic code.
- Move Go recipe mapping out of planner core.
- Move React recipe mapping into the same registry path.
- Keep the planner responsible for orchestration, risk, verification, and IDs.

Suggested shape:

```go
type DiagnosticRecipe struct {
    Code string
    Recipe string
    Risk func(ir.Diagnostic) ir.Severity
    Verify func(ir.Diagnostic) []string
    Action func(ir.Diagnostic) string
}
```

Implementation order:

1. Create `internal/recipes/diagnostics.go`.
2. Add Go mappings first.
3. Move React mappings.
4. Make planner call `recipes.StepForDiagnostic(diag)`.
5. Delete the large switch once parity is confirmed.

Verification:

- Snapshot compare representative React plan output before/after.
- Snapshot compare Go plan output before/after.
- `go test ./...`

Risk: medium. Touches core planner behavior, but can be done incrementally.

## Priority 4: Go Context Pack Support

Problem: Go analysis currently fills basic `FileIR` fields. `girl pack --lang go`
cannot yet choose compact function snippets around diagnostics.

Deliverables:

- Include Go function summaries in `FileIR` or a language-neutral symbol model.
- Select snippets by diagnostic `File`, `Line`, and `EndLine`.
- Add package/file summaries to packs.
- Keep packs private-safe: no absolute paths, no hidden/private fixture leakage.

Implementation order:

1. Decide whether to extend `FileIR` or add `SymbolIR`.
2. Store function ranges from Go parser.
3. Update packer snippet selection to use diagnostic ranges.
4. Add `--privacy private` path redaction checks for Go packs.
5. Add smoke fixtures under synthetic `testdata/` only.

Verification:

- `./girl pack . --lang go --budget 12000 --output markdown`
- Confirm the pack includes high-value functions like parser/planner hotspots.
- `go test ./...`

Risk: medium. Requires careful IR shape so TSX and Go do not diverge.

## Priority 5: Lower-Noise Go Diagnostics

Problem: ignored-error detection currently treats any ignored call result as an
ignored error. That can over-report without type information.

Deliverables:

- Reduce false positives without requiring heavy dependencies by default.
- Add optional typed analysis later if needed.

Implementation order:

1. First-pass heuristic improvements:
   - Only report `_` in multi-return assignments.
   - Prioritize functions that themselves return `error`.
   - Ignore obvious non-error helpers if no error signal exists.
2. Add tests for common patterns:
   - `value, _ := strconv.Atoi(x)` should report.
   - `_, ok := m[k]` should not report.
   - `x, _ := pureTuple()` should not report unless known error-like.
3. Later optional mode: add typed analysis with `go/packages` behind a flag.

Verification:

- `go test ./...`
- Compare current GIRL self-analysis count before/after.
- Manually inspect any remaining `go.ignored-error` findings.

Risk: medium. Better precision may reduce count, which is good if documented.

## Priority 6: Safer Language Detection And Walking

Problem: auto-detect walks too broadly and defaults mixed repos to TypeScript.

Deliverables:

- Shared directory skip policy across analyzers.
- Explicit mixed-repo behavior.
- Better defaults for monorepos.

Implementation order:

1. Add shared `ShouldSkipDir(base string) bool` helper.
2. Skip `.git`, `.grp`, `node_modules`, `vendor`, `dist`, `build`, `.next`, and
   generated output dirs.
3. Return `go` when `go.mod` is present at target root.
4. For mixed repos without `go.mod`, prefer explicit error/warning or require
   `--lang` for deterministic behavior.
5. Add tests for root Go repo, TS app, and mixed fixture.

Verification:

- `go test ./...`
- `./girl analyze . --output text`
- `./girl analyze testdata/real --output text`

Risk: low. Improves performance and predictability.

## Priority 7: Documentation And Spec Alignment

Problem: docs describe the original React-only flow and older recipe identifier
rules. Go support and new roadmap need clear public-facing docs.

Deliverables:

- README documents `--lang auto|go|ts`.
- README lists current Go diagnostics and recipes.
- GRP spec clarifies recipe naming for language-level refactors.
- Docs state that examples are synthetic and private paths stay redacted.

Verification:

- Manual README/spec review.
- Run all documented commands against synthetic fixtures or GIRL itself.

Risk: low.

## Two-Week Execution Order

Week 1:

1. Structured diagnostics.
2. Stable step IDs.
3. Safer language detection.
4. README/spec alignment.

Week 2:

1. Recipe registry.
2. Go context pack support.
3. Lower-noise Go diagnostics.
4. Dogfood on GIRL parser/planner refactors.

## Dogfood Milestone

After priorities 1 through 4 land, use GIRL on itself:

```bash
./girl analyze . --lang go --output text
./girl plan . --lang go --goal "refactor GIRL parser and planner hotspots" --output markdown
./girl pack . --lang go --budget 12000 --output markdown
go build ./...
go vet ./...
go test ./...
```

Expected outcome: the generated GRP plan should be specific enough that an agent
can extract helpers from `internal/parsertsx/parser.go` and simplify
`internal/planner/planner.go` without reading the whole repo.
