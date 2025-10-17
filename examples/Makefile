.PHONY: all build build-rust build-collector run test test-logs test-traces clean install-tools help

# Configuration
RUST_FFI_DIR := ../exporter/azuregigwarmexporter/geneva_ffi_bridge
RUST_LIB_DIR := $(RUST_FFI_DIR)/target/release
COLLECTOR_BIN := ./bin/otelcol-azuregigwarm
CONFIG_FILE := config.yaml

# Default target
all: build

# Build everything (Rust FFI + Collector)
build: build-rust build-collector
	@echo "✅ Build complete!"
	@echo ""
	@echo "Run the collector with:"
	@echo "  make run"
	@echo ""
	@echo "Or directly:"
	@echo "  $(COLLECTOR_BIN) --config $(CONFIG_FILE)"

# Build Rust FFI bridge (generates both .so and .a)
build-rust:
	@echo "🦀 Building Rust FFI bridge..."
	cd $(RUST_FFI_DIR) && cargo build --release
	@echo "✓ Rust library built: $(RUST_LIB_DIR)/libgeneva_ffi_bridge.a"

# Build collector with static linking
build-collector: build-rust
	@echo "🔨 Building OpenTelemetry Collector with static linking..."
	CGO_ENABLED=1 \
	CGO_LDFLAGS="-L$(RUST_LIB_DIR) -lgeneva_ffi_bridge" \
	ocb --config builder-config.yaml
	@echo "✓ Collector built: $(COLLECTOR_BIN)"

# Run the collector
run: build
	@echo "🚀 Starting OpenTelemetry Collector..."
	@echo "   OTLP HTTP endpoint: http://localhost:4318"
	@echo "   OTLP gRPC endpoint: http://localhost:4317"
	@echo ""
	$(COLLECTOR_BIN) --config $(CONFIG_FILE)

# Run all tests (logs + traces)
test: test-logs test-traces

# Send test logs to the collector
test-logs:
	@echo "📝 Sending test logs..."
	./send-test-logs.sh

# Send test traces to the collector
test-traces:
	@echo "📊 Sending test traces..."
	./send-test-traces.sh

# Clean all build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf ./bin
	cd $(RUST_FFI_DIR) && cargo clean
	@echo "✓ Clean complete"

# Install required build tools
install-tools:
	@echo "📦 Installing required tools..."
	@command -v cargo >/dev/null 2>&1 || (echo "❌ Rust not found. Install from https://rustup.rs" && exit 1)
	@command -v go >/dev/null 2>&1 || (echo "❌ Go not found. Install from https://go.dev" && exit 1)
	@echo "Installing OpenTelemetry Collector Builder..."
	go install go.opentelemetry.io/collector/cmd/builder@latest
	@echo "✓ Tools installed"

# Check prerequisites
check-prereqs:
	@echo "🔍 Checking prerequisites..."
	@command -v cargo >/dev/null 2>&1 || (echo "❌ Rust not found. Install from https://rustup.rs" && exit 1)
	@command -v go >/dev/null 2>&1 || (echo "❌ Go not found. Install from https://go.dev" && exit 1)
	@command -v ocb >/dev/null 2>&1 || (echo "⚠️  ocb not found. Run 'make install-tools'" && exit 1)
	@echo "✓ All prerequisites met"

# Rebuild from scratch
rebuild: clean build

# Help message
help:
	@echo "Azure GigWarm Exporter - Examples Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make all              - Build everything (default)"
	@echo "  make build            - Build Rust FFI + Collector"
	@echo "  make build-rust       - Build only Rust FFI bridge"
	@echo "  make build-collector  - Build only collector"
	@echo "  make run              - Build and run collector"
	@echo "  make test             - Send test logs and traces"
	@echo "  make test-logs        - Send test logs only"
	@echo "  make test-traces      - Send test traces only"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make rebuild          - Clean and rebuild everything"
	@echo "  make install-tools    - Install required tools (ocb)"
	@echo "  make check-prereqs    - Check if prerequisites are installed"
	@echo "  make help             - Show this help message"
	@echo ""
	@echo "Quick start:"
	@echo "  1. make check-prereqs  # Verify tools are installed"
	@echo "  2. make build          # Build everything"
	@echo "  3. make run            # Start the collector"
	@echo "  4. make test           # (In another terminal) Send test data"
	@echo ""
	@echo "Configuration:"
	@echo "  Edit config.yaml to customize the collector configuration"
	@echo "  Edit builder-config.yaml to add/remove components"
