# GRP Core

**Grammar Refactoring Protocol — Plan Envelope**

Version 0.1 — Core specification.

## Framing

**GRP Core is the envelope. Bindings define how specific languages, frameworks,
and tools speak it.**

GRP Core defines only the universal handoff format for source-grounded
refactoring plans:

- `plan` — top-level envelope
- `diagnostic` — structured finding
- `step` — ordered refactoring action
- `verification` — repo-native command
- `extensions` — binding/tool namespace
- `conformance` — capability levels

It does **not** define:

- parser or AST format
- grammar engine
- codemod runtime
- AI agent
- language-specific rules or diagnostics

These belong in **bindings** (GRP-Go, GRP-TypeScript, etc.) or external tools.

## Invariant

A valid GRP plan must be understandable without knowledge of the internal
implementation that produced it.

## Plan Document

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

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `specversion` | string | GRP spec version, e.g. `"0.1"` |
| `id` | string | Content-derived deterministic identifier, prefixed `grp_` |
| `type` | string | Plan type discriminator, e.g. `"dev.refactor.plan"` |
| `source` | string | Producer identifier, e.g. `"github.com/canadian-ai/girl"` |
| `subject` | string | Repo-relative target path |
| `language` | string | Language tag: `"go"`, `"ts"`, `"js"`, or `"auto"` |
| `goal` | string | Description of what the refactor achieves |
| `risk` | string | One of: `"low"`, `"medium"`, `"high"` |
| `diagnostics` | array | Array of [Diagnostic](#diagnostic) objects |
| `steps` | array | Array of [Step](#step) objects |
| `verification` | array | Array of [Verification](#verification) entries |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `time` | string | ISO 8601 timestamp of plan generation |
| `repository` | string | Repository URL |
| `commit` | string | Commit hash the plan was generated against |
| `tool` | string | Tool name and version, e.g. `"girl v0.1.0"` |
| `extensions` | object | Binding/tool-specific metadata |
| `requiredExtensions` | array | List of extension keys consumers MUST support |
| `context` | object | Producer-chosen context or preamble |
| `artifacts` | array | Attached artifact references |

## Bindings

A **binding** maps a language, framework, or tool into GRP by defining:

- **Diagnostic codes** — namespaced identifiers like `go.high-complexity`
- **Recipe identifiers** — namespaced actions like `go.extract-function`
- **Verification rules** — repo-native commands for that ecosystem
- **Parser/analyzer choices** — GRP does not mandate a parser; each binding
  selects its own (e.g. `go/parser`, TypeScript Compiler API, Tree-sitter)

Bindings use the extension system to carry binding-specific metadata.
Consumers that do not support a given binding MUST ignore its extension fields.

### GRP-Go
- Analyzer: `go/parser` + `go/ast` + `go/types`
- Diagnostics: `go.long-function`, `go.high-complexity`, `go.deep-nesting`,
  `go.large-file`, `go.ignored-error`, `go.too-many-params`
- Verification: `go test ./...`, `go vet ./...`, `go build ./...`

### GRP-TypeScript
- Analyzer: TypeScript Compiler API, ts-morph, Tree-sitter, or Babel/SWC
- Diagnostics: `ts.large-function`, `ts.complex-conditional`, `ts.unsafe-any`,
  `ts.large-file`, `ts.duplicated-logic`
- Verification: TypeScript compiler check, tsconfig lint rules

### GRP-React
- Framework binding layered on TS/JS analysis
- Diagnostics: `react.large-component`, `react.too-many-hooks`,
  `react.repeated-jsx`, `react.mixed-responsibilities`
- Recipes: `react.split-large-component`, `react.extract-repeated-jsx`,
  `react.extract-custom-hook`

## Risk Levels

| Level | Description |
|-------|-------------|
| `low` | Mechanical change, low chance of breakage. Safe to apply automatically. |
| `medium` | Behavioral change, requires test verification. Review recommended. |
| `high` | Structural change, may affect other components. Human review required. |

## Verification

The [Verification spec](verification.md) defines the verification shape and detection rules.
