#!/usr/bin/env bash
set -euo pipefail

# Source common setup
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Configuration
CONCURRENT=16
DURATION="60s" 
PROFILE_SECONDS=30

log "Initializing profiling run for GET..."
common_init

# Create a temporary modified config for shorter duration
cat "$SCRIPT_DIR/get-1MiB.yml" | sed "s/duration: 300s/duration: $DURATION/" > "$SCRIPT_DIR/get-1MiB-profile.yml"

log "Starting warp benchmark (get-1MiB, c=$CONCURRENT, duration=$DURATION)..."
# warp get will populate objects first if they don't exist
warp run "$SCRIPT_DIR/get-1MiB-profile.yml" \
  -var Host="$HOSTPORT" \
  -var AccessKey="$ACCESS_KEY" \
  -var SecretKey="$SECRET_KEY" \
  -var Region="us-east-1" \
  -var Bucket="$BUCKET" \
  -var TLS="$USE_TLS" \
  -var Concurrent="$CONCURRENT" \
  -var BenchData="profile-get-1MiB.csv.zst" &
WARP_PID=$!

log "Waiting 80s for benchmark to warm up (populate objects)..."
sleep 80

log "Capturing CPU profile for ${PROFILE_SECONDS}s..."
curl -o "cpu_get.prof" "http://localhost:6060/debug/pprof/profile?seconds=${PROFILE_SECONDS}"

log "Waiting for benchmark to finish..."
wait "$WARP_PID"

log "Profiling complete. Results:"
log "- Profile: cpu_get.prof"
log "- Benchmark Data: profile-get-1MiB.csv.zst"
log ""
log "Analyze with: go tool pprof -http=:8081 cpu_get.prof"

# Cleanup temporary file
rm "$SCRIPT_DIR/get-1MiB-profile.yml"
