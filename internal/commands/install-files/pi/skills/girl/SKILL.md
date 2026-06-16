---
name: girl
description: Grammar-Informed Refactoring Language (GIRL) — analyze code, detect refactoring opportunities, and generate structured GRP plans for AI coding agents. Use when refactoring Go/TS/React code, decomposing large diffs, or creating token-budgeted context packs.
---

# GIRL

GIRL analyzes code and generates structured GRP refactoring plans.

## CLI

- `girl analyze <path>` — scan for refactoring opportunities
- `girl plan <path>` — generate structured GRP refactor plan
- `girl pack <path>` — create token-budgeted agent context pack
- `girl validate <file>` — validate a GRP plan JSON file
- `girl verify <path>` — detect available verification commands
- `girl review` — check diff reviewability
- `girl decompose` — decompose large diffs

## Workflow

1. Analyze: `girl analyze . --output text`
2. Plan: `girl plan . --goal "<goal>" --output markdown`
3. Apply: follow plan steps
4. Review: `git diff | girl review --stdin`
5. Verify: `girl verify . --output text`

## Privacy Modes

- `private` (default) — no change, all analysis is local
- `redacted` — redact paths, API keys, tokens, emails
- `public` — sanitize "private"/"secret" path segments

## Diagnostics

- Go: `go.long-function`, `go.high-complexity`, `go.deep-nesting`, `go.ignored-error`, `go.large-param-list`, `go.large-file`
- React/TS: `react.large-component`, `react.repeated-jsx`, `react.too-many-hooks`, `react.too-many-state-vars`, `react.mixed-responsibilities`, `react.missing-prop-types`, `react.hardcoded-data`, `react.complex-conditional`
