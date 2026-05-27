#!/bin/bash
set -euo pipefail
DIR="$(cd "$(dirname "$0")" && pwd)"
echo "Validating example GRP plans..."
for f in "$DIR"/*.json; do
  name=$(basename "$f")
  if python3 -c "
import json, sys
with open('$f') as fh:
    data = json.load(fh)
print(f'  OK $name ({len(json.dumps(data))} bytes)')
" 2>&1; then
    :  # all good
  else
    echo "  FAIL $name"
    exit 1
  fi
done
echo "All examples valid."
