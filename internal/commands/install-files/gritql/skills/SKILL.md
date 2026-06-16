---
name: gritql
description: GritQL pattern generation from GIRL diagnostics. GIRL detects refactoring patterns and generates GritQL queries for automated code transformation.
---

# GritQL + GIRL

GIRL generates GritQL patterns from refactoring diagnostics.

## How It Works

1. `girl analyze <path>` detects refactoring patterns
2. GIRL generates GritQL patterns matching the diagnosed code
3. Apply with `grit apply pattern.grit`

## Diagnostics

- `gritql.pattern-available` — Diagnostic maps to a GritQL pattern
- `gritql.generate-pattern` — Generate GritQL pattern from diagnostic
- `gritql.apply-transform` — Apply GritQL transformation
