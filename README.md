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
| `girl plan <path>` | Generate structured GRP refactor plan |
| `girl pack <path>` | Create token-budgeted agent context pack |
| `girl verify <path>` | Detect available verification commands |

### `girl analyze`

Detects: large components, repeated JSX, too many hooks, too many state
variables, mixed responsibilities, complex conditionals, hardcoded data,
missing prop types.

Output: JSON, text, or markdown.

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
