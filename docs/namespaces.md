# GRP Namespace Guide

GRP diagnostics and recipes use a dot-separated namespace to identify the origin and domain of each code.

## Namespace classes

| Namespace    | Class             | Examples                                      |
|-------------|-------------------|-----------------------------------------------|
| `go.*`      | Language binding  | `go.high-complexity`, `go.deep-nesting`       |
| `ts.*`      | Language binding  | `ts.missing-type`, `ts.any-usage`             |
| `react.*`   | Framework binding | `react.large-component`, `react.too-many-hooks` |
| `next.*`    | Framework binding | `next.static-props-missing`, `next.client-component-rule` |
| `framework.*` | Generic framework | Reserved for framework-agnostic extensions    |
| `tool.*`    | External tool     | `tool.gritql.pattern-match`, `tool.eslint.no-unused-vars` |
| `vendor.*`  | Vendor-owned      | Reserved for third-party binding maintainers  |

## Language bindings

Language bindings encode findings from the core language itself, independent of any framework or tool.

- `go.high-complexity` — Cyclomatic complexity exceeds threshold
- `go.deep-nesting` — Control-flow nesting depth exceeds threshold
- `go.long-function` — Function body exceeds line limit
- `go.large-file` — File exceeds line limit
- `go.ignored-error` — Error return value discarded with `_`
- `go.too-many-params` — Function parameter count exceeds limit
- `ts.any-usage` — Type `any` used where a concrete type is available
- `ts.missing-return-type` — Function lacks explicit return type annotation

## Framework bindings

Framework bindings encode findings specific to a particular framework or library.

- `react.large-component` — Component exceeds line limit
- `react.too-many-hooks` — Component uses more hooks than the configured limit
- `react.too-many-effects` — Component has more `useEffect` calls than the configured limit
- `react.too-many-state-vars` — Component declares more state variables than the configured limit
- `react.repeated-jsx` — JSX element is duplicated more than the minimum threshold
- `react.mixed-responsibilities` — Component mixes state, effects, and rendering excessively
- `react.hardcoded-data` — Component embeds literal data that should be external
- `react.missing-prop-types` — Component is missing type annotations for its props
- `next.static-props-missing` — Server component missing `getStaticProps` or equivalent
- `next.client-component-rule` — Client component rules violated (e.g., `'use client'` directive)

## Tool bindings

Tool bindings encode findings from external analysis tools mapped into GRP.

- `tool.gritql.<pattern>` — GritQL pattern match
- `tool.eslint.<rule-id>` — ESLint rule violation
- `tool.semgrep.<rule-id>` — Semgrep rule match
- `tool.sarif.<tool>.<rule>` — Any SARIF-compatible tool finding

## Choosing a namespace

1. If the finding comes from the language parser/evaluator itself, use the language namespace (`go.*`, `ts.*`).
2. If the finding comes from a framework-specific analysis (React, Vue, Angular), use the framework namespace (`react.*`, `vue.*`, `angular.*`).
3. If the finding comes from an external tool (ESLint, GritQL, Semgrep), use the tool namespace (`tool.*`).
4. If no existing namespace fits and the finding is framework-neutral, use `framework.*`.
5. Vendors maintaining their own binding may use `vendor.<vendor-name>.*` with documentation.

## Recipe names

Recipes follow the same naming convention but use verbs:

- `go.flatten-nesting`
- `react.split-large-component`
- `react.extract-custom-hook`
- `tool.gritql.apply-pattern`

Recipe names should be imperative and describe the refactoring action.
