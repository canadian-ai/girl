# Python Diagnostics (GRP-Python v0.1)

| Code | Detection | Default limit | Suggested recipe |
|---|---|---|---|
| `py.long-function` | Function or async function source span | 80 lines | `py.extract-function` |
| `py.high-complexity` | AST branch/boolean path count | 10 | `py.simplify-branches` |
| `py.deep-nesting` | AST control-flow nesting | 4 | `py.flatten-nesting` |
| `py.too-many-params` | Positional, keyword-only, variadic parameters | 5 | `py.group-parameters` |
| `py.large-file` | Source file length | 500 lines | `py.split-module` |
| `py.syntax-error` | Python AST parse failure | N/A | Fix the syntax |

Findings use `py.*` codes, source-relative file paths, line and end-line spans, symbols, and actionable suggestions. Syntax errors are reported rather than silently ignored.

**Caveats:** Branch counts are heuristics, not coverage metrics; `match` arms and exception handling may be undercounted. Framework entrypoints, fixtures, and data-model functions may be intentionally long. Method parameter counts include `self` / `cls`. AST analysis cannot detect dynamic behavior or runtime type errors.
