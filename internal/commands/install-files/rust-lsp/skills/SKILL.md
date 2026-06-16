---
name: rust-lsp
description: Rust LSP integration — export GIRL diagnostics in LSP-compatible format for use with rust-analyzer and Rust IDE tooling.
---

# Rust-LSP + GIRL

GIRL exports diagnostics in LSP-compatible format for use with rust-analyzer and Rust IDE tooling.

## How It Works

1. `girl analyze src/ --lang rust` runs Rust analysis
2. Diagnostics export in LSP diagnostic format
3. IDE tooling consumes GIRL diagnostics alongside rust-analyzer

## Diagnostics

- `rust-lsp.long-function` — Function exceeds complexity threshold
- `rust-lsp.high-complexity` — Function has high cyclomatic complexity
- `rust-lsp.deep-nesting` — Nesting depth exceeds threshold
- `rust-lsp.large-file` — File exceeds line count threshold
- `rust-lsp.export-diagnostics` — Export diagnostics in LSP format
