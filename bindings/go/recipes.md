# GRP-Go Recipes

- Binding name: `GRP-Go v0.1`
- Recipes are binding-owned, not core-owned.

## go.extract-function

| Field | Value |
|-------|-------|
| **Description** | Extract a block of inline code into a named function |
| **When to use** | Repeated logic, deeply nested blocks, or single long functions exceeding 40 lines |
| **Diagnostic mapping** | `go.large-function`, `go.duplicated-logic` |
| **Risk level** | Low – mechanical extraction preserves behavior |
| **Verification** | `go build ./...`, `go test ./...` |

## go.simplify-branches

| Field | Value |
|-------|-------|
| **Description** | Reduce nested conditionals by returning early, merging branches, or using switch |
| **When to use** | If-else chains deeper than 3 levels, or functions with multiple exit points that can be consolidated |
| **Diagnostic mapping** | `go.complex-conditional` |
| **Risk level** | Medium – control flow changes require care |
| **Verification** | `go build ./...`, `go test ./...` |

## go.flatten-nesting

| Field | Value |
|-------|-------|
| **Description** | Restructure deeply nested blocks by extracting inner logic or inverting conditions |
| **When to use** | Nesting depth > 4 in any single function |
| **Diagnostic mapping** | `go.large-function`, `go.complex-conditional` |
| **Risk level** | Medium – may change variable scoping |
| **Verification** | `go build ./...`, `go test ./...` |

## go.split-file

| Field | Value |
|-------|-------|
| **Description** | Split a monolith file into multiple files by type, responsibility, or logical grouping |
| **When to use** | Single file exceeds 800 lines or contains 5+ unrelated types |
| **Diagnostic mapping** | `go.large-file` |
| **Risk level** | Low – restructures without changing runtime behavior |
| **Verification** | `go build ./...` |

## go.handle-error

| Field | Value |
|-------|-------|
| **Description** | Add proper Go error handling where errors are silently discarded or panics are used instead of returned errors |
| **When to use** | Ignored error returns, bare panics in library code, missing error propagation |
| **Diagnostic mapping** | `go.ignored-error` |
| **Risk level** | High – retyping error contracts changes API surface |
| **Verification** | `go build ./...`, `go vet ./...`, `go test ./...` |

## go.extract-options-struct

| Field | Value |
|-------|-------|
| **Description** | Replace a long parameter list with a functional options struct |
| **When to use** | Constructor or function with 4+ parameters of the same type, or optional parameters |
| **Diagnostic mapping** | `go.large-function`, `go.too-many-params` |
| **Risk level** | Medium – API change requires updating callers |
| **Verification** | `go build ./...`, `go test ./...` |
