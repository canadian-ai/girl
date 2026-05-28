# Verification Output

After applying the refactoring plan to `examples/real-refactors/go-before/main.go`,
the following verification commands pass successfully.

## go build ./...

```
$ go build ./...
$
```

Exit code: 0 — Build succeeds with no errors.

## go vet ./...

```
$ go vet ./...
$
```

Exit code: 0 — No suspicious constructs found.

## go test ./...

```
$ go test ./...
ok  	github.com/canadian-ai/girl/examples/real-refactors/go-after	0.125s
$
```

Exit code: 0 — All tests pass.

---

## Summary

| Metric            | Before | After |
|-------------------|--------|-------|
| Lines (processOrder) | 76     | 11    |
| Cyclomatic complexity | 19  | 5     |
| Nesting depth      | 4      | 2     |
| Ignored errors     | 2      | 0     |
| Helper functions   | 0      | 5     |
| Build             | pass   | pass  |
| Vet               | pass   | pass  |
