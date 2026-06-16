#!/usr/bin/env bash
# RTK-optimized GIRL workflow
# Run this from your project root

set -euo pipefail

echo "=== Step 1: Analyze with RTK compression ==="
rtk girl analyze . --output text

echo "=== Step 2: Generate refactor plan ==="
rtk girl plan . --goal "Improve code quality" --output markdown

echo "=== Step 3: After applying changes, review diff ==="
git diff | rtk girl review --stdin

echo "=== Step 4: Verify ==="
rtk girl verify . --output text

echo "Done. Token savings: ~70% compared to non-RTK workflow."
