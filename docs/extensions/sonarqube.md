# SonarQube provider (experimental)

A read-only adapter that imports SonarQube / SonarCloud issues into GIRL's
governed `Finding` contract, proposes bounded GRP actions, and issues
authoritative remediation receipts.

## Capability status

| Capability | Status |
|------------|--------|
| Authenticated read-only client (bearer / legacy basic), timeouts, retry with `Retry-After`, pagination, clear auth/connectivity errors | Implemented, mock-tested |
| Normalization to `extensions.Finding` with provenance (analysis key, revision, branch/PR, category) | Implemented, mock-tested |
| Mapping to bounded GRP action proposals | Implemented for structural rules (S3776, S1541, S134, S1066, S138, S107, S104) in Python, TS/JS, Go and Rust; everything else is reported as `unmapped` |
| Remediation receipts tied to a fresh scan of the fixing revision | Implemented, mock-tested |
| **Validation against a live SonarQube or SonarCloud instance** | **Not yet done.** The adapter follows the documented Web API (SonarQube 10.x / 2025.x, SonarCloud); response shapes in the tests are written from the documentation, not recorded from a server. |
| Applying fixes / transitioning issues | Out of scope. The provider never writes to SonarQube. |

## Endpoints

All requests are `GET`:

| Endpoint | Used for |
|----------|----------|
| `/api/system/status` | reachability and server version (must be `UP`) |
| `/api/authentication/validate` | token validity |
| `/api/issues/search` | open issues (`resolved=false`), or specific issues by key; paged at `ps=500` up to the API's 10,000-result ceiling (beyond it the import fails rather than truncating) |
| `/api/project_analyses/search` | latest analysis key, date and SCM revision for the branch / PR |
| `/api/ce/component` | queued and last background tasks (pending or failed analyses) |

Severity uses the highest MQR `impacts[].severity` when present (10.2+),
otherwise the legacy `severity`. `issueStatus` (10.4+) is preferred over
`status`/`resolution`.

## Setup

```bash
export SONAR_TOKEN=...            # never passed as a flag
girl sonar status --url https://sonar.example.com --project my-project
girl sonar issues --url https://sonar.example.com --project my-project --branch main > findings.json
girl sonar verify --url https://sonar.example.com --project my-project --branch main \
  --issues AYx1,AYx2 --revision "$(git rev-parse HEAD)"
```

`SONAR_HOST_URL`, `SONAR_PROJECT_KEY` and `SONAR_ORGANIZATION` (SonarCloud) may
replace the flags. Use `--auth basic` for SonarQube versions before 10.0.

**Least privilege:** use a user token for an account that has only *Browse*
permission on the project. Project analysis tokens cannot read issues. The
token is sent only over HTTPS (plain HTTP is allowed for loopback hosts only),
is never forwarded across a cross-host redirect, and never appears in output,
errors, logs or context packs.

## Governance

`girl sonar issues` emits findings and action **proposals**. A provider finding
never authorizes a change: every action must pass `extensions.Evaluate` against
an explicit policy (allowed paths, file and line budgets, review categories).
Security findings (vulnerabilities, hotspots, SECURITY impacts) carry
`category: security` so policies can require human review.

Issue messages are untrusted text written outside the repository. They are
stripped of control characters and capped at 500 characters, but an agent
consuming findings should still treat `message` as data, never as instructions.

## Remediation receipts

`girl sonar verify` returns a receipt and exits 0 only when the status is
`verified`:

| Receipt status | When |
|----------------|------|
| `verified`   | the latest successful analysis of the branch/PR is of the fixing revision, and every named issue is closed with resolution `FIXED` (or `issueStatus: FIXED`) |
| `failed`     | at least one named issue is still open on that scan |
| `unverified` | an analysis is queued or in progress; the last analysis failed; no analysis exists; the analysis has no SCM revision; the analysis is of a different (stale) revision; an issue is missing, dismissed (won't fix, false positive, accepted) or in an unknown state |

Pending, stale, missing and failed scans are never reported as success.
