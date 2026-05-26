# GRP Plan Format

GRP (Grammar Refactoring Protocol) plans structure refactoring intent so that
AI coding agents can execute refactors safely, step by step.

## Specification

### Plan Document

```json
{
  "planId": "grp_<unix_timestamp>",
  "goal": "string — description of what the refactor achieves",
  "risk": "low | medium | high",
  "target": "string — file or directory path",
  "tokenEstimate": "int — approximate input tokens for this plan",
  "fileCount": "int — number of files to be touched",
  "diagnostics": [
    {
      "code": "react.large-component",
      "severity": "low | medium | high",
      "message": "Human-readable diagnostic message",
      "file": "path/to/file.tsx",
      "line": 42,
      "component": "ComponentName",
      "suggestion": "How to fix"
    }
  ],
  "steps": [
    {
      "id": "step_<recipe_code>",
      "recipe": "recipe identifier",
      "action": "What to do",
      "file": "affected file",
      "risk": "low | medium | high",
      "verify": ["typecheck", "tests", "lint"]
    }
  ],
  "verification": [
    "npm run typecheck",
    "npm run lint",
    "npm test"
  ]
}
```

### Step Order

Steps should be executed in order. Each step builds on the previous.

1. Safe mechanical refactors first (extract JSX, rename).
2. Behavioral refactors second (extract hooks, split components).
3. Verification always after each meaningful change.

### Recipe Identifiers

Format: `<language>.<domain>.<pattern>`

Examples:
- `react.split-large-component`
- `react.extract-repeated-jsx`
- `react.extract-custom-hook`
- `react.reduce-state-vars`
- `react.consolidate-effects`
- `react.add-prop-types`
- `react.extract-constants`

### Verification Commands

After a refactor, run verification:

- `typecheck` — ensure type correctness
- `tests` — run unit/integration tests
- `lint` — check code style
- `build` — verify the project builds

### Risk Levels

- `low` — mechanical change, low chance of breakage
- `medium` — behavioral change, requires test verification
- `high` — structural change, may affect other components
