# GIRL

**Grammar-Informed Refactoring Language** — a CLI for analyzing code and
generating GRP refactoring plans.

> **GRP** (Grammar Refactoring Protocol) is the protocol/schema for structured,
> source-grounded refactoring plans. **GIRL** is the reference CLI
> implementation of GRP, with [binding maturity](docs/bindings/maturity.md):
> **Reference** (Go runtime), **Stable** (TypeScript spec), and
> **Experimental** (React spec).

GIRL analyzes code, detects refactoring opportunities, and generates structured
GRP plans that make agent refactoring safe, repeatable, and token-efficient.

[![Release](https://img.shields.io/github/v/release/canadian-ai/girl)](https://github.com/canadian-ai/girl/releases/latest)
[![CI](https://github.com/canadian-ai/girl/actions/workflows/ci.yml/badge.svg)](https://github.com/canadian-ai/girl/actions/workflows/ci.yml)

## Design Philosophy

GRP is language-agnostic. GIRL uses tree-sitter for TypeScript/React/JavaScript
parsing and `go/ast` for Go analysis. These choices are implementation details —
GRP plans are parser-independent.

## GRP vs GIRL

| | GRP | GIRL |
|---|---|---|
| **Role** | Protocol/schema for source-grounded refactoring plans | Reference CLI implementation of GRP |
| **Scope** | Plan envelope, diagnostics, steps, verification | Analyzers (go/ast, tree-sitter), recipe engine, CLI |
| **Extensible** | Via binding namespaced codes (`go.*`, `react.*`) | Register recipes and diagnostics in code |
| **Language** | Language-agnostic | Go (go/ast) [Reference], TypeScript (tree-sitter) [Stable], React (tree-sitter) [Experimental] |

**Non-goals for GRP Core:**
- parser or AST format
- grammar engine
- codemod runtime
- AI agent
- language-specific rules

Binding maturity is tracked in [docs/bindings/maturity.md](docs/bindings/maturity.md).

## Architecture

GIRL refactoring follows a pipeline that maps source code through analysis
into structured GRP plans, then into agent context packs with verification:

```txt
source code
  -> analyzers (go/ast for Go, tree-sitter + sitter queries for TS/JS/TSX/JSX)
  -> binding maps findings into GRP diagnostics via recipe engine
  -> GRP plan generator (ordered steps with risk, requires, verify)
  -> context packer (token-optimized agent input with heuristic estimator)
  -> agent/codemod/human
  -> verification (typecheck/lint/test)
```

## Why

- **Prompt-based refactoring** is vague and unpredictable.
- **AST-only tools** are rigid and miss semantic intent.
- **GIRL** combines tree-sitter grammar queries, code structure analysis, and
  verification into a compact protocol for AI agents.

## Quick Start

```bash
# Install (latest release)
go install github.com/canadian-ai/girl@latest

# Or build from source
go build -o girl ./cmd/girl/

# Test installation
girl --help

# Analyze a file or directory
./girl analyze examples/messy-react-form --output text

# Analyze Go code explicitly, or use --lang auto to detect Go/TS
./girl analyze . --lang go --output text

# Create shareable benchmark and proof reports
./girl benchmark . --lang go --output markdown
./girl prove . --output text

# Generate a GRP refactor plan (JSON, Markdown, or GRP JSON)
./girl plan examples/messy-react-form --output markdown
./girl plan . --lang go --output grp-json

# Validate a GRP plan file
./girl validate examples/grp/minimal-plan.json

# Create a token-budgeted agent context pack
./girl pack examples/messy-react-form --budget 8000 --output markdown

# Detect available verification commands
./girl verify examples/messy-react-form --output text
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `girl analyze <path>` | Scan code for refactoring opportunities |
| `girl benchmark <path>` | Summarize GIRL findings across a repo |
| `girl prove <path>` | Generate a shareable repository health proof report |
| `girl nodes <path>` | List semantic nodes from TS/TSX files |
| `girl refs <path>` | List reference nodes, optionally filtered by symbol |
| `girl plan <path>` | Generate structured GRP refactor plan |
| `girl pack <path>` | Create token-budgeted agent context pack |
| `girl install <framework>` | Install agents/skills for a coding framework |
| `girl validate <file>` | Validate a GRP plan JSON file |
| `girl verify <path>` | Detect available verification commands |
| `girl review` | Check diff reviewability against a budget |
| `girl decompose` | Decompose a large diff into smaller reviewable tasks |

### `girl analyze`

Detects: large components, repeated JSX, too many hooks, too many state
variables, mixed responsibilities, complex conditionals, hardcoded data,
missing prop types, Go long functions, high complexity, deep nesting, large
files, ignored errors, and large parameter lists.

Output: JSON, text, or markdown. Use `--lang auto|ts|go` to choose the analyzer.


### `girl benchmark`

Summarizes analyzer output across a repository: files scanned, diagnostic totals, severity counts, top diagnostic codes, and worst files.

```bash
girl benchmark . --lang go --output markdown
```

### `girl prove`

Builds on the same summary model as `benchmark` and adds a 0-100 repository health score with a screenshot-friendly status label.

```bash
girl prove . --output text
```

### `girl plan`

Generates an ordered GRP plan with step-by-step refactoring actions, risk
levels, and required verification commands.

Output formats:
- `--output json` (default) — internal IR JSON
- `--output markdown` — human-readable Markdown
- `--output grp-json` — valid GRP Core JSON with deterministic IDs and requires linkage

### `girl pack`

Creates a token-budgeted context pack optimized for AI coding agents.
Includes file summaries, selected component snippets, diagnostics, steps,
risks, and verification commands.

Token estimation is heuristic (`len(text)/3`). A real tokenizer (tiktoken,
tokenizers) can be swapped in via the `tokens.Estimator` interface.

### `girl validate`

Validates a GRP plan JSON file against the core requirements:
required fields, valid enum values, deterministic ID formatting, diagnostic
uniqueness, step-diagnostic linkage, and relative file paths.

### `girl verify`

Detects available verification commands for a project by inspecting
`package.json`, `go.mod`, and project structure.

### `girl review`

Checks a unified diff against a reviewability budget to determine if it's safe for human review.

```bash
# Review a diff file
girl review --diff-file large-change.diff --output text

# Review from stdin
git diff main..feature | girl review --stdin --output markdown

# Fail CI if over budget
git diff main..feature | girl review --stdin --fail-on-over-budget
```

### `girl decompose`

Splits a large diff into atomic reviewable tasks by file category and dependency order.

```bash
# Decompose from diff file
girl decompose --diff-file large-change.diff --output markdown

# Write decomposition JSON for use with `girl pack --task`
girl decompose --diff-file large-change.diff --output-file .grp/decomposition.json

# Use task-scoped pack
girl pack . --task task_001_go --task-file .grp/decomposition.json --output markdown
```

## GIRL Recipes

Recipes are the unit of refactoring knowledge. Thresholds (lines, counts) are
configured in Go code via `internal/recipes.Thresholds` and its
`DefaultThresholds()` function — not YAML or config files.

- `react.split-large-component` — Split components by responsibility boundary
- `react.extract-repeated-jsx` — Extract repeated JSX into reusable components
- `react.extract-custom-hook` — Extract related logic into custom hooks
- `react.reduce-state-vars` — Consolidate state into reducer/grouped state
- `react.consolidate-effects` — Merge related useEffect calls
- `react.add-prop-types` — Add TypeScript interfaces for component props
- `react.extract-constants` — Move hardcoded data to external files
- `go.extract-function` — Split long Go functions into focused helpers
- `go.simplify-branches` — Reduce branching with guard clauses or smaller units
- `go.flatten-nesting` — Reduce deep nesting in Go functions
- `go.split-file` — Split large Go files by responsibility
- `go.handle-error` — Replace ignored errors with explicit handling
- `go.extract-options-struct` — Group large parameter lists

## Project Board

Track progress across milestones on the [GIRL/GRP GitHub Project](https://github.com/orgs/canadian-ai/projects/6):

- **GRP Core v0.1** — spec, schemas, pkg/grp types, deterministic GRP output, validate command
- **Bindings v0.1** — Go, TypeScript, React binding docs and verification detection
- **Context + Trust** — context pack improvements, CI, golden tests, privacy, dogfooding

Issues: [github.com/canadian-ai/girl/issues](https://github.com/canadian-ai/girl/issues)

## Roadmap

See `docs/roadmap/high-impact-plan.md` for the full timeline.

| Phase | Status | Target |
|-------|--------|--------|
| Initial scaffolding, Go self-hosting, core productionization | **Done** | May 18-25 |
| Dogfood recursion (0 GIRL self-diagnostics) | **Done** | May 26 |
| GRP Core v0.1 — spec, schemas, pkg/grp, grp-json, validate | **Done** | May 26 - Jun 1 |
| GRP-Go binding v0.1 | **Done** | Jun 2 - Jun 8 |
| GRP-TypeScript binding v0.1 | **Done** | Jun 2 - Jun 8 |
| GRP-React binding v0.1 | **Done** | Jun 2 - Jun 8 |
| GIRL context packs (privacy modes, budget tiers) | **Done** | Jun 9 - Jun 15 |
| Repo-native verification | **Done** | Jun 9 - Jun 15 |
| Golden tests and conformance | **Done** | Jun 9 - Jun 15 |
| LICENSE and NOTICE (Apache 2.0, branding reservations) | **Done** | Jun 14 |
| Tree-sitter TSX parser replacement | **Done** | Jun 15 |
| CI pipeline (CGo-enabled build, vet, test) | **Done** | Jun 15 |
| Production release | **Planned** | Jun 16+ |

Track via [GitHub Project](https://github.com/orgs/canadian-ai/projects/6) or see `docs/project.md` for issue details.

## GRP Plan Format

GRP Core is a minimal plan envelope. The full specification is in `docs/spec/`:

- **[Core](docs/spec/core.md)** — plan envelope, fields, risk levels, bindings
- **[Diagnostics](docs/spec/diagnostics.md)** — diagnostic shape, rules, severity/confidence
- **[Steps](docs/spec/steps.md)** — step shape, ID rules, execution modes
- **[Verification](docs/spec/verification.md)** — verification shape, types, detection rules
- **[Extensions](docs/spec/extensions.md)** — extension system, `requiredExtensions`, namespacing
- **[Lifecycle](docs/spec/lifecycle.md)** — artifact model: plan, context pack, verification result
- **[Conformance](docs/spec/conformance.md)** — Core and Binding conformance levels
- **[Schemas](schemas/grp-plan.v0.1.schema.json)** — JSON Schema files for Plan, Diagnostic, Step, Verification
- **[Examples](examples/grp/)** — minimal, GRP-Go, and GRP-React example plans
- **[Real Refactors](examples/real-refactors/)** — Go and React before/after refactoring demos with GRP plans, context packs, and verification
- **[Namespaces](docs/namespaces.md)** — diagnostic and recipe naming conventions

A real GRP plan from `examples/real-refactors/go-grp-plan.json`:

```json
{
  "specversion": "0.1",
  "id": "grp_f2a9c1e4",
  "type": "dev.refactor.plan",
  "source": "github.com/canadian-ai/girl",
  "subject": "examples/real-refactors/go-before",
  "language": "go",
  "goal": "Refactor order processing: reduce complexity, flatten nesting, fix ignored errors",
  "risk": "medium",
  "tool": "girl v0.1.0",
  "diagnostics": [
    {
      "id": "diag_001",
      "code": "go.high-complexity",
      "severity": "high",
      "confidence": "high",
      "message": "Function processOrder has cyclomatic complexity 19 (limit: 10)",
      "file": "examples/real-refactors/go-before/main.go",
      "span": { "startLine": 21, "startColumn": 1, "endLine": 96, "endColumn": 2 },
      "symbol": { "kind": "function", "name": "processOrder" },
      "metadata": { "complexity": "19", "threshold": "10" }
    },
    {
      "id": "diag_002",
      "code": "go.deep-nesting",
      "severity": "medium",
      "confidence": "high",
      "message": "Function processOrder has nesting depth 4 (limit: 3)",
      "file": "examples/real-refactors/go-before/main.go",
      "span": { "startLine": 31, "startColumn": 3, "endLine": 43, "endColumn": 4 },
      "symbol": { "kind": "function", "name": "processOrder" },
      "metadata": { "depth": "4", "threshold": "3" }
    }
  ],
  "steps": [
    {
      "id": "step_001_go.high-complexity_processOrder",
      "recipe": "go.simplify-branches",
      "title": "Simplify branching in processOrder",
      "action": "Simplify branching logic in processOrder with guard clauses and early returns",
      "target": { "file": "examples/real-refactors/go-before/main.go", "symbol": "processOrder", "kind": "function" },
      "risk": "high",
      "requires": ["diag_001"],
      "verify": [
        { "command": "go build ./...", "required": true, "type": "build" },
        { "command": "go test ./...", "required": true, "type": "test" }
      ],
      "execution": { "mode": "agent-assisted" }
    }
  ],
  "verification": [
    { "command": "go build ./...", "required": true, "type": "build" },
    { "command": "go vet ./...",   "required": true, "type": "lint" },
    { "command": "go test ./...",  "required": true, "type": "test" }
  ]
}
```

See [Namespaces](docs/namespaces.md) for the complete namespace convention.

## Tool Recipes

GIRL includes diagnostic recipes for complementary refactoring tools.
Install with `girl install <tool>`:

| Tool | Install | Description |
|------|---------|-------------|
| **OpenRewrite** | `girl install openrewrite` | Export diagnostics as OpenRewrite YAML recipes for Java refactoring |
| **RTK** | `girl install rtk` | Pipe GIRL through RTK for 60-90% token compression |
| **GritQL** | `girl install gritql` | Generate GritQL patterns from GIRL diagnostics |
| **Rust-LSP** | `girl install rust-lsp` | Export GIRL diagnostics in LSP format for rust-analyzer |

### OpenRewrite

```bash
girl install openrewrite
# Analyze and generate OpenRewrite YAML recipe
girl analyze src/main --lang java --output text
girl plan src/main --recipe openrewrite.export-yaml-recipe --output markdown
# Apply: mvn rewrite:run -Drewrite.activeRecipes=dev.refactor.GirlGeneratedRecipe
```

### RTK

```bash
girl install rtk
# All GIRL commands pipe through RTK automatically
rtk girl analyze . --output text
rtk girl plan . --goal "Refactor" --output markdown
rtk girl verify . --output text
```

### GritQL

```bash
girl install gritql
# Generate and apply GritQL patterns
girl analyze src/ --output json
girl plan src/ --recipe gritql.generate-pattern --output markdown
# Apply: grit apply generated-patterns.grit
```

### Rust-LSP

```bash
girl install rust-lsp
# Export diagnostics in LSP format for IDE consumption
girl analyze src/ --lang rust --output json
# Consumed by rust-analyzer and Rust IDE tooling
```

## Future Tool Bindings (post-v0.1)

- ESLint binding
- SARIF binding

## Framework Integrations

GIRL ships with first-class support for multiple AI coding frameworks. Install
agents/skills for your framework of choice:

```bash
# Coding frameworks
girl install opencode    # OpenCode agents
girl install claude      # Claude Code skill
girl install codex       # Codex skill
girl install pi          # Pi skill

# Refactoring tools
girl install openrewrite # OpenRewrite recipe generation
girl install rtk         # RTK token optimization
girl install gritql      # GritQL pattern generation
girl install rust-lsp    # Rust-LSP diagnostics
```

Or copy files manually:

| Framework/Tool | Source | Target |
|----------------|--------|--------|
| **OpenCode** | `opencode/agents/` | `.opencode/agents/` |
| **Claude Code** | `claude/` | `.claude/` |
| **Codex** | `codex/` | `.codex/` |
| **Pi** | `pi/` | `.pi/agent/` |
| **OpenRewrite** | `openrewrite/` | `.openrewrite/` |
| **RTK** | `rtk/` | `.rtk/` |
| **GritQL** | `gritql/` | `.gritql/` |
| **Rust-LSP** | `rust-lsp/` | `.rust-lsp/` |

### OpenCode

```bash
girl install opencode
# or manually: cp -r opencode/agents/* .opencode/agents/
```

Use `@girl-planner`, `@girl-implementer`, or `@girl-reviewer` agents.

```txt
@girl-planner analyze examples/messy-react-form and generate a GRP plan
```

### Claude Code

```bash
girl install claude
# or manually: cp -r claude/* .claude/
```

The GIRL skill registers via `skills/girl/SKILL.md`.

```txt
/girl analyze this component and plan the refactor
```

### Codex

```bash
girl install codex
# or manually: cp -r codex/* .codex/
```

The GIRL skill registers via `skills/girl/SKILL.md`.

### Pi

```bash
girl install pi
# or manually: cp -r pi/* .pi/agent/
```

The GIRL skill registers via `skills/girl/SKILL.md`.

## Privacy

- No source code uploaded by default.
- All analysis is local.
- Private eval suites stay in `evals/private/` (gitignored).
- Path redaction available for reports.

## License

Apache 2.0
