# Quick Start Guide

Get started with the Azure GigWarm exporter in 5 minutes.

## Directory Structure

```
otel-azuregigwarm-exporter/
├── exporter/
│   └── azuregigwarmexporter/          # Main exporter code
│       ├── config.go                   # Configuration
│       ├── factory.go                  # Exporter factory
│       ├── logsexporter.go             # Logs exporter
│       ├── tracesexporter.go           # Traces exporter
│       ├── geneva_ffi_bridge/          # Rust FFI bridge
│       │   ├── Cargo.toml
│       │   └── src/lib.rs
│       └── internal/cgo/               # CGO bindings
│           ├── geneva_ffi.go
│           ├── c_helpers.c
│           └── headers/
│               ├── geneva_ffi.h
│               └── geneva_errors.h
├── examples/                           # Example configurations and test scripts
│   ├── builder-config.yaml             # OCB configuration
│   ├── config.yaml                     # Collector configuration
│   ├── test-logs.json                  # Sample log payload
│   ├── test-traces.json                # Sample trace payload
│   ├── send-test-logs.sh               # Script to send test logs
│   └── send-test-traces.sh             # Script to send test traces
├── README.md                           # Full documentation
├── TESTING.md                          # Detailed testing guide
└── go.mod                              # Go module definition
```

## Prerequisites

Install required tools:

```bash
# Go 1.21+
go version

# Rust toolchain
rustc --version

# OpenTelemetry Collector Builder
go install go.opentelemetry.io/collector/cmd/builder@latest

# For testing
which curl openssl
```

## Step 1: Build the Rust FFI Bridge

```bash
cd exporter/azuregigwarmexporter/geneva_ffi_bridge
cargo build --release
cd ../../..
```

**Expected output:**
```
   Compiling geneva_ffi_bridge v0.1.0
    Finished release [optimized] target(s) in 27.65s
```

## Step 2: Verify Go Build

```bash
cd exporter/azuregigwarmexporter
CGO_ENABLED=1 go build ./...
cd ../..
```

**Expected:** No errors

## Step 3: Create a Test Collector

### Create build configuration

```bash
mkdir test-build
cd test-build
```

Create `builder-config.yaml`:

```yaml
dist:
  name: test-collector
  output_path: ./bin

exporters:
  - gomod: github.com/open-telemetry/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter v0.1.0
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.137.0

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.137.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.137.0
```

### Build with local module

```bash
# Generate collector code
ocb --config builder-config.yaml

# Use local module for testing
cd test-collector
go mod edit -replace github.com/open-telemetry/otel-azuregigwarm-exporter=/path/to/otel-azuregigwarm-exporter

# Build
CGO_ENABLED=1 go build -o ../bin/test-collector .
cd ..
```

## Step 4: Configure

Copy example config and update:

```bash
cp ../examples/config.yaml config.yaml
```

Update these critical fields in `config.yaml`:

```yaml
exporters:
  azuregigwarm:
    endpoint: "https://gcs.ppe.monitoring.core.windows.net"  # Your Geneva endpoint
    account: "YourAccount"                                    # UPDATE THIS
    namespace: "YourNamespace"                                # UPDATE THIS
    tenant: "your-tenant-id"                                  # UPDATE THIS
    role_name: "your-role"                                    # UPDATE THIS
    role_instance: "instance-1"                               # UPDATE THIS

    # For certificate auth
    auth_method: 1
    cert_path: "/path/to/cert.p12"                           # UPDATE THIS
    cert_password: "your-password"                            # UPDATE THIS
```

## Step 5: Run

```bash
./bin/test-collector --config config.yaml
```

**Expected output:**
```
2025-10-16T16:07:14.688-0700    info    Starting otelcol...
2025-10-16T16:07:14.688-0700    info    Starting AzureGigWarm exporter
2025-10-16T16:07:14.689-0700    info    Everything is ready. Begin running and processing data.
```

## Step 6: Test

In a new terminal:

```bash
cd ../examples

# Send test log
./send-test-logs.sh

# Send test trace
./send-test-traces.sh
```

**Expected output:**
```
✓ Log sent successfully!
Response: {"partialSuccess":{}}
```

Check collector terminal for export confirmation.

## Quick Troubleshooting

### "library not found"

```bash
# Build Rust FFI bridge
cd exporter/azuregigwarmexporter/geneva_ffi_bridge
cargo build --release
```

### "Certificate error"

Update `config.yaml`:
```yaml
cert_path: "/correct/path/to/cert.p12"
```

Or use MSI:
```yaml
auth_method: 0  # System MSI (no cert needed)
```

### "Connection refused"

Check endpoint and network:
```bash
curl -v https://gcs.ppe.monitoring.core.windows.net
```

## Next Steps

1. ✅ **Read** [README.md](README.md) for full documentation
2. ✅ **Read** [TESTING.md](TESTING.md) for comprehensive testing guide
3. ✅ **Publish** your module to make it available to others
4. ✅ **Deploy** to your environment with real credentials

## Publishing Your Module

When ready to publish:

```bash
# Tag a release
git tag v0.1.0
git push origin v0.1.0

# GitHub will create a release
# Go modules will be available at:
# github.com/yourorg/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter v0.1.0
```

Then others can use it in their `builder-config.yaml`:

```yaml
exporters:
  - gomod: github.com/yourorg/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter v0.1.0
```

## Support

- 📖 Read the full [README.md](README.md)
- 🧪 Check [TESTING.md](TESTING.md) for testing details
- 🐛 Report issues on GitHub
- 📝 Check example configs in `examples/`

## Common Commands Reference

```bash
# Build Rust bridge
cd exporter/azuregigwarmexporter/geneva_ffi_bridge && cargo build --release

# Test Go build
cd exporter/azuregigwarmexporter && CGO_ENABLED=1 go build ./...

# Build collector with local module
ocb --config builder-config.yaml
cd test-collector
go mod edit -replace github.com/open-telemetry/otel-azuregigwarm-exporter=/path/to/module
CGO_ENABLED=1 go build -o ../bin/test-collector .

# Run collector
./bin/test-collector --config config.yaml

# Send test data
./examples/send-test-logs.sh
./examples/send-test-traces.sh
```

You're all set! 🎉
