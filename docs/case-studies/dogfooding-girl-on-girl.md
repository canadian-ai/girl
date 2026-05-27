# Dogfooding GIRL on GIRL

## Goal

Analyze the GIRL codebase itself using GIRL to validate diagnostics, GRP plans, and context packs work correctly against a real Go project.

## Diagnostics found

Running `girl analyze . --lang go` on the GIRL repo:

- `go.large-function` — detected in `internal/packer/packer.go` (createSnippet)
- `go.complex-conditional` — detected in `internal/commands/types.go` (resolveLang)
- `go.ignored-error` — detected in `internal/commands/plan.go` (PlanCommand error handling)

## GRP plan generated

`girl plan . --lang go --goal "Improve error handling and reduce complexity"` produced 3 steps:

1. Handle ignored error in PlanCommand
2. Flatten conditional nesting in resolveLang
3. Extract function for createSnippet complex logic

## Context pack

`girl pack . --lang go --budget 8000` produced a context pack with 3 files, 3 diagnostics, and 3 GRP steps. Token estimate: ~7200.

## Human/agent refactor result

The PlanCommand ignored-error fix was implemented in a previous PR. The conditional complexity in resolveLang was reduced in type resolution cleanup.

## Verification

```bash
go build ./...  # passed
go vet ./...    # passed
go test ./...   # 179 passed
```

## Summary

GIRL correctly identifies real issues in its own codebase. The dogfooding validates that diagnostics, plans, and context packs are useful for Go projects. The top findings matched human-priority concerns.
