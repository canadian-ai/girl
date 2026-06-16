---
name: girl
description: Grammar-Informed Refactoring Language (GIRL) — analyze code, detect refactoring opportunities, and generate structured GRP plans for AI coding agents. Use when refactoring Go/TS/React code or creating token-optimized agent context.
---

# GIRL — Grammar-Informed Refactoring Language

GIRL analyzes code, detects refactoring opportunities, and generates structured GRP plans. Use it for safe, repeatable refactoring with any AI coding agent.

## CLI Commands

| Command | Description |
|---------|-------------|
| `girl analyze <path>` | Scan code for refactoring opportunities |
| `girl plan <path>` | Generate structured GRP refactor plan |
| `girl pack <path>` | Create token-budgeted agent context pack |
| `girl validate <file>` | Validate a GRP plan JSON file |
| `girl verify <path>` | Detect available verification commands |
| `girl review` | Check diff reviewability |
| `girl decompose` | Decompose large diffs |

## Workflow for Codex

1. `girl analyze <path> --output text` — identify refactoring smells
2. Review diagnostics and choose target
3. `girl plan <path> --goal "<goal>" --output markdown` — generate plan
4. Apply plan changes via Codex edit tools
5. `girl review --stdin` — check diff is reviewable
6. `girl verify <path> --output text` — confirm verification commands pass

## Agent Context Packs

For large refactors, use `girl pack` to create a compact, token-efficient context:

```bash
girl pack <path> --budget 8000 --output markdown
```

Output includes: file summaries, component snippets, diagnostics, risks, verification commands.

## Privacy

All GIRL analysis is local. No code leaves the machine. Use `--privacy redacted` for path/content redaction in reports.

## Supported Languages

- Go: long functions, high complexity, deep nesting, ignored errors, large parameter lists
- TypeScript/React: large components, repeated JSX, too many hooks/state vars, mixed responsibilities, missing prop types
