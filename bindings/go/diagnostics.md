# GRP-Go Binding v0.1

**Binding identity:**
- Name: `GRP-Go v0.1`
- Source analyzer: `go/parser` + `go/ast` + `go/types` (Go stdlib)
- GRP Core output: namespaced diagnostic codes like `go.long-function`
- Verification: `go build ./...`, `go vet ./...`, `go test ./...`

---

## `go.long-function`

**Diagnostic code:** `go.long-function`
**Recipe:** `go.extract-function`

### Description

Function exceeds the configurable line threshold (default: 80 lines).
Long functions are harder to read, test, and maintain.

### Severity

- **high** if > 150 lines
- **medium** if > 80 lines

### Confidence

Always `high`. Direct line-count measurement on known AST boundaries.

### Metadata fields

| Field | Type | Description |
|-------|------|-------------|
| `lines` | int | Function line count |
| `threshold` | int | Configured threshold |

### Verification

```bash
go build ./...
go vet ./...
go test ./...
```

### Recommended recipe

`go.extract-function` — Extract cohesive blocks into named helper functions.

### False positive risks

- Init functions and `main()` are often legitimately longer.
- Table-driven tests with inline test cases may exceed thresholds.
- Godoc-style comments add visual length without complexity.

---

## `go.high-complexity`

**Diagnostic code:** `go.high-complexity`
**Recipe:** `go.simplify-branches`

### Description

Function has high cyclomatic complexity (default threshold: 15).
Complex functions are more likely to contain bugs and are harder to test.

### Severity

- **high** if complexity > 30
- **medium** if complexity > 15

### Confidence

Always `high`. Cyclomatic complexity is computed directly from AST branch nodes.

### Metadata fields

| Field | Type | Description |
|-------|------|-------------|
| `complexity` | int | McCabe cyclomatic complexity |
| `threshold` | int | Configured threshold |

### Verification

```bash
go build ./...
go vet ./...
go test ./...
```

### Recommended recipe

`go.simplify-branches` — Use guard clauses, early returns, and smaller helper functions.

### False positive risks

- Switch statements and type switches inflate counts but are often readable.
- Generated code (protobuf, stringer) may have unavoidable complexity.

---

## `go.deep-nesting`

**Diagnostic code:** `go.deep-nesting`
**Recipe:** `go.flatten-nesting`

### Description

Function has deep nesting (default threshold: 4 levels past function body).
Deeply nested code is hard to read and reason about.

### Severity

- **high** if nesting > 7
- **medium** if nesting > 4

### Confidence

Always `high`. Nesting depth is measured directly from AST block structure.

### Metadata fields

| Field | Type | Description |
|-------|------|-------------|
| `maxDepth` | int | Maximum nesting depth |
| `threshold` | int | Configured threshold |

### Verification

```bash
go build ./...
go vet ./...
go test ./...
```

### Recommended recipe

`go.flatten-nesting` — Extract nested blocks into helper functions, use early returns.

### False positive risks

- Error handling chains (`if err != nil`) within a single function are idiomatic Go.
- Protocol handlers with multiple validation steps can be nested legitimately.

---

## `go.large-file`

**Diagnostic code:** `go.large-file`
**Recipe:** `go.split-file`

### Description

File exceeds the line threshold (default: 500 lines).
Large files often mix unrelated responsibilities.

### Severity

- **high** if > 1000 lines
- **medium** if > 500 lines

### Confidence

Always `high`. File length is a direct line count.

### Metadata fields

| Field | Type | Description |
|-------|------|-------------|
| `lines` | int | File line count |
| `threshold` | int | Configured threshold |

### Verification

```bash
go build ./...
go test ./...
```

### Recommended recipe

`go.split-file` — Split file by type/function responsibility into multiple files.

### False positive risks

- Generated files (protobuf, stringer, mocks) are excluded by convention.
- Files with many constants or type definitions may be large but not complex.

---

## `go.ignored-error`

**Diagnostic code:** `go.ignored-error`
**Recipe:** `go.handle-error`

### Description

Return value of a function call in the error-returning pattern is discarded.
Ignored errors can mask failures.

### Severity

**medium** — always medium severity.

### Confidence

**high** when the call target is a known error-returning function.
**medium** when inferred from `(T, error)` return pattern without cross-package check.

### Verification

```bash
go vet ./...
go build ./...
```

### Recommended recipe

`go.handle-error` — Assign the error value and handle it explicitly.

### False positive risks

- `fmt.Fprint*` calls where the write error is intentionally ignored.
- Logging calls where failure is non-critical.
- `db.Close()` and similar cleanup in `defer` where the error is intentionally discarded.

---

## `go.too-many-params`

**Diagnostic code:** `go.too-many-params`
**Recipe:** `go.extract-options-struct`

### Description

Function has more parameters than the threshold (default: 6).
Large parameter lists make call sites hard to read and evolve.

### Severity

- **high** if > 10 parameters
- **medium** if > 6 parameters

### Confidence

Always `high`. Parameter count is measured directly from function signature AST.

### Metadata fields

| Field | Type | Description |
|-------|------|-------------|
| `count` | int | Number of parameters |
| `threshold` | int | Configured threshold |

### Verification

```bash
go build ./...
go test ./...
```

### Recommended recipe

`go.extract-options-struct` — Group related parameters into a configuration struct.

### False positive risks

- Constructor functions (`NewXxx`) often take many dependencies.
- Middleware and handler wrappers naturally have many arguments.
