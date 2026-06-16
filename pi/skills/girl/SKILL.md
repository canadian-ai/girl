---
name: girl
description: Grammar-Informed Refactoring Language (GIRL) — analyze code, detect refactoring opportunities, and generate structured GRP plans for AI coding agents. Use when refactoring Go/TS/React code, decomposing large diffs, or creating token-budgeted context packs.
---

# GIRL — Grammar-Informed Refactoring Language

**GIRL** (Grammar-Informed Refactoring Language) is a CLI for analyzing code and generating structured GRP refactoring plans. **GRP** (Grammar Refactoring Protocol) is the protocol/schema for source-grounded refactoring plans.

## CLI Reference

```bash
girl analyze <path>      # Scan code for refactoring opportunities
girl plan <path>         # Generate structured GRP refactor plan
girl pack <path>         # Create token-budgeted agent context pack
girl validate <file>     # Validate a GRP plan JSON file
girl verify <path>       # Detect available verification commands
girl review              # Check diff reviewability
girl decompose           # Decompose large diffs into atomic tasks
```

## Flags

- `--output json|text|markdown|grp-json` — output format
- `--lang auto|go|ts` — language selection
- `--budget <tokens>` — token budget for context packs (4000/8000/12000/16000+)
- `--privacy private|redacted|public` — privacy mode for context packs
- `--goal "<goal>"` — refactoring goal for plan generation
- `--stdin` / `--diff-file <file>` — diff input for review/decompose
- `--task <id>` / `--task-file <file>` — task-scoped context packs from decomposition

## GIRL Workflow for Pi

```
[analyze] -> [plan] -> [apply] -> [review] -> [verify]
```

1. **Analyze**: `girl analyze . --output text` to detect smells
2. **Plan**: `girl plan . --goal "<goal>" --output markdown` for structured plan
3. **Apply**: Follow plan steps using Pi edit tools
4. **Review**: `git diff | girl review --stdin` to check reviewability
5. **Verify**: `girl verify . --output text` to confirm tests/lint pass

## Context Packs

For large codebases where token efficiency matters:

```bash
girl pack . --budget 8000 --output markdown
```

Produces: file summaries, relevant snippets, diagnostics by severity, ordered steps, risks, verification commands.

## Privacy Modes

| Mode | Behavior |
|------|----------|
| `private` (default) | No change — all analysis is local |
| `redacted` | Redact paths, API keys, tokens, emails |
| `public` | Sanitize "private"/"secret" path segments |

## Diagnostics by Language

**Go**: `go.long-function`, `go.high-complexity`, `go.deep-nesting`, `go.ignored-error`, `go.large-param-list`, `go.large-file`

**React/TS**: `react.large-component`, `react.repeated-jsx`, `react.too-many-hooks`, `react.too-many-state-vars`, `react.mixed-responsibilities`, `react.missing-prop-types`, `react.hardcoded-data`, `react.complex-conditional`
