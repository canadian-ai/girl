#!/usr/bin/env bash
# Load-test `girl serve` with k6 on simulated hardware profiles.
#
#   scripts/k6-hardware.sh                       # all profiles that fit this host
#   PROFILES="edge laptop" DURATION=20s scripts/k6-hardware.sh
#   MODE=local scripts/k6-hardware.sh            # no Docker: taskset + GOMAXPROCS
#
# Each profile runs GIRL in a container capped with --cpus/--memory (Go sizes
# its scheduler from the cgroup CPU quota, so worker counts follow the cap),
# then k6 simulates agents calling it. Results land in $OUT/<profile>.{json,md}
# and a combined table in $OUT/summary.md.
#
# Environment: PROFILES, DURATION (30s), VUS (4), MAX_CONCURRENT (2),
# COPIES (corpus size, 250), OUT (.girl-scale/k6-results), MODE (docker|local),
# K6_IMAGE (grafana/k6:0.54.0), RUNTIME_IMAGE (gcr.io/distroless/static-debian12:nonroot).
set -euo pipefail

cd "$(dirname "$0")/.."
repo=$(pwd)

# name cpus memory
all_profiles=(
  "edge        1 1g"
  "laptop      2 4g"
  "ci          4 8g"
  "workstation 8 16g"
)
PROFILES="${PROFILES:-edge laptop ci workstation}"
DURATION="${DURATION:-30s}"
VUS="${VUS:-4}"
MAX_CONCURRENT="${MAX_CONCURRENT:-2}"
COPIES="${COPIES:-250}"
OUT="${OUT:-.girl-scale/k6-results}"
MODE="${MODE:-docker}"
K6_IMAGE="${K6_IMAGE:-grafana/k6:0.54.0}"
RUNTIME_IMAGE="${RUNTIME_IMAGE:-gcr.io/distroless/static-debian12:nonroot}"
corpus="${CORPUS:-.girl-scale/k6-corpus}"
host_cpus=$(getconf _NPROCESSORS_ONLN)

mkdir -p "$OUT"
out_abs=$(cd "$OUT" && pwd)
[[ -d "$corpus/large" ]] || scripts/k6-corpus.sh "$COPIES" "$corpus"
corpus_abs=$(cd "$corpus" && pwd)

bin=".girl-scale/girl-linux-static"
echo "building static girl binary" >&2
CGO_ENABLED=1 go build -tags netgo,osusergo \
  -ldflags '-s -w -linkmode external -extldflags "-static"' -o "$bin" ./cmd/girl
bin_abs="$repo/$bin"

# k6 flags for one profile: k6_args <profile> <url> <summary-dir>
k6_args() {
  printf '%s ' -e "PROFILE=$1" -e "GIRL_URL=$2" -e SMALL_PATH=small -e LARGE_PATH=large \
    -e "VUS=$VUS" -e "DURATION=$DURATION" -e "SUMMARY_DIR=$3" --quiet
}

# run_k6 <profile> <url>: prefer a local k6, else the k6 container (as the
# invoking user, so it can write results).
run_k6() {
  if command -v k6 >/dev/null; then
    # shellcheck disable=SC2046
    k6 run $(k6_args "$1" "$2" "$out_abs") bench/k6/girl.js || true
  else
    # shellcheck disable=SC2046
    docker run --rm --network host --user "$(id -u):$(id -g)" \
      -v "$repo/bench/k6:/scripts:ro" -v "$out_abs:/out" "$K6_IMAGE" \
      run $(k6_args "$1" "$2" /out) /scripts/girl.js || true
  fi
}

wait_healthy() {
  local url=$1
  for _ in $(seq 1 50); do
    if curl -fsS "$url/healthz" >/dev/null 2>&1; then return 0; fi
    sleep 0.2
  done
  echo "girl serve did not become healthy at $url" >&2
  return 1
}

run_docker() {
  local name=$1 cpus=$2 mem=$3 port=$4
  local cname="girl-k6-$name"
  docker rm -f "$cname" >/dev/null 2>&1 || true
  docker run -d --name "$cname" --cpus "$cpus" --memory "$mem" --memory-swap "$mem" \
    -p "127.0.0.1:$port:7777" \
    -v "$bin_abs:/girl:ro" -v "$corpus_abs:/corpus:ro" \
    "$RUNTIME_IMAGE" /girl serve --addr 0.0.0.0:7777 --root /corpus --max-concurrent "$MAX_CONCURRENT" >/dev/null
  wait_healthy "http://127.0.0.1:$port"
  run_k6 "$name" "http://127.0.0.1:$port"
  docker stats --no-stream --format '{{.MemUsage}}' "$cname" > "$out_abs/$name.mem" 2>/dev/null || true
  docker rm -f "$cname" >/dev/null
}

run_local() {
  local name=$1 cpus=$2 port=$3
  local last=$((cpus - 1))
  GOMAXPROCS="$cpus" taskset -c "0-$last" "$bin_abs" serve --addr "127.0.0.1:$port" \
    --root "$corpus_abs" --max-concurrent "$MAX_CONCURRENT" 2>/dev/null &
  local pid=$!
  wait_healthy "http://127.0.0.1:$port"
  run_k6 "$name" "http://127.0.0.1:$port"
  kill "$pid" 2>/dev/null; wait "$pid" 2>/dev/null || true
}

port=17777
ran=()
for entry in "${all_profiles[@]}"; do
  read -r name cpus mem <<<"$entry"
  [[ " $PROFILES " == *" $name "* ]] || continue
  if (( cpus > host_cpus )); then
    echo "skip $name: needs $cpus CPUs, host has $host_cpus" >&2
    continue
  fi
  echo "=== $name ($cpus CPU, $mem) ===" >&2
  rm -f "$out_abs/$name.md" "$out_abs/$name.json"
  if [[ "$MODE" == "local" ]]; then
    run_local "$name" "$cpus" "$port"
  else
    run_docker "$name" "$cpus" "$mem" "$port"
  fi
  ran+=("$name")
  port=$((port + 1))
done

{
  echo "| Profile | Workers | Agent p50 (ms) | Agent p95 (ms) | Agent req/s | Queue p95 (ms) | Repo scan avg | Errors | Thresholds | Memory after run |"
  echo "|---|---:|---:|---:|---:|---:|---:|---:|---|---:|"
  for name in "${ran[@]}"; do
    if [[ -f "$out_abs/$name.md" ]]; then
      mem=$(cut -d/ -f1 "$out_abs/$name.mem" 2>/dev/null | tr -d ' ')
      printf '%s %s |\n' "$(tr -d '\n' < "$out_abs/$name.md")" "${mem:-n/a}"
    else
      echo "| $name | - | - | - | - | - | - | - | no result | - |"
    fi
  done
} > "$out_abs/summary.md"
cat "$out_abs/summary.md"

# STRICT=1 (CI): fail when any profile missed its thresholds or did not run.
if [[ "${STRICT:-0}" == "1" ]] && grep -qE '\| (FAIL|no result) \|' "$out_abs/summary.md"; then
  echo "k6: thresholds failed for at least one hardware profile" >&2
  exit 1
fi
