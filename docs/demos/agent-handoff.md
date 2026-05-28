# Agent Handoff Demo: GIRL → AI Coder

## Overview

This demo shows the end-to-end flow of analyzing Go code with GIRL, generating a
GRP plan and context pack, then handing structured context to an AI coding agent
for implementation.

The flow: `girl analyze` → `girl plan` → `girl pack` → agent prompt → verification.

## Step 1: Input code

The target is a Go file with a `complexFunc` whose cyclomatic complexity exceeds
the recommended limit of 10 due to repeated `if` branches.

Source: `testdata/golden/go-high-complexity/input/main.go`

```go
package testdata

func complexFunc() {
	if true {
		_ = 1
	}
	if true {
		_ = 1
	}
	if true {
		_ = 1
	}
	if true {
		_ = 1
	}
	if true {
		_ = 1
	}
	if true {
		_ = 1
	}
	if true {
		_ = 1
	}
	if true {
		_ = 1
	}
	if true {
		_ = 1
	}
	if true {
		_ = 1
	}
	if true {
		_ = 1
	}
}
```

## Step 2: Analyze

```bash
girl analyze testdata/golden/go-high-complexity/input --output text
```

Expected output:

```
Found 1 issue(s):

  High:   0
  Medium: 0
  Low:    1

[LOW] Function complexFunc has cyclomatic complexity 13 (limit: 10)
      Reduce branching with early returns, guard clauses, or table-driven tests.
```

## Step 3: Generate GRP plan

```bash
girl plan testdata/golden/go-high-complexity/input --output grp-json
```

Expected plan (`testdata/golden/go-high-complexity/expected.plan.json`):

```json
{
  "specversion": "0.1",
  "id": "grp_e8286824",
  "type": "dev.refactor.plan",
  "source": "github.com/canadian-ai/girl",
  "subject": "testdata/golden/go-high-complexity/input",
  "language": "go",
  "goal": "Improve code quality",
  "risk": "low",
  "diagnostics": [
    {
      "id": "diag_001",
      "code": "go.high-complexity",
      "severity": "low",
      "confidence": "high",
      "message": "Function complexFunc has cyclomatic complexity 13 (limit: 10)",
      "file": "testdata/golden/go-high-complexity/input/main.go",
      "line": 3,
      "symbol": {
        "kind": "function",
        "name": "complexFunc"
      }
    }
  ],
  "steps": [
    {
      "id": "step_001_go.high-complexity_complexfunc",
      "recipe": "go.simplify-branches",
      "title": "go.simplify-branches",
      "action": "Simplify branching logic in complexFunc with guard clauses and early returns",
      "target": {
        "file": "testdata/golden/go-high-complexity/input/main.go"
      },
      "risk": "low",
      "requires": ["diag_001"],
      "verify": [
        { "command": "go build ./...", "required": true, "source": "binding-default", "confidence": "medium" },
        { "command": "go vet ./...",   "required": true, "source": "binding-default", "confidence": "medium" },
        { "command": "go test ./...",  "required": true, "source": "binding-default", "confidence": "medium" }
      ]
    }
  ],
  "verification": [
    { "command": "go build ./...", "required": true, "source": "binding-default", "confidence": "medium" },
    { "command": "go vet ./...",   "required": true, "source": "binding-default", "confidence": "medium" },
    { "command": "go test ./...",  "required": true, "source": "binding-default", "confidence": "medium" }
  ]
}
```

## Step 4: Pack context for agent

```bash
girl pack testdata/golden/go-high-complexity/input --budget 8000 --output markdown
```

The context pack bundles goal, file summaries, diagnostics, steps, and
verification into a single token-budgeted document:

```
# GIRL Context Pack

**Goal:** Refactor testdata/golden/go-high-complexity/input: simplify complex functions

**Token budget:** 8000
**Token estimate:** 43

## Files

- `testdata/golden/go-high-complexity/input/main.go`

## Summaries

- `testdata/golden/go-high-complexity/input/main.go`: Module with 41 lines,
  1 diagnostics, 0 hooks, 0 imports

## Diagnostics

- [LOW] Function complexFunc has cyclomatic complexity 13 (limit: 10)

## Steps

- step_001_go.simplify-branches_simplify-branching-logic-in-complexfunc-:
  Simplify branching logic in complexFunc with guard clauses and early returns

## Verification

```bash
go build ./...
```

```bash
go vet ./...
```

```bash
go test ./...
```
```

## Step 5: Agent prompt

Below is the actual prompt you'd give an AI coding agent. It combines the GRP
plan and context pack into a structured instruction:

```
You are a Go refactoring expert. Here is a GRP plan and context pack.

GOAL: Improve code quality

DIAGNOSTICS:
  1. [LOW] Function complexFunc has cyclomatic complexity 13 (limit: 10)
         File: main.go, Line: 3
         Reduce branching with early returns, guard clauses, or table-driven tests.

STEPS:
  1. step_001_go.high-complexity_complexfunc — Simplify branching logic in
     complexFunc with guard clauses and early returns
     Recipe: go.simplify-branches
     Target: main.go
     Risk: low

VERIFICATION:
  1. go build ./...   (required)
  2. go vet ./...     (required)
  3. go test ./...    (required)

Context pack:
  Token budget: 8000
  Token estimate: 43
  Files: testdata/golden/go-high-complexity/input/main.go (41 lines)

Please implement the steps in order and verify each one.
```

## Step 6: Verification

After the refactoring, run the verification commands:

```bash
$ go build ./...
$    # exit code 0 — build succeeds

$ go vet ./...
$    # exit code 0 — no suspicious constructs

$ go test ./...
$    # exit code 0 — all tests pass
```

## Summary

| Metric           | Before | After |
|------------------|--------|-------|
| ComplexFunc lines| 40     | ~5    |
| Complexity       | 13     | ~2    |
| Build            | pass   | pass  |
| Vet              | pass   | pass  |

The GRP context pack bridges GIRL's static analysis and the AI agent's
implementation — the agent receives a focused goal, ordered steps with
verification, and a token-budgeted code snippet without manually copying
files or reading raw diagnostic output.

## File

`docs/demos/agent-handoff.md`
