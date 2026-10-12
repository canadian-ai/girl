#!/usr/bin/env bash
# Build a React/TSX load-test corpus from GIRL's own TSX fixtures.
#
#   scripts/k6-corpus.sh [copies] [out-dir]
#
# Every copy is a workspace package holding all TSX fixtures from testdata/
# and examples/ (real-world panels, golden multi-diagnostic components, the
# messy form), so findings stay realistic as the corpus grows. The output
# directory contains:
#   small/  one copy  - the size of a typical agent change
#   large/  N copies  - a monorepo-sized scan
set -euo pipefail

cd "$(dirname "$0")/.."
copies="${1:-250}"
out="${2:-.girl-scale/k6-corpus}"

fixtures=()
while IFS= read -r f; do fixtures+=("$f"); done < <(
  find testdata examples -name '*.tsx' -not -path '*/node_modules/*' | sort
)
if [[ ${#fixtures[@]} -eq 0 ]]; then
  echo "no TSX fixtures found" >&2
  exit 1
fi

rm -rf "$out"
write_copy() {
  local dest="$1"
  mkdir -p "$dest/src"
  for f in "${fixtures[@]}"; do
    # Flatten fixture paths into unique, stable file names.
    name=$(echo "$f" | tr '/' '_')
    cp "$f" "$dest/src/$name"
  done
  printf '{"name":"%s","private":true}\n' "$(basename "$dest")" > "$dest/package.json"
}

write_copy "$out/small/app"
for i in $(seq -f '%04g' 1 "$copies"); do
  write_copy "$out/large/packages/app-$i"
done
printf '{"name":"k6-corpus","private":true,"workspaces":["packages/*"]}\n' > "$out/large/package.json"

files=$(find "$out/large" -name '*.tsx' | wc -l | tr -d ' ')
echo "corpus: $out (small: ${#fixtures[@]} files, large: $files files from ${#fixtures[@]} fixtures x $copies)" >&2
