# GRP-CAI Binding

## Overview

The GRP-CAI binding maps GIRL's Grammar Refactoring Protocol (GRP) to the
Canadian AI (CAI) agent-safety framework. This enables policy-compliant,
reviewable, and gated AI-assisted development.

## Concepts

### SIGIL (Safety Integrity Guardrails and Instrumentation Layer)

SIGIL is the metadata layer that tracks safety posture across the development
lifecycle. Every agent change carries a SIGIL manifest.

| GRP Concept | SIGIL Mapping |
|-------------|---------------|
| plan.extensions | sigil.json manifest |
| reviewability | sigil.review |
| verification | sigil.gates |
| decomposition | sigil.workorders |

### Tenancy

Tenancy defines the deployment context and isolation boundaries for agent
changes. It maps to GRP context:

```json
{
  "grp": {
    "extensions": {
      "cai.tenancy": {
        "environment": "development",
        "isolation": "project-local",
        "deploymentGates": ["preflight", "review", "launchkit", "prove"]
      }
    }
  }
}
```

### Launch Kit

A launch kit is a collection of quality gates that must pass before
deployment. Each gate maps to a GRP verification step:

| Gate | GRP Mapping | Command |
|------|-------------|---------|
| preflight | verification step | `girl preflight` |
| review | reviewability budget | `girl review --stdin --output sigil-json` |
| launchkit_validate | self-validation | `girl launchkit validate` |
| prove_app | readiness check | `girl prove-app` |

### Deployment Gates

Deployment gates are GRP verification steps with required=true that gate
the merge/deploy pipeline. They are serialized as GRP verification entries.

## CLI Integration

| Command | CAI Concept | Output Format |
|---------|-------------|---------------|
| `girl preflight` | Readiness check | JSON, text, markdown |
| `girl review --output sigil-json` | SIGIL review | SIGIL JSON |
| `girl launchkit validate` | Gate validation | JSON, text, markdown |
| `girl receipt` | Change receipt | JSON, text, markdown |
| `girl prove-app` | Deployment readiness | JSON, text, markdown |
| `girl workorder` | Agent task spec | JSON, markdown |
| `girl install cai` | Project scaffold | Files on disk |

## Go Types

The `pkg/cai` package provides Go types for all CAI concepts:

- `SigilManifest` — SIGIL safety manifest
- `TenancyConfig` — Deployment tenancy configuration
- `LaunchKit` — Launch kit with quality gates
- `DeploymentGate` — Individual deployment gate
- `WorkOrder` — Agent-ready task specification
- `AgentReceipt` — Agent-change receipt
- `ProveAppResult` — Deployment readiness result
- `PreflightResult` — Preflight check result

## Security

- All CAI analysis runs locally. No code leaves the machine.
- SIGIL manifests do not contain secrets or credentials.
- Receipts include content hashes for integrity verification.
- Tenancy configurations must not contain access keys or tokens.
