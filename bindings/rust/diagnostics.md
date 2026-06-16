# Rust Diagnostics

All Rust diagnostics use the `rust.*` namespace.

## Diagnostic Codes

| Code | Severity | Description |
|------|----------|-------------|
| `rust.long-function` | low–high | Function exceeds line count threshold |
| `rust.high-complexity` | low–high | Cyclomatic complexity exceeds threshold |
| `rust.deep-nesting` | low–high | Control-flow nesting depth exceeds threshold |
| `rust.large-file` | low–high | File exceeds line count threshold |
| `rust.too-many-params` | low–medium | Function parameter count exceeds threshold |

## Severity Escalation

Severity scales with how far the metric exceeds the threshold:

- **Low**: Metric exceeds threshold
- **Medium**: Metric exceeds 1.5× threshold (nesting: threshold + 2)
- **High**: Metric exceeds 2× threshold (nesting: threshold + 4)

## Thresholds (DefaultConfig)

| Metric | Default Limit |
|--------|---------------|
| Max function lines | 80 |
| Max cyclomatic complexity | 10 |
| Max nesting depth | 4 |
| Max file lines | 500 |
| Max parameters | 5 |

## Metadata

Each diagnostic includes:
- `Code` — diagnostic code (e.g., `rust.long-function`)
- `File` — relative file path
- `Line` — start line of the function or file
- `Symbol` — qualified function name (e.g., `Counter::increment` or `add`)
- `Kind` — `function` or `file`
- `Suggestion` — human-readable refactoring advice
- `Span` — start/end line range (for function diagnostics)
