---
name: girl
description: Grammar-Informed Refactoring Language (GIRL) — analyze code, detect refactoring opportunities, and generate structured GRP plans. Use when refactoring Go/TS/React code, decomposing large diffs, creating token-budgeted agent context packs, or verifying refactor safety.
---

# GIRL — Grammar-Informed Refactoring Language

GIRL analyzes code, detects refactoring opportunities, and generates structured GRP (Grammar Refactoring Protocol) plans. Use it for safe, repeatable, token-efficient refactoring.

## Commands

```bash
# Analyze for refactoring opportunities
girl analyze <path> --output text

# Generate a structured refactor plan
girl plan <path> --goal "<goal>" --output markdown

# Create a token-budgeted agent context pack
girl pack <path> --budget 8000 --output markdown

# Validate a GRP plan
girl validate <file>

# Detect available verification commands
girl verify <path> --output text

# Check diff reviewability
girl review --diff-file <diff>

# Decompose large diffs
girl decompose --diff-file <diff>
```

## GIRL Workflow

1. **Analyze** — `girl analyze .` to detect refactoring smells (large components, repeated JSX, high complexity, etc.)
2. **Plan** — `girl plan . --goal "<goal>"` to generate ordered GRP plan with steps, risks, verification
3. **Pack** — `girl pack . --budget 12000` for token-optimized agent context
4. **Verify** — `girl verify .` to detect available typecheck/lint/test commands
5. **Review** — `girl review --stdin` after applying changes to check budget

## Use with Claude Code

- `girl analyze` before refactoring to identify what needs work
- `girl plan` to generate a structured plan Claude Code can follow step-by-step
- `girl pack` to create compact context when working with large codebases
- `girl verify` before and after to confirm verification commands pass

## Bindings

- Go (`go.*` diagnostics): long functions, high complexity, deep nesting, ignored errors, large parameter lists
- TypeScript/React (`react.*` diagnostics): large components, repeated JSX, too many hooks/state vars, mixed responsibilities, missing prop types

## Resources

- Repo: https://github.com/canadian-ai/girl
- Spec: `docs/spec/` in the GIRL repo
- OpenCode agents: `opencode/agents/` in the GIRL repo
