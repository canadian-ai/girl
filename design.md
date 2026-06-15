# GIRL v0.1 Hardening Plan

## Goal
Harden GIRL for a credible v0.1 release. Focus on correctness, tests, and trust — no flashy new features.

## Priority 1 — Parser Regression Tests

### Problem
`internal/parsertsx/` has zero tests. TSX/TS/JS/JSX parsing is core to the React analysis pipeline but has no regression coverage.

### Plan
- Add `internal/parsertsx/parser_test.go`
- Create small fixture files for each extension under `testdata/parsertsx/`
- Cover: function components, arrow components, exported components, default exports, hooks, useState, useEffect, JSX children, event handlers, imports
- Edge cases: `React.memo`, `forwardRef`, generic arrows, optional chaining, template literal attrs, nested components
- Acceptance: `go test ./internal/parsertsx/...` passes

### Files changed
- `internal/parsertsx/parser_test.go` (new)

## Priority 2 — Replace Fake Token Estimation

### Problem
`len(content) / 3` repeated in 4 places (analyzer, planner, packer × 2). No shared interface.

### Plan
- Create `internal/tokens/estimator.go` with `Estimator` interface + `HeuristicEstimator`
- Export: `NewHeuristicEstimator()`, `(e *Estimator) Estimate(content string) int`, `(e *Estimator) EstimateBytes(content []byte) int`
- Replace all 4 call sites
- Add tests: ASCII prose, code snippets, long identifiers, non-ASCII, truncated
- Acceptance: no `len(...)/3` outside `internal/tokens`

### Files changed
- `internal/tokens/estimator.go` (new)
- `internal/tokens/estimator_test.go` (new)
- `internal/analyzer/analyzer.go` (use Estimator)
- `internal/planner/planner.go` (use Estimator)
- `internal/packer/packer.go` (use Estimator)

## Priority 3 — Make Recipe Thresholds Configurable

### Problem
Recipe thresholds duplicated between `internal/analyzer/analyzer.go` (Config struct) and `internal/recipes/recipe.go` (hardcoded: 200 lines, 3 reps, 5 hooks, 4 state vars, 2 effects). They can diverge.

### Plan
- Add `Thresholds` struct to `internal/recipes/` with fields: `LargeComponentLines`, `RepeatedJSXCount`, `MaxHooks`, `MaxStateVars`, `MaxEffects`
- Constructor `DefaultThresholds()` matches analyzer defaults
- Make each recipe struct accept `Thresholds` (function option or struct field)
- Pass `Thresholds` from planner (or from NewEngine)
- Add tests proving changing thresholds changes matching
- Acceptance: recipe logic has one source of truth

### Files changed
- `internal/recipes/types.go` (new — Thresholds struct)
- `internal/recipes/recipe.go` (use Thresholds)
- `internal/recipes/recipe_test.go` (threshold change tests)
- `internal/planner/planner.go` (thread thresholds)

## Priority 4 — Tighten Language Handling

### Problem
`languageTag()` in parser returns `typescriptreact`, `typescript`, `javascriptreact`, `javascript`. `resolveLang()` in commands returns `go`, `ts`. GRP output may say `auto`. These values are inconsistent.

### Plan
- Define canonical language constants in a shared package (e.g., `internal/lang/`)
- `--lang auto` resolves before analysis, so GRP output never says `auto`
- For `grp-json` output, use canonical values (`typescript`, `typescriptreact`, `go`)
- Add tests for `--lang auto` with Go, TS, and mixed dirs
- Acceptance: `grp-json` output never says `auto`

### Files changed
- `internal/lang/types.go` (new — constants + helpers)
- `internal/commands/types.go` (use lang package)
- `internal/commands/plan.go` (ensure concrete language in GRP)
- `internal/commands/types_test.go` (lang tests)

## Priority 5 — Tree-Sitter Query Correctness Per Grammar

### Problem
All 18 queries compiled against TSX grammar. `grammarFor()` selects correct language for parsing but queries use TSX grammar for TS, JS, JSX. Some queries may not match correctly in JS or TS.

### Plan
- Audit each query against `.ts`, `.js`, `.jsx` grammars
- Update `lazyInit()` to compile query sets per language (TSX, TS, JS)
- Add tests that parse `.ts`, `.js`, `.jsx` fixtures and verify correct results
- Acceptance: parser behavior deterministic across all 4 extensions

### Files changed
- `internal/parsertsx/parser.go` (query sets per grammar)

## Priority 6 — Add Golden Output Tests

### Problem
Golden tests exist only for planner (inside `internal/planner/`). CLI commands (`analyze`, `plan --output grp-json`, `pack --output grp-context-json`, `validate`) lack CLI-level golden tests.

### Plan
- Add `internal/commands/golden_test.go`
- Use small fixtures in `testdata/` 
- Golden files for analyze JSON, plan GRP-JSON, pack GRP-context JSON
- Keep human-readable
- Acceptance: future refactors show clear diffs

### Files changed
- `internal/commands/golden_test.go` (new)
- `testdata/golden/commands/` (new fixtures)

## Priority 7 — Add Release-Readiness Checks

### Problem
No CI workflow. `.github/` exists but empty.

### Plan
- Add `.github/workflows/ci.yml` with:
  - `go build ./...`
  - `go vet ./...`
  - `go test ./...`
  - Smoke test: build binary, run against Go + React fixture
- Acceptance: CI proves CLI works

### Files changed
- `.github/workflows/ci.yml` (new)

## Priority 8 — Update Docs Honestly

### Plan
- README: note tree-sitter usage, heuristic token estimation, code-configured recipes, GRP reference implementation
- Acceptance: docs match implementation

### Files changed
- `README.md`

## Implementation Order

1. Priority 1 (parser tests) — standalone, no deps
2. Priority 2 (tokens package) — no deps
3. Priority 4 (language handling) — no deps on 3 or 5
4. Priority 3 (recipe thresholds) — needs priority 4
5. Priority 5 (query per grammar) — standalone
6. Priority 6 (golden CLI tests) — needs 1, 2, 4
7. Priority 7 (CI) — standalone
8. Priority 8 (docs) — last, after all changes

## Verification

After each priority: `go build ./... && go vet ./... && go test ./...`
Final: build binary, run against examples, validate output.
