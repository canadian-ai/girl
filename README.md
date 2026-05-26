# GIRL

**Grammar-Informed Refactoring Language** for AI coding agents.

GIRL analyzes code, detects refactoring opportunities, and generates structured
GRP (Grammar Refactoring Protocol) plans that make agent refactoring safe,
repeatable, and token-efficient.

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

# Generate a refactor plan
./girl plan examples/messy-react-form --output markdown

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
| `girl verify <path>` | Detect available verification commands |

### `girl analyze`

Detects: large components, repeated JSX, too many hooks, too many state
variables, mixed responsibilities, complex conditionals, hardcoded data,
missing prop types, Go long functions, high complexity, deep nesting, large
files, ignored errors, and large parameter lists.

Output: JSON, text, or markdown. Use `--lang auto|ts|go` to choose the analyzer.

### `girl nodes`

Parses TS/TSX files into the semantic node graph and lists nodes. Use
`--kind component`, `--kind hook`, `--kind state`, `--kind jsx`, or
`--kind reference` to focus output.

### `girl refs`

Lists reference nodes from the semantic graph. Use `--symbol <name>` to focus on
one identifier.

### `girl plan`

Generates an ordered GRP plan with step-by-step refactoring actions, risk
levels, and required verification commands.

### `girl pack`

Creates a token-budgeted context pack optimized for AI coding agents.
Includes file summaries, selected component snippets, diagnostics, steps,
risks, and verification commands.

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

## Roadmap

See `docs/roadmap/high-impact-plan.md` for the next high-impact implementation
plan: structured diagnostics, stable GRP step IDs, recipe registry, Go context
packing, lower-noise diagnostics, safer language detection, and docs/spec
alignment.

## GRP Plan Format

A GRP plan is a JSON document containing:

```json
{
  "planId": "grp_1234567890",
  "goal": "Refactor component X: reduce component size and extract hooks",
  "risk": "medium",
  "steps": [
    {
      "id": "step_react.large-component",
      "recipe": "react.split-large-component",
      "action": "Split ComponentName into smaller focused components",
      "file": "src/Component.tsx",
      "risk": "medium",
      "verify": ["typecheck", "tests"]
    }
  ],
  "verification": ["npm run typecheck", "npm run lint", "npm test"]
}
```

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
