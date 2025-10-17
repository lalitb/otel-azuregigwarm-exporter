# Azure GIG/warm Exporter for OpenTelemetry Collector

This exporter sends OpenTelemetry logs and traces to Azure Geneva Warm Path using a Rust FFI bridge.

## Features

- ✅ Export OTLP logs to Geneva Warm
- ✅ Export OTLP traces to Geneva Warm
- ✅ Multiple authentication methods (MSI, Certificate, Workload Identity)
- ✅ Configurable retry and batching
- ✅ Rust FFI bridge for high-performance encoding and upload

## Prerequisites

- Go 1.21 or later
- Rust toolchain (for building the FFI bridge)
- CGO enabled (`CGO_ENABLED=1`)

## Installation

### As a Go Module

Add to your `go.mod`:

```go
require github.com/open-telemetry/otel-azuregigwarm-exporter v0.1.0
```

### In OpenTelemetry Collector Builder

Add to your `builder-config.yaml`:

```yaml
exporters:
  - gomod: github.com/open-telemetry/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter v0.1.0
```

## Building the Rust FFI Bridge

Before building the collector, you must build the Rust FFI bridge:

```bash
cd exporter/azuregigwarmexporter/geneva_ffi_bridge
cargo build --release
cd ../../..
```

## Configuration

### Basic Configuration

```yaml
exporters:
  azuregigwarm:
    endpoint: "https://abc.windows.net"
    environment: "Production" or "Test" or ...
    account: "YourAccount"
    namespace: "YourNamespace"
    region: "eastus"
    config_major_version: 2
    tenant: "your-tenant-id"
    role_name: "your-role"
    role_instance: "instance-1"
    auth_method: 0  # 0=MSI, 1=Certificate, 2=WorkloadIdentity

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [azuregigwarm]
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [azuregigwarm]
```

### Authentication Methods

#### System Managed Identity (MSI)

```yaml
exporters:
  azuregigwarm:
    auth_method: 0
    # No additional configuration needed
```

#### Certificate Authentication

```yaml
exporters:
  azuregigwarm:
    auth_method: 1
    cert_path: "/path/to/certificate.p12"
    cert_password: "your-password"
```

#### Workload Identity

```yaml
exporters:
  azuregigwarm:
    auth_method: 2
    workload_identity_resource: "https://monitor.azure.com"
```

### Advanced Configuration

```yaml
exporters:
  azuregigwarm:
    endpoint: "https://gcs.prod.monitoring.core.windows.net"
    environment: "Production"
    account: "YourAccount"
    namespace: "YourNamespace"
    region: "eastus"
    config_major_version: 2
    tenant: "your-tenant-id"
    role_name: "your-role"
    role_instance: "instance-1"
    auth_method: 0

    # Queue configuration
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 1000

    # Retry configuration
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s

    # Batch retry configuration (for individual batches)
    batch_retry:
      enabled: true
      max_retries: 3
      initial_interval: 100ms
      max_interval: 5s
      multiplier: 2.0
```

## Building a Collector with This Exporter

For a complete working example with all configuration files and test scripts, see:

**📁 [examples/](examples/)** - Complete examples and helper scripts

**📘 [examples/README.md](examples/README.md)** - Detailed quick start guide

### Quick Build Steps

1. Build the Rust FFI bridge:
   ```bash
   cd exporter/azuregigwarmexporter/geneva_ffi_bridge
   cargo build --release
   ```

2. Use OpenTelemetry Collector Builder:
   ```bash
   ocb --config examples/builder-config.yaml
   ```

3. Run the collector:
   ```bash
   export CGO_ENABLED=1
   export LD_LIBRARY_PATH=./exporter/azuregigwarmexporter/geneva_ffi_bridge/target/release
   ./examples/bin/otelcol-azuregigwarm --config examples/config.yaml
   ```

See [examples/README.md](examples/README.md) for detailed instructions and troubleshooting.

## Testing

### Send Test Logs

```bash
curl -X POST http://localhost:4318/v1/logs \
  -H "Content-Type: application/json" \
  -d '{
    "resourceLogs": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": {"stringValue": "test-service"}
        }]
      },
      "scopeLogs": [{
        "scope": {"name": "test"},
        "logRecords": [{
          "timeUnixNano": "'$(date +%s%N)'",
          "severityText": "INFO",
          "body": {"stringValue": "Test log message"}
        }]
      }]
    }]
  }'
```

### Send Test Traces

```bash
curl -X POST http://localhost:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{
    "resourceSpans": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": {"stringValue": "test-service"}
        }]
      },
      "scopeSpans": [{
        "scope": {"name": "test"},
        "spans": [{
          "traceId": "5b8aa5a2d2c872e8321cf37308d69df2",
          "spanId": "051581bf3cb55c13",
          "name": "test-span",
          "kind": 1,
          "startTimeUnixNano": "'$(date +%s%N)'",
          "endTimeUnixNano": "'$(date +%s%N)'"
        }]
      }]
    }]
  }'
```

## Deploying to Azure Kubernetes Service (AKS)

For a complete guide on deploying this exporter to AKS with Azure Workload Identity authentication, see:

**📘 [AKS_DEPLOYMENT.md](docs/AKS_DEPLOYMENT.md)**

The guide covers:
- Setting up Workload Identity on AKS
- Creating and configuring Managed Identity
- Registering with Geneva
- Building and deploying Docker images
- Complete Kubernetes manifests
- Troubleshooting common issues

## Architecture

```
┌─────────────────────────────────────────────┐
│   OpenTelemetry Collector                  │
│                                             │
│  ┌───────────────────────────────────────┐ │
│  │  Azure GigWarm Exporter (Go)          │ │
│  │                                        │ │
│  │  ┌──────────────────────────────────┐ │ │
│  │  │  CGO Bridge                      │ │ │
│  │  └──────────────────────────────────┘ │ │
│  │              │                         │ │
│  │              ▼                         │ │
│  │  ┌──────────────────────────────────┐ │ │
│  │  │  Rust FFI Bridge                 │ │ │
│  │  │  - OTLP parsing                  │ │ │
│  │  │  - Geneva encoding               │ │ │
│  │  │  - Compression (gzip/deflate)    │ │ │
│  │  │  - HTTP upload                   │ │ │
│  │  └──────────────────────────────────┘ │ │
│  └───────────────────────────────────────┘ │
│                  │                          │
│                  ▼                          │
│     Azure Geneva Warm Path                 │
└─────────────────────────────────────────────┘
```

## Error Handling

The exporter provides graceful error handling with detailed error messages from the Rust layer:

- Configuration errors (missing fields, invalid auth)
- Connection errors
- Upload failures
- Encoding errors

All errors are properly propagated through the FFI boundary with descriptive messages.

## Development

### Project Structure

```
otel-azuregigwarm-exporter/
├── exporter/
│   └── azuregigwarmexporter/
│       ├── config.go              # Configuration
│       ├── factory.go             # Exporter factory
│       ├── logsexporter.go        # Logs exporter
│       ├── tracesexporter.go      # Traces exporter
│       ├── geneva_ffi_bridge/     # Rust FFI bridge
│       │   ├── Cargo.toml
│       │   └── src/
│       │       └── lib.rs
│       └── internal/
│           └── cgo/
│               ├── geneva_ffi.go  # Go FFI bindings
│               ├── c_helpers.c    # C helper functions
│               └── headers/
│                   ├── geneva_ffi.h
│                   └── geneva_errors.h
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

### Running Tests

```bash
cd exporter/azuregigwarmexporter
go test ./...
```

### Local Development

For local development and testing before publishing:

```bash
# In your test collector directory
go mod edit -replace github.com/open-telemetry/otel-azuregigwarm-exporter=/path/to/local/otel-azuregigwarm-exporter
```

## Contributing

Contributions are welcome! Please open an issue or pull request.

## Support

For issues and questions:
- Open an issue on GitHub
- Check existing issues and documentation

## References

- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [OTLP Specification](https://github.com/open-telemetry/opentelemetry-proto)
