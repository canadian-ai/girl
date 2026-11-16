# Context-pack privacy

`girl pack --privacy private|redacted|public` controls what a context pack may
contain. Unknown modes are rejected.

GIRL's secret detection is heuristic. It is conservative (it prefers hiding a
harmless value to leaking a credential) but **it is not perfect**. Do not treat
a sanitized pack as safe to publish without review.

| Mode       | Guarantee |
|------------|-----------|
| `private`  | Nothing is altered. For use on this machine only. Default. |
| `redacted` | Every exported string passes the shared sanitizer (`internal/redact`) at the export boundary: goal, file paths, summaries, snippets, diagnostics (message, suggestion, symbol, metadata, related info, fixes), steps (action, verify), risks, verification commands. Absolute and home paths are hidden. Contents of sensitive files (`.env*`, private keys, `*.pem`, `credentials*`, `secrets*`, `*.tfvars`, …) are omitted. Any snippet on which a detector still fires after sanitization is omitted. |
| `public`   | Everything in `redacted`, plus public path sanitization. |

Both sanitizing modes **fail closed**: after sanitization an independent scan
runs over every string in the pack, and if any detector still fires the export
is refused with an error naming only the field path and detector, never the
value.

## What is detected

- PEM private-key blocks (multiline, including unterminated blocks)
- Provider token formats: Anthropic, OpenAI, Stripe, GitHub, GitLab, Slack
  tokens and webhooks, AWS access key IDs, Google API keys, npm tokens, JWTs
- `Authorization: Bearer|Basic|Token …` values
- Credentials in URLs and connection strings (`scheme://user:pass@host`)
- Values assigned to sensitive keys (password, secret, token, api key, access
  key, private key, client secret, credential, auth, session, signing key,
  salt, DSN, connection string, database URL, webhook URL) in env, shell
  `export`, YAML (including block scalars), TOML, INI, JSON, Python
  (including triple-quoted strings), JavaScript and Go (`:=`) syntax
- Sensitive keywords followed by a letter-and-digit value in prose
  ("use password …")
- Long, high-entropy mixed-case tokens (hex hashes such as commit IDs are not
  matched)
- Email addresses

## Tests

`internal/redact/redact_test.go` and `internal/packer/privacy_test.go` use
synthetic credentials assembled at runtime. They assert that the secrets are
absent from serialized JSON (both the context pack and the GRP context pack),
that harmless code and prose survive, that unknown modes are rejected, and
that errors never contain secret values.
