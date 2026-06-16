# agent.structural — Structural Overhead Extension

**Extension key:** `agent.structural`
**Namespace:** `agent.*`
**Version:** v0.1-draft

## Purpose

Augment reviewability budget with structural awareness: detect diffs heavy on
ceremonial scaffolding, scattered across unrelated concerns, or carrying an
unreviewable test-to-code ratio. Provides a structured breakdown of hunk
classifications, ratios, and cohesion metrics for tools and reviewers.

## Extension Rules

- **GRP consumers MUST ignore this extension** unless `agent.structural` is
  listed in `requiredExtensions`.
- **Core GRP validation must pass** regardless of whether this extension is
  present.
- **All fields are optional.** Consumers may emit a subset of the shape.
  Missing fields imply the analysis was unavailable, not zero.

## Extension Value Shape

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

## Fields

### `added`
Line counts per classification bucket (added lines only, not changed or
deleted). Present when a diff was analyzed. Omitted if no diff was provided.

| Field | Type | Description |
|-------|------|-------------|
| `logic` | int | Behavior change, control flow, business logic |
| `test` | int | Test assertions, test cases, test data |
| `reusable_support` | int | Builders, factories, shared helpers, exported test utilities |
| `ephemeral_support` | int | One-off wiring, inline mocks, scoped setup |
| `config_data` | int | Static values, CSS, package metadata |
| `config_structural` | int | CI, build scripts, schema, dependency changes |
| `generated` | int | Vended code, protobuf stubs, lockfiles |

### `ratios`
Computed ratios. Present when at least one of `logic`, `test`,
`reusable_support`, or `ephemeral_support` is non-zero.

| Field | Type | Description |
|-------|------|-------------|
| `structural_overhead` | float | (ephemeral_support + config_structural) / (all non-test, non-config-data, non-generated added). Range [0, 1]. |
| `test_to_logic` | float | test / (logic + reusable_support). Range [0, ∞). |
| `productive_scaffold` | float | reusable_support / (reusable_support + ephemeral_support). Range [0, 1]. 0 when denominator is 0. |

### `cohesion`
Topic cohesion across touched files. Present when ≥2 files are touched.

| Field | Type | Description |
|-------|------|-------------|
| `variance` | float | Mean pairwise Jaccard distance over normalized path segment tokens. Range [0, 1]. |
| `suggested_clusters` | array | List of path-topic groups. Each cluster is an array of file paths. Present when `variance > 0.6`. |

## Diagnostics (from this extension)

These diagnostics MAY be emitted as top-level plan diagnostics when the
extension is present. See [design doc](../design/structural-overhead.md) for
trigger thresholds.

| Namespace | Level | Trigger |
|-----------|-------|---------|
| `agent.high-overhead` | WARN | `structural_overhead_ratio > 0.5` |
| `agent.low-cohesion` | WARN | `cohesion_variance > 0.6` |
| `agent.test-to-code-imbalance` | WARN | `test_to_logic_ratio > 3.0` AND `ephemeral > reusable` |
| `agent.ceremonial-noise` | HIGH | `high-overhead` AND `low-cohesion` |
| `agent.productive-scaffold` | INFO | `productive_scaffold_ratio > 0.5` AND `reusable_support >= 20` |

## Classification Algorithm

### File-path patterns (primary)

| Pattern | Bucket |
|---------|--------|
| `*_test.go`, `*_test.ts`, `*.spec.*`, `test/`, `__tests__/` | `test` |
| `*.json` (exclude config: `tsconfig*`, `.babelrc*`), `*.css`, `*.scss` | `config_data` |
| `tsconfig*`, `.babelrc*`, `.browserslistrc`, `.eslintrc*`, `.prettierrc*`, `Makefile`, `Dockerfile`, `.github/`, `.gitlab-ci.yml`, `go.mod`, `go.sum`, `Gemfile`, `Gemfile.lock`, `Package.resolved` | `config_structural` |
| `vendor/`, `node_modules/`, `gen/`, `*.pb.go`, `*.pb.ts`, `*.pb.swift`, `yarn.lock`, `package-lock.json`, `Cargo.lock` | `generated` |

### Line-content heuristics (secondary, same-file hunks)

| Pattern | Bucket |
|---------|--------|
| Lines consisting only of `import`, `require`, `using`, `#include` directives | `ephemeral_support` |
| Function/type definitions that are exported or referenced from ≥2 hunks or files | `reusable_support` |
| Test functions, `it(...)`, `describe(...)`, `t.Run(...)`, `t.Log(...)`, `assert.*`, `expect(...)` | `test` |

When both file-path and line-content apply, file-path wins (e.g. everything in
a `*_test.go` is `test` — even inline mocks are counted as test, not support).

## Example Plan with Extension

```json
{
  "specversion": "0.1.0",
  "type": "grp.refactor",
  "subject": "reviewability",
  "goal": "Add structural overhead detection to reviewability",
  "diagnostics": [
    {
      "code": "agent.high-overhead",
      "severity": "warn",
      "confidence": "high",
      "message": "Structural overhead ratio is 0.24 (budget: 0.5)"
    }
  ],
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
