# Repository Instructions

- Use Go 1.22+ for development.
- Use `go build`, `go test`, `go mod tidy`.
- The `girl` binary goes in the repo root.
- Keep `.gitignore` up to date for private eval data.
- All analysis runs locally. No code leaves the machine.
- Private evals in `evals/private/` are gitignored.
- Never commit absolute paths or secret values.

## Building

```bash
go build -o girl ./cmd/girl/
```

## Testing

```bash
go test ./...
```

## Running Examples

```bash
make full-example
```

## OpenCode Integration

GIRL agents are in `opencode/agents/`. Copy them to any project:

```bash
cp opencode/agents/* .opencode/agents/
```

The GIRL CLI must be on PATH for OpenCode agents to use it.

## graphify

This project has a graphify knowledge graph at graphify-out/ (if present).

## Security

- Do not expose secrets or env values.
- Do not commit private eval fixtures.
- Do not include absolute paths in reports.
- Use `--privacy private` for private context packs.
