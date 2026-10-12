# GIRL

**The code-quality tool built for coding agents.** Point Claude Code, Codex, or any agent at a repository of any size and GIRL hands back the five fixes that matter most — with the exact line span, the code, what to do, and the commands that prove it worked.

[![Release](https://img.shields.io/github/v/release/canadian-ai/girl)](https://github.com/canadian-ai/girl/releases/latest)
[![CI](https://github.com/canadian-ai/girl/actions/workflows/ci.yml/badge.svg)](https://github.com/canadian-ai/girl/actions/workflows/ci.yml)

```bash
go install github.com/canadian-ai/girl/cmd/girl@latest
girl next          # what should I fix first?
```

```text
$ cd kubernetes && girl next
GIRL next: 5 of 3068 locations (3967 diagnostics, 5489 files, all scope)

1. [HIGH] staging/src/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation/validation.go:999 (ValidateCustomResourceDefinitionOpenAPISchema)
   do: Reduce nesting depth in ValidateCustomResourceDefinitionOpenAPISchema by extracting helper functions
   why: Function ValidateCustomResourceDefinitionOpenAPISchema has nesting depth 9 (limit: 4)
   why: Function ValidateCustomResourceDefinitionOpenAPISchema has cyclomatic complexity 113 (limit: 10)
   why: Function ValidateCustomResourceDefinitionOpenAPISchema is 292 lines (limit: 80)
   why: Function ValidateCustomResourceDefinitionOpenAPISchema has 6 parameters (limit: 5)
...
```

## Why agents want it

Linters were built for humans scrolling an IDE. Agents have different constraints: a finite context window, no memory between sessions, and a habit of "fixing" whatever they see first. GIRL is designed around those constraints.

| An agent needs | GIRL gives it |
|---|---|
| **Bounded output** — not 5,000 findings in the context window | `girl next` returns the top *N* items and a `remaining` count |
| **Ranking** — fix what matters, not what's first alphabetically | Severity × number of rules hit × span size, merged per function |
| **Somewhere to act** — not just a message | `file`, `line`, `endLine`, `symbol`, and a numbered code `excerpt` |
| **A next step** | A concrete `action` and named `recipe` for every item |
| **Proof** | The repository's own build / lint / test commands, detected automatically |
| **No noise** | Generated code, `vendor/`, `node_modules/`, `testdata/`, `.min.js`, `.d.ts` are skipped |
| **Stable contracts** | Versioned JSON schemas (`girl.next/v1`, `girl.capabilities/v1`), deterministic ordering |
| **Self-discovery** | `girl capabilities` describes every command and flag as JSON |
| **Privacy** | 100% local. No code leaves the machine. No API keys |

### The agent loop

```bash
girl next -o json                 # 1. ranked work: span, excerpt, action, recipe, verify commands
#                                   2. agent fixes ONE item
girl check --changed -o json      # 3. analysis + complexity ratchet + reviewability + repo verification
girl next --changed -o json       # 4. confirm it's gone; nothing new introduced
```

Each `girl next` item, abridged:

```json
{
  "rank": 1,
  "id": "examples/real-refactors/go-before/main.go#processOrder",
  "severity": "high",
  "file": "examples/real-refactors/go-before/main.go",
  "line": 27, "endLine": 105,
  "symbol": "processOrder",
  "codes": ["go.high-complexity", "go.deep-nesting", "go.ignored-error"],
  "problems": ["Function processOrder has cyclomatic complexity 26 (limit: 10)", "..."],
  "action": "Simplify branching logic in processOrder with guard clauses and early returns",
  "recipe": "go.simplify-branches",
  "excerpt": "   27  func processOrder(o Order) error {\n   28  \tif o.ID == \"\" {\n ..."
}
```

Flags: `--limit N`, `--changed`, `--base main`, `--min-severity medium`, `--excerpt-lines 30`, `-o text|json|markdown`.

### Plug it into your agent

```bash
girl install claude      # Claude Code skill  (.claude/skills/girl/SKILL.md)
girl install codex       # Codex skill
girl install opencode    # OpenCode planner / implementer / reviewer sub-agents
girl agent sync          # add the GIRL loop to AGENTS.md without touching your own instructions
```

Any other agent: `girl capabilities` is the whole manual in one JSON call.

Long-running agents, editors, and agent fleets can keep one warm process instead of starting the CLI per call:

```bash
girl serve --root .                      # http://127.0.0.1:7777, loopback only
curl -s -X POST localhost:7777/v1/next -d '{"path":"apps/web","limit":5}'
```

`POST /v1/next` takes the same options as `girl next`, `POST /v1/benchmark` returns repo-wide counts, and `GET /v1/capabilities` / `GET /healthz` describe the server. Every request path is confined to `--root` (symlinks resolved), and `--max-concurrent` bounds simultaneous analyses; `Server-Timing` reports queue time separately from analysis time.

## Tested on large codebases

Every CI run analyzes a 6,000-file synthetic monorepo (planted hotspots plus generated, vendored, and fixture noise) and the full Go standard library, and asserts: exact file accounting, no findings in generated or vendored code, the planted hotspot ranked first, identical output across runs and worker counts, no absolute paths in output, and a valid GRP plan. `make scale-bench` repeats this against pinned real-world repositories:

| Repository | Language | Files analyzed | Findings | v0.1.29 | v0.1.30 | Now |
|---|---|---:|---:|---:|---:|---:|
| Kubernetes v1.31.0 | Go | 5,489 | 3,967 | 15.2s | 3.6s | **1.6s** |
| Go 1.25 standard library | Go | 3,344 | 5,652 | 41.8s | 15.3s | **1.5s** |
| VS Code 1.95.0 | TypeScript | 4,965 | 4,561 | 61.3s (1 finding) | 26.3s | **10.5s** |
| TSX fixture monorepo (`k6-corpus`) | React | 2,250 | 7,500 | 16.9s | 7.8s | **3.1s** |
| Excalidraw v0.18.0 | TypeScript / React | 531 | 731 | 3.7s | 1.9s | **0.8s** |
| ripgrep 14.1.1 | Rust | 98 | 280 | 0.7s | 0.3s | **0.3s** |

<sub>`girl benchmark`, one fresh process per run, 4 vCPU Linux, warm file and Go build caches. v0.1.29 also analyzed generated files: in Kubernetes, 2,753 of 8,295 Go files carry a `Code generated ... DO NOT EDIT` header, and those produced roughly half of its findings. Every speed-up since v0.1.30 was checked byte-for-byte against the previous `analyze`/`next` output on these corpora.</sub>

```bash
make scale          # 10x synthetic monorepo + Go standard library (runs in CI)
make scale-bench    # + Kubernetes, VS Code, Excalidraw, ripgrep at pinned tags
GIRL_SCALE=1 GIRL_SCALE_CORPUS=/path/to/your/repo go test ./internal/commands -run TestScale -v
```

## Load-tested on different hardware (k6)

`scripts/k6-hardware.sh` runs `girl serve` in containers capped to different CPU and memory budgets and drives each with [k6](https://k6.io), simulating agents in their edit loop (`girl next` on a change-sized target, 0.5s think time) alongside a repo-wide rescan. The corpus is GIRL's own TSX fixtures (`testdata/real`, the golden React components, `examples/`): one copy as the agent's change, 250 copies (2,250 files, 7,500 findings) as the monorepo.

| Profile | CPUs / memory | Agent p50 | Agent p95 | Queue p95 | Repo scan (2,250 TSX files) | Memory after run |
|---|---|---:|---:|---:|---:|---:|
| edge | 1 / 1 GB | 149 ms | 424 ms | 197 ms | 12.0 s | 203 MiB |
| laptop | 2 / 4 GB | 49 ms | 72 ms | 7 ms | 6.4 s | 183 MiB |
| ci | 4 / 8 GB | 34 ms | 53 ms | 3 ms | 3.4 s | 211 MiB |

<sub>4 agents + 1 concurrent repo scan per profile, `--max-concurrent 2`, 30 s each, 0% errors. Agent throughput is bounded by think time (~7 req/s for 4 agents). On one CPU, agent requests queue behind the repo scan. That is the cost of sharing a single core, and `Server-Timing` exposes it.</sub>

```bash
make k6                                          # one local run (k6 or Docker)
make k6-hardware                                 # edge, laptop, ci, workstation (skips profiles larger than the host)
PROFILES="edge laptop" DURATION=60s VUS=8 scripts/k6-hardware.sh
MODE=local scripts/k6-hardware.sh                # no Docker: taskset + GOMAXPROCS
```

GIRL sizes its worker pool from the container's CPU quota (cgroup v1 and v2), so a 1-CPU container runs one worker instead of one per host core. The `k6 hardware profiles` workflow runs edge, laptop, and ci on pull requests that touch analysis code, and publishes the table to the job summary.

## Project quality contract

```bash
girl init                       # detect stack, workspaces, verification scripts, agent frameworks
girl check                      # the daily command
girl check --changed            # changed files only
girl check --base master        # compare against a Git base
girl check --ci                 # CI-friendly defaults
girl check --no-verify          # analysis/quality gates only
girl check --output json        # machine-readable result
```

`girl check` combines changed-file scoping, multi-language analysis, complexity regression checks, diff reviewability, project preflight, and repository-native verification.

> **GRP** (Grammar Refactoring Protocol) is the language-agnostic plan format GIRL emits. **GIRL** (Grammar-Informed Refactoring Language) is the reference CLI.

## Project configuration

GIRL keeps project policy in `.girl/config.yaml`:

```yaml
version: "1"
profiles:
  - cai
  - next
  - convex
workspaces:
  - apps/*
  - packages/*
analysis:
  changed_only: true
  exclude:
    - .next
    - node_modules
complexity:
  max: 10
  baseline: .girl/complexity-baseline.json
  fail_on: regression
reviewability:
  max_diff_lines: 1500
  max_touched_files: 12
  max_risk: medium
verify:
  typecheck:
    - bun run type-check
  lint:
    - bun run lint
  build:
    - bun run build:ci
agents:
  opencode: true
  codex: true
  claude: false
```

GIRL recognizes common project aliases such as `typecheck` / `type-check`, prefers `build:ci` over `build`, and detects CAI gates such as `lint:copy-wrap`, `provenance:check`, and `css:budget` when present.

## Core commands

| Command | Purpose |
|---|---|
| `girl next [path]` | Ranked, bounded, agent-ready work queue |
| `girl capabilities` | Every command and flag as JSON, for agent tool discovery |
| `girl serve` | Local HTTP API (`/v1/next`, `/v1/benchmark`) for agents, editors, and k6 |
| `girl init [path]` | Detect a project and create `.girl/config.yaml` |
| `girl check [path]` | Run the project quality contract |
| `girl agent sync` | Sync GIRL agent instructions safely |
| `girl analyze <path>` | Find refactoring opportunities |
| `girl complexity <path>` | Measure and ratchet TS/JS/React complexity |
| `girl benchmark <path>` | Summarize findings across a repository |
| `girl prove <path>` | Generate a repository health proof report |
| `girl plan <path>` | Generate a structured GRP refactor plan |
| `girl pack <path>` | Build a token-budgeted agent context pack |
| `girl verify <path>` | Detect repository verification commands |
| `girl review` | Check diff reviewability |
| `girl decompose` | Split a large diff into reviewable tasks |
| `girl workorder` | Generate agent-ready work orders |
| `girl preflight` | Check repository readiness |
| `girl launchkit validate` | Validate launch-kit quality gates |
| `girl validate <file>` | Validate a GRP plan |
| `girl receipt` | Produce execution evidence |
| `girl update` | Update GIRL from GitHub releases |

Run `girl <command> --help` for command-specific flags.

## Analysis and GRP

GIRL uses:

- `go/ast` for Go
- tree-sitter for TypeScript, JavaScript, TSX, JSX, and Rust
- parallel parsing (`GIRL_JOBS` sets the worker count; output is identical at any value)
- repository-native verification discovered from project files

The pipeline is:

```text
source / git diff
  -> analyzers
  -> diagnostics + recipes
  -> GRP plan
  -> context/work order
  -> agent or human change
  -> repository verification
  -> receipt
```

GRP keeps plans source-grounded and parser-independent. A plan contains diagnostics, ordered steps, risk, dependencies, and verification commands. Specification docs live under `docs/spec/`.

## Refactoring recipes

- **Go:** long functions, branching, nesting, file size, ignored errors, large parameter lists
- **TypeScript / JavaScript:** long functions and high complexity in any function, method, or callback (`ts.*`)
- **React:** large components, repeated JSX, hooks, state, effects, prop types, hardcoded data
- **Rust:** long functions, branching, nesting, file size, large parameter lists

Use a specific recipe when needed:

```bash
girl plan . --recipe react.split-large-component --output grp-json
```

## Complexity ratchet

```bash
# Establish a baseline
girl complexity . --lang ts --write-baseline .girl/complexity-baseline.json

# Fail only on regressions or newly-complex functions
girl complexity . --lang ts \
  --baseline .girl/complexity-baseline.json \
  --fail-on regression
```

Once the baseline exists, `girl check` enforces the configured complexity policy automatically.

## Reviewability

```bash
git diff main..HEAD | girl review --stdin --fail-on-over-budget

girl decompose --diff-file change.diff --output-file .grp/decomposition.json
girl pack . --task task_001 --task-file .grp/decomposition.json
```

`girl check` also evaluates the current Git diff against the configured reviewability budget, keeping large agent-generated changes human-reviewable.

## Privacy

GIRL runs locally. Source code does not need to leave the machine.

For sensitive context packs, use the available privacy controls and never include secrets or private evaluation data in generated artifacts.

## Development

Requires Go 1.25+.

```bash
go build -o girl ./cmd/girl/
go vet ./...
go test ./...
```

Repository instructions are in `AGENTS.md`. Binding maturity and deeper protocol documentation live under `docs/`.

## License

Apache 2.0. See `LICENSE` and `NOTICE`.
