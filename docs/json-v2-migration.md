# Go 1.27 and JSON v2 migration

Girl builds with Go 1.27.4 in CI, releases, and load tests. The module requires
Go 1.27 and recommends the patched 1.27.4 toolchain.

HTTP POST bodies now require a single complete JSON value. Send `{}` for default
requests. Empty bodies, unknown or incorrectly cased fields, duplicate member
names, invalid UTF-8, trailing values, and bodies larger than 1 MiB return 400.
Responses are encoded before headers are committed, so encoding errors return a
JSON 500 response. Existing response field omission, null collections, escaping,
and map ordering are preserved.

CLI JSON, saved GRP plans, and SARIF exports use the v2 API with v1 compatibility
options. Existing persisted GRP input decoding also retains v1 semantics. Other
legacy JSON sites remain compatible and can be migrated incrementally with the
codemod below. This is not a claim that analysis itself becomes faster.

References: [Go 1.27 release notes](https://go.dev/doc/go1.27) and
[JSON v2 migration guide](https://go.dev/doc/jsonv2-migration).

## Reusable codemod

Run locally with Go 1.27.4 or newer:

```sh
go run ./cmd/jsonv2-migrate ./path/to/package
go run ./cmd/jsonv2-migrate --write ./path/to/package
```

The default reports candidates without writing. Pass files or directories; directory
walks skip hidden directories, vendor, node_modules, and symlinks. Review the diff
and run the target repository's tests after writing.

The codemod uses Go's parser and type information to identify direct
`encoding/json.Marshal` and `Unmarshal` calls, including aliased imports. It adds
`encoding/json/v2` and passes `encoding/json.DefaultOptionsV1()` to retain existing
wire behavior. It chooses an unused compatibility alias, preserves comments, and
leaves shadowed local identifiers alone. Running it again makes no further edits.

Files using encoders, decoders, JSON types, function references, dot imports,
mixed v1/v2 imports, or unresolved JSON imports are reported for manual review and
left unchanged. This deliberately trades coverage for safety. Each file is checked
independently; generated and test files are included when explicitly present in
the requested tree, so choose the input paths carefully. Unrelated unresolved local
imports do not authorize any edits: target package references must be proven by
the type checker. No project source is uploaded.

This is an API migration, not an automatic switch to stricter JSON semantics.
Remove compatibility options only after testing nulls, omitempty, map ordering,
custom marshalers, case matching, duplicate keys, and invalid UTF-8. HTTP request
decoders require separate review for unknown fields, size limits, empty bodies,
and trailing data. No performance improvement is assumed without benchmarks.
