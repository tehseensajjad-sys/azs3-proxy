
#!/usr/bin/env bash
set -euo pipefail

# Source common setup functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

# ---------------------------
# Rabata-specific Config
# ---------------------------
REGION="${REGION:-south-india}"

log "Region         : $REGION"

# ---------------------------
# Initialize common setup
# ---------------------------
common_init

# ---------------------------
# Run Rabata benchmarks
# ---------------------------
# Example: Mixed 1MiB test (matches Rabata headline "Mixed ops")
log "Running Rabata Mixed 1MiB benchmark..."
warp run "$SCRIPT_DIR/mixed-1MiB.yml" \
  -var Host="$HOSTPORT" \
  -var AccessKey="$ACCESS_KEY" \
  -var SecretKey="$SECRET_KEY" \
  -var Region="$REGION" \
  -var Bucket="$BUCKET" \
  -var TLS="$USE_TLS"

# PUT throughput (Upload)
log "Running Rabata PUT 1MiB benchmark..."
warp run "$SCRIPT_DIR/put-1MiB.yml" \
  -var Host="$HOSTPORT" \
  -var AccessKey="$ACCESS_KEY" \
  -var SecretKey="$SECRET_KEY" \
  -var Region="$REGION" \
  -var Bucket="$BUCKET" \
  -var TLS="$USE_TLS"

# GET throughput (Download)
log "Running Rabata GET 1MiB benchmark..."
warp run "$SCRIPT_DIR/get-1MiB.yml" \
  -var Host="$HOSTPORT" \
  -var AccessKey="$ACCESS_KEY" \
  -var SecretKey="$SECRET_KEY" \
  -var Region="$REGION" \
  -var Bucket="$BUCKET" \
  -var TLS="$USE_TLS"

# Small‑object ops/sec
log "Running Rabata Small 128KiB benchmark..."
warp run "$SCRIPT_DIR/small-128KiB.yml" \
  -var Host="$HOSTPORT" \
  -var AccessKey="$ACCESS_KEY" \
  -var SecretKey="$SECRET_KEY" \
  -var Region="$REGION" \
  -var Bucket="$BUCKET" \
  -var TLS="$USE_TLS"

log "Done. Results in: $RUN_DIR"
ls -lh "$RUN_DIR"
