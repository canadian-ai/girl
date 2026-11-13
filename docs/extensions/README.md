# Governed extensions for GIRL (experimental)

This incremental layer represents findings from existing analyzers as bounded,
auditable action proposals. It is not an executable plugin loader.

- `Finding`: source provider, external ID, rule, canonical relative location,
  severity and deterministic fingerprint.
- `Action`: proposed GRP recipe, touched paths, changed-line budget and risk.
- `Policy`: explicit allowed paths, change budgets and review-required categories.
- `Evaluate`: denies missing policies, traversal, unapproved paths and budget
  overruns; requests review for critical or security-sensitive changes.

## Cloud agent integration model

1. Analyze using a read-only SonarQube connector or an isolated Ruff/Clippy runner.
2. Normalize source findings, keeping the provider ID and rule.
3. Produce a small, explicit, versioned action using a deterministic recipe when possible.
4. Apply policy outside the agent's prompt. Neither a finding nor an agent can grant access.
5. Apply only approved actions in an ephemeral worktree, then run tests and gates.
6. Record baseline commit, changed paths, verification results and source issue IDs.
7. Mark resolved only after a fresh scan by the authoritative provider.

Use the existing experimental Python and Rust analyzers; do not fork them.
A future `girl grill` mode should ask one question at a time for ambiguous or
high-risk refactors, producing reviewable decision records.

## Current scope

The Go contracts and unit tests above are implemented. SonarQube API ingestion,
Ruff/Clippy subprocesses, interactive grill CLI, recipe executor, CI enforcement
and verified cloud-agent execution are NOT yet implemented.

Measure success using issues resolved by authoritative rescans, regression rate,
agent tokens per verified fix and approval overhead.
