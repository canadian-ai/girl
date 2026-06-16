---
name: girl
description: Grammar-Informed Refactoring Language (GIRL) — analyze code, detect refactoring opportunities, and generate structured GRP plans. Use when refactoring Go/TS/React code, decomposing large diffs, creating token-budgeted agent context packs, or verifying refactor safety.
---

# GIRL

GIRL analyzes code, detects refactoring opportunities, and generates structured GRP plans.

## Commands

- `girl analyze <path>` — detect refactoring smells
- `girl plan <path> --goal "<goal>"` — generate refactor plan
- `girl pack <path> --budget 8000` — create token-optimized context
- `girl validate <file>` — validate a GRP plan
- `girl verify <path>` — detect available verification commands
- `girl review --stdin` — check diff reviewability
- `girl decompose --diff-file <file>` — break large diffs into tasks

## Workflow

1. `girl analyze .` to find issues
2. `girl plan . --goal "<goal>"` for structured steps
3. Apply changes
4. `girl review --stdin` for budget check
5. `girl verify .` for typecheck/lint/test commands

## Supported

- Go: long functions, high complexity, deep nesting, ignored errors, large param lists
- React/TS: large components, repeated JSX, too many hooks/state vars, mixed responsibilities, missing prop types
