---
name: girl
description: Local code-quality work queue for coding agents. Use when asked to refactor, clean up, reduce complexity, find what to fix, review a change, or keep a Go / TypeScript / React / Rust codebase healthy — especially in large repositories. `girl next -o json` returns the few highest-value fixes with code excerpts and verify commands; `girl check --changed` gates the result.
---

# GIRL (Codex)

GIRL turns a codebase of any size into a short, ranked list of fixes an agent can act on, then proves the fix with the repository's own checks. Everything runs locally; generated code, vendored dependencies, and test fixtures are excluded automatically.

## The loop

```bash
girl next -o json                  # 1. top 5 fixes: file, line span, excerpt, action, recipe, verify commands
#                                    2. fix ONE item; keep the diff small
girl check --changed -o json       # 3. analysis + complexity ratchet + reviewability + repo verification
girl next --changed -o json        # 4. confirm it is gone and nothing new appeared
```

Repeat until `girl next --changed` is empty and `girl check --changed` passes.

## Reading `girl next -o json` (schema `girl.next/v1`)

- `items[]` — ranked work. Each has `file`, `line`, `endLine`, `symbol`, `codes`, `problems`, `action`, `recipe`, and a numbered `excerpt` of the code.
- Several rules on one function are merged into a single item: fix the function once.
- `remaining` — how many more locations exist. Do not fetch them all; fix the top items first.
- `verify` — the repository's own build/lint/test commands. Run them after the change.

Useful flags: `--limit N`, `--changed`, `--base main`, `--min-severity medium`, `--excerpt-lines 30`, `-o markdown`.

## When to use the other commands

| Need | Command |
|---|---|
| Discover every command and flag as JSON | `girl capabilities` |
| A multi-step, risk-ordered refactor plan | `girl plan <path> --output grp-json` |
| Token-budgeted context for a sub-agent | `girl pack <path> --budget 8000 --output markdown` |
| Is my diff small enough to review? | `git diff \| girl review --stdin` |
| Split a large diff into tasks | `girl decompose --diff-file change.diff` |
| Repo-wide health numbers | `girl benchmark <path>` |

## Rules

- Preserve behavior. A refactor that changes output is a bug.
- Fix the highest-ranked item first; do not start a repo-wide rewrite.
- Never edit generated files (`// Code generated ... DO NOT EDIT`, `@generated`), `vendor/`, or `node_modules/`.
- Run the `verify` commands before reporting done, and report what you ran.

## What it detects

- Go: long functions, high cyclomatic complexity, deep nesting, ignored errors, many parameters, large files
- TypeScript / JavaScript: long functions and high complexity in any function, method, or callback
- React: large components, repeated JSX, too many hooks / state / effects, mixed responsibilities, missing prop types, hardcoded data
- Rust: long functions, high complexity, deep nesting, many parameters, large files
