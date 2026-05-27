# GRP-Go Verification

- Binding name: `GRP-Go v0.1`

## Default verification commands

- `go test ./...`
- `go vet ./...`
- `go build ./...`

## Detection rules

| Rule | Match | Verification |
|------|-------|-------------|
| Go project | `go.mod` exists | `go build ./...`, `go vet ./...`, `go test ./...` |
| Makefile test target | `Makefile` contains a `test:` target | `make test` (optional, additional) |
| golangci-lint | `.golangci.yml` or `.golangci.yaml` exists | `golangci-lint run` (optional, additional) |

## Binding-owned recommendations

Verification recommendations are binding-specific. GRP Core defines the verification shape; GRP-Go defines what commands to run for Go repos. Commands are listed in order of decreasing confidence.
