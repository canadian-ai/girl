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
- **Future Tool Bindings** — GritQL, Tree-sitter, OpenRewrite, ESLint, SARIF, LSP

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

| # | Title | Milestone |
|---|-------|-----------|
| 2 | Add GRP Core v0.1 spec skeleton | GRP Core v0.1 |
| 3 | Add GRP diagnostics spec | GRP Core v0.1 |
| 4 | Add GRP steps spec | GRP Core v0.1 |
| 5 | Add GRP verification spec | GRP Core v0.1 |
| 6 | Add extensions and conformance specs | GRP Core v0.1 |
| 7 | Add GRP JSON schemas | GRP Core v0.1 |
| 8 | Add example GRP plans | GRP Core v0.1 |
| 9 | Add pkg/grp protocol types | GRP Core v0.1 |
| 10 | Add pkg/grp structural validation | GRP Core v0.1 |
| 11 | Add deterministic GRP normalization | GRP Core v0.1 |
| 12 | Add girl plan --output grp-json | GRP Core v0.1 |
| 13 | Replace timestamp plan IDs with content-derived IDs | GRP Core v0.1 |
| 14 | Add girl validate command | GRP Core v0.1 |
| 15 | Add GRP validation and determinism tests | GRP Core v0.1 |
| 16 | Update README with GRP vs GIRL product boundary | GRP Core v0.1 |
| 17 | Document Go binding diagnostics | Bindings v0.1 |
| 18 | Document Go binding recipes | Bindings v0.1 |
| 19 | Document Go binding verification | Bindings v0.1 |
| 20 | Document TypeScript binding diagnostics and recipes | Bindings v0.1 |
| 21 | Document React binding diagnostics and recipes | Bindings v0.1 |
| 22 | Improve package-manager and script-based verification detection | Bindings v0.1 |
| 29 | Document future tool binding model | Bindings v0.1 |
| 23 | Add grp-context-json output for girl pack | Context + Trust |
| 24 | Add context pack privacy modes | Context + Trust |
| 25 | Add budget-aware snippet selection tiers | Context + Trust |
| 26 | Add GitHub Actions CI | Context + Trust |
| 27 | Add golden GRP plan tests | Context + Trust |
| 28 | Add dogfooding case study | Context + Trust |

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
