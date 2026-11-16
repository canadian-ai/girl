# Changelog

All notable changes to GIRL (Grammar-Informed Refactoring Language) are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed (fail-closed enforcement, #119–#126)
- `girl check` fails (non-zero exit) on unparseable source in every mode, including `--no-verify`. Python (`ast`) and Go (`go/parser`) failures emit blocking `<lang>.syntax-error` diagnostics with line/column instead of being dropped or downgraded to warnings; a missing `python3` is a blocking `py.analysis-error`. TypeScript/JavaScript and Rust use an error-recovering grammar that rejects some valid source, so they emit advisory `ts.parse-recovery` / `rust.parse-recovery` instead (#119)
- `verificationRun` is true only when a command actually executed. New `verificationSummary` (verified, failed, not-configured, blocked, skipped) with configured/executed/passed/failed/errored/not-run counts; each command records exit code, `error` for commands that could not start, `not-run` for the `--fail-fast` remainder, and provenance. Discovery errors are surfaced instead of swallowed (#120)
- Changed-file discovery includes untracked non-ignored files, resolves paths from the Git root and constrains them to the requested subdirectory on segment boundaries, reads NUL-delimited output, skips symlinks, falls back to `origin/<base>`, and reports failures instead of returning an empty scan. `check` and `next` share it (#121)
- Project exclusions are applied before any file is read by every analyzer, the repeated-literal scan and the complexity engine, in all/changed/nested-workspace modes, using the nearest `.girl/config.yaml`; documented directory/glob/anchored semantics. Effective exclusions and skip counts are reported in `scope` (#122)
- Context packs: a shared export-boundary sanitizer covers every exported field; broader credential detection (PEM blocks, provider tokens, URL credentials, many assignment syntaxes, YAML block scalars, triple-quoted values, prose, high-entropy tokens); sensitive files are omitted; `redacted` and `public` fail closed; unknown privacy modes are rejected. Absolute diagnostic paths no longer collapse to an empty filename (#123)
- Python diagnostics map to GRP recipes (`py.repair-syntax` for parser failures); unmapped findings are reported in a plan's `unplanned` section instead of being dropped; `grp.CheckExecutionReadiness` and `girl validate --strict` separate structural validity from execution readiness (#124)
- A missing, unreadable, corrupt or incompatible complexity baseline is `unenforced` (blocking under `--strict`) with remediation, never a silent `skip`; baseline SHA-256 and schema are attached; non-JS projects report `not-applicable` instead of omitting the result (#125)

### Added
- `girl check --strict` (implied by `--ci`) and an evidence `verdict` (`verified`, `failed`, `blocked`, `unverified`) with `reasons`; invalid config values are reported in `configProblems`
- Repo-native Python verification detection (`compileall`; `pytest`, `ruff`, `mypy` only when declared)
- Read-only SonarQube provider and `girl sonar status|issues|verify`: normalized findings with provenance, bounded GRP action proposals, and remediation receipts tied to a fresh scan of the fixing revision. Mock-tested against the documented Web API; not yet validated against a live instance (#126)
- `docs/enforcement.md`, `docs/privacy.md`, `docs/extensions/sonarqube.md`

### Security
- In base/CI mode the complexity baseline is read from the base ref, so a change cannot rewrite its own baseline; changes to `.girl/config.yaml` or the baseline are reported in `policyFilesChanged`
- Analyzers never read through symlinks; Git base refs cannot be interpreted as options; project config is not inherited from parent directories outside a repository
- Verification command output is sanitized and size-capped before it reaches JSON results; SonarQube issue text is treated as untrusted (control characters stripped, length capped)
- `girl serve`: local HTTP API (`POST /v1/next`, `POST /v1/benchmark`, `GET /v1/capabilities`, `GET /healthz`). Loopback by default, request paths confined to `--root`, `--max-concurrent` queueing with `Server-Timing` queue/analysis split
- k6 load testing: `bench/k6/girl.js` (agent-loop + repo-scan scenarios, p95 thresholds), `scripts/k6-corpus.sh` (React corpus from the TSX fixtures in `testdata/` and `examples/`), `scripts/k6-hardware.sh` (edge / laptop / ci / workstation profiles via Docker `--cpus`/`--memory`, or `MODE=local` with taskset), `make k6` / `make k6-hardware`, and a `k6 hardware profiles` CI workflow
- Scale tests build their synthetic monorepo's React code from the real TSX fixtures in `testdata/`
- `girl next`: a bounded, ranked work queue for coding agents. Merges findings per function/component and returns the top items with line span, numbered code excerpt, action, recipe, and the repository's own verification commands (`girl.next/v1` JSON schema). Supports `--limit`, `--changed`, `--base`, `--min-severity`, `--excerpt-lines`
- `girl capabilities`: every command and flag as JSON (`girl.capabilities/v1`), generated from the live command tree, plus the recommended agent loop
- `ts.high-complexity` and `ts.long-function` diagnostics for any TypeScript/JavaScript function, method, or callback, with `ts.simplify-branches` / `ts.extract-function` recipes. Plain TypeScript codebases (e.g. VS Code) previously produced almost no findings
- Scale test suite: a synthetic mixed Go/TS monorepo with planted hotspots and generated/vendored/fixture noise runs in every `go test`; `make scale` (10x synthetic + Go standard library, in CI) and `make scale-bench` (Kubernetes, VS Code, Excalidraw, ripgrep at pinned tags)
- `GIRL_JOBS` sets the number of parallel analysis workers
- `girl update`: reads the latest `vX.Y.Z` from the GitHub Releases API and installs it. Verifies the SHA256SUMS checksum (when published) and that the binary runs, then atomically replaces itself; automatically falls back to building from source via `go install` when the release asset is incompatible with the host (e.g. an older glibc), so updates work regardless of the local libc. `--check --json` exposes status for tooling
- GRP Core reduction/collection slice (#90): semantic node identity with opaque provider-supplied capability IDs (no prefix enforced; e.g. `cap.booking`, `booking.create`), canonical targets, source-grounded reference evidence, standard compression blocks with inputs/outputs, and gated collect steps (`grp.reduction.collect`)
- Validator rejects unsafe collect ordering: a collect step must require at least one `grp.reduction.migrate` step and carry a verification gate; migration steps must also carry verification
- `schemas/grp-reduction.v0.1.schema.json` and `reduction` property on the plan schema; deterministic `reduction-duplicate`, `reduction-contract`, and `invalid-unsafe-collect` conformance fixtures
- Human-readable GRP plan rendering (`grp.RenderPlanMarkdown`) with safety rationale; `girl plan --output grp-markdown` and `girl validate --output markdown`
- Context pack reduction: `girl pack --reduction-file` prefers a reduced canonical block instead of dumping duplicate implementation internals
- Normalization now remaps step-to-step `requires` links so reduction plans survive renumbering deterministically
- `docs/spec/reduction.md` documenting the GRP Core reduction/collection lifecycle and the public CAI Lang/TEMPLE adapter contract

### Changed
- TypeScript analysis traverses each file once: the 11 whole-file query patterns run as one multi-pattern tree-sitter query, the five per-component patterns as another, and conditionals and loops are counted in one walk. Parser output is byte-identical on 5,505 files
- Each TS/JS file is parsed once; function complexity is measured on the React extractor's tree (`complexity.MeasureTree`)
- Compiled tree-sitter queries are cached per grammar for the process; analyzing a small change no longer spends ~95% of its time compiling queries (264 ms -> 7 ms on `testdata/real`)
- Go type-checking runs only for files that assign a call result to `_` (the only place type information is read), and standard-library export data is resolved with one batched `go list` instead of one subprocess per package (Go standard library: 15.7 s -> 1.7 s)
- GOMAXPROCS follows the container CPU quota on cgroup v1 layouts the Go runtime misses, so constrained CI runners and containers do not oversubscribe
- React component diagnostics (too-many-hooks, mixed-responsibilities, complex-conditionals, …) carry the component's line span, so `girl next` items always have a location and excerpt
- Go, TypeScript, Rust, and complexity analysis parse files in parallel; Go type-checking reuses one importer per worker. Go standard library: 41.8s → 15.3s; VS Code: 61.3s → 26.3s; Kubernetes: 15.2s → 3.6s (fresh `girl benchmark` process, 4 vCPU)
- Directory scans skip generated code (`// Code generated ... DO NOT EDIT.`, `@generated`, `*.pb.go`, `*.min.js`, `*.bundle.js`, `*.d.ts`); Go scans also skip `testdata/`. The TypeScript walker now uses the shared skip list (adds `vendor/`, `out/`, `.turbo/`, …)
- Go and Rust function diagnostics carry `endLine`/`span`, so context packs and `girl next` excerpts include the whole function
- `go.long-function` / `rust.long-function` messages no longer embed the file path (it is already in `file`)
- `girl analyze -o text|markdown` prints `file:line` for every finding
- Benchmark/proof reports and plan subjects no longer contain absolute paths
- Diagnostic ordering is fully deterministic (file/line tie-breakers)
- Skills for Claude Code, Codex, and Pi, the OpenCode agents, and the `girl agent sync` block now lead with the `girl next` → fix → `girl check --changed` loop

### Fixed
- `girl check --changed` aborted with `unsupported file extension: .go` whenever a `_test.go` file was in the diff; changed-file runs now skip tests and generated files like full scans do
- `girl plan <absolute path> --output grp-json` produced an invalid GRP plan (absolute `subject`)
- Structured diagnostics: Kind, Symbol, Span, EndLine, Package, Metadata, Tags, Related, Fixes on Diagnostic
- DiagnosticTarget() helper that prefers Symbol then Component then File
- Recipe registry: 14 diagnostic-code-to-recipe mappings (8 React + 6 Go)
- Deterministic GRP step IDs with ordinal + recipe + slug format
- Shared directory skip policy (internal/shared): .git, .grp, node_modules, vendor, dist, build, .next
- SARIF 2.1.0 exporter (internal/sarif)
- Malformed input test suite (10 cases) — no panics on bad input
- Component parser split: parser.go (459 lines) + component.go (401 lines)
- Expanded test coverage: ir, analyzer, goanalysis, planner, recipes, shared, parsertsx, packer, sarif
- Safer language detection with --lang auto detection improvements

### Fixed
- Planner no longer parses diagnostic messages to find targets
- extractTarget() message-parsing hack removed
- Parser robustness: no crashes on empty, malformed, deeply nested input
- Error handling hygiene: ignored errors in verifier, packer, plan command identified

### Changed
- Planner generateStepsFromDiagnostics uses recipe registry instead of switch
- assignStepIDs now produces deterministic unique IDs
- Go analyzer walk uses shared.ShouldSkipDir
- TSX parser walk uses shared.ShouldSkipDir
- Underlying types improved but existing CLI output format preserved
