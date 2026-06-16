---
name: rtk
description: RTK (Rust Token Killer) integration — pipe GIRL commands through RTK for token-efficient CLI output, reducing context consumption by 60-90%.
---

# RTK + GIRL

RTK (Rust Token Killer) compresses CLI output by 60-90%, reducing token consumption when running GIRL commands inside AI coding sessions.

## How It Works

RTK hooks into bash to transparently rewrite commands. GIRL commands are automatically compressed:

```bash
# Without RTK: ~1200 tokens
girl analyze src/ --output text

# With RTK: ~300-500 tokens  
rtk girl analyze src/ --output text
```

## Use with GIRL

```bash
# Analyze with token compression
rtk girl analyze . --output text

# Generate plans with compressed output
rtk girl plan . --goal "Refactor" --output markdown

# Verify with compressed results
rtk girl verify . --output text

# Full workflow
rtk girl analyze . --output text | rtk girl plan . --goal "Simplify" --output markdown
```

## GIRL Workflow (RTK-optimized)

```
rtk girl analyze source/     ~400 tokens
  -> rtk girl plan .         ~800 tokens  
  -> git diff | rtk girl review --stdin  ~200 tokens
  -> rtk girl verify .        ~150 tokens
Total: ~1550 tokens (vs ~4000 without RTK)
```

## Installation

```bash
# RTK is pre-configured for Claude Code via ~/.claude/RTK.md
# For other frameworks, pipe explicitly: rtk girl <command>
```

## Diagnostics

- `rtk.optimize-commands` — Pipe GIRL commands through RTK for token efficiency
- `rtk.hook-configure` — Configure RTK hook for transparent command rewriting
