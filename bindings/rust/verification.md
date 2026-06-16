# Rust Verification Detection

## Detection Rules

GIRL detects Rust verification commands by looking for `Cargo.toml` in the project root.

## Detected Commands

| Command | Required | Type | Source |
|---------|----------|------|--------|
| `cargo build` | Yes | build | `Cargo.toml` |
| `cargo clippy` | No | lint | `Cargo.toml` |
| `cargo test` | Yes | test | `Cargo.toml` |

## Package Manager

When `Cargo.toml` is present, the package manager is detected as `cargo`.

## Lockfile Detection

The verification system checks for `Cargo.toml` (not `Cargo.lock`) as the primary signal, since `Cargo.toml` is always present in valid Rust projects.
