# Rust Refactoring Recipes

Each Rust diagnostic maps to a refactoring recipe in the GRP plan.

## Recipe Catalog

### rust.extract-function
- **Trigger**: `rust.long-function`
- **Action**: Extract helper functions from the target function to reduce its line count
- **Verification**: `cargo build`, `cargo test`

### rust.simplify-branches
- **Trigger**: `rust.high-complexity`
- **Action**: Reduce branching with guard clauses, simpler match arms, or early returns
- **Verification**: `cargo clippy`, `cargo test`

### rust.flatten-nesting
- **Trigger**: `rust.deep-nesting`
- **Action**: Flatten deep nesting with early returns or extracted helper functions
- **Verification**: `cargo build`, `cargo test`

### rust.split-file
- **Trigger**: `rust.large-file`
- **Action**: Split the file into smaller modules by responsibility
- **Verification**: `cargo build`, `cargo test`

### rust.extract-options-struct
- **Trigger**: `rust.too-many-params`
- **Action**: Group parameters into a builder or options struct
- **Verification**: `cargo build`, `cargo test`
