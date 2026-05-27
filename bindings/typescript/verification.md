# GRP-TypeScript Verification v0.1

- Binding name: `GRP-TypeScript v0.1`

## Detection rules

| Rule | Match | Verification |
|------|-------|-------------|
| npm project | `package-lock.json` exists | `npm run {script}` |
| yarn project | `yarn.lock` exists | `yarn {script}` |
| pnpm project | `pnpm-lock.yaml` exists | `pnpm {script}` |
| bun project | `bun.lock` / `bun.lockb` exists | `bun run {script}` |
| TypeScript config | `tsconfig.json` exists | `npx tsc --noEmit` |

## Configuration detection

Commands are resolved from `package.json` scripts. Standard scripts checked: `typecheck`, `lint`, `test`, `build`, `format`. Each detected command includes:

- `command`: the full shell command
- `required`: `true` for build/typecheck/test, `false` for lint/format
- `source`: `"package.json"` or `"lockfile"` or `"binding-default"`
- `confidence`: `"high"` when lockfile matches package.json, `"medium"` for inferred commands
