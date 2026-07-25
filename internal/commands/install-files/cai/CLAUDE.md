# CAI Agent Safety Requirements

You are a CAI (Coding Agent Interface) agent. You MUST follow these safety rules.

## Mandatory Preflight Checks

Before making ANY changes to the codebase, run a preflight check:

```
girl preflight
```

This validates the tenancy, risk level, and gate readiness for the current project context.

## SIGIL Manifest

Every project MUST have a `sigil.json` manifest at the project root. The SIGIL (Safety Integrity Guardrails and Instrumentation Layer) manifest declares:

- Project identity and tenancy
- Risk level (low/medium/high)
- Required gates (preflight, review, receipt, prove)

The SIGIL manifest is generated during project setup via `girl install cai`. Update the `sigil.json` file directly or re-run:

```
girl install cai
```

## Launch Kit Validation Gates

Before any deployment or significant change, validate the launch kit:

```
girl launchkit validate
```

The launch kit defines the quality gates that must pass:

| Gate                  | Required | Command                           |
|-----------------------|----------|-----------------------------------|
| preflight             | yes      | `girl preflight`                  |
| review                | yes      | `girl review --stdin --output sigil-json` |
| launchkit_validate    | yes      | `girl launchkit validate`         |
| prove_app             | yes      | `girl prove-app`                  |

## Receipt Generation

After every change, generate a receipt to record what was done:

```
girl receipt
```

Receipts provide audit trail and reviewability for all modifications.

## Tenancy-Aware Development

All development MUST respect the tenancy configuration (`.cai/tenancy.yaml`):

- **environment**: The deployment context (development/staging/production)
- **isolation**: The boundary level (project-local/team-wide/organization-wide)
- **deployment_gates**: The gates that must pass before deployment

Run `girl preflight` to check current tenancy context.

## Available Commands

- `girl workorder` - Generate agent-ready task work orders from a plan or decomposition
- `girl preflight` - Run CAI repo readiness checks
- `girl receipt --stdin` - Generate a change receipt from a diff
- `girl launchkit validate` - Validate launch kit gates
- `girl prove-app` - Prove application readiness
