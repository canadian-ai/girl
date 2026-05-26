# Changelog

All notable changes to GIRL (Grammar-Informed Refactoring Language) are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Structured diagnostics: Kind, Symbol, Span, EndLine, Package, Metadata, Tags, Related, Fixes on Diagnostic
- DiagnosticTarget() helper that prefers Symbol then Component then File
- Recipe registry: 14 diagnostic-code-to-recipe mappings (8 React + 6 Go)
- Deterministic GRP step IDs with ordinal + recipe + slug format
- Shared directory skip policy (internal/shared): .git, .grp, node_modules, vendor, dist, build, .next
- SARIF 2.1.0 exporter (internal/sarif)
- Malformed input test suite (10 cases) — no panics on bad input
- Component parser split: parser.go (459 lines) + component.go (401 lines)
- Expanded test coverage: ir, analyzer, goanalysis, planner, recipes, shared, parsertsx, packer, sarif
- Safer language detection with --lang auto detection improvements

### Fixed
- Planner no longer parses diagnostic messages to find targets
- extractTarget() message-parsing hack removed
- Parser robustness: no crashes on empty, malformed, deeply nested input
- Error handling hygiene: ignored errors in verifier, packer, plan command identified

### Changed
- Planner generateStepsFromDiagnostics uses recipe registry instead of switch
- assignStepIDs now produces deterministic unique IDs
- Go analyzer walk uses shared.ShouldSkipDir
- TSX parser walk uses shared.ShouldSkipDir
- Underlying types improved but existing CLI output format preserved
