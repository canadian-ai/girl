# GRP-TypeScript Diagnostics v0.1

## ts.large-function

| Field | Value |
|-------|-------|
| **Code** | `ts.large-function` |
| **Title** | Function is too large |
| **Description** | Function exceeds 40 lines. Large functions are harder to test, reason about, and maintain. |
| **Severity** | `medium` |
| **Confidence** | `high` |
| **Recipes** | `ts.extract-function`, `ts.split-function` |
| **False positive risks** | Configuration objects, generated code, or switch-based dispatchers may be large by design. |

## ts.complex-conditional

| Field | Value |
|-------|-------|
| **Code** | `ts.complex-conditional` |
| **Title** | Conditional logic is too complex |
| **Description** | Conditional nesting exceeds 3 levels or cyclomatic complexity exceeds 10 per function. |
| **Severity** | `medium` |
| **Confidence** | `high` |
| **Recipes** | `ts.simplify-branches`, `ts.extract-condition` |
| **False positive risks** | State machines and parsers may legitimately have high cyclomatic complexity. |

## ts.unsafe-any

| Field | Value |
|-------|-------|
| **Code** | `ts.unsafe-any` |
| **Title** | Unsafe use of `any` type |
| **Description** | Using `any` bypasses TypeScript's type checking. Prefer `unknown` with proper narrowing. |
| **Severity** | `high` |
| **Confidence** | `high` |
| **Recipes** | `ts.replace-any-with-unknown`, `ts.add-type-guard` |
| **False positive risks** | Third-party type definitions may force `any` usage. |

## ts.large-file

| Field | Value |
|-------|-------|
| **Code** | `ts.large-file` |
| **Title** | File exceeds recommended size |
| **Description** | File exceeds 400 lines. Large files tend to accumulate unrelated concerns. |
| **Severity** | `low` |
| **Confidence** | `medium` |
| **Recipes** | `ts.split-file` |
| **False positive risks** | Generated files, type definition files, and configuration files are intentionally larger. |

## ts.duplicated-logic

| Field | Value |
|-------|-------|
| **Code** | `ts.duplicated-logic` |
| **Title** | Duplicated logic detected |
| **Description** | The same or structurally similar code appears in multiple locations. |
| **Severity** | `medium` |
| **Confidence** | `medium` |
| **Recipes** | `ts.extract-function`, `ts.extract-utility` |
| **False positive risks** | Test assertions, mocks, and intentionally repeated boilerplate. |
