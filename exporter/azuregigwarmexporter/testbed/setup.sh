#!/bin/bash
# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# Setup script for GigWarm load testing

set -e

echo "=== GigWarm Exporter Load Test Setup ==="
echo ""

# Check if we're in the right directory
if [ ! -d "../exporter/azuregigwarmexporter" ]; then
    echo "ERROR: Must run from testbed directory"
    echo "Usage: cd gigwarmexporter/opentelemetry-collector/testbed && ./setup.sh"
    exit 1
fi

# Build the collector with GigWarm support
echo "Step 1: Building collector with GigWarm exporter..."
cd ..
if make AZUREGIGWARM=1 otelcorecol; then
    echo "✓ Collector built successfully"
else
    echo "✗ Failed to build collector"
    exit 1
fi

# Check if telemetrygen is available
echo ""
echo "Step 2: Checking for telemetrygen (optional)..."
if command -v telemetrygen &> /dev/null; then
    echo "✓ telemetrygen found: $(which telemetrygen)"
else
    echo "⚠ telemetrygen not found (optional for manual testing)"
    echo "  Install with: go install github.com/open-telemetry/opentelemetry-collector-contrib/cmd/telemetrygen@latest"
fi

# Return to testbed directory
cd testbed

echo ""
echo "Step 3: Downloading Go dependencies..."
if go mod download; then
    echo "✓ Dependencies downloaded"
else
    echo "✗ Failed to download dependencies"
    exit 1
fi

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Available commands:"
echo "  make list-tests              - List all available tests"
echo "  make run-gigwarm-tests       - Run all GigWarm tests"
echo "  make run-trace-tests         - Run trace load tests"
echo "  make run-log-tests           - Run log load tests"
echo "  make run-high-throughput-test - Run stress test"
echo ""
echo "Example:"
echo "  make run-trace-tests"
echo ""
