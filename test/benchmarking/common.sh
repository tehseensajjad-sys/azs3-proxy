#!/usr/bin/env bash
# Common setup and utility functions for benchmarking scripts
# This script should be sourced by other benchmark scripts, not executed directly

set -euo pipefail

# ---------------------------
# Load .env if exists
# ---------------------------
PROXY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
if [[ -f "$PROXY_ROOT/.env" ]]; then
  echo "Loading configuration from $PROXY_ROOT/.env"
  set -a
  source "$PROXY_ROOT/.env"
  set +a
fi

# ---------------------------
# Default Config (can be overridden)
# ---------------------------
ENDPOINT_URL="${PROXY_URL:-http://localhost:8080}"
ACCESS_KEY="${S3_ACCESS_KEY:?S3_ACCESS_KEY must be set}"
SECRET_KEY="${S3_SECRET_KEY:?S3_SECRET_KEY must be set}"
BUCKET="${BUCKET_NAME:-${BUCKET:-warp-benchmark-bucket}}"

# Install location
WORKDIR="${WORKDIR:-$PWD/warp_runs}"
INSTALL_DIR="${INSTALL_DIR:-$WORKDIR/bin}"
export PATH="$INSTALL_DIR:$PATH"

# ---------------------------
# Helpers
# ---------------------------
log() { echo "[$(date -Is)] $*"; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing dependency: $1"; exit 1; }
}

# WARP expects host:port (no scheme) and uses --tls to enable HTTPS.
# Convert http://localhost:8080 -> localhost:8080
endpoint_to_hostport() {
  local url="$1"
  url="${url#http://}"
  url="${url#https://}"
  url="${url%%/*}"
  echo "$url"
}

# ---------------------------
# Preflight checks
# ---------------------------
check_dependencies() {
  need_cmd curl
  need_cmd tar
  need_cmd uname
  need_cmd go

  ARCH="$(uname -m)"
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"

  if [[ "$OS" != "linux" ]]; then
    echo "This script currently targets Linux. Detected OS=$OS"
    exit 1
  fi

  case "$ARCH" in
    x86_64|amd64) WARP_ASSET="warp_Linux_x86_64.tar.gz" ;;
    aarch64|arm64) WARP_ASSET="warp_Linux_arm64.tar.gz" ;;
    *)
      echo "Unsupported arch: $ARCH (supported: x86_64/amd64, aarch64/arm64)"
      exit 1
      ;;
  esac
}

# ---------------------------
# Setup workspace
# ---------------------------
setup_workspace() {
  HOSTPORT="$(endpoint_to_hostport "$ENDPOINT_URL")"
  USE_TLS="false"
  if [[ "$ENDPOINT_URL" == https://* ]]; then
    USE_TLS="true"
  fi

  log "Endpoint URL   : $ENDPOINT_URL"
  log "WARP host:port : $HOSTPORT"
  log "TLS            : $USE_TLS"
  log "Bucket         : $BUCKET"

  mkdir -p "$WORKDIR"
  mkdir -p "$INSTALL_DIR"
  RUN_DIR="$WORKDIR/run_$(date +%Y%m%d_%H%M%S)"
  mkdir -p "$RUN_DIR"
}

# ---------------------------
# Start Proxy
# ---------------------------
start_proxy() {
  PROXY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

  log "Starting proxy using 'make run'..."

  # Check for Azure credentials
  if [[ -z "${AZURE_STORAGE_ACCOUNT:-}" ]]; then
    echo "Error: AZURE_STORAGE_ACCOUNT is not set."
    echo "Please set AZURE_STORAGE_ACCOUNT and AZURE_STORAGE_KEY (or other auth vars) before running this script."
    exit 1
  fi

  # Start in background
  (cd "$PROXY_ROOT" && make run) > "$WORKDIR/proxy.log" 2>&1 &
  MAKE_PID=$!
  log "Make run started with PID $MAKE_PID. Logs: $WORKDIR/proxy.log"

  # Wait for proxy
  log "Waiting for proxy to be ready at $ENDPOINT_URL..."
  MAX_RETRIES=30
  count=0
  while ! curl -s "$ENDPOINT_URL/health" >/dev/null; do
    sleep 1
    count=$((count+1))
    if [[ $count -ge $MAX_RETRIES ]]; then
      echo "Proxy failed to start. Check logs at $WORKDIR/proxy.log"
      cat "$WORKDIR/proxy.log"
      exit 1
    fi
  done
  log "Proxy is ready!"
}

# ---------------------------
# Monitor Proxy
# ---------------------------
start_proxy_monitor() {
  PROXY_PID=$(lsof -ti :8080 | head -n 1)
  if [[ -z "$PROXY_PID" ]]; then
    log "Error: Could not find proxy process on port 8080."
    exit 1
  fi

  log "Monitoring proxy process (PID $PROXY_PID)..."
  (
    while true; do
      if ! kill -0 "$PROXY_PID" 2>/dev/null; then
        echo ""
        log "CRITICAL: Proxy process $PROXY_PID died unexpectedly!"
        kill $$
        exit 1
      fi
      sleep 1
    done
  ) &
  MONITOR_PID=$!
}

# ---------------------------
# Cleanup
# ---------------------------
setup_cleanup() {
  trap cleanup EXIT
}

cleanup() {
  if [[ -n "${MONITOR_PID:-}" ]]; then
    kill "$MONITOR_PID" 2>/dev/null || true
  fi
  if [[ -n "${MAKE_PID:-}" ]]; then
    kill "$MAKE_PID" 2>/dev/null || true
  fi
  # Find process on port 8080 and kill it
  pid=$(lsof -ti :8080 || true)
  if [[ -n "$pid" ]]; then
    kill "$pid" 2>/dev/null || true
  fi
  if [[ -n "${TMPDIR:-}" ]]; then
    rm -rf "$TMPDIR"
  fi
}

# ---------------------------
# Ensure Bucket Exists
# ---------------------------
ensure_bucket() {
  log "Checking if bucket '$BUCKET' exists..."

  EXAMPLE_DIR="$PROXY_ROOT/examples/python-s3-client"

  log "Ensuring bucket '$BUCKET' exists using python example..."
  if command -v python3 >/dev/null 2>&1; then
    # Check if boto3 is installed
    if python3 -c "import boto3" >/dev/null 2>&1; then
      # Ensure the python script uses the same bucket as this script
      export BUCKET_NAME="$BUCKET"
      
      python3 "$EXAMPLE_DIR/create_bucket.py"
    else
      log "Warning: boto3 not found. Skipping bucket creation check. Warp might fail if bucket doesn't exist."
    fi
  else
    log "Warning: python3 not found. Skipping bucket creation check."
  fi
}

# ---------------------------
# Install/Update WARP
# ---------------------------
install_warp() {
  log "Fetching latest WARP release info..."
  # GitHub "latest release" endpoint redirects to the current tag
  LATEST_TAG="$(curl -fsSLI https://github.com/minio/warp/releases/latest | awk -F'/' '/^location:/ {print $NF}' | tr -d '\r')"
  if [[ -z "$LATEST_TAG" ]]; then
    echo "Failed to detect latest WARP tag"
    exit 1
  fi

  log "Latest WARP tag : $LATEST_TAG"

  WARP_URL="https://github.com/minio/warp/releases/download/${LATEST_TAG}/${WARP_ASSET}"
  log "Downloading     : $WARP_URL"

  TMPDIR="$(mktemp -d)"

  curl -fL "$WARP_URL" -o "$TMPDIR/$WARP_ASSET"
  tar -xzf "$TMPDIR/$WARP_ASSET" -C "$TMPDIR"

  # The tar contains a 'warp' binary
  chmod +x "$TMPDIR/warp"

  log "Installing warp -> $INSTALL_DIR/warp (may require sudo)..."
  if [[ -w "$INSTALL_DIR" ]]; then
    mv "$TMPDIR/warp" "$INSTALL_DIR/warp"
  else
    sudo mv "$TMPDIR/warp" "$INSTALL_DIR/warp"
  fi

  log "warp version:"
  warp --version || true
}

# ---------------------------
# Setup common WARP flags
# ---------------------------
setup_warp_common_flags() {
  COMMON_FLAGS=(--host "$HOSTPORT" --access-key "$ACCESS_KEY" --secret-key "$SECRET_KEY" --bucket "$BUCKET")
  if [[ "$USE_TLS" == "true" ]]; then
    COMMON_FLAGS+=(--tls)
  fi
}

# ---------------------------
# Main initialization function
# Call this from your benchmark scripts to set everything up
# ---------------------------
common_init() {
  check_dependencies
  setup_workspace
  setup_cleanup
  start_proxy
  start_proxy_monitor
  ensure_bucket
  install_warp
  setup_warp_common_flags
}
