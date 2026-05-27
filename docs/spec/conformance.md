# GRP Conformance

**Grammar Refactoring Protocol — Conformance Levels**

Version 0.1 — Core specification.

## Conformance Levels

| Level | Capability | Core/Binding |
|-------|-----------|--------------|
| 0 | Can read/write valid GRP JSON | Core |
| 1 | Emits structured diagnostics conforming to GRP diagnostic shape | Core |
| 2 | Emits deterministic plans with valid step IDs | Core |
| 3 | Emits token-budgeted context packs | Core |
| 4 | Emits dry-run patches for steps | Core |
| 5 | Runs verification and reports results | Core |

## Core vs Binding Conformance

- A tool can be Level 2 **Core** conformant and also implement the **GRP-Go binding**.
- **Core conformance** means a tool correctly implements the GRP Core spec (plan, diagnostic, step, verification, extensions, conformance).
- **Binding conformance** means a tool correctly implements a specific binding's diagnostic/recipe set (e.g., all GRP-Go diagnostics, all GRP-TypeScript diagnostics).
- A tool may be conformant at different levels for Core vs a given binding. E.g., "Level 3 Core, Level 1 GRP-Go"

## Claiming Conformance

A tool MAY self-certify its conformance level. The spec does not define a certification authority.

## Each Level Includes All Lower Levels

Level 3 implies Level 0, 1, and 2 conformance.
