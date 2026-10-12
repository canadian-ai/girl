#!/usr/bin/env bash
# Run GIRL's real-corpus scale tests against pinned open-source repositories.
#
#   scripts/scale-bench.sh                 # all corpora
#   scripts/scale-bench.sh ripgrep vscode  # a subset
#
# Repositories are shallow-cloned once into $GIRL_SCALE_CACHE (default:
# .girl-scale/, gitignored). The Go standard library is always included.
set -euo pipefail

cd "$(dirname "$0")/.."
cache="${GIRL_SCALE_CACHE:-.girl-scale}"
mkdir -p "$cache"

# name repo tag
corpora=(
  "kubernetes kubernetes/kubernetes v1.31.0"
  "vscode     microsoft/vscode      1.95.0"
  "excalidraw excalidraw/excalidraw v0.18.0"
  "ripgrep    BurntSushi/ripgrep    14.1.1"
)

selected=("$@")
paths=()
for entry in "${corpora[@]}"; do
  read -r name repo tag <<<"$entry"
  if [[ ${#selected[@]} -gt 0 && ! " ${selected[*]} " =~ " ${name} " ]]; then
    continue
  fi
  dir="$cache/$name"
  if [[ ! -d "$dir" ]]; then
    echo "cloning $repo@$tag" >&2
    git -c advice.detachedHead=false clone --quiet --depth 1 --branch "$tag" "https://github.com/$repo.git" "$dir"
  fi
  paths+=("$(cd "$dir" && pwd)")
done

corpus_list=$(IFS=:; echo "${paths[*]}")
echo "workers: ${GIRL_JOBS:-$(getconf _NPROCESSORS_ONLN)}" >&2
GIRL_SCALE=1 GIRL_SCALE_CORPUS="$corpus_list" \
  go test ./internal/commands -run 'TestScale' -count=1 -v -timeout 30m \
  | sed -n 's/^ *scale_test.go:[0-9]*: //p'
