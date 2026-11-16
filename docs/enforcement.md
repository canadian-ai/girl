# Enforcement contract

`girl check` is evidence-driven and fails closed. **The absence of a failure is
never treated as proof of success.** Every gate either produces evidence or says
plainly that it could not.

## Verdicts

Every check result carries a `status` (`pass` | `warn` | `fail`, kept for
compatibility) and an evidence `verdict`:

| Verdict      | Meaning                                                                 | Exit code |
|--------------|-------------------------------------------------------------------------|-----------|
| `verified`   | Every required check executed and passed.                               | 0         |
| `failed`     | A required check failed (including unparseable source).                 | 1         |
| `blocked`    | A prerequisite or policy requirement is missing (strict mode only).     | 1         |
| `unverified` | Required evidence was not produced (advisory mode, or `--no-verify`).   | 0 (status `warn`) |

`reasons` lists every contribution to the verdict, prefixed `fail:`, `blocked:`,
`unverified:` or `warn:`. Text and JSON output carry the same information and
the process exit code always follows `status`.

## Strict mode

`--strict` (implied by `--ci`) turns missing evidence into a blocking failure:

- no verification commands configured or detected;
- repo-native verification discovery failed;
- a required complexity baseline is missing, unreadable, corrupt or incompatible;
- `.girl/config.yaml` contains invalid values.

Without `--strict` these are visible warnings and the verdict is `unverified`.
An explicit `--no-verify` is always visible and can never yield `verified`.

## Blocking analysis failures

Failures of an **authoritative** parser are a separate class from refactoring
severity. They are tagged `blocking` and always fail the gate, with or without
verification and in every mode:

| Code                  | Cause                                                                 |
|-----------------------|-----------------------------------------------------------------------|
| `py.syntax-error`     | Python's `ast` rejected the file (includes syntax newer than the interpreter) |
| `py.analysis-error`   | Python parser failed for another reason (null bytes, recursion), or `python3` is not installed (`tags: runtime-missing`) |
| `go.syntax-error`     | `go/parser` rejected the file                                         |

Each carries the file, line and (where the parser reports it) column. Analyzed
Python is parsed only, never imported or executed.

TypeScript/JavaScript and Rust are parsed with tree-sitter, an error-recovering
grammar that is not the language's own compiler and rejects some valid source
(for example a bare `&` in JSX text). Its findings are therefore **advisory**:
`ts.parse-recovery` and `rust.parse-recovery` are low severity, low confidence
and never blocking. For these languages the authoritative syntax check is the
project's own `typecheck` / `build` command in verification, which is one more
reason a gate without executed verification is never `verified`.

## Verification

`verificationSummary.status`:

| Status           | Meaning |
|------------------|---------|
| `verified`       | every configured command ran and exited 0 |
| `failed`         | a command exited non-zero, could not start, or was not run |
| `not-configured` | no commands in config and none detected |
| `blocked`        | repo-native detection failed |
| `skipped`        | `--no-verify` |

Each command records `role`, `command`, `status` (`pass`, `fail`, `error` for
"could not start / not found / not executable", `not-run` for the `--fail-fast`
remainder), `exitCode`, and provenance (`source`, `confidence`). Captured
output passes the export sanitizer (see [privacy.md](privacy.md)) and is capped
at 64 KiB. Commands currently run without a timeout.
`verificationRun` is `true` only when at least one command actually executed.

Repo-native detection covers JavaScript/TypeScript package scripts, Go
(`go build/vet/test`), Rust (`cargo build/clippy/test`) and Python
(`compileall`, plus `pytest`, `ruff` and `mypy` only when the repository
declares them). GIRL never invents a command.

## Complexity ratchet

`complexity.enforcement` is one of:

| Enforcement   | Meaning |
|---------------|---------|
| `enforced`    | baseline loaded and compared (`baselineSha256`, `baselineSchemaVersion` attached) |
| `threshold`   | `fail_on: threshold`; no baseline needed |
| `unenforced`  | the configured `regression` policy could not be applied; `status` is `unenforced` (advisory) or `blocked` (strict) with `reason` and `remediation` |
| `disabled`    | `fail_on: never` |
| `unsupported` | not a JS/TS project; `status: not-applicable`. Other bindings report complexity as analyzer diagnostics |

In base mode (`--base`, `--ci`) the baseline is read from the base ref
(`baselineSource: base:<ref>`), not the working tree, so a change cannot
rewrite its own yardstick. If the change touches `.girl/config.yaml` or the
baseline, the files are listed in `policyFilesChanged` and flagged for human
review. Note that the configuration itself (exclusions, verify commands,
thresholds) is still read from the checked-out tree: for untrusted pull
requests, review `policyFilesChanged` before trusting the result, and remember
that `verify` commands execute code from the change.

A gate never writes or resets a baseline. Create one intentionally on a trusted
revision and commit it:

```bash
girl complexity . --write-baseline .girl/complexity-baseline.json
```

## Scope and exclusions

`scope` reports what was actually analyzed: `mode` (`all`, `working-tree`,
`base`), `discovered`, `analyzed`, `skipped` (by reason: `excluded`,
`unsupported`, `generated`, `symlink`, `missing`, …), `outsideScope`, the
effective `exclusions` and the `config` file they came from.

### Changed-file discovery

- **Working-tree mode** (no `--base`): staged, unstaged and untracked
  non-ignored files versus `HEAD`. Renames report the new path.
- **Base mode** (`--base REF`, or `GITHUB_BASE_REF` under `--ci`):
  `merge-base(REF, HEAD)..HEAD`, committed changes only. `REF` falls back to
  `origin/REF`. A missing base is an error, not an empty scan.
- Paths are resolved from the Git root, constrained to the requested subtree on
  path-segment boundaries, and read NUL-delimited (spaces and Unicode survive).
- Deleted and ignored files are never reported. Symlinks are never followed, in
  changed or full scans, so a link cannot pull a file from outside the tree
  into analysis.
- Base refs beginning with `-` are rejected; a ref can never be read as a Git
  option.
- If discovery fails, `check` analyzes the full project instead and reports
  `scope.discoveryError`; it never looks like a clean scan.

### Exclusion semantics (`analysis.exclude`)

Exclusions are applied before any file is read, identically by `check`,
`next`, every language analyzer, the repeated-literal scan and the complexity
engine, in all, changed and nested-workspace modes. The nearest enclosing
`.girl/config.yaml` (up to the Git root) applies. Outside a Git repository only
the target directory's own config is used; parents are never searched.

- A pattern without `/` (`generated`, `*.pb.go`) matches any single path
  segment: a directory excludes its subtree, a file matches by base name.
- A pattern with `/` (`src/legacy`, `packages/*/fixtures`) is anchored at the
  config directory and excludes that path and everything below it. `**`
  matches any number of segments.
- Matching respects segment boundaries: `gen` never excludes `generic/`.
- A config without an `exclude` list keeps the defaults
  (`.next`, `node_modules`, `dist`, `build`, `coverage`, `generated`);
  `exclude: []` clears them. Built-in skips (VCS, dependency and build
  directories) always apply.
- An explicit single-file target (`girl next path/to/file.ts`) is analyzed even
  if it matches an exclusion; naming a file is an explicit request.
- Invalid patterns are reported in `configProblems` (blocking in strict mode).
