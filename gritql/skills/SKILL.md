---
name: gritql
description: GritQL pattern generation from GIRL diagnostics. GIRL detects refactoring patterns and generates GritQL queries for automated code transformation.
---

# GritQL + GIRL

GIRL generates GritQL patterns from refactoring diagnostics for automated code transformation.

## How It Works

1. `girl analyze <path>` detects refactoring patterns
2. GIRL generates GritQL patterns matching the diagnosed code
3. Apply with `grit apply pattern.grit`

## Generated GritQL Patterns

GIRL maps diagnostics to GritQL patterns:

```
`react.large-component`  ->  Split component patterns
`react.repeated-jsx`     ->  Extract repeated JSX patterns  
`go.long-function`       ->  Extract function patterns
`go.deep-nesting`        ->  Flatten nesting patterns
```

Example: `go.long-function` generates:

```grit
// Extract long functions into smaller units
pattern extract_long_function() {
  `function $name($params) {
    $body
  }` where {
    $body <: contains repeated `return` within 50 lines
  }
}
```

## Use with GritQL

```bash
# Analyze and generate GritQL patterns
girl analyze src/ --output json > diagnostics.json
girl plan src/ --recipe gritql.generate-pattern --output markdown

# Apply with GritQL
grit apply generated-patterns.grit
```

## Diagnostics

- `gritql.pattern-available` — Diagnostic maps to a GritQL pattern
- `gritql.generate-pattern` — Generate GritQL pattern from diagnostic
- `gritql.apply-transform` — Apply GritQL transformation
