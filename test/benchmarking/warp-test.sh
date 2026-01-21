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

for CONCURRENT in 64 16 8; do
    log "---------------------------------------------------"
    log "Starting benchmarks with Concurrency: $CONCURRENT"
    log "---------------------------------------------------"

    for TEST_FILE in "$SCRIPT_DIR"/*.yml; do
        BASENAME=$(basename "$TEST_FILE")
        # Ensure we skip if glob fails or matches non-files
        if [[ ! -f "$TEST_FILE" ]]; then continue; fi

        # Get test name without extension (e.g. get-100MiB)
        TEST_NAME="${BASENAME%.yml}"
        
        BENCH_DATA="proxy-${TEST_NAME}-c${CONCURRENT}.csv.zst"
        METRICS_FILE="${BENCH_DATA}.metrics.csv"
        
        log "Running azs3-proxy $TEST_NAME benchmark (c=$CONCURRENT)..."
        start_collecting_metrics "$METRICS_FILE"
        warp run "$TEST_FILE" \
          -var Host="$HOSTPORT" \
          -var AccessKey="$ACCESS_KEY" \
          -var SecretKey="$SECRET_KEY" \
          -var Region="$REGION" \
          -var Bucket="$BUCKET" \
          -var TLS="$USE_TLS" \
          -var Concurrent="$CONCURRENT" \
          -var BenchData="$BENCH_DATA"
        stop_collecting_metrics
    done

done

log "Done. Results in $PWD"
ls -lh proxy-*.csv.zst*

log "Generating report..."
/bin/python3 "$SCRIPT_DIR/generate_report.py" "$PWD"

