
#!/usr/bin/env bash
set -euo pipefail

# Source common setup functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# ---------------------------
# MinIO-specific Config
# ---------------------------
# Benchmark params (tune)
DURATION="${DURATION:-2m}"          # how long each test runs
CONCURRENT="${CONCURRENT:-64}"      # concurrency level
OBJ_SIZE="${OBJ_SIZE:-64MiB}"       # default object size for PUT/GET
AUTOTERM="${AUTOTERM:-true}"        # auto stop when stable results

# Multipart params
MP_OBJ_SIZE="${MP_OBJ_SIZE:-512MiB}"   # object size for multipart
MP_PART_SIZE="${MP_PART_SIZE:-64MiB}"  # part size

log "Duration       : $DURATION"
log "Concurrent     : $CONCURRENT"
log "Obj size       : $OBJ_SIZE"
log "Multipart size : $MP_OBJ_SIZE (part $MP_PART_SIZE)"

# ---------------------------
# Initialize common setup
# ---------------------------
common_init

# ---------------------------
# MinIO-specific WARP flags
# ---------------------------
COMMON_FLAGS+=(--duration "$DURATION" --concurrent "$CONCURRENT" --no-color)
if [[ "$AUTOTERM" == "true" ]]; then
  COMMON_FLAGS+=(--autoterm)
fi

# ---------------------------
# Run benchmarks
# ---------------------------
log "Running PUT benchmark..."
warp put "${COMMON_FLAGS[@]}" --obj.size "$OBJ_SIZE" --benchdata "$RUN_DIR/put.csv.zst" | tee "$RUN_DIR/put.out"

log "Running GET benchmark..."
warp get "${COMMON_FLAGS[@]}" --obj.size "$OBJ_SIZE" --benchdata "$RUN_DIR/get.csv.zst" | tee "$RUN_DIR/get.out"

log "Running Multipart PUT benchmark..."
# WARP supports multipart and multipart-put benchmarks. We'll run multipart-put since you said multipart upload works. [3](https://docs.min.io/enterprise/minio-warp/reference/cli/)[4](https://pkg.go.dev/github.com/minio/warp)
warp multipart-put "${COMMON_FLAGS[@]}" --obj.size "$MP_OBJ_SIZE" --part.size "$MP_PART_SIZE" --benchdata "$RUN_DIR/multipart-put.csv.zst" | tee "$RUN_DIR/multipart-put.out"

# Optional: mixed workload (comment in if you want)
# log "Running MIXED benchmark..."
# warp mixed "${COMMON_FLAGS[@]}" --obj.size "$OBJ_SIZE" --benchdata "$RUN_DIR/mixed.csv.zst" | tee "$RUN_DIR/mixed.out"

# ---------------------------
# Analyze results (optional but useful)
# ---------------------------
log "Analyzing results..."
warp analyze "$RUN_DIR/put.csv.zst" | tee "$RUN_DIR/put.analyze"
warp analyze "$RUN_DIR/get.csv.zst" | tee "$RUN_DIR/get.analyze"
warp analyze "$RUN_DIR/multipart-put.csv.zst" | tee "$RUN_DIR/multipart-put.analyze"

log "Done. Results in: $RUN_DIR"
ls -lh "$RUN_DIR"
