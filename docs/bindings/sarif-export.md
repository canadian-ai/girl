# GRP to SARIF Export

## Motivation

SARIF is a standard format for static analysis results. Supporting SARIF export enables:
- GitHub Code Scanning integration
- VS Code SARIF viewer
- CI pipeline ingestion
- Cross-tool result aggregation

## Mapping

| GRP Field | SARIF Field |
|-----------|-------------|
| diagnostic.id | result.ruleId |
| diagnostic.code | result.message.text |
| diagnostic.severity | result.level (error/warning/note) |
| diagnostic.file | result.locations[0].physicalLocation.artifactLocation.uri |
| diagnostic.span | result.locations[0].physicalLocation.region |
| step | result.fixes[0] |

## Severity mapping

- high → error
- medium → warning
- low → note

## Non-goals for v0.1

- Full SARIF run/tool/artifact objects
- ThreadFlow locations
- Code flow
- Attachments
