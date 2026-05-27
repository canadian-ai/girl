# Future Tool Binding Model

GRP Core does not care how you parse code. GRP Core only cares that you can describe the refactor clearly. Bindings map tool-specific findings into GRP diagnostics and steps.

## How a tool binding works

1. **Analyze input** — The tool (e.g. ESLint, GritQL, Tree-sitter) processes source code
2. **Map findings to diagnostics** — Each tool-specific finding is mapped to a GRP `Diagnostic` with:
   - `code`: namespaced as `tool.<tool-name>.<diagnostic-code>`
   - `severity`, `message`, `file`, `line`, `span`
3. **Map operations to steps** — Each tool-specific refactor operation is mapped to a GRP `Step` with:
   - `recipe`: namespaced as `tool.<tool-name>.<recipe-name>`
   - `action`, `file`, `risk`, `verify`

## Extension naming convention

- Diagnostics: `tool.<tool-name>.<diagnostic-id>`
- Recipes: `tool.<tool-name>.<recipe-name>`

## Conformance expectations

A conformant tool binding must:
- Produce valid GRP diagnostics
- Produce valid GRP steps when refactoring
- Document its supported diagnostic codes and recipes
- Version itself independently of GRP Core

## Implementation priority

| Priority | Binding | Notes |
|----------|---------|-------|
| P0 | GRP-Go | Already implemented |
| P1 | GRP-TypeScript | Suggested diagnostics defined in `bindings/typescript/` |
| P2 | GRP-React | Suggested diagnostics defined in `bindings/react/` |
| P3 | ESLint binding | Map existing ESLint rules to GRP diagnostics |
| P4 | GritQL binding | Map Grit patterns to GRP diagnostics/steps |
| P5 | Tree-sitter binding | Map grammar queries to GRP diagnostics |
| P6 | SARIF binding | Ingest SARIF output as GRP diagnostics |
| P7 | LSP binding | Live LSP diagnostics → GRP diagnostics |
| P8 | OpenRewrite binding | Map OpenRewrite recipes to GRP steps |

## Future tool binding design docs

Each tool binding should have a one-page design doc that documents:

- Binding name and version
- Supported diagnostic codes and their GRP equivalents
- Supported recipes and their GRP equivalents
- Verification commands
- Known limitations
- False positive risks

Design docs live in `docs/bindings/<tool-name>.md`.
