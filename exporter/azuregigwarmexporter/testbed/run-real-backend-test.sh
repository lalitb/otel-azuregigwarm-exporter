#!/bin/bash
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# Run load tests against real Azure Geneva backend
# This script requires environment variables to be set (see .env.template)
#
# Usage:
#   ./run-real-backend-test.sh [TYPE] [LEVEL|COUNT]
#
# Arguments:
#   TYPE      - Type of telemetry: trace|traces|log|logs (default: trace)
#   LEVEL     - Load level or item count:
#               Preset levels (duration-based):
#                 light     - 1000 items/sec for 60s (60K total)
#                 moderate  - 5000 items/sec for 300s (1.5M total)
#                 heavy     - 10000 items/sec for 600s (6M total)
#                 stress    - 50000 items/sec for 600s (30M total)
#               Count mode (exact number):
#                 Any number - Send exactly that many items at 1000/sec
#
# Examples:
#   # Duration-based (preset levels)
#   ./run-real-backend-test.sh trace light          # 60K traces in 60s
#   ./run-real-backend-test.sh logs moderate        # 1.5M logs in 300s
#
#   # Count-based (exact number of items)
#   ./run-real-backend-test.sh trace 50000          # Exactly 50K traces (takes 50s)
#   ./run-real-backend-test.sh logs 10000           # Exactly 10K logs (takes 10s)
#   ./run-real-backend-test.sh trace 100000         # Exactly 100K traces (takes 100s)
#
# Environment:
#   Set credentials and config in .env file (see .env.template)
#   TEST_SCENARIO can be set in .env or will auto-generate based on TYPE
#
# Output:
#   - Collector logs: collector.log
#   - Test results displayed at end
#   - Query Geneva using test.scenario and test.timestamp attributes

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Show help if requested
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    cat << 'EOF'
Azure GigWarm Real Backend Load Test

Usage:
  ./run-real-backend-test.sh [TYPE] [LEVEL|COUNT]

Arguments:
  TYPE      - Type of telemetry: trace|traces|log|logs (default: trace)
  LEVEL     - Load level or item count:
              Preset levels (duration-based):
                light     - 1000 items/sec for 60s (60K total)
                moderate  - 5000 items/sec for 300s (1.5M total)
                heavy     - 10000 items/sec for 600s (6M total)
                stress    - 50000 items/sec for 600s (30M total)
              Count mode (exact number):
                Any number - Send exactly that many items at 1000/sec

Examples:
  # Duration-based (preset levels)
  ./run-real-backend-test.sh trace light          # 60K traces in 60s
  ./run-real-backend-test.sh logs moderate        # 1.5M logs in 300s
  ./run-real-backend-test.sh trace heavy          # 6M traces in 600s

  # Count-based (exact number of items)
  ./run-real-backend-test.sh trace 50000          # Exactly 50K traces (50s @ 1000/sec)
  ./run-real-backend-test.sh logs 10000           # Exactly 10K logs (10s @ 1000/sec)
  ./run-real-backend-test.sh trace 100000         # Exactly 100K traces (100s @ 1000/sec)

Environment:
  Set credentials and config in .env file (see .env.template)
  TEST_SCENARIO can be set in .env or will auto-generate based on TYPE:
    - Traces: "gigwarm-traces-loadtest"
    - Logs:   "gigwarm-logs-loadtest"

Output:
  - Collector logs: collector.log
  - Test results displayed at end
  - Query Geneva using test.scenario and test.timestamp attributes

Geneva Query Example:
  source
  | where test_scenario == "gigwarm-traces-loadtest"
  | summarize count()

EOF
    exit 0
fi

echo "=== GigWarm Real Backend Load Test ==="
echo ""

# Check for .env file
if [ ! -f ".env" ]; then
    echo "ERROR: .env file not found"
    echo ""
    echo "Please create .env from .env.template:"
    echo "  cp .env.template .env"
    echo "  # Edit .env with your Geneva credentials"
    echo ""
    exit 1
fi

# Load environment variables
echo "Loading environment variables from .env..."
set -a
source .env
set +a

# Validate required variables
REQUIRED_VARS=(
    "GENEVA_ENDPOINT"
    "GENEVA_ENVIRONMENT"
    "GENEVA_ACCOUNT"
    "GENEVA_NAMESPACE"
    "GENEVA_REGION"
    "GENEVA_TENANT"
    "GENEVA_ROLE_NAME"
    "GENEVA_ROLE_INSTANCE"
    "GENEVA_AUTH_METHOD"
)

MISSING_VARS=()
for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var}" ]; then
        MISSING_VARS+=("$var")
    fi
done

if [ ${#MISSING_VARS[@]} -ne 0 ]; then
    echo "ERROR: Missing required environment variables:"
    for var in "${MISSING_VARS[@]}"; do
        echo "  - $var"
    done
    echo ""
    echo "Please set these in your .env file"
    exit 1
fi

# Validate auth method and cert if needed
if [ "$GENEVA_AUTH_METHOD" = "1" ]; then
    if [ -z "$GENEVA_CERT_PATH" ] || [ ! -f "$GENEVA_CERT_PATH" ]; then
        echo "ERROR: Certificate authentication selected but GENEVA_CERT_PATH is not set or file doesn't exist"
        echo "  GENEVA_CERT_PATH: $GENEVA_CERT_PATH"
        exit 1
    fi
    echo "✓ Using certificate authentication: $GENEVA_CERT_PATH"
else
    echo "✓ Using MSI authentication"
fi

# Set test timestamp
export TEST_TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build collector if not already built or if make is requested
# Check if COLLECTOR_BIN is already set (e.g., to use core collector)
if [ -z "$COLLECTOR_BIN" ]; then
    # For contrib, use otelcontribcol from the root (3 levels up from testbed/)
    COLLECTOR_BIN="../../../bin/otelcontribcol_$(go env GOOS)_$(go env GOARCH)"
fi

if [ ! -f "$COLLECTOR_BIN" ] || [ "$1" == "--build" ]; then
    # Only build if COLLECTOR_BIN wasn't set externally
    if [[ "$COLLECTOR_BIN" == *"otelcontribcol"* ]]; then
        echo ""
        echo "Building contrib collector with GigWarm exporter..."
        cd ../../..
        make otelcontribcol
        cd exporter/azuregigwarmexporter/testbed
        echo "✓ Collector built: $COLLECTOR_BIN"
    else
        echo ""
        echo "✓ Using external collector: $COLLECTOR_BIN"
    fi
fi

echo ""
echo "=== Configuration Summary ==="
echo "Endpoint: $GENEVA_ENDPOINT"
echo "Account: $GENEVA_ACCOUNT"
echo "Namespace: $GENEVA_NAMESPACE"
echo "Region: $GENEVA_REGION"
echo "Auth Method: $([ "$GENEVA_AUTH_METHOD" = "0" ] && echo "MSI" || echo "Certificate")"
echo "Test Scenario: ${TEST_SCENARIO:-real-backend-loadtest}"
echo "Test Timestamp: $TEST_TIMESTAMP"
echo ""

# Parse command line arguments
TEST_TYPE="${1:-trace}"
LOAD_LEVEL="${2:-moderate}"

# Set scenario name based on test type (only if not set in .env)
if [ -z "$TEST_SCENARIO" ]; then
    case "$TEST_TYPE" in
        trace|traces)
            TEST_SCENARIO="gigwarm-traces-loadtest"
            ;;
        log|logs)
            TEST_SCENARIO="gigwarm-logs-loadtest"
            ;;
        *)
            TEST_SCENARIO="gigwarm-loadtest"
            ;;
    esac
fi
export TEST_SCENARIO

# Check if LOAD_LEVEL is a number (count mode) or a preset (duration mode)
if [[ "$LOAD_LEVEL" =~ ^[0-9]+$ ]]; then
    # Count mode: user specified exact number of items
    TOTAL_ITEMS="$LOAD_LEVEL"
    ITEMS_PER_SEC=1000  # Default rate
    DURATION=$((TOTAL_ITEMS / ITEMS_PER_SEC))
    USE_COUNT_MODE=true
    echo "Test Type: $TEST_TYPE"
    echo "Mode: Count-based"
    echo "Total Items: $TOTAL_ITEMS"
    echo "Rate: $ITEMS_PER_SEC items/sec"
    echo "Estimated Duration: ${DURATION}s"
    echo ""
else
    # Duration mode: use preset levels
    USE_COUNT_MODE=false
    case "$LOAD_LEVEL" in
        light)
            ITEMS_PER_SEC=1000
            DURATION=60
            ;;
        moderate)
            ITEMS_PER_SEC=5000
            DURATION=300
            ;;
        heavy)
            ITEMS_PER_SEC=10000
            DURATION=600
            ;;
        stress)
            ITEMS_PER_SEC=50000
            DURATION=600
            ;;
        *)
            echo "Unknown load level: $LOAD_LEVEL"
            echo "Usage: $0 [trace|logs] [light|moderate|heavy|stress|COUNT]"
            echo "  Examples:"
            echo "    $0 trace light          # 1000/sec for 60s"
            echo "    $0 logs 50000           # Send exactly 50000 logs"
            exit 1
            ;;
    esac
    echo "Test Type: $TEST_TYPE"
    echo "Load Level: $LOAD_LEVEL ($ITEMS_PER_SEC items/sec for $DURATION seconds)"
    echo ""
fi

# Generate config from template
CONFIG_FILE="config-real-backend.yaml"
echo "Generating config from template..."

# Set storage directory (supports env override)
STORAGE_DIR="${OTEL_FILE_STORAGE_DIR:-./storage}"

# Use sed to replace environment variables (works without envsubst)
sed -e "s|\${GENEVA_ENDPOINT}|${GENEVA_ENDPOINT}|g" \
    -e "s|\${GENEVA_ENVIRONMENT}|${GENEVA_ENVIRONMENT}|g" \
    -e "s|\${GENEVA_ACCOUNT}|${GENEVA_ACCOUNT}|g" \
    -e "s|\${GENEVA_NAMESPACE}|${GENEVA_NAMESPACE}|g" \
    -e "s|\${GENEVA_REGION}|${GENEVA_REGION}|g" \
    -e "s|\${GENEVA_CONFIG_VERSION:-1}|${GENEVA_CONFIG_VERSION:-1}|g" \
    -e "s|\${GENEVA_AUTH_METHOD:-0}|${GENEVA_AUTH_METHOD:-0}|g" \
    -e "s|\${GENEVA_TENANT}|${GENEVA_TENANT}|g" \
    -e "s|\${GENEVA_ROLE_NAME}|${GENEVA_ROLE_NAME}|g" \
    -e "s|\${GENEVA_ROLE_INSTANCE}|${GENEVA_ROLE_INSTANCE}|g" \
    -e "s|\${GENEVA_CERT_PATH}|${GENEVA_CERT_PATH}|g" \
    -e "s|\${GENEVA_CERT_PASSWORD}|${GENEVA_CERT_PASSWORD}|g" \
    -e "s|\\\${OTEL_FILE_STORAGE_DIR:-\./storage}|${STORAGE_DIR}|g" \
    -e "s|\${TEST_SCENARIO:-loadtest}|${TEST_SCENARIO:-real-backend-loadtest}|g" \
    -e "s|\${TEST_TIMESTAMP}|${TEST_TIMESTAMP}|g" \
    config-real-backend.yaml.template > "$CONFIG_FILE"

echo "✓ Generated config: $CONFIG_FILE"

# Create storage directory for persistent queue (use same as in config)
mkdir -p "$STORAGE_DIR"
chmod 755 "$STORAGE_DIR"
echo "✓ Created storage directory: $STORAGE_DIR"

# Start the collector in background
echo ""
echo "Starting collector..."
COLLECTOR_BIN="../../../bin/otelcontribcol_$(go env GOOS)_$(go env GOARCH)"
$COLLECTOR_BIN --config "$CONFIG_FILE" > collector.log 2>&1 &
COLLECTOR_PID=$!
echo "✓ Collector started (PID: $COLLECTOR_PID)"

# Wait for collector to be ready
echo "Waiting for collector to be ready..."
sleep 5

# Check if collector is still running
if ! kill -0 $COLLECTOR_PID 2>/dev/null; then
    echo "✗ Collector failed to start. Check collector.log:"
    tail -20 collector.log
    exit 1
fi
echo "✓ Collector is ready"

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "Cleaning up..."
    if [ -n "$COLLECTOR_PID" ] && kill -0 $COLLECTOR_PID 2>/dev/null; then
        echo "Stopping collector (PID: $COLLECTOR_PID)..."
        kill $COLLECTOR_PID
        wait $COLLECTOR_PID 2>/dev/null || true
    fi
    echo "✓ Cleanup complete"

    echo ""
    echo "Collector log (last 50 lines):"
    tail -50 collector.log
}
trap cleanup EXIT INT TERM

# Check if telemetrygen is available
if ! command -v telemetrygen &> /dev/null; then
    echo "ERROR: telemetrygen not found"
    echo "Install with: go install github.com/open-telemetry/opentelemetry-collector-contrib/cmd/telemetrygen@latest"
    exit 1
fi

# Run the load test
echo ""
echo "=== Starting Load Test ==="
if [ "$USE_COUNT_MODE" = true ]; then
    echo "Total Items: $TOTAL_ITEMS"
    echo "Rate: $ITEMS_PER_SEC items/sec"
    echo "Estimated Duration: ${DURATION}s"
else
    echo "Duration: ${DURATION}s"
    echo "Rate: $ITEMS_PER_SEC items/sec"
fi
echo ""

START_TIME=$(date +%s)

case "$TEST_TYPE" in
    trace|traces)
        if [ "$USE_COUNT_MODE" = true ]; then
            telemetrygen traces \
                --otlp-endpoint 127.0.0.1:4317 \
                --otlp-insecure \
                --rate $ITEMS_PER_SEC \
                --traces $TOTAL_ITEMS \
                --telemetry-attributes test.scenario=\"${TEST_SCENARIO}\" \
                --telemetry-attributes test.timestamp=\"${TEST_TIMESTAMP}\" \
                --status-code Ok
        else
            telemetrygen traces \
                --otlp-endpoint 127.0.0.1:4317 \
                --otlp-insecure \
                --rate $ITEMS_PER_SEC \
                --duration ${DURATION}s \
                --telemetry-attributes test.scenario=\"${TEST_SCENARIO}\" \
                --telemetry-attributes test.timestamp=\"${TEST_TIMESTAMP}\" \
                --status-code Ok
        fi
        ;;
    log|logs)
        if [ "$USE_COUNT_MODE" = true ]; then
            telemetrygen logs \
                --otlp-endpoint 127.0.0.1:4317 \
                --otlp-insecure \
                --rate $ITEMS_PER_SEC \
                --logs $TOTAL_ITEMS \
                --telemetry-attributes test.scenario=\"${TEST_SCENARIO}\" \
                --telemetry-attributes test.timestamp=\"${TEST_TIMESTAMP}\" \
                --body "Load test message from GigWarm testbed"
        else
            telemetrygen logs \
                --otlp-endpoint 127.0.0.1:4317 \
                --otlp-insecure \
                --rate $ITEMS_PER_SEC \
                --duration ${DURATION}s \
                --telemetry-attributes test.scenario=\"${TEST_SCENARIO}\" \
                --telemetry-attributes test.timestamp=\"${TEST_TIMESTAMP}\" \
                --body "Load test message from GigWarm testbed"
        fi
        ;;
    *)
        echo "Unknown test type: $TEST_TYPE"
        echo "Usage: $0 [trace|logs] [light|moderate|heavy|stress|COUNT]"
        exit 1
        ;;
esac

END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

echo ""
echo "=== Test Complete ==="
echo "Total Duration: ${ELAPSED}s"
if [ "$USE_COUNT_MODE" = true ]; then
    echo "Items Sent: $TOTAL_ITEMS"
else
    echo "Items Sent: $((ITEMS_PER_SEC * ELAPSED))"
fi
echo "Average Rate: $((ITEMS_PER_SEC)) items/sec"
echo ""
echo "Check your Azure Geneva dashboard to verify data arrival"
echo "Test identifiers:"
echo "  - test.scenario: ${TEST_SCENARIO}"
echo "  - test.timestamp: ${TEST_TIMESTAMP}"
echo ""
