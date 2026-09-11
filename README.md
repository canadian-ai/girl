# GIRL

**Grammar-Informed Refactoring Language** — a local-first project quality contract for developers and AI coding agents.

GIRL analyzes source code, creates structured GRP refactoring plans, scopes changes, runs repository verification, and keeps agent workflows grounded in the same project rules.

> **GRP** (Grammar Refactoring Protocol) is the language-agnostic plan format. **GIRL** is the reference CLI implementation.

[![Release](https://img.shields.io/github/v/release/canadian-ai/girl)](https://github.com/canadian-ai/girl/releases/latest)
[![CI](https://github.com/canadian-ai/girl/actions/workflows/ci.yml/badge.svg)](https://github.com/canadian-ai/girl/actions/workflows/ci.yml)

## Start here

```bash
# Install
go install github.com/canadian-ai/girl@latest

# Initialize the current project
girl init

# Run the project quality contract
girl check
```

`girl init` detects the stack, workspaces, verification scripts, and installed agent frameworks, then writes `.girl/config.yaml`.

`girl check` is the normal daily command. It combines changed-file scoping, code analysis, CAI/project preflight checks, and the verification commands defined for the repo.

Useful variants:

```bash
girl check --changed            # changed files only
girl check --base master        # compare against a Git base
girl check --all                # whole project
girl check --ci                 # CI-friendly defaults
girl check --no-verify          # analysis/preflight only
girl check --output json        # machine-readable result
```

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

## Agent integration

Keep GIRL-owned skills synchronized without replacing hand-written project instructions:

```bash
girl agent sync
girl agent sync --framework opencode
girl agent sync --dry-run
```

GIRL updates its managed block in `AGENTS.md` and syncs GIRL-specific framework files. Existing repository instructions remain intact.

Legacy framework installation remains available through `girl install <framework>`.

## Core commands

| Command | Purpose |
|---|---|
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
- tree-sitter for TypeScript, JavaScript, TSX, and JSX
- Rust analysis for supported Rust repositories
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

Current recipes include React/TypeScript rules for large components, repeated JSX, hooks, state, effects, prop types, and hardcoded data; Go rules cover long functions, branching, nesting, file size, ignored errors, and large parameter lists.

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

## Reviewability

```bash
git diff main..HEAD | girl review --stdin --fail-on-over-budget

girl decompose --diff-file change.diff --output-file .grp/decomposition.json
girl pack . --task task_001 --task-file .grp/decomposition.json
```

This lets GIRL keep large agent-generated changes within a human-reviewable budget.

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
