---
name: rust-lsp
description: Rust LSP integration — export GIRL diagnostics in LSP-compatible format for use with rust-analyzer and Rust IDE tooling.
---

# Rust-LSP + GIRL

GIRL exports diagnostics in LSP-compatible format for use with rust-analyzer and Rust IDE tooling.

## How It Works

1. `girl analyze src/ --lang rust` runs Rust analysis (future: uses rust-analyzer)
2. Diagnostics can be exported in LSP diagnostic format
3. IDE tooling can consume GIRL diagnostics alongside rust-analyzer

## LSP Diagnostic Format

```json
{
  "jsonrpc": "2.0",
  "method": "textDocument/publishDiagnostics",
  "params": {
    "uri": "file:///path/to/src/main.rs",
    "diagnostics": [
      {
        "range": {
          "start": { "line": 10, "character": 0 },
          "end": { "line": 45, "character": 0 }
        },
        "severity": 2,
        "code": "rust.long-function",
        "source": "girl",
        "message": "Function 'process_data' is 120 lines (threshold: 60). Consider extracting helper functions."
      }
    ]
  }
}
```

## Use with Rust Analyzer

```bash
# Analyze Rust code
girl analyze src/ --lang rust --output text

# Export diagnostics in LSP format
girl analyze src/ --lang rust --output json > diagnostics.json

# Generate refactor plan
girl plan src/ --goal "Simplify complex functions" --output markdown
```

## Pipeline Integration

```bash
# CI gate: fail if GIRL finds high-severity issues
girl analyze src/ --lang rust --output text | grep "high" && exit 1
```

## Diagnostics

- `rust-lsp.long-function` — Function exceeds complexity threshold
- `rust-lsp.high-complexity` — Function has high cyclomatic complexity
- `rust-lsp.deep-nesting` — Nesting depth exceeds threshold
- `rust-lsp.large-file` — File exceeds line count threshold
- `rust-lsp.export-diagnostics` — Export diagnostics in LSP format
