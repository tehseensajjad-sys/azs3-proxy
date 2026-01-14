#!/usr/bin/env bash
set -euo pipefail

# Source common setup functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# ---------------------------
# azs3-proxy-specific Config
# ---------------------------
REGION="${REGION:-south-india}"
CONCURRENT="${CONCURRENT:-64}"
PROFILE_DURATION="${PROFILE_DURATION:-30}"

log "Region         : $REGION"

# ---------------------------
# Initialize common setup
# ---------------------------
common_init

log "---------------------------------------------------"
log "Starting Profiling Run with Concurrency: $CONCURRENT"
log "---------------------------------------------------"

# Start profiling in background
log "Starting CPU profile capture for ${PROFILE_DURATION}s..."
# Ensure pprof is reachable
if ! curl -s "http://localhost:6060/debug/pprof/symbol" > /dev/null; then
    log "Warning: pprof endpoint not reachable on localhost:6060. Is the proxy running with --pprof?"
else
    curl -s -o "$PWD/cpu-profile-c${CONCURRENT}.prof" "http://localhost:6060/debug/pprof/profile?seconds=${PROFILE_DURATION}" &
    CURL_PID=$!
    log "Profiler started (PID: $CURL_PID)"
fi

# GET throughput 1MiB
log "Running azs3-proxy GET 1MiB benchmark (c=$CONCURRENT)..."
warp run "$SCRIPT_DIR/get-1MiB.yml" \
  -var Host="$HOSTPORT" \
  -var AccessKey="$ACCESS_KEY" \
  -var SecretKey="$SECRET_KEY" \
  -var Region="$REGION" \
  -var Bucket="$BUCKET" \
  -var TLS="$USE_TLS" \
  -var Concurrent="$CONCURRENT" \
  -var BenchData="proxy-get-1MiB-c${CONCURRENT}-profile.csv.zst"

if [[ -n "${CURL_PID:-}" ]]; then
    log "Waiting for profiler to finish..."
    wait $CURL_PID || true
    log "Profile saved to $PWD/cpu-profile-c${CONCURRENT}.prof"
fi

log "Generating report..."
/bin/python3 "$SCRIPT_DIR/generate_report.py" "$PWD"
