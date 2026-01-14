#!/usr/bin/env bash
set -euo pipefail

# Source common setup
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Configuration
CONCURRENT=16
DURATION="120s" 
PROFILE_SECONDS=60

log "Initializing profiling run for GET (Large 100MB Objects)..."
common_init

# warp get will populate objects first if they don't exist
warp run "$SCRIPT_DIR/get-100MiB.yml" \
  -var Host="$HOSTPORT" \
  -var AccessKey="$ACCESS_KEY" \
  -var SecretKey="$SECRET_KEY" \
  -var Region="us-east-1" \
  -var Bucket="$BUCKET" \
  -var TLS="$USE_TLS" \
  -var Concurrent="$CONCURRENT" \
  -var BenchData="profile-get-100MiB.csv.zst" &
WARP_PID=$!

log "Waiting 120s for benchmark to warm up (populate objects)..."
sleep 120

log "Capturing CPU profile for ${PROFILE_SECONDS}s..."
curl -o "cpu_get_large.prof" "http://localhost:6060/debug/pprof/profile?seconds=${PROFILE_SECONDS}"

log "Waiting for benchmark to finish..."
wait "$WARP_PID"

log "Profiling complete. Results:"
log "- Profile: cpu_get_large.prof"
log "- Benchmark Data: profile-get-100MiB.csv.zst"
log ""
log "Analyze with: go tool pprof -http=:8081 cpu_get_large.prof"
