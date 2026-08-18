# GRP Core Reduction & Collection

**Grammar Refactoring Protocol — Software Graph Garbage Collection Runtime**

Version 0.1 — Core specification.

## Framing

GRP Core extends from plan generation into a safe maintenance substrate: it
can identify redundant, duplicate, obsolete, and dead software graph nodes,
migrate references to canonical equivalents, verify behavior, and only then
propose collection of the obsolete implementations.

This is **plan-first and non-destructive**. GRP Core only ever emits a *plan*
describing reduction/collection work. Nothing is deleted by emitting the plan.
Collection is expressed as a step that is statically invalid unless it links
migration and verification gates.

## Lifecycle

```
discover → trace refs → classify → propose canonicalization → migrate → verify → collect
```

| Stage | GRP artifact |
|-------|--------------|
| discover | `reduction.nodes` — semantic node identities (capability IDs) |
| trace refs | `reduction.nodes[].references` — source-grounded reference edges |
| classify | `reduction.nodes[].garbageClass` — reachability / duplication / redundancy |
| propose canonicalization | `reduction.nodes[].canonicalID` — canonical target |
| migrate | steps with recipe `grp.reduction.migrate` |
| verify | `verify` entries on migrate/collect steps and plan-level `verification` |
| collect | steps with recipe `grp.reduction.collect` |

## Garbage classes

| Class | Meaning |
|-------|---------|
| `unreachable` | no active feature/route/workflow reference |
| `duplicate` | canonical primitive supersedes it |
| `obsolete` | version/migration replaced it |
| `redundant` | overlapping semantic capability |
| `dead-api` | dead API/interface |
| `dead-policy` | dead policy/capability |
| `dead-schema-field` | dead schema/data field |
| `dead-dependency-adapter` | dead dependency/provider adapter |

## Plan schema

A reduction plan is a GRP `Plan` carrying an optional `reduction` object
(see `schemas/grp-reduction.v0.1.schema.json`):

```json
{
  "specversion": "0.1",
  "id": "grp_reduction_notification_001",
  "type": "dev.refactor.reduction",
  "goal": "Canonicalize duplicate notification implementations",
  "steps": [ ... ],
  "verification": [ ... ],
  "reduction": {
    "nodes": [
      {
        "id": "cap_notification",
        "kind": "capability",
        "reachable": true,
        "refCount": 9,
        "symbol": "NotificationService",
        "file": "notifications/notification.go"
      },
      {
        "id": "cap_notifier_member",
        "kind": "capability",
        "garbageClass": "duplicate",
        "canonicalID": "cap_notification",
        "reachable": false,
        "symbol": "MemberNotifier",
        "file": "notifications/member_notifier.go",
        "references": [
          { "from": "cap_notifier_member", "to": "cap_notification",
            "kind": "duplicate-of", "file": "notifications/member_notifier.go", "line": 12 }
        ]
      }
    ],
    "blocks": [
      {
        "id": "blk_notification",
        "capabilityId": "cap_notification",
        "standard": true,
        "inputs": ["memberId", "booking"],
        "outputs": ["notification"],
        "nodes": ["cap_notification", "cap_notifier_member"]
      }
    ]
  }
}
```

### Semantic node identity

`reduction.nodes[].id` is the **capability ID** (`cap_` prefix). GRP Core defines
only the shape; a graph provider supplies the IDs. See the adapter contract
below. Node IDs are unique within a plan and deterministic when produced from
the same graph.

### Canonical target

`reduction.nodes[].canonicalID` points at the canonical capability that
supersedes a duplicate/redundant/obsolete node. The canonical target must exist
in the same plan and must be `reachable`. Nodes classified
`duplicate`/`obsolete`/`redundant` **must** declare a canonical target.

### Reference evidence

`reduction.nodes[].references[]` is deterministic, source-grounded evidence:
`from`/`to` node IDs, reference `kind`, and the repo-relative `file`/`line`
where the reference was observed. Both endpoints must reference known nodes.

### Standard blocks

`reduction.blocks[]` are compression metadata for collapsing repeated subgraphs
into a standard block with explicit `inputs` and `outputs`. When present, a
context pack prefers the reduced canonical block instead of dumping every
duplicate implementation (see Context packs).

## Collection safety

A step is a **collect step** when its recipe (or action) is
`grp.reduction.collect`. `ValidatePlan` rejects a collect step unless:

1. it **requires at least one migration step** (`grp.reduction.migrate`) via
   `requires`, and
2. it **carries a verification gate** (`verify` entries).

Migration steps must also carry a verification gate. This guarantees the plan
cannot express "delete first, verify later": reference migration and behavior
verification are static prerequisites of any collection step.

Existing GRP clients remain backwards compatible: `reduction` is an optional
top-level field, every new field is `omitempty`, and plans without reduction
metadata validate exactly as before.

## Context packs

`girl pack` can accept a reduction file (`--reduction-file`) carrying the same
shape as the plan's `reduction` section. When a component matches a reduced node
(a node with a `canonicalID`), the pack emits a reduced placeholder
(`// reduced: <name> collapses into canonical <id>`) instead of dumping the
duplicate implementation's internals. The canonical node's snippet and the
`reduction` metadata (including standard blocks with inputs/outputs) are
included in both `dev.refactor.context` JSON and GRP context JSON.

Unmatched reduction metadata never drops unrelated components — a component is
only suppressed when both its file and symbol match a reduced node.

## Human-readable output

`grp.RenderPlanMarkdown` renders a plan as markdown. For reduction plans it
explains, per node: reachability, classification, canonical target, reference
evidence, and whether collection is safe and under what gates.

```bash
girl validate .grp/plan.json --output markdown
girl plan <path> --output grp-markdown
```

## Adapter contract for CAI Lang / TEMPLE graphs

GIRL (and GRP Core) must not depend on CAI-specific application code. A future
CAI Lang/TEMPLE graph can feed canonical capability IDs into GIRL through this
public, language-agnostic contract:

1. **Shape** — emit JSON conforming to `schemas/grp-reduction.v0.1.schema.json`
   (a `reduction` object with `nodes`, `references`, `canonicalID`, `blocks`).
2. **Identity** — capability IDs are opaque strings; GRP Core never interprets
   them, only checks prefix/uniqueness.
3. **Reachability** — the graph marks a node `reachable` and supplies
   `refCount` plus reference edges as evidence.
4. **Canonicalization** — the graph classifies nodes and assigns `canonicalID`;
   GRP Core validates targets exist and are reachable.
5. **Non-destructive** — the graph produces a plan; GIRL validates it and can
   render it. No executor deletes anything.
6. **Compatibility** — plans without `reduction` are byte-identical in behavior
   to current GRP Core.

## First proof

`testdata/conformance/reduction-duplicate/plan.json` demonstrates three
notification implementations (`MemberNotifier`, `BookingEmail`, `NotifyMember`)
canonicalizing toward one `cap_notification` capability while preserving
distinct behavior where actually necessary (each retains its own migration
step, and collection is gated on migrate + verify). It normalizes
deterministically (step-to-step `requires` links survive renumbering) and
passes `ValidatePlan`.