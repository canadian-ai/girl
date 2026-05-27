# GRP-TypeScript Recipes v0.1

## ts.extract-function

| Field | Value |
|-------|-------|
| **Description** | Extract inline logic into a named function |
| **When to use** | Repeated logic, long functions, deeply nested blocks |
| **Diagnostic mapping** | `ts.large-function`, `ts.duplicated-logic` |
| **Risk level** | Low |
| **Verification** | `npm run build`, `npm test` |

## ts.simplify-branches

| Field | Value |
|-------|-------|
| **Description** | Reduce nesting via early returns, guard clauses, or switch |
| **When to use** | Conditionals nested 3+ levels deep |
| **Diagnostic mapping** | `ts.complex-conditional` |
| **Risk level** | Medium |
| **Verification** | `npm run build`, `npm test` |

## ts.extract-condition

| Field | Value |
|-------|-------|
| **Description** | Extract a complex boolean expression into a named predicate |
| **When to use** | Boolean conditions spanning 3+ sub-expressions |
| **Diagnostic mapping** | `ts.complex-conditional` |
| **Risk level** | Low |
| **Verification** | `npm run build`, `npm test` |

## ts.split-function

| Field | Value |
|-------|-------|
| **Description** | Split a monolith function into smaller focused functions |
| **When to use** | Function exceeds 50 lines or handles 3+ responsibilities |
| **Diagnostic mapping** | `ts.large-function` |
| **Risk level** | Medium |
| **Verification** | `npm run build`, `npm test` |

## ts.replace-any-with-unknown

| Field | Value |
|-------|-------|
| **Description** | Replace `any` annotations with `unknown` and add type guards |
| **When to use** | Functions accepting `any` parameters or returning `any` |
| **Diagnostic mapping** | `ts.unsafe-any` |
| **Risk level** | Medium |
| **Verification** | `npx tsc --noEmit`, `npm test` |

## ts.split-file

| Field | Value |
|-------|-------|
| **Description** | Split a file into multiple files by concern |
| **When to use** | File exceeds 400 lines with multiple unrelated exports |
| **Diagnostic mapping** | `ts.large-file` |
| **Risk level** | Low |
| **Verification** | `npm run build`, `npm test` |

## ts.extract-utility

| Field | Value |
|-------|-------|
| **Description** | Extract duplicated logic into a shared utility function |
| **When to use** | Same pattern appears in 3+ locations |
| **Diagnostic mapping** | `ts.duplicated-logic` |
| **Risk level** | Low |
| **Verification** | `npm run build`, `npm test` |
