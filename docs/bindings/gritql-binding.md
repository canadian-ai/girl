# GritQL Binding (Draft)

## Vision

GritQL as an optional query/rewrite execution binding for GRP.

## Design

- GRP provides the plan + verification + context envelope
- GritQL provides query matching and rewrite execution
- step.execution.engine = "gritql" indicates GritQL-powered steps
- step.execution.config contains GritQL-specific config

## Example

```json
{
  "id": "step_001_react.large-component_dashboard",
  "recipe": "react.split-large-component",
  "execution": {
    "mode": "gritql",
    "config": {
      "pattern": "`function $name(props) { ... }`",
      "rewrite": "split_component.grit"
    }
  }
}
```

## Non-goals for v0.1

- Shipping GritQL as a dependency
- GritQL runtime
- Pattern auto-generation
