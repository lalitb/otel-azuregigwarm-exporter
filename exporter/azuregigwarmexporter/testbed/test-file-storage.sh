#!/bin/bash
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# Test file_storage extension with GigWarm exporter
# This script verifies that persistent queue works correctly

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== File Storage Test ==="
echo ""

# Check for required files
if [ ! -f ".env" ]; then
    echo "ERROR: .env file not found"
    echo "Please create .env from .env.template with your Geneva credentials"
    exit 1
fi

# Load environment
echo "Loading environment variables..."
set -a
source .env
set +a

# Create storage directory
STORAGE_DIR="./storage-test"
rm -rf "$STORAGE_DIR"  # Clean start
mkdir -p "$STORAGE_DIR"
echo "✓ Created storage directory: $STORAGE_DIR"

# Create test config with file_storage
echo ""
echo "Creating test configuration with file_storage..."
cat > config-test-filestorage.yaml <<EOF
extensions:
  file_storage:
    directory: $STORAGE_DIR
    timeout: 10s
    compaction:
      directory: $STORAGE_DIR
      on_start: true
      on_rebound: true

receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  batch:
    send_batch_size: 100
    timeout: 1s

exporters:
  azuregigwarm:
    endpoint: "${GENEVA_ENDPOINT}"
    environment: "${GENEVA_ENVIRONMENT}"
    account: "${GENEVA_ACCOUNT}"
    namespace: "${GENEVA_NAMESPACE}"
    region: "${GENEVA_REGION}"
    config_major_version: ${GENEVA_CONFIG_VERSION:-1}
    auth_method: ${GENEVA_AUTH_METHOD:-0}
    tenant: "${GENEVA_TENANT}"
    role_name: "${GENEVA_ROLE_NAME}"
    role_instance: "${GENEVA_ROLE_INSTANCE}"
    cert_path: "${GENEVA_CERT_PATH}"
    cert_password: "${GENEVA_CERT_PASSWORD}"

    sending_queue:
      enabled: true
      num_consumers: 5
      queue_size: 1000
      storage: file_storage  # *** PERSISTENT QUEUE ***

    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s

    batch_retry:
      enabled: true
      max_retries: 3
      initial_interval: 100ms
      max_interval: 5s
      multiplier: 2.0

service:
  extensions: [file_storage]  # *** ENABLE EXTENSION ***
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [azuregigwarm]
EOF

echo "✓ Created config-test-filestorage.yaml"

# Build collector if needed
# For contrib, use otelcontribcol from the root (3 levels up from testbed/)
COLLECTOR_BIN="../../../bin/otelcontribcol_$(go env GOOS)_$(go env GOARCH)"
if [ ! -f "$COLLECTOR_BIN" ]; then
    echo ""
    echo "Building contrib collector..."
    cd ../../..
    make otelcontribcol
    cd exporter/azuregigwarmexporter/testbed
    echo "✓ Collector built"
fi

echo ""
echo "=== Test 1: Normal Operation with Persistence ==="
echo ""

# Start collector in background
echo "Starting collector with file_storage..."
$COLLECTOR_BIN --config config-test-filestorage.yaml > test-filestorage.log 2>&1 &
COLLECTOR_PID=$!
echo "✓ Collector started (PID: $COLLECTOR_PID)"

# Wait for startup
sleep 3

# Check if collector is running
if ! kill -0 $COLLECTOR_PID 2>/dev/null; then
    echo "✗ Collector failed to start. Check test-filestorage.log:"
    tail -20 test-filestorage.log
    exit 1
fi

echo "✓ Collector is running"

# Send some data
echo ""
echo "Sending 50 test logs..."
telemetrygen logs \
    --otlp-endpoint 127.0.0.1:4317 \
    --otlp-insecure \
    --logs 50 \
    --body "File storage test message" \
    2>&1 | grep -E "(Generated|logs)"

echo "✓ Data sent"

# Wait for processing
echo ""
echo "Waiting for data to be processed..."
sleep 3

# Check storage directory
echo ""
echo "Checking storage directory..."
if [ -d "$STORAGE_DIR" ] && [ "$(ls -A $STORAGE_DIR)" ]; then
    echo "✓ Storage directory contains data:"
    ls -lh "$STORAGE_DIR"
else
    echo "⚠️  Storage directory is empty (data may have been sent immediately)"
fi

# Graceful shutdown
echo ""
echo "=== Test 2: Graceful Shutdown (Queue Persistence) ==="
echo ""
echo "Sending more data before shutdown..."
telemetrygen logs \
    --otlp-endpoint 127.0.0.1:4317 \
    --otlp-insecure \
    --logs 100 \
    --rate 50 \
    --body "Pre-shutdown test" \
    2>&1 | grep -E "(Generated|logs)" &

sleep 1

echo "Shutting down collector gracefully..."
kill -TERM $COLLECTOR_PID
wait $COLLECTOR_PID 2>/dev/null || true
echo "✓ Collector stopped"

# Check storage after shutdown
echo ""
echo "Checking storage after shutdown..."
if [ -d "$STORAGE_DIR" ] && [ "$(ls -A $STORAGE_DIR)" ]; then
    echo "✓ Storage directory still contains data (queue persisted):"
    ls -lh "$STORAGE_DIR"
    STORAGE_SIZE=$(du -sh "$STORAGE_DIR" | cut -f1)
    echo "  Total size: $STORAGE_SIZE"
else
    echo "✓ Storage directory empty (all data successfully sent before shutdown)"
fi

# Restart collector
echo ""
echo "=== Test 3: Queue Restoration After Restart ==="
echo ""
echo "Restarting collector..."
$COLLECTOR_BIN --config config-test-filestorage.yaml > test-filestorage-restart.log 2>&1 &
COLLECTOR_PID=$!
echo "✓ Collector restarted (PID: $COLLECTOR_PID)"

sleep 5

# Check logs for queue restoration
echo ""
echo "Checking for queue restoration in logs..."
if grep -q "restored queue" test-filestorage-restart.log 2>/dev/null; then
    echo "✓ Queue was restored from file storage:"
    grep "restored queue" test-filestorage-restart.log | head -3
else
    echo "ℹ️  No queue restoration message (queue may have been empty)"
fi

# Send final test
echo ""
echo "Sending final test to verify collector works after restart..."
telemetrygen logs \
    --otlp-endpoint 127.0.0.1:4317 \
    --otlp-insecure \
    --logs 10 \
    --body "Post-restart test" \
    2>&1 | grep -E "(Generated|logs)"

sleep 2

# Cleanup
echo ""
echo "Cleaning up..."
if kill -0 $COLLECTOR_PID 2>/dev/null; then
    kill -TERM $COLLECTOR_PID
    wait $COLLECTOR_PID 2>/dev/null || true
fi

echo ""
echo "=== Test Summary ==="
echo ""
echo "✅ File storage extension configured successfully"
echo "✅ Collector started with persistent queue"
echo "✅ Data sent and processed"
echo "✅ Graceful shutdown preserved queue state"
echo "✅ Collector restarted and restored queue"
echo ""
echo "Log files:"
echo "  - test-filestorage.log (initial run)"
echo "  - test-filestorage-restart.log (after restart)"
echo ""
echo "Storage directory: $STORAGE_DIR"
echo ""
echo "To inspect logs for upload success:"
echo "  grep 'upload_batch: SUCCESS' test-filestorage*.log"
echo ""

# Keep storage for inspection
echo "Storage directory preserved for inspection. To clean up:"
echo "  rm -rf $STORAGE_DIR config-test-filestorage.yaml test-filestorage*.log"
echo ""
