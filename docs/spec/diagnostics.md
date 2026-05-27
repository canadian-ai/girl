# GRP Diagnostics

**Grammar Refactoring Protocol — Diagnostic Shape**

Version 0.1 — Core specification.

## Diagnostic

A structured finding produced by a language/tool analyzer. Diagnostics are the
input to planning: each diagnostic describes a code smell, refactoring
opportunity, or quality concern.

```json
{
  "id": "diag_001",
  "code": "go.high-complexity",
  "severity": "high",
  "confidence": "high",
  "message": "Function handleRequest has cyclomatic complexity 22 (threshold: 10)",
  "file": "internal/server/handler.go",
  "span": {
    "startLine": 42,
    "startColumn": 1,
    "endLine": 89,
    "endColumn": 2
  },
  "symbol": {
    "kind": "function",
    "name": "handleRequest"
  },
  "metadata": {
    "complexity": 22,
    "threshold": 10
  }
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Stable diagnostic ID within the plan, e.g. `"diag_001"` |
| `code` | string | yes | Stable across major versions, e.g. `"go.high-complexity"` |
| `severity` | string | yes | One of: `"high"`, `"medium"`, `"low"` |
| `confidence` | string | yes | One of: `"high"`, `"medium"`, `"low"` |
| `message` | string | yes | Human-readable description. **Display-only** — planner logic must not parse message text. |
| `file` | string | yes | Repo-relative path. Must not be absolute. |
| `span` | object | no | Source location span (preferred over `line`) |
| `span.startLine` | integer | with span | 1-based start line |
| `span.startColumn` | integer | with span | 1-based start column |
| `span.endLine` | integer | with span | 1-based end line |
| `span.endColumn` | integer | with span | 1-based end column |
| `line` | integer | no | Single line number (fallback when span is unavailable) |
| `endLine` | integer | no | End line (single-line fallback companion) |
| `symbol` | object | no | Symbol-level target |
| `symbol.kind` | string | with symbol | e.g. `"function"`, `"component"`, `"variable"`, `"type"` |
| `symbol.name` | string | with symbol | Symbol identifier |
| `metadata` | object | no | Extensible key-value pairs for binding-specific data |
| `tags` | array | no | List of human-readable tag strings |
| `related` | array | no | Related diagnostic IDs within the same plan |
| `fixes` | array | no | Suggested fix descriptions (informational, not executable) |

### Rules

1. **Message is display-only.** Planner, step generator, and recipe logic must
   never parse, match, or branch on diagnostic message text. All structured
   data required for planning must be in `code`, `metadata`, `symbol`, or
   `span`.

2. **File must be repo-relative.** Absolute paths must not appear in
   `file`. Consumers should reject or repair absolute paths on read.

3. **Line numbers are 1-based.** Start line, end line, start column, and
   end column all use 1-based indexing.

4. **Code stability.** Diagnostic codes must be stable within a major spec
   version. A code like `go.high-complexity` must not be renamed or removed
   in a 0.x release without a deprecation notice.

5. **Span is preferred over single-line.** Producers should compute `span`
   when the analyzer provides range information. Single `line` is a fallback.

6. **Symbol is preferred for symbol-level diagnostics.** When a diagnostic
   targets a specific function, component, or type, `symbol` should be
   populated so planners can group diagnostics by symbol.

### Severity

| Value | Meaning |
|-------|---------|
| `high` | Likely bug, performance problem, or correctness risk |
| `medium` | Maintainability concern, moderate complexity |
| `low` | Style or minor readability opportunity |

### Confidence

| Value | Meaning |
|-------|---------|
| `high` | Analyzer is certain this is a genuine issue (e.g. complexity threshold exceeded) |
| `medium` | Analyzer has reasonable evidence but may produce false positives |
| `low` | Heuristic or pattern-based suggestion, review recommended |
