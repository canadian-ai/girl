# GRP-CAI Binding

## Overview

The GRP-CAI binding maps GIRL's Grammar Refactoring Protocol (GRP) to the
Canadian AI (CAI) agent-safety framework. This enables policy-compliant,
reviewable, and gated AI-assisted development.

GRP-CAI is a safety and governance layer that integrates with existing tools.
It does **not** replace:

- **SIGIL**: SIGIL remains the canonical safety manifest format. GRP-CAI maps
  SIGIL concepts into the GRP plan lifecycle for agent consumption.
- **Clerk**: Authentication and user management remain handled by Clerk.
  GRP-CAI does not duplicate auth middleware.
- **Convex**: Convex remains the backend and real-time data layer. GRP-CAI
  adds preflight checks that verify Convex configuration.
- **Vercel**: Deployment and hosting remain on Vercel. GRP-CAI adds deployment
  gate validation that runs before Vercel deploys.
- **Deployment tooling**: CI/CD pipelines (GitHub Actions, etc.) remain the
  deployment orchestrator. GRP-CAI provides review artifacts and receipts
  that feed into these pipelines.

## Diagnostic Catalogue

CAI-specific diagnostic codes used in GRP `diagnostics[]` entries:

| Code | Severity | Confidence | Description | Example |
|------|----------|------------|-------------|---------|
| CAI-PREFLIGHT-001 | high | high | Preflight check not configured | `.cai/preflight.yaml` missing |
| CAI-PREFLIGHT-002 | medium | medium | Tenancy isolation not set | `.cai/tenancy.yaml` missing isolation field |
| CAI-SIGIL-001 | high | high | SIGIL manifest not found | `sigil.json` missing from project root or `.cai/` |
| CAI-SIGIL-002 | medium | medium | SIGIL gates not fully enabled | One or more gates set to `false` |
| CAI-LAUNCHKIT-001 | high | high | Launch kit gate missing | Required gate not configured |
| CAI-LAUNCHKIT-002 | medium | medium | Launch kit version missing | `launchkit.yaml` has no version field |
| CAI-RECEIPT-001 | low | medium | No receipt generated for change | Change was made without `girl receipt` |
| CAI-REVIEW-001 | medium | high | Review exceeds budget | Diff exceeds reviewability budget thresholds |
| CAI-PROVE-001 | high | medium | Build check failed | Application does not compile |
| CAI-PROVE-002 | high | medium | Test check failed | Tests are not passing |
| CAI-TENANCY-001 | high | high | Environment mismatch | Change targets production without staging verification |
| CAI-AGENT-001 | medium | medium | Agent instruction file missing | No `AGENTS.md` or `CLAUDE.md` found |

## Recipe Types

GRP recipes that map to CAI operations:

| Recipe | CAI Mapping | Description |
|--------|-------------|-------------|
| `cai.preflight.init` | Initialization | Scaffold preflight configuration for a new project |
| `cai.sigil.generate` | Generation | Create or update SIGIL manifest |
| `cai.launchkit.validate` | Validation | Run launch kit quality gate validation |
| `cai.receipt.generate` | Auditing | Generate agent-change receipt after modification |
| `cai.review.check` | Reviewability | Check diff reviewability against CAI budget |
| `cai.prove.readiness` | Verification | Verify application deployment readiness |

## Sample GRP Plan with CAI Extensions

```json
{
  "specversion": "0.1",
  "id": "grp_cai_example_001",
  "type": "dev.refactor.plan",
  "source": "github.com/canadian-ai/girl",
  "subject": "examples/cai-project",
  "language": "typescript",
  "goal": "Add authentication middleware with CAI safety gates",
  "risk": "medium",
  "diagnostics": [
    {
      "id": "diag_0",
      "code": "CAI-PREFLIGHT-001",
      "severity": "medium",
      "confidence": "high",
      "message": "Preflight check not configured for auth change",
      "file": ".cai/preflight.yaml"
    }
  ],
  "steps": [
    {
      "id": "step_0",
      "recipe": "cai.preflight.init",
      "title": "Initialize CAI preflight configuration",
      "action": "Create .cai/preflight.yaml with default checks",
      "target": { "file": ".cai/preflight.yaml" },
      "risk": "low",
      "requires": ["diag_0"],
      "verify": [
        {
          "command": "girl preflight --profile cai-next-convex",
          "required": true,
          "source": "cai-binding",
          "confidence": "high"
        }
      ]
    },
    {
      "id": "step_1",
      "recipe": "cai.sigil.generate",
      "title": "Generate SIGIL safety manifest",
      "action": "Create sigil.json with project identity and gates",
      "target": { "file": "sigil.json" },
      "risk": "low"
    }
  ],
  "verification": [
    {
      "command": "girl preflight --strict",
      "required": true,
      "source": "cai-binding",
      "confidence": "high"
    },
    {
      "command": "girl review --stdin --output sigil-json",
      "required": true,
      "source": "cai-binding",
      "confidence": "high"
    }
  ],
  "extensions": {
    "cai": {
      "tenancy": {
        "environment": "development",
        "isolation": "project-local",
        "deploymentGates": ["preflight", "review", "launchkit", "prove"]
      },
      "sigil": {
        "version": "0.1",
        "gates": {
          "preflight": true,
          "review": true,
          "receipt": true,
          "prove": true
        }
      },
      "launchKit": {
        "version": "0.1",
        "gates": {
          "preflight": { "required": true, "command": "girl preflight" },
          "review": { "required": true, "command": "girl review --stdin --output sigil-json" },
          "launchkit_validate": { "required": true, "command": "girl launchkit validate" },
          "prove_app": { "required": true, "command": "girl prove-app" }
        }
      }
    }
  },
  "requiredExtensions": ["cai"],
  "artifacts": [".grp/plan.json", ".grp/receipt.json"]
}
```

## Validation

GRP-CAI plans can be validated using the standard GRP schema
(`schemas/grp-plan.v0.1.schema.json`). The `extensions.cai` field accepts any
valid JSON object, so no additional schema is required for the CAI extension.
However, for stricter validation, a dedicated
`schemas/cai-plan.v0.1.schema.json` may be added in a future release.

You can validate CAI-enhanced plans with:
```bash
girl validate .grp/plan.json
```

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
