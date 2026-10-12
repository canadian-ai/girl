# Changelog

All notable changes to GIRL (Grammar-Informed Refactoring Language) are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
- Go, TypeScript, Rust, and complexity analysis parse files in parallel; Go type-checking reuses one importer per worker. Go standard library: 85s → 15s; VS Code: 62s → 26s; Kubernetes: 15s → 1.1s (4 vCPU)
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
