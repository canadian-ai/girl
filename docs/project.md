# GIRL/GRP Project

**GitHub Project Board**: [github.com/orgs/canadian-ai/projects/6](https://github.com/orgs/canadian-ai/projects/6)
**Issues**: [github.com/canadian-ai/girl/issues](https://github.com/canadian-ai/girl/issues)

## Milestones

### GRP Core v0.1
GitHub Project column: **GRP Core v0.1**

The foundational protocol standard. All issues prefixed with GRP protocol work:
- Spec docs (`spec/`) — core, diagnostics, steps, verification, extensions, conformance
- JSON schemas (`schemas/`) — plan, diagnostic, step, verification
- Example plans (`examples/grp/`) — minimal, Go, React
- Public types (`pkg/grp/`) — Plan, Diagnostic, Step, Verification, normalization, validation
- CLI — `girl plan --output grp-json`, `girl validate`, deterministic IDs
- README — GRP vs GIRL boundary

Issues: `#2`–`#16`

### Bindings v0.1
GitHub Project column: **Bindings v0.1**

Language binding documentation that maps specific languages/frameworks into GRP:

- **GRP-Go** — Go diagnostics, recipes, verification (`bindings/go/`)
- **GRP-TypeScript** — TypeScript diagnostics, recipes, verification (`bindings/typescript/`)
- **GRP-React** — React diagnostics, recipes, verification (`bindings/react/`)
- Verification detection improvements (package manager, script discovery)
- **Future Tool Bindings** — GritQL, OpenRewrite, ESLint, SARIF, LSP

Issues: `#17`–`#22`

### Context + Trust
GitHub Project column: **Context + Trust**

Production-hardening features that make GIRL reliable in real workflows:
- `girl pack --output grp-context-json`
- Privacy modes (`--privacy private|redacted|public`)
- Budget-aware snippet selection tiers
- GitHub Actions CI
- Golden GRP plan tests
- Dogfooding case study

Issues: `#23`–`#28`

## Issue Index

All items completed. Remaining work tracked via Production release milestone.

## Implementation Notes

- **Parsing**: Go analysis uses `go/ast` + `go/types`. TypeScript/React/JS
  analysis uses tree-sitter (`github.com/smacker/go-tree-sitter`) with sitter
  query patterns for component/hook/state/JSX detection.
- **Token estimation**: Heuristic `len(text)/3`. The `tokens.Estimator`
  interface in `internal/tokens/` allows swapping in a real tokenizer.
- **Recipe thresholds**: Configured in Go code via `internal/recipes.Thresholds`
  and `DefaultThresholds()`. No YAML files.

| # | Title | Milestone | Status |
|---|-------|-----------|--------|
| 2-16 | GRP Core v0.1 (spec, schemas, types, CLI, tests) | GRP Core v0.1 | Done |
| 17-22, 29 | Bindings v0.1 (Go, TS, React docs + verification) | Bindings v0.1 | Done |
| 23-28 | Context + Trust (pack, privacy, budget, CI, goldens, case study) | Context + Trust | Done |

## Labels

- `spec` — GRP protocol specification documents
- `schema` — JSON schema files
- `grp` — GRP protocol implementation
- `cli` — GIRL CLI commands and flags
- `docs` — Documentation, README, binding docs
- `bindings` — Language binding docs (Go, TypeScript, React)
- `context-pack` — Context packing and snippet selection
- `verification` — Command detection and verification
- `testing` — Tests, CI, golden fixtures
