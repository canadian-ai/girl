# GRP-Python Refactoring Recipes

| Recipe | Action | Verify |
|---|---|---|
| `py.extract-function` | Split long functions into cohesive helpers | `python3 -m compileall -q .`, test suite |
| `py.simplify-branches` | Simplify conditional paths and isolate branches | tests |
| `py.flatten-nesting` | Prefer guard clauses, early returns and small helpers | tests |
| `py.group-parameters` | Consider typed configuration objects for related arguments | tests |
| `py.split-module` | Separate modules by responsibility | imports and tests |

No recipe is applied automatically. GIRL provides ranked, source-grounded recommendations for agents to implement and verify.
