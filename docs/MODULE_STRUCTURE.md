# Module Structure & Documentation Guide

This document explains the structure of the `otel-azuregigwarm-exporter` standalone module.

## Directory Structure

```
otel-azuregigwarm-exporter/
├── README.md                           # 📖 Main module documentation
├── QUICK_START.md                      # 🚀 5-minute quick start guide
├── TESTING.md                          # 🧪 Comprehensive testing guide
├── MODULE_STRUCTURE.md                 # 📋 This file
├── .gitignore                          # 🚫 Git ignore rules
├── go.mod                              # 📦 Root module definition
│
├── exporter/
│   └── azuregigwarmexporter/           # Main exporter package
│       ├── README.md                   # 📄 Exporter-specific documentation
│       ├── go.mod                      # 📦 Exporter go module
│       ├── go.sum                      # 🔒 Dependency checksums
│       ├── metadata.yaml               # 📊 Component metadata
│       ├── Makefile                    # 🔨 Build helpers
│       │
│       ├── config.go                   # ⚙️ Configuration structures
│       ├── factory.go                  # 🏭 Exporter factory (CGO build)
│       ├── factory_nocgo.go            # 🏭 Factory stub (no CGO)
│       ├── logsexporter.go             # 📝 Logs exporter implementation
│       ├── tracesexporter.go           # 📊 Traces exporter implementation
│       │
│       ├── internal/
│       │   └── cgo/                    # 🔗 CGO/FFI bridge
│       │       ├── geneva_ffi.go       # Go FFI bindings
│       │       ├── c_helpers.c         # C helper functions
│       │       └── headers/
│       │           ├── geneva_ffi.h    # FFI function declarations
│       │           └── geneva_errors.h # Error code definitions
│       │
│       ├── geneva_ffi_bridge/          # 🦀 Rust FFI bridge
│       │   ├── README.md               # Rust bridge documentation
│       │   ├── Cargo.toml              # Rust package manifest
│       │   ├── Cargo.lock              # Rust dependency lock
│       │   ├── build.rs                # Rust build script
│       │   ├── src/
│       │   │   └── lib.rs              # Rust FFI implementation
│       │   └── target/                 # Rust build artifacts (gitignored)
│       │       └── release/
│       │           └── libgeneva_ffi_bridge.a  # Static library
│       │
│       └── testbed/                    # 🧪 Test infrastructure
│           ├── README.md
│           └── ...
│
└── examples/                           # 📚 Example configurations
    ├── builder-config.yaml             # OCB builder configuration
    ├── config.yaml                     # Sample collector config
    ├── test-logs.json                  # Sample log payload
    ├── test-traces.json                # Sample trace payload
    ├── send-test-logs.sh               # 🔨 Send test logs script
    └── send-test-traces.sh             # 🔨 Send test traces script
```

## Documentation Map

### For Users (Quick Start)

1. **Start here:** [QUICK_START.md](QUICK_START.md)
   - 5-minute setup guide
   - Prerequisites
   - Build steps
   - Quick test

2. **Then read:** [README.md](README.md)
   - Complete documentation
   - All configuration options
   - Authentication methods
   - Architecture overview

3. **For testing:** [TESTING.md](TESTING.md)
   - Local development testing
   - Integration testing
   - Load testing
   - Troubleshooting

### For Developers

1. **Exporter code:** [exporter/azuregigwarmexporter/](exporter/azuregigwarmexporter/)
   - Start with [exporter/azuregigwarmexporter/README.md](exporter/azuregigwarmexporter/README.md)
   - Configuration: `config.go`
   - Logs implementation: `logsexporter.go`
   - Traces implementation: `tracesexporter.go`

2. **FFI Bridge:** [exporter/azuregigwarmexporter/internal/cgo/](exporter/azuregigwarmexporter/internal/cgo/)
   - Go side: `geneva_ffi.go`
   - C helpers: `c_helpers.c`
   - Headers: `headers/*.h`

3. **Rust implementation:** [exporter/azuregigwarmexporter/geneva_ffi_bridge/](exporter/azuregigwarmexporter/geneva_ffi_bridge/)
   - Start with [geneva_ffi_bridge/README.md](exporter/azuregigwarmexporter/geneva_ffi_bridge/README.md)
   - Implementation: `src/lib.rs`

## Module Publishing

### Module Paths

This module uses a nested structure:

- **Root module:** `github.com/open-telemetry/otel-azuregigwarm-exporter`
- **Exporter package:** `github.com/open-telemetry/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter`

Users will reference the **exporter package** in their builder config:

```yaml
exporters:
  - gomod: github.com/open-telemetry/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter v0.1.0
```

### Publishing Workflow

1. **Prepare for first release:**
   ```bash
   cd /path/to/otel-azuregigwarm-exporter

   # Initialize git if not done
   git init
   git add .
   git commit -m "Initial commit: Azure GigWarm exporter"
   ```

2. **Create GitHub repository:**
   - Go to GitHub and create new repository
   - Name: `otel-azuregigwarm-exporter`
   - Public or Private (your choice)
   - Do NOT initialize with README (you already have one)

3. **Push to GitHub:**
   ```bash
   git remote add origin https://github.com/YOUR_ORG/otel-azuregigwarm-exporter.git
   git branch -M main
   git push -u origin main
   ```

4. **Tag first release:**
   ```bash
   # Tag the release
   git tag v0.1.0
   git push origin v0.1.0
   ```

5. **Go module becomes available:**
   - Go modules will automatically pick up your tagged release
   - Users can now reference:
     ```yaml
     - gomod: github.com/YOUR_ORG/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter v0.1.0
     ```

6. **For subsequent releases:**
   ```bash
   # Make changes
   git add .
   git commit -m "Description of changes"

   # Tag new version
   git tag v0.2.0
   git push origin main
   git push origin v0.2.0
   ```

## Testing Before Publishing

### Local Development

Test with `replace` directive before publishing:

```bash
# In your test collector directory
go mod edit -replace github.com/open-telemetry/otel-azuregigwarm-exporter=/path/to/local/otel-azuregigwarm-exporter

# Build and test
CGO_ENABLED=1 go build ./...
```

### Verify Module Structure

```bash
cd /path/to/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter
go mod verify
go mod tidy
```

## Build Requirements

### For Users Building Collectors

Users need:
- Go 1.21+
- Rust toolchain (to build the FFI bridge)
- CGO enabled

### Build Process

1. Build Rust FFI bridge:
   ```bash
   cd exporter/azuregigwarmexporter/geneva_ffi_bridge
   cargo build --release
   ```

2. Build with OpenTelemetry Collector Builder:
   ```bash
   ocb --config builder-config.yaml
   cd <generated-collector>
   CGO_ENABLED=1 go build -o collector .
   ```

## Configuration Files

### examples/builder-config.yaml

Template for building collectors that include this exporter.

**Key fields:**
- `dist.name`: Output binary name
- `exporters`: List of exporters including this one
- `receivers`: OTLP, etc.
- `processors`: batch, etc.

### examples/config.yaml

Sample runtime configuration for the collector.

**Critical fields to update:**
- `endpoint`: Your Geneva endpoint
- `account`: Your Geneva account
- `namespace`: Your Geneva namespace
- `tenant`: Your Azure tenant ID
- `role_name`: Your role
- `role_instance`: Your instance
- `cert_path` / `cert_password`: For certificate auth

## Key Features

### Authentication Methods

1. **System MSI** (`auth_method: 0`)
   - Default
   - Uses Azure Managed Identity
   - No credentials needed

2. **Certificate** (`auth_method: 1`)
   - Requires `.p12` certificate file
   - Requires certificate password

3. **Workload Identity** (`auth_method: 2`)
   - For Kubernetes workloads
   - Requires `workload_identity_resource`

### Resilience Features

1. **Persistent Queue** (via file_storage extension)
   - Write-Ahead Log
   - Survives collector restarts

2. **Two-Level Retry**
   - Export-level: Retry entire export with backoff
   - Batch-level: Retry individual failed batches

3. **Graceful Error Handling**
   - Detailed error messages from Rust layer
   - No crashes on configuration errors

## Common Tasks

### Add New Feature

1. Update Rust code in `geneva_ffi_bridge/src/lib.rs`
2. Update FFI headers in `internal/cgo/headers/`
3. Update Go bindings in `internal/cgo/geneva_ffi.go`
4. Update exporter in `logsexporter.go` / `tracesexporter.go`
5. Update `config.go` if new config fields needed
6. Update documentation in `README.md`
7. Add tests

### Update Dependencies

```bash
# Update Go dependencies
cd exporter/azuregigwarmexporter
go get -u ./...
go mod tidy

# Update Rust dependencies
cd geneva_ffi_bridge
cargo update
```

### Run Tests

```bash
# Go tests
cd exporter/azuregigwarmexporter
CGO_ENABLED=1 go test ./...

# Rust tests
cd geneva_ffi_bridge
cargo test
```

## Support & Contributing

- 📖 Read the [README.md](README.md) for usage
- 🐛 Report issues on GitHub Issues
- 💬 For questions, open a GitHub Discussion
- 🤝 For contributions, open a Pull Request

## License

Apache License 2.0 - See [LICENSE](LICENSE)

## Links

- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [OpenTelemetry Collector Builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder)
- [Geneva Documentation](https://eng.ms/docs/products/geneva)
