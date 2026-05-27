# GIRL/GRP Roadmap

**Project board**: [github.com/orgs/canadian-ai/projects/6](https://github.com/orgs/canadian-ai/projects/6)
**Issues**: [github.com/canadian-ai/girl/issues](https://github.com/canadian-ai/girl/issues)

```

Phase                     | Status        | Timeline
--------------------------|---------------|-----------------
Initial scaffolding       | DONE          | May 18-20
Go self-hosting           | DONE          | May 20-21
Productionize core        | DONE          | May 22-23
Parser robustness         | DONE          | May 24
SARIF + packer + gov      | DONE          | May 25
Dogfood recursion         | DONE          | May 26
GRP Core v0.1             | IN PROGRESS   | May 26 - Jun 1
Bindings v0.1             | PLANNED       | Jun 2 - Jun 8
Context + Trust           | PLANNED       | Jun 9 - Jun 15
Production release        | PLANNED       | Jun 16+

```

## Completed

### May 18-20: Initial scaffolding

- Repo skeleton, CLI skeleton, TSX parser, node graph
- OpenCode skills, docs, roadmap
- GitHub repo created

### May 20-21: Go self-hosting

- Go analyzer via stdlib `go/parser` + `go/ast`
- Go diagnostics: long-function, high-complexity, deep-nesting, large-file, ignored-error, too-many-params
- Go recipes: extract-function, simplify-branches, flatten-nesting, split-file, handle-error, extract-options-struct
- Go verification: `go build`, `go vet`, `go test`
- Synthetic fixtures (`testdata/real/`)

### May 22-23: Productionize core

- **Structured diagnostics**: `ir.Diagnostic` extended with `Kind`, `Symbol`, `EndLine`, `Package`, `Span`, `Metadata`, `Related`, `Fixes`. Planner no longer parses message text. Diagnostic-target helper prefers symbol → component → file.
- **Recipe registry**: 14 mappings in `internal/recipes/diagnostics.go`. Planner calls `recipes.StepForDiagnostic(diag)` instead of a large switch. Adding new diagnostics no longer grows planner.
- **Stable step IDs**: `step_001_go.high-complexity_buildComponentFromBody` format. Deterministic after sorting.
- **Safer language detection**: `shared.ShouldSkipDir` covers `.git`, `.grp`, `node_modules`, `vendor`, `dist`, `build`, `.next`, `.turbo`. Go detected via `go.mod`.

### May 24: Parser robustness

- Split 855-line `parser.go` into `parser.go` (459 lines) + `component.go` (401 lines)
- 10 malformed-input tests: empty, unclosed JSX, unmatched braces, garbage, unterminated strings, deep nesting, non-ASCII — none panic

### May 25: SARIF + packer + governance

- **SARIF 2.1.0 exporter** (`internal/sarif/exporter.go`): level mapping, rule dedup, span fallback
- **Rich context packer**: diagnostic-range snippet selection, relative path privacy, `DiagnosticCounts` + `TopCodes` in `ContextPack`
- **Governance files**: CHANGELOG.md, CONTRIBUTING.md, SECURITY.md, CODEOWNERS

### May 26: Dogfood recursion

- Refactored 17 internal files across 7 packages: parser, packer, SARIF, goanalysis, verifier, command, planner, node, visitor, recipes
- 1130 insertions, 902 deletions
- GIRL self-analysis: **0 issues**
- Tests: 141/141 passed
- Dogfood: `girl analyze` → 0 issues, `girl plan` → empty (no unresolved diagnostics), `girl verify` → go/ts detection
- **All 7 original roadmap priorities complete**

## In Progress

### May 26 - Jun 1: GRP Core v0.1 (+15 issues)

GitHub project column: **GRP Core v0.1**

Protocol standard work that makes GRP a real specification:

- **Spec docs** (`spec/`): core plan envelope, diagnostic shape, step shape, verification shape, extension rules, conformance levels
- **JSON schemas** (`schemas/`): plan, diagnostic, step, verification
- **Example plans** (`examples/grp/`): minimal, Go, React
- **Public types** (`pkg/grp/`): Plan, Diagnostic, Step, Verification, Span, Symbol, Target, Execution
- **GRP normalization**: deterministic sorting, content-hash plan IDs, stable diagnostic IDs (`diag_001`), step `requires` linking
- **CLI**: `girl plan --output grp-json`, `girl validate`
- **Tests**: determinism, schema validation, round-trip

Issues: [#2](https://github.com/canadian-ai/girl/issues/2) - [#16](https://github.com/canadian-ai/girl/issues/16)

## Planned

### Jun 2 - Jun 8: Bindings v0.1 (+7 issues)

GitHub project column: **Bindings v0.1**

Language binding documentation that maps specific languages/frameworks into GRP:

- **GRP-Go**: diagnostics, recipes, verification docs (`bindings/go/`)
- **GRP-TypeScript**: diagnostics, recipes, verification docs (`bindings/typescript/`)
- **GRP-React**: diagnostics, recipes, verification docs (`bindings/react/`)
- **Future Tool Bindings**: GritQL, Tree-sitter, OpenRewrite, ESLint, SARIF, LSP (`docs/bindings/tool-bindings.md`)
- **Verification detection**: package-manager-aware (npm/pnpm/yarn/bun), package.json script discovery, confidence levels

Issues: [#17](https://github.com/canadian-ai/girl/issues/17) - [#22](https://github.com/canadian-ai/girl/issues/22), [#29](https://github.com/canadian-ai/girl/issues/29)

#### Binding names

- **GRP-Go** — diagnostics, recipes, verification docs for Go
- **GRP-TypeScript** — diagnostics, recipes, verification docs for TypeScript
- **GRP-React** — diagnostics, recipes, verification docs for React
- **Future Tool Bindings** — document how GritQL, Tree-sitter, OpenRewrite, ESLint, SARIF, LSP will map into GRP

GRP Core does not care how you parse code. GRP only cares that you can describe the refactor clearly.
Go uses `go/parser`/`go/ast`/`go/types`. TypeScript can use TS Compiler API, ts-morph, Tree-sitter, Babel/SWC.
React defines framework-specific diagnostics on top of TS analysis.

### Jun 9 - Jun 15: Context + Trust (+6 issues)

GitHub project column: **Context + Trust**

Production-hardening for reliable agent use:

- **Context pack GRP format**: `girl pack --output grp-context-json`
- **Privacy modes**: `--privacy private|redacted|public`
- **Budget tiers**: 4k, 8k, 16k, 32k snippet selection
- **CI**: GitHub Actions workflow
- **Golden tests**: deterministic GRP output fixtures
- **Dogfooding case study**: documented results

Issues: [#23](https://github.com/canadian-ai/girl/issues/23) - [#28](https://github.com/canadian-ai/girl/issues/28)

### Jun 16+: Production release

- False positive rate documentation and control
- SARIF export verified against real repos
- Context packs privacy-verified
- Dry-run patch mode
- Rollback metadata
- GitHub Action published
- Homebrew tap (optional)
- Public announcement

## Dogfooded Milestone

GIRL now analyzes and plans refactors for itself with **zero diagnostic findings**:

```bash
./girl analyze . --lang go --output text    # 0 issues
./girl plan . --lang go --output grp-json     # valid GRP, empty step list
./girl verify . --output text                 # go detection, no scripts
go build ./... && go vet ./... && go test ./...  # all pass
```

## Key Metric

| Metric | May 19 | May 26 | Target |
|--------|--------|--------|--------|
| GIRL self-diagnostics | 38 | 0 | 0 |
| Tests passing | 30 | 141 | 200+ |
| Go packages analyzed | 0 | 2 | 3 |
| TS/React packages analyzed | 2 | 2 | 3 |
| Language bindings documented | 0 | 0 | 3 (GRP-Go, GRP-TypeScript, GRP-React) |
| CI | none | none | green |
| GRP conformance level | 0 | 2 | 3 |
| Dogfooded | no | yes | continuous |
