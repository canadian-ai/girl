# Cyclomatic complexity in GIRL

## Executive finding

Cyclomatic complexity is best treated as a **control-flow and testing signal**, not a complete code-quality or readability score. GIRL therefore measures it per function, reports size beside it, and supports a regression ratchet instead of requiring an inherited repository to pass a universal threshold immediately.

The TypeScript/JavaScript implementation covers `.ts`, `.tsx`, `.js`, and `.jsx`, including React function components, hooks and callbacks, ordinary functions, arrow functions, generators, and class methods.

## Definition

McCabe defined cyclomatic complexity from a control-flow graph:

```text
M = E - N + 2P
```

- `E` is the number of graph edges.
- `N` is the number of graph nodes.
- `P` is the number of connected components. For one function, `P = 1`.

For structured source code, this is commonly implemented as `1 + decision points`. The result is the dimension of a basis set of linearly independent control-flow paths. It is **not** the total number of possible runtime paths: loops alone can make that number unbounded.

This makes the measure useful when deciding how many independent paths deserve focused testing. It does not prove that exactly `M` tests are sufficient for every data combination, state interaction, or fault.

Primary references:

- Thomas J. McCabe, [“A Complexity Measure” (1976)](https://doi.org/10.1109/TSE.1976.233837)
- NIST SP 500-99, [“Structured Testing: A Software Testing Methodology Using the Cyclomatic Complexity Metric”](https://www.nist.gov/publications/structured-testing-software-testing-methodology-using-cyclomatic-complexity-metric)
- NIST IR 5737, [basis-path construction](https://nvlpubs.nist.gov/nistpubs/Legacy/IR/nistir5737.pdf)

## TypeScript and React counting model

Each function begins at `1`. GIRL adds one for each source construct that creates another independent path:

| Construct | Increment | Notes |
|---|---:|---|
| `if` / `else if` | +1 each | A final `else` adds no decision. |
| `for`, `for…in`, `for…of`, `while`, `do…while` | +1 | Array calls such as `.map()` are calls, not branches. Their callbacks are measured separately. |
| non-default `case` | +1 | `default` does not add another predicate. |
| `catch` | +1 | Models the exceptional control-flow branch. |
| ternary `?:` | +1 | Includes conditional JSX rendering. |
| `&&`, `||`, `??` | +1 | These operators short-circuit. |
| `&&=`, `||=`, `??=` | +1 | Logical assignment also short-circuits. |
| default parameter or destructuring value | +1 | The default is conditionally evaluated. |
| each optional-chain segment `?.` | +1 | Matches modern ESLint complexity behavior. |

The implementation intentionally aligns with current [ESLint complexity semantics](https://eslint.org/docs/latest/rules/complexity) where JavaScript has evolved beyond McCabe's original language examples. GIRL uses tree-sitter directly, so no project ESLint configuration or TypeScript compilation is required.

### Function boundaries matter

Nested functions do not increase the parent function's score. For example:

```tsx
function List({ items }: Props) {             // CC 2: base + if
  if (!items) return null;
  return items.map(item =>                    // callback measured separately
    item.active ? <Active/> : <Inactive/>     // callback CC 2
  );
}
```

This avoids charging a React component for every branch inside event handlers, effects, memo callbacks, or collection callbacks. GIRL still reports each callback, so the paths remain visible rather than disappearing from the repository total.

## Thresholds

McCabe proposed splitting modules above `10`, and NIST structured-testing guidance retained `10` as a useful default while allowing justified exceptions. ESLint currently defaults to `20`. These are policies, not natural constants.

GIRL defaults to `10` because it is conservative and matches the existing Go/Rust analyzer, but exposes `--max` for team calibration:

```bash
girl complexity src --max 15
```

A threshold should trigger review, testing, or an explanation. It should not automatically prove that code is defective.

## Tracking policy

Raw repository totals mostly grow with the codebase and should not be used as a standalone health score. A large empirical study across more than 1.2 million C, C++, and Java files found that lines of code explained roughly 90% of cyclomatic-complexity variance. See Jay et al., [“Cyclomatic Complexity and Lines of Code”](https://doi.org/10.4236/jsea.2009.23020).

GIRL therefore records:

- per-function complexity and decision points;
- function size and source location;
- average, maximum, total, and count above threshold;
- stable file-and-symbol identities;
- increases, decreases, additions, removals, and net change against a baseline.

The regression policy fails when:

1. an existing function's complexity increases; or
2. a new function is introduced above the configured threshold.

A new small function does not fail merely because every function has a base complexity of one. Removed functions do not fail. This makes the policy practical for gradual improvement:

```bash
# Establish a baseline
girl complexity . --lang ts --output json \
  --write-baseline .girl/complexity-baseline.json

# Ratchet in CI or a pull request
girl complexity . --lang ts \
  --baseline .girl/complexity-baseline.json \
  --fail-on regression --output markdown
```

Anonymous callbacks use a deterministic ordinal inside their enclosing function. Moving or reordering callbacks can therefore appear as an add/remove pair; named callbacks provide more stable history.

## Limits and companion signals

Cyclomatic complexity does not measure nesting, naming, domain difficulty, coupling, data flow, or how hard code feels to a person. Early returns can make a function easier to read without reducing its cyclomatic score. Extracting helpers reduces per-function scores but may leave repository total complexity unchanged.

Research also shows why GIRL should not equate the metric with understandability:

- SonarSource introduced [Cognitive Complexity](https://www.sonarsource.com/resources/cognitive-complexity/) specifically to model human comprehension rather than path count.
- Hotspot research combines complexity with change history because stable complex code and frequently changed complex code carry different maintenance risk: Willenbring et al., [“Using Complexity Metrics with Hotspot Analysis”](https://ieeexplore.ieee.org/document/9985034/).

GIRL should eventually combine cyclomatic complexity with nesting, code churn, coverage, ownership, and defect history. This release deliberately keeps those signals separate so the meaning of each metric remains inspectable.
