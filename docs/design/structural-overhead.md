# Structural Overhead — Reviewability Dimension

**Design doc** · June 2026

## Problem

Current reviewability budget measures **surface** (diff lines, files touched) but not **structure** — whether the change is mostly logic, mostly scaffolding, or scattered across concerns.

Two 50-line diffs with identical file counts have very different reviewability:

| Diff | Content | Review overhead | Semantic signal |
|------|---------|----------------|-----------------|
| A | 50 lines of new business logic in one function | High — must understand semantics | High |
| B | 30 lines of test builder, 15 lines of config, 5 lines of logic | High — mostly infrastructure ceremony | Low |

Diff B has low semantic signal but high review overhead: reviewers must verify all the infrastructure is correctly wired before they can evaluate the 5 lines of logic. Uncle Bob's observation: *excessive test code means you're testing the production API wrong*. Extended: **excessive ceremonial scaffolding in a diff means the change surface is not independently reviewable**.

## Classification Buckets

Every hunk is classified into exactly one bucket. The buckets replace ad-hoc ratios with a flat taxonomy.

| Bucket | Counts toward overhead? | Counts as productive? | Examples |
|--------|:----------------------:|:---------------------:|----------|
| `logic` | No | Yes | Behavior change, control flow, business logic, data transformations |
| `test` | No | Neutral | Test assertions, test cases, test data, test setup |
| `reusable_support` | No | Yes | Builder functions, factory helpers, shared mocks, exported API wrappers, test utilities |
| `ephemeral_support` | Yes | No | One-off wiring, interface stubs, inline mocks, import additions, registration calls |
| `config_data` | No | Neutral | CSS variables, static JSON, package metadata entries, feature flag values |
| `config_structural` | Yes | No | CI workflows, build scripts, `tsconfig` behavior changes, `go.mod` dependency changes, DB schema |
| `generated` | Excluded | Excluded | Vendored code, `node_modules`, protobuf stubs, lockfiles |

### Classification rules

**File path (primary):** cheap, high signal.

| Pattern | Bucket |
|---------|--------|
| `*_test.go`, `*_test.ts`, `*.spec.*`, `test/`, `__tests__/` | `test` |
| `*.json` (except `tsconfig`/`.babelrc`/config), `*.css`, `*.scss` | `config_data` |
| `.github/`, `Makefile`, `Dockerfile`, `tsconfig.*`, `.babelrc.*`, `go.mod` | `config_structural` |
| `vendor/`, `node_modules/`, `gen/`, `*.pb.go`, `*.pb.ts`, lockfiles | `generated` |

**Line content (secondary):** for same-file mixed hunks.

| Pattern | Bucket |
|---------|--------|
| `import`, `require`, `using` directives | `ephemeral_support` if single-use; `logic` if part of new dependency injection |
| Builder function, exported helper, shared mock constructor | `reusable_support` |
| Inline mock, one-off test data struct, setup block scoped to one test | `ephemeral_support` |

**Ephemeral vs. reusable heuristic:** a new function/type in a `_test.go` file that is referenced from multiple hunks or exported → `reusable_support`. A setup block or inline mock inside a single test function → `ephemeral_support`.

## Metrics

All ratios use **added lines** as the denominator. Added lines are stable across hunk boundaries and simpler than hunks or changed lines.

### 1. `structural_overhead_ratio`

What fraction of the change is unreviewable ceremony?

```
(ephemeral_support_added + config_structural_added)
/
max(1, logic_added + reusable_support_added + ephemeral_support_added + config_structural_added)
```

Denominator excludes `test`, `config_data`, `generated` — test is the validation layer, static data has negligible review cost, generated is excluded.

### 2. `test_to_logic_ratio`

Does the diff carry its test weight or is it testing the wrong abstraction?

```
test_added
/
max(1, logic_added + reusable_support_added)
```

`reusable_support` is logic-adjacent: if test pressure forced a builder API, that is productive, not noise.

### 3. `productive_scaffold_ratio`

How much of the scaffold is reusable vs. ephemeral?

```
reusable_support_added
/
max(1, reusable_support_added + ephemeral_support_added)
```

High ratio → tests are driving good API design. Low ratio → ceremony sludge.

### 4. `cohesion_variance`

Are changes concentrated in one concern or scattered?

Jaccard distance over normalized path segment tokens:

- `internal/server/handler.go` → `{internal, server, handler}`
- `internal/db/schema.sql` → `{internal, db, schema}`
- Distance = 1 - (common / union) = 1 - (1/5) = 0.8

Mean pairwise Jaccard distance across all touched files. 0 = all files share the same path prefix. 1 = maximally scattered.

When variance > 0.6, the analyser emits `suggested_clusters` — path-topic groups the diff naturally splits into, computed by hierarchical clustering on path tokens.

## Diagnostics

| Namespace | Level | Trigger |
|-----------|-------|---------|
| `agent.high-overhead` | WARN | `structural_overhead_ratio > 0.5` |
| `agent.low-cohesion` | WARN | `cohesion_variance > 0.6` |
| `agent.test-to-code-imbalance` | WARN | `test_to_logic_ratio > 3.0` AND `ephemeral > reusable` |
| `agent.ceremonial-noise` | HIGH | `high-overhead` AND `low-cohesion` simultaneously |
| `agent.productive-scaffold` | INFO | `productive_scaffold_ratio > 0.5` AND `reusable_support_added >= 20` |
| `agent.repeated-boilerplate` | WARN | (deferred — requires corpus analysis to calibrate) |

## Risk Integration

Structural metrics adjust the base risk from the surface budget:

| Condition | Adjustment |
|-----------|------------|
| `structural_overhead_ratio > 0.5` | +1 level (e.g. medium → high) |
| `cohesion_variance > 0.6` | +1 level |
| `test_to_logic_ratio > 3.0` AND `ephemeral > reusable` | +1 level |
| `productive_scaffold_ratio > 0.5` AND `reusable_support_added >= 20` | -1 level (productive) |

When `cohesion_variance > 0.6`, the decomposer receives suggested split boundaries from path clustering.

## Key Decisions

1. **Added lines are the denominator for all ratios.** Not hunks, not changed lines. Simplest stable unit.
2. **Ephemeral vs. reusable uses reference-count heuristic.** Multi-hunk or exported → reusable. Single-setup → ephemeral. No temporal decay needed for v0.
3. **Config is split into data and structural.** Static data has negligible review cost; build/CI/Db schema changes do not.
4. **Structural field lives under `extensions.agent.structural`**, not GRP Core v0.1. Promoted after ≥2 dogfood runs confirm shape stability.
5. **Cohesion variance drives decomposition suggestions, not failure.** Only ceremonial-noise (overhead + scattered) triggers HIGH.
6. **Thresholds are defaults, not constants.**

## Output Shape

```json
{
  "extensions": {
    "agent.structural": {
      "added": {
        "logic": 18,
        "test": 42,
        "reusable_support": 25,
        "ephemeral_support": 15,
        "config_data": 6,
        "config_structural": 4,
        "generated": 0
      },
      "ratios": {
        "structural_overhead": 0.24,
        "test_to_logic": 0.98,
        "productive_scaffold": 0.62
      },
      "cohesion": {
        "variance": 0.4,
        "suggested_clusters": []
      }
    }
  }
}
```

## Next

- Prototype hunk classifier as `internal/structural/classifier.go`
- Add structural test fixtures with known bucket splits
- Implement cohesion variance as Jaccard path-topic distance
- Implement `suggested_clusters` via single-linkage path clustering
- Dogfood on the reviewability PR diff (3500 insertions, 35 files)
- After 2+ dogfood runs, promote `extensions.agent.structural` to `reviewability.structural` if shape stabilizes
