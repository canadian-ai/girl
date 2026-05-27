# GRP Steps

**Grammar Refactoring Protocol — Step Shape**

Version 0.1 — Core specification.

## Step

An ordered refactoring action recommended to address one or more diagnostics.
Steps are the executable unit of a GRP plan: each step describes what to do,
where to do it, and how to verify it.

```json
{
  "id": "step_001_go.high-complexity_buildComponentFromBody",
  "recipe": "go.simplify-branches",
  "title": "Simplify branching in buildComponentFromBody",
  "action": "Extract guard clauses and reduce nesting in buildComponentFromBody",
  "target": {
    "file": "internal/parsertsx/parser.go",
    "symbol": "buildComponentFromBody",
    "kind": "function"
  },
  "risk": "medium",
  "requires": ["diag_001"],
  "verify": [
    {
      "command": "go test ./...",
      "required": true,
      "source": "go",
      "confidence": "high"
    }
  ],
  "execution": {
    "mode": "agent-assisted"
  }
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Deterministic step identifier. Must not derive from action text. |
| `recipe` | string | no | Recipe identifier, e.g. `"go.simplify-branches"` |
| `title` | string | yes | Human-readable title |
| `action` | string | yes | Description of what to do |
| `target` | object | yes | Refactoring target |
| `target.file` | string | yes | Repo-relative file path |
| `target.symbol` | string | no | Symbol name within the file |
| `target.kind` | string | no | Symbol kind: `"function"`, `"component"`, `"type"`, etc. |
| `risk` | string | yes | One of: `"low"`, `"medium"`, `"high"` |
| `requires` | array | no | List of diagnostic IDs this step addresses |
| `verify` | array | no | Verification entries specific to this step |
| `execution` | object | no | Execution mode hint |

### Step ID Rules

1. **Must not derive from action text.** Step IDs are deterministic from
   diagnostic code + target symbol + ordinal, not from freeform description.

2. **Duplicate diagnostic codes must produce unique step IDs.** When a
   diagnostic fires N times for N different symbols, each gets its own
   ordinal.

3. **Format:** `step_<ordinal>_<diagnostic-code>_<target-slug>`

   Examples:
   - `step_001_go.high-complexity_buildComponentFromBody`
   - `step_002_go.high-complexity_findComponentEnd`
   - `step_001_react.large-component_UserProfileForm`

### Execution Modes

| Mode | Description |
|------|-------------|
| `manual` | Human applies the change by hand |
| `agent-assisted` | AI agent suggests or applies with human review |
| `codemod` | Automated codemod tool applies the change |
| `automated` | Safe to apply automatically (low-risk, mechanical) |
| `detect-only` | Informational; no action required |

### Execution Mode Guidelines

- `automated` — only for `low` risk steps that are purely mechanical (rename,
  extract constant, add type annotation). Safe to run without human review.
- `agent-assisted` — `medium` or `high` risk steps where an AI agent proposes
  the change and a human approves.
- `manual` — `high` risk structural changes that need human judgment.
- `codemod` — steps that have a known automated transformation in a tool like
  GritQL, OpenRewrite, or jscodeshift.
- `detect-only` — steps that inform the user but require no change.

### Risk Levels

See [Core specification](core.md#risk-levels) for risk definitions.
