# GRP Verification

**Grammar Refactoring Protocol — Verification Shape**

Version 0.1 — Core specification.

## Verification

A verification entry declares a repo-native command that confirms the refactor
produced a correct result. Each entry is detected from the repository, not
guessed — the planner discovers build, test, lint, and typecheck commands from
lockfiles, manifest files, and build configuration.

```json
{
  "command": "go test ./...",
  "required": true,
  "source": "go",
  "confidence": "high",
  "type": "test"
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | string | yes | Shell command to run |
| `required` | boolean | yes | Whether this must pass before the plan is complete |
| `source` | string | yes | Detection source: `"go"`, `"package.json"`, `"Makefile"`, etc. |
| `confidence` | string | yes | One of: `"high"`, `"medium"`, `"low"` |
| `type` | string | no | Verification type: `"build"`, `"test"`, `"lint"`, `"typecheck"`, `"format"`, `"security"`, `"custom"` |

### Verification Types

| Type | Description |
|------|-------------|
| `build` | Compilation or build command (e.g. `go build`, `tsc --noEmit`) |
| `test` | Test runner command (e.g. `go test`, `npm test`) |
| `lint` | Linter invocation (e.g. `golangci-lint run`, `eslint`) |
| `typecheck` | Type checker (e.g. `tsc --noEmit`, `pyright`) |
| `format` | Formatter check (e.g. `gofmt -l`, `prettier --check`) |
| `security` | Security scanner (e.g. `gosec`, `npm audit`) |
| `custom` | Any other verification command |

### Confidence

| Value | Meaning |
|-------|---------|
| `high` | Command is confirmed present in the repo's lockfile, manifest, or build configuration |
| `medium` | Command is inferred from ecosystem conventions (e.g. `go.mod` suggests `go test`) |
| `low` | Command is guessed from filename patterns or partial evidence |

### Detection Rules

1. **Commands are detected from the repo, not guessed.** Every verification
   entry must be traceable to a file in the repository (go.mod, package.json,
   Makefile, Dockerfile, etc.).

2. **Go repos** default to `go test ./...`, `go build ./...`, and `go vet ./...`
   when `go.mod` is present and no `Makefile` overrides them.

3. **Package managers** are detected from lockfiles: `package-lock.json`
   (npm), `pnpm-lock.yaml` (pnpm), `yarn.lock` (yarn), `bun.lock` (bun).

4. **package.json scripts** are used when present. The `test`, `build`,
   `lint`, `typecheck`, and `format` scripts are extracted as verification
   entries.

5. **Confidence is high** when the script or command exists in a manifest
   matched to its lockfile (e.g. `package.json` with matching
   `package-lock.json`).

6. **Multiple sources** may produce duplicate commands. Duplicates with the
   same `command` string should be deduplicated, keeping the highest
   `confidence` entry.
