# Contributing to GIRL

Welcome. Contributions to GIRL — bug reports, feature requests, documentation, and code — are appreciated.

## Development Setup

Requires **Go 1.22+**.

```bash
# Build the CLI
go build -o girl ./cmd/girl/

# Run all tests
go test ./...

# Format and vet
go fmt ./...
go vet ./...
```

## Code Style

- Run `go fmt` and `go vet` before committing.
- Use meaningful names. Avoid unnecessary abbreviations.
- Keep package surface small. Export only what callers need.
- Write tests alongside new features or fixes.

## Pull Request Process

1. Tests must pass (`go test ./...`).
2. No new `go vet` warnings.
3. Run `go mod tidy` if dependencies change.
4. Keep PRs focused on a single concern.

## GIRL/GRP Gate

If your change touches core analysis, planning, or diagnostic logic, run the GIRL gate before committing:

```bash
girl analyze . --output text
```

This ensures your change doesn't silently degrade analysis quality.

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
