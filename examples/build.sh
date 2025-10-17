#!/bin/bash
# Build script for Azure GigWarm Exporter
# This script builds the Rust FFI bridge and OpenTelemetry Collector with static linking

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
RUST_FFI_DIR="../exporter/azuregigwarmexporter/geneva_ffi_bridge"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${BLUE}🔧 Azure GigWarm Exporter Build Script${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

if ! command -v cargo &> /dev/null; then
    echo -e "${RED}❌ Rust not found. Please install from https://rustup.rs${NC}"
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go not found. Please install from https://go.dev${NC}"
    exit 1
fi

if ! command -v builder &> /dev/null; then
    echo -e "${YELLOW}⚠️  OpenTelemetry Collector Builder not found.${NC}"
    echo -e "Installing builder..."
    go install go.opentelemetry.io/collector/cmd/builder@latest
fi

echo -e "${GREEN}✓ Prerequisites met${NC}"
echo ""

# Build Rust FFI bridge
echo -e "${BLUE}🦀 Building Rust FFI bridge...${NC}"
cd "${SCRIPT_DIR}/${RUST_FFI_DIR}"
cargo build --release

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Rust library built successfully${NC}"
    echo -e "  Static library: ${SCRIPT_DIR}/${RUST_FFI_DIR}/target/release/libgeneva_ffi_bridge.a"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo -e "  Dynamic library: ${SCRIPT_DIR}/${RUST_FFI_DIR}/target/release/libgeneva_ffi_bridge.dylib"
    else
        echo -e "  Dynamic library: ${SCRIPT_DIR}/${RUST_FFI_DIR}/target/release/libgeneva_ffi_bridge.so"
    fi
else
    echo -e "${RED}❌ Rust build failed${NC}"
    exit 1
fi
echo ""

# Build collector
cd "${SCRIPT_DIR}"
echo -e "${BLUE}🔨 Building OpenTelemetry Collector with static linking...${NC}"

export CGO_ENABLED=1
export CGO_LDFLAGS="-L${SCRIPT_DIR}/${RUST_FFI_DIR}/target/release -lgeneva_ffi_bridge"

builder --config builder-config.yaml

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Collector built successfully${NC}"
    echo -e "  Binary: ${SCRIPT_DIR}/bin/otelcol-azuregigwarm"
else
    echo -e "${RED}❌ Collector build failed${NC}"
    exit 1
fi
echo ""

# Success message
echo -e "${GREEN}✅ Build complete!${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Edit config.yaml to configure your Geneva account details"
echo -e "  2. Run the collector:"
echo -e "     ${BLUE}./bin/otelcol-azuregigwarm --config config.yaml${NC}"
echo -e "  3. Send test data:"
echo -e "     ${BLUE}./send-test-logs.sh${NC}"
echo -e "     ${BLUE}./send-test-traces.sh${NC}"
echo ""
echo -e "${YELLOW}Or use the Makefile:${NC}"
echo -e "     ${BLUE}make run${NC}         # Build and run"
echo -e "     ${BLUE}make test${NC}        # Send test data"
echo ""
