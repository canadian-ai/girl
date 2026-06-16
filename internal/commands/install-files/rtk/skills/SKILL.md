---
name: rtk
description: RTK (Rust Token Killer) integration — pipe GIRL commands through RTK for token-efficient CLI output, reducing context consumption by 60-90%.
---

# RTK + GIRL

RTK compresses CLI output by 60-90%, reducing token consumption when running GIRL commands.

## Usage

```bash
rtk girl analyze . --output text
rtk girl plan . --goal "Refactor" --output markdown
rtk girl verify . --output text
```

## Diagnostics

- `rtk.optimize-commands` — Pipe GIRL commands through RTK for token efficiency
- `rtk.hook-configure` — Configure RTK hook for transparent command rewriting
