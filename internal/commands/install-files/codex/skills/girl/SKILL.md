---
name: girl
description: Grammar-Informed Refactoring Language (GIRL) — analyze code, detect refactoring opportunities, and generate structured GRP plans for AI coding agents. Use when refactoring Go/TS/React code or creating token-optimized agent context.
---

# GIRL

GIRL analyzes code, detects refactoring opportunities, and generates structured GRP plans.

## CLI

- `girl analyze <path>` — scan for refactoring opportunities
- `girl plan <path>` — generate structured GRP refactor plan
- `girl pack <path>` — create token-budgeted agent context pack
- `girl validate <file>` — validate a GRP plan JSON file
- `girl verify <path>` — detect available verification commands
- `girl review` — check diff reviewability
- `girl decompose` — decompose large diffs

## Workflow for Codex

1. `girl analyze <path>` to identify smells
2. `girl plan <path> --goal "<goal>"` for structured plan
3. Apply plan steps
4. `girl review --stdin` to verify diff budget
5. `girl verify <path>` to confirm commands pass

## Context Packs

```bash
girl pack <path> --budget 8000 --output markdown
```

Includes: file summaries, component snippets, diagnostics, risks, verification commands.

## Privacy

All analysis is local. Use `--privacy redacted` for path/content redaction.
