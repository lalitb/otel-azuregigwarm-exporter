#!/bin/bash
# Script to send test logs to OpenTelemetry Collector
# Usage: ./send-test-logs.sh

set -e

COLLECTOR_URL="${COLLECTOR_URL:-http://localhost:4318}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Get current timestamp in nanoseconds
TIMESTAMP=$(date +%s%N)

# Read the template and replace timestamp
PAYLOAD=$(cat "${SCRIPT_DIR}/test-logs.json" | sed "s/TIMESTAMP_PLACEHOLDER/${TIMESTAMP}/g")

echo "Sending test log to ${COLLECTOR_URL}/v1/logs..."

# Send the request
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${COLLECTOR_URL}/v1/logs" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ Log sent successfully!"
    echo "Response: $BODY"
    exit 0
else
    echo "✗ Failed to send log (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi
