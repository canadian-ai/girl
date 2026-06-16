# GRP Binding Maturity Levels

**GRP spec version:** 0.1

Each GRP binding progresses through four maturity levels. This document defines the levels and records the current maturity of every binding in the GIRL project.

## Maturity levels

| Level | Description |
|-------|-------------|
| **Draft** | Documented idea only — namespace reserved, design notes exist, no implementation |
| **Experimental** | Emits valid GRP plans and diagnostics — limited test coverage, no golden tests |
| **Stable** | Documented, tested, examples included — covered by unit tests, binding README complete |
| **Reference** | Used by GIRL itself — covered by golden tests, used in production pipelines |

## Current binding maturity

| Binding | Maturity | Tests | Documentation | Golden tests | Used by GIRL |
|---------|----------|-------|---------------|-------------|--------------|
| Go (`go.*`) | **Reference** | `internal/goanalysis/analyzer_test.go`, `internal/planner/planner_test.go`, `internal/planner/golden_test.go` | `bindings/go/diagnostics.md`, `recipes.md`, `verification.md` | `grp-go`, `go-high-complexity`, `minimal-core` | Yes — `girl analyze`, `girl plan` |
| TypeScript (`ts.*`) | **Stable** | `internal/parsertsx/parser_test.go`, `component_test.go`, `malformed_test.go`, `pkg/grp/` | `bindings/typescript/diagnostics.md`, `recipes.md`, `verification.md`, `README.md` | None | No |
| React (`react.*`) | **Experimental** | `internal/recipes/diagnostics_test.go` (1 test) | `bindings/react/diagnostics.md`, `recipes.md`, `verification.md`, `README.md` | `grp-react`, `react-too-many-hooks` | No |
| Rust (`rust.*`) | **Experimental** | `internal/rustanalysis/analyzer_test.go` | `bindings/rust/diagnostics.md`, `recipes.md`, `verification.md`, `README.md` | None | No |
| ESLint (`tool.eslint.*`) | **Draft** | None | `docs/bindings/tool-bindings.md` (concept) | None | No |
| GritQL (`tool.gritql.*`) | **Draft** | None | `docs/bindings/tool-bindings.md` (concept) | None | No |
| Semgrep (`tool.semgrep.*`) | **Draft** | None | `docs/bindings/tool-bindings.md` (concept) | None | No |
| SARIF (`tool.sarif.*`) | **Draft** | None | `docs/bindings/tool-bindings.md` (concept) | None | No |
| OpenRewrite (`tool.openrewrite.*`) | **Draft** | None | `docs/bindings/tool-bindings.md` (concept) | None | No |

## Level definitions

### Draft

The namespace is reserved and a high-level design exists. No code, no tests, no generated GRP output.

- Namespace allocation in `docs/namespaces.md`
- Design sketch in `docs/bindings/` or `docs/bindings/tool-bindings.md`
- No implementation files in `bindings/`
- No test files
- May be contributed by external teams

### Experimental

A working implementation exists that produces valid GRP output. Tests cover the happy path but edge cases, golden tests, and documentation are incomplete.

- Binding directory in `bindings/<name>/`
- Diagnostic codes emit valid GRP `Diagnostic` structs
- Plan steps emit valid GRP `Step` structs
- At least one unit test per diagnostic
- No golden test scenario required
- Binding `README.md` may be partial

### Stable

The binding is fully documented and tested. It can be relied on by downstream consumers without contacting the maintainer.

- `bindings/<name>/diagnostics.md` — all diagnostics documented with severity, confidence, metadata, false positive risks
- `bindings/<name>/recipes.md` — all refactoring recipes documented
- `bindings/<name>/verification.md` — verification commands for the target ecosystem
- `bindings/<name>/README.md` — binding overview, parser choice, version
- Every diagnostic has a corresponding unit test
- Edge cases tested (malformed input, empty files, boundary thresholds)
- GRP output validates against the spec (`pkg/grp/validate.go`)
- No golden test required (but recommended)

### Reference

The binding is used by GIRL's own analysis pipeline. It is the highest quality tier and serves as a model for other bindings.

- All Stable requirements
- At least one golden test in `testdata/golden/<binding-name>/` with an `expected.plan.json` that is verified by `internal/planner/golden_test.go`
- The binding is wired into the CLI (`cmd/girl/`) as a default analyzer
- Golden tests are run as part of CI (`go test ./...`)
- Breaking changes to the binding require updating golden fixtures

## How to advance a level

### Draft → Experimental

1. Create `bindings/<name>/` directory with initial diagnostic, recipe, and verification stubs
2. Implement at least one diagnostic that emits a valid GRP `Diagnostic`
3. Write one unit test that exercises the diagnostic
4. Verify output validates against `pkg/grp/validate.go`

### Experimental → Stable

1. Complete all binding documentation files (diagnostics, recipes, verification, README)
2. Add unit tests for every diagnostic, including edge cases
3. Add unit tests for every recipe
4. Ensure all GRP output passes schema validation
5. Document known false positive risks for each diagnostic

### Stable → Reference

1. Add golden test fixtures in `testdata/golden/<binding-name>/`
2. Wire the binding into `internal/planner/golden_test.go`
3. Wire the binding into `cmd/girl/` so it runs as a default analyzer
4. Run `go test ./...` and verify golden tests pass
5. Submit a PR updating this maturity table
