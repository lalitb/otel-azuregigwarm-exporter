#!/bin/bash
# Script to send test traces to OpenTelemetry Collector
# Usage: ./send-test-traces.sh

set -e

COLLECTOR_URL="${COLLECTOR_URL:-http://localhost:4318}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Generate random trace and span IDs
TRACE_ID=$(openssl rand -hex 16)
SPAN_ID=$(openssl rand -hex 8)

# Get current timestamp in nanoseconds (works on Linux and macOS)
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS doesn't support %N, use seconds and add zeros
    START_TIME=$(date +%s)000000000
else
    # Linux supports nanoseconds
    START_TIME=$(date +%s%N)
fi
END_TIME=$((START_TIME + 1000000000))  # 1 second duration

# Read the template and replace placeholders
PAYLOAD=$(cat "${SCRIPT_DIR}/test-traces.json" | \
    sed "s/TRACE_ID_PLACEHOLDER/${TRACE_ID}/g" | \
    sed "s/SPAN_ID_PLACEHOLDER/${SPAN_ID}/g" | \
    sed "s/START_TIME_PLACEHOLDER/${START_TIME}/g" | \
    sed "s/END_TIME_PLACEHOLDER/${END_TIME}/g")

echo "Sending test trace to ${COLLECTOR_URL}/v1/traces..."
echo "Trace ID: ${TRACE_ID}"
echo "Span ID: ${SPAN_ID}"

# Send the request
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${COLLECTOR_URL}/v1/traces" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD")

HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ Trace sent successfully!"
    echo "Response: $BODY"
    exit 0
else
    echo "✗ Failed to send trace (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi
