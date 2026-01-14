#!/bin/bash
set -e

# Default values
SIZE=${1:-"64M"}    # Size of the file (default 64MB)
DURATION=${2:-"30s"} # Duration of the test (default 30 seconds)
PROXY_URL="http://127.0.0.1:8080"
PROFILE_URL="http://localhost:6060/debug/pprof/profile?seconds=20"

# Check if proxy is running with pprof enabled
if ! curl -s "http://localhost:6060/debug/pprof" > /dev/null; then
    echo "Error: Proxy not running with pprof enabled on port 6060"
    exit 1
fi

echo "Starting CPU profilng test..."
echo "File Size: $SIZE"
echo "Duration: $DURATION"

# Generate random file
echo "Generating test file..."
dd if=/dev/urandom of=test_file_$SIZE.dat bs=$SIZE count=1 status=none

# Configure aws-cli
export AWS_ACCESS_KEY_ID="test-key"
export AWS_SECRET_ACCESS_KEY="test-secret"
export AWS_DEFAULT_REGION="us-east-1"

# Create bucket if not exists
aws --endpoint-url $PROXY_URL s3 mb s3://test-bucket 2>/dev/null || true

# Start load generator in background
echo "Starting load generator..."
end_time=$((SECONDS + 30))
while [ $SECONDS -lt $end_time ]; do
    aws --endpoint-url $PROXY_URL s3 cp test_file_$SIZE.dat s3://test-bucket/obj_$SECONDS >/dev/null 2>&1 &
    aws --endpoint-url $PROXY_URL s3 cp s3://test-bucket/obj_$SECONDS /dev/null >/dev/null 2>&1 &
    sleep 0.1
done &
LOAD_PID=$!

# Capture profile
echo "Capturing CPU profile (20s)..."
curl -o cpu_profile.prof "$PROFILE_URL"

# Wait for load generator
kill $LOAD_PID 2>/dev/null || true
wait $LOAD_PID 2>/dev/null || true

# Cleanup
rm test_file_$SIZE.dat

echo "Profiling complete. File saved to cpu_profile.prof"
echo "Run 'pprof -http=:8081 cpu_profile.prof' to analyze"
