# GRP-Rust Binding

- Binding name: `GRP-Rust v0.1`
- Parser: [tree-sitter](https://tree-sitter.github.io/tree-sitter/) with Rust grammar
- Maturity: **Experimental**

## Files

| File | Purpose |
|------|---------|
| `diagnostics.md` | Rust diagnostics and their specification |
| `recipes.md` | Rust-specific refactoring recipes |
| `verification.md` | Verification detection rules for Rust/Cargo projects |

## Overview

The Rust binding analyzes `.rs` files using tree-sitter and produces GRP diagnostics under the `rust.*` namespace. It detects refactoring opportunities in Rust code including long functions, high cyclomatic complexity, deep nesting, large files, and excessive parameter counts.

## Analyzer

- Walks `.rs` files in a directory tree (skips `target/`, `.git/`, etc.)
- Parses each file with tree-sitter Rust grammar
- Extracts function metadata: line ranges, parameters, complexity, nesting, modifiers
- Detects impl block receivers for method diagnostics

## Usage

```bash
# Analyze Rust code
girl analyze src/ --lang rust --output text

# Generate a GRP refactor plan
girl plan src/ --lang rust --output markdown

# Create a context pack
girl pack src/ --lang rust --output markdown
```
