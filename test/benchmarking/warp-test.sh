#!/usr/bin/env bash
set -euo pipefail

# Source common setup functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# ---------------------------
# azs3-proxy-specific Config
# ---------------------------
REGION="${REGION:-south-india}"

log "Region         : $REGION"

# ---------------------------
# Initialize common setup
# ---------------------------
common_init

# ---------------------------
# Run azs3-proxy benchmarks
# ---------------------------

for CONCURRENT in 8 16 64; do
    log "---------------------------------------------------"
    log "Starting benchmarks with Concurrency: $CONCURRENT"
    log "---------------------------------------------------"

    # Mixed 1MiB
    log "Running azs3-proxy Mixed 1MiB benchmark (c=$CONCURRENT)..."
    warp run "$SCRIPT_DIR/mixed-1MiB.yml" \
      -var Host="$HOSTPORT" \
      -var AccessKey="$ACCESS_KEY" \
      -var SecretKey="$SECRET_KEY" \
      -var Region="$REGION" \
      -var Bucket="$BUCKET" \
      -var TLS="$USE_TLS" \
      -var Concurrent="$CONCURRENT" \
      -var BenchData="proxy-mixed-1MiB-c${CONCURRENT}.csv.zst"

    # PUT throughput 1MiB
    log "Running azs3-proxy PUT 1MiB benchmark (c=$CONCURRENT)..."
    warp run "$SCRIPT_DIR/put-1MiB.yml" \
      -var Host="$HOSTPORT" \
      -var AccessKey="$ACCESS_KEY" \
      -var SecretKey="$SECRET_KEY" \
      -var Region="$REGION" \
      -var Bucket="$BUCKET" \
      -var TLS="$USE_TLS" \
      -var Concurrent="$CONCURRENT" \
      -var BenchData="proxy-put-1MiB-c${CONCURRENT}.csv.zst"

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
      -var BenchData="proxy-get-1MiB-c${CONCURRENT}.csv.zst"

    # Small 128KiB
    log "Running azs3-proxy Small 128KiB benchmark (c=$CONCURRENT)..."
    warp run "$SCRIPT_DIR/small-128KiB.yml" \
      -var Host="$HOSTPORT" \
      -var AccessKey="$ACCESS_KEY" \
      -var SecretKey="$SECRET_KEY" \
      -var Region="$REGION" \
      -var Bucket="$BUCKET" \
      -var TLS="$USE_TLS" \
      -var Concurrent="$CONCURRENT" \
      -var BenchData="proxy-small-put-128KiB-c${CONCURRENT}.csv.zst"

done

log "Done. Results in $PWD"
ls -lh proxy-*.csv.zst
