# GIRL

**Grammar-Informed Refactoring Language** for AI coding agents.

> **GRP Core** is the envelope. Bindings define how specific languages, frameworks, and tools speak it.
>
> **GIRL** is the first reference implementation of GRP, starting with Go, TypeScript, and React bindings.

GIRL analyzes code, detects refactoring opportunities, and generates structured
GRP (Grammar Refactoring Protocol) plans that make agent refactoring safe,
repeatable, and token-efficient.

## Design Philosophy

GRP does not care how you parse code.
GRP only cares that you can describe the refactor clearly.

GIRL is not a parser, linter, codemod engine, or AI agent. It is the handoff protocol between them.

## GRP vs GIRL

| | GRP | GIRL |
|---|---|---|
| **Role** | Minimal protocol for source-grounded refactoring plans | First reference implementation of GRP |
| **Scope** | Plan envelope, diagnostics, steps, verification | Go and TypeScript analyzers, recipe engine, CLI |
| **Extensible** | Yes — binding namespaced codes like `go.*`, `react.*` | Yes — register new recipes and diagnostics |
| **Language** | Language-agnostic | Focused on Go and TypeScript/React |

**Non-goals for GRP Core:**
- parser or AST format
- grammar engine
- codemod runtime
- AI agent
- language-specific rules

Binding maturity is tracked in [docs/bindings/maturity.md](docs/bindings/maturity.md).

## Architecture

```txt
Source code
  -> language/tool-specific analyzer
  -> binding maps findings into GRP diagnostics
  -> GRP plan
  -> GIRL context pack
  -> agent/codemod/human
  -> verification
```

## Why

- **Prompt-based refactoring** is vague and unpredictable.
- **AST-only tools** are rigid and miss semantic intent.
- **GIRL** combines grammar rules, code structure, semantic analysis, and
  verification into a compact protocol for AI agents.

## Quick Start

```bash
# Build
go build -o girl ./cmd/girl/

# Analyze a file or directory
./girl analyze examples/messy-react-form --output text

# Analyze Go code explicitly, or use --lang auto to detect Go/TS
./girl analyze . --lang go --output text

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
| `girl nodes <path>` | List semantic nodes from TS/TSX files |
| `girl refs <path>` | List reference nodes, optionally filtered by symbol |
| `girl plan <path>` | Generate structured GRP refactor plan |
| `girl pack <path>` | Create token-budgeted agent context pack |
| `girl validate <file>` | Validate a GRP plan JSON file |
| `girl verify <path>` | Detect available verification commands |

### `girl analyze`

Detects: large components, repeated JSX, too many hooks, too many state
variables, mixed responsibilities, complex conditionals, hardcoded data,
missing prop types, Go long functions, high complexity, deep nesting, large
files, ignored errors, and large parameter lists.

Output: JSON, text, or markdown. Use `--lang auto|ts|go` to choose the analyzer.

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

### `girl validate`

Validates a GRP plan JSON file against the core requirements:
required fields, valid enum values, deterministic ID formatting, diagnostic
uniqueness, step-diagnostic linkage, and relative file paths.

### `girl verify`

Detects available verification commands for a project by inspecting
`package.json`, `go.mod`, and project structure.

## GIRL Recipes

Recipes are the unit of refactoring knowledge:

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
| GRP-Go binding v0.1 | **In progress** | Jun 2 - Jun 8 |
| GRP-TypeScript binding v0.1 | **Planned** | Jun 2 - Jun 8 |
| GRP-React binding v0.1 | **Planned** | Jun 2 - Jun 8 |
| GIRL context packs | **Planned** | Jun 9 - Jun 15 |
| Repo-native verification | **Planned** | Jun 9 - Jun 15 |
| Golden tests and conformance | **Planned** | Jun 9 - Jun 15 |
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

A minimal GRP plan:

```json
{
  "specversion": "0.1",
  "id": "grp_8f41c2b9",
  "type": "dev.refactor.plan",
  "source": "github.com/canadian-ai/girl",
  "subject": ".",
  "language": "go",
  "goal": "Refactor long functions into smaller focused units",
  "risk": "medium",
  "diagnostics": [],
  "steps": [],
  "verification": []
}
```

See [Namespaces](docs/namespaces.md) for the complete namespace convention.

## Future Tool Bindings (post-v0.1)

- GritQL binding
- Tree-sitter binding
- OpenRewrite binding
- ESLint binding
- SARIF binding
- LSP binding

## Use with OpenCode

Copy the GIRL agents into your project:

```bash
cp -r opencode/agents/* .opencode/agents/
```

Then in OpenCode:

```txt
@girl-planner analyze examples/messy-react-form and generate a GRP plan
```

Or via the GIRL skill:

```txt
/girl analyze this component and plan the refactor
```

## Architecture

```txt
source code
  -> Go parser (AST analysis)
  -> visitor pattern (responsibility detection)
  -> recipe engine (pattern matching)
  -> GRP plan generator (structured plan)
  -> context packer (token-optimized agent input)
  -> agent coding harness (safe apply)
  -> verification (typecheck/lint/test)
```

## Privacy

- No source code uploaded by default.
- All analysis is local.
- Private eval suites stay in `evals/private/` (gitignored).
- Path redaction available for reports.

## License

Apache 2.0
