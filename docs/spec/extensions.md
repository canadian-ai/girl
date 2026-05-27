# GRP Extensions

**Grammar Refactoring Protocol — Extension System**

Version 0.1 — Core specification.

## Extension Rules

1. **GRP consumers MUST ignore unknown extensions** unless listed in `requiredExtensions`.

2. **Bindings** use namespaced diagnostic/recipe codes like `go.high-complexity`, `react.large-component`.

3. **The `extensions` field** on a Plan is a JSON object (map) for binding/tool-specific metadata.

4. **The `requiredExtensions` field** is an array of extension key strings. Consumers that do not support a listed extension MUST reject the plan with a clear error.

5. **Core GRP validation must pass** for any plan, regardless of which binding extensions are present.

6. **Binding extensions** may carry any JSON value (objects, arrays, strings, numbers, booleans).

7. **Extension naming convention**: tool or binding namespaced, e.g. `girl.analysis`, `grp-go.metrics`.

## Example

```json
{
  "extensions": {
    "girl.analysis": {
      "analyzerVersion": "0.1.0",
      "parser": "go/ast"
    },
    "grp-go.metrics": {
      "totalFunctions": 42,
      "avgComplexity": 8.3
    }
  },
  "requiredExtensions": ["girl.analysis"]
}
```

## Plan Rejection Rule

If `requiredExtensions` contains `"some-binding"` and a consumer does not support it, the consumer must reject the plan. If `requiredExtensions` is absent or empty, no extension is required.
