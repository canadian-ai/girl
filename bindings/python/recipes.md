# GRP-Python Refactoring Recipes

| Recipe | Action | Verify |
|---|---|---|
| `py.extract-function` | Split long functions into cohesive helpers | `python3 -m compileall -q .`, test suite |
| `py.simplify-branches` | Simplify conditional paths and isolate branches | tests |
| `py.flatten-nesting` | Prefer guard clauses, early returns and small helpers | tests |
| `py.group-parameters` | Consider typed configuration objects for related arguments | tests |
| `py.split-module` | Separate modules by responsibility | imports and tests |
| `py.repair-syntax` | Repair syntax / parser failures before any refactor (`py.syntax-error`, `py.analysis-error`) | `python3 -m compileall -q .` |

Diagnostic mapping (enforced by `internal/planner/python_contract_test.go` for every code the analyzer emits):

| Diagnostic | Recipe |
|---|---|
| `py.long-function` | `py.extract-function` |
| `py.high-complexity` | `py.simplify-branches` |
| `py.deep-nesting` | `py.flatten-nesting` |
| `py.too-many-params` | `py.group-parameters` |
| `py.large-file` | `py.split-module` |
| `py.syntax-error`, `py.analysis-error` | `py.repair-syntax` (blocking) |

Findings without a recipe are listed in the plan's `unplanned` section instead of being dropped. `girl validate --strict` rejects a plan whose actionable findings have no step; a clean no-op plan stays valid and ready.

No recipe is applied automatically. GIRL provides ranked, source-grounded recommendations for agents to implement and verify.
