#!/usr/bin/env bash
set -euo pipefail

# Source common setup
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# Configuration
CONCURRENT=16
DURATION="60s" # Override duration for profiling run to be shorter
PROFILE_SECONDS=30

log "Initializing profiling run..."
common_init

# Create a temporary modified config for shorter duration
cat "$SCRIPT_DIR/put-1MiB.yml" | sed "s/duration: 300s/duration: $DURATION/" > "$SCRIPT_DIR/put-1MiB-profile.yml"

log "Starting warp benchmark (put-1MiB, c=$CONCURRENT, duration=$DURATION)..."
warp run "$SCRIPT_DIR/put-1MiB-profile.yml" \
  -var Host="$HOSTPORT" \
  -var AccessKey="$ACCESS_KEY" \
  -var SecretKey="$SECRET_KEY" \
  -var Region="us-east-1" \
  -var Bucket="$BUCKET" \
  -var TLS="$USE_TLS" \
  -var Concurrent="$CONCURRENT" \
  -var BenchData="profile-put-1MiB.csv.zst" &
WARP_PID=$!

log "Waiting 10s for benchmark to warm up..."
sleep 10

log "Capturing CPU profile for ${PROFILE_SECONDS}s..."
curl -o "cpu.prof" "http://localhost:6060/debug/pprof/profile?seconds=${PROFILE_SECONDS}"

log "Waiting for benchmark to finish..."
wait "$WARP_PID"

log "Profiling complete. Results:"
log "- Profile: cpu.prof"
log "- Benchmark Data: profile-put-1MiB.csv.zst"
log ""
log "Analyze with: go tool pprof -http=:8081 cpu.prof"

# Cleanup temporary file
rm "$SCRIPT_DIR/put-1MiB-profile.yml"
