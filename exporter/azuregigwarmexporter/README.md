# Azure GigWarm (Geneva Warm) Exporter

| Status        |           |
| ------------- |-----------|
| Stability     | [alpha]: traces, logs   |
| Module        | `github.com/open-telemetry/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter` |

[alpha]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#alpha

This exporter sends OpenTelemetry traces and logs to [Azure Geneva Warm (GigWarm)](https://eng.ms/docs/products/geneva/collect/instrument/opentelemetryotlp) using a Rust FFI bridge for high-performance encoding and upload.

**Note**: This is a standalone module. See the [root README](../../README.md) for complete documentation and installation instructions.

## Prerequisites

**Important**: This exporter requires CGO to be enabled due to its Rust FFI bridge dependency.

- Go 1.24+
- Rust toolchain (for building the FFI bridge)
- CGO enabled (`CGO_ENABLED=1`)

## Configuration

The Azure GigWarm Exporter requires the following configuration parameters:

### Required Parameters

- `endpoint` (no default): Geneva GCS endpoint URL (e.g., `https://gcs.ppe.monitoring.core.windows.net`)
- `environment` (no default): Environment name (e.g., `Production`, `Test`)
- `account` (no default): Geneva account name
- `namespace` (no default): Geneva namespace
- `region` (no default): Azure region (e.g., `eastus`, `westus2`)
- `tenant` (no default): Azure tenant ID
- `role_name` (no default): Role name for the service
- `role_instance` (no default): Role instance identifier
- `config_major_version` (default = 1): Geneva configuration version
- `auth_method` (default = 0): Authentication method
  - `0` = System Managed Service Identity (MSI)
  - `1` = Certificate
  - `2` = Workload Identity

### Authentication

#### MSI Authentication (default)

```yaml
exporters:
  azuregigwarm:
    auth_method: 0
    endpoint: "https://gcs.monitoring.core.windows.net"
    environment: "Production"
    account: "MyAccount"
    namespace: "MyNamespace"
    region: "eastus"
    tenant: "00000000-0000-0000-0000-000000000000"
    role_name: "MyService"
    role_instance: "instance-1"
```

#### Certificate Authentication

```yaml
exporters:
  azuregigwarm:
    auth_method: 1
    cert_path: "/path/to/certificate.p12"
    cert_password: "certificate_password"
    endpoint: "https://gcs.monitoring.core.windows.net"
    environment: "Production"
    account: "MyAccount"
    namespace: "MyNamespace"
    region: "eastus"
    tenant: "00000000-0000-0000-0000-000000000000"
    role_name: "MyService"
    role_instance: "instance-1"
```

#### Workload Identity Authentication

```yaml
exporters:
  azuregigwarm:
    auth_method: 2
    workload_identity_resource: "https://monitor.azure.com"
    endpoint: "https://gcs.monitoring.core.windows.net"
    environment: "Production"
    account: "MyAccount"
    namespace: "MyNamespace"
    region: "eastus"
    tenant: "00000000-0000-0000-0000-000000000000"
    role_name: "MyService"
    role_instance: "instance-1"
```

### Optional Parameters

#### Sending Queue

The exporter supports persistent queuing to prevent data loss during collector restarts. This feature requires the `file_storage` extension.

```yaml
extensions:
  file_storage:
    directory: /var/lib/otelcol/storage
    timeout: 10s

exporters:
  azuregigwarm:
    # ... required config ...
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 5000
      storage: file_storage  # Enable persistence

service:
  extensions: [file_storage]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [azuregigwarm]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [azuregigwarm]
```

#### Retry Configuration

The exporter provides two levels of retry for maximum resilience:

**Export-level retry** (entire export operation):
```yaml
exporters:
  azuregigwarm:
    # ... required config ...
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s
```

**Batch-level retry** (individual batches within an export):
```yaml
exporters:
  azuregigwarm:
    # ... required config ...
    batch_retry:
      enabled: true
      max_retries: 3
      initial_interval: 100ms
      max_interval: 5s
      multiplier: 2.0
```

### Complete Configuration Example

```yaml
extensions:
  file_storage:
    directory: /var/lib/otelcol/storage
    timeout: 10s
    compaction:
      on_start: true
      on_rebound: true
      rebound_needed_threshold_mib: 5
      rebound_trigger_threshold_mib: 10

receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 10s
    send_batch_size: 1024

exporters:
  azuregigwarm:
    endpoint: "https://gcs.monitoring.core.windows.net"
    environment: "Production"
    account: "MyAccount"
    namespace: "MyNamespace"
    region: "eastus"
    tenant: "00000000-0000-0000-0000-000000000000"
    role_name: "MyService"
    role_instance: "instance-1"
    config_major_version: 1
    auth_method: 0

    # Persistent queue (recommended for production)
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 5000
      storage: file_storage

    # Export-level retry
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 30s
      max_elapsed_time: 300s

    # Batch-level retry
    batch_retry:
      enabled: true
      max_retries: 3
      initial_interval: 100ms
      max_interval: 5s
      multiplier: 2.0

service:
  extensions: [file_storage]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [azuregigwarm]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [azuregigwarm]
```

## Architecture

The exporter uses a Rust FFI bridge for high-performance encoding and upload:

1. **Go Layer**: Receives OTLP data from the collector pipeline
2. **CGO Bridge**: Passes data to Rust via FFI
3. **Rust Layer**: Encodes to Geneva format, compresses, and uploads to GCS endpoint

### Resilience Features

The exporter implements multiple layers of resilience:

1. **Persistent Queue** (via file_storage): Write-Ahead Log prevents data loss during collector crashes
2. **Batch-level Retry**: Individual failed batches are retried without re-encoding successful batches
3. **Export-level Retry**: Entire export operation is retried with exponential backoff
4. **Concurrent Upload**: Multiple batches uploaded in parallel for high throughput

## Installation

### As a Go Module Dependency

Add to your collector's builder configuration:

```yaml
exporters:
  - gomod: github.com/open-telemetry/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter v0.1.0
```

### Building

**Prerequisites**: Rust toolchain and CGO enabled.

First, build the Rust FFI bridge:

```bash
cd geneva_ffi_bridge
cargo build --release
cd ..
```

Then build the Go code:

```bash
CGO_ENABLED=1 go build ./...
```

The Rust FFI library is statically linked by default, so no additional runtime dependencies are required.

## Testing

See [../../docs/TESTING.md](../../docs/TESTING.md) for comprehensive testing guide.

### Quick Test

Using the provided test scripts in the root `examples/` directory:

```bash
# Start collector with your config
./bin/test-collector --config config.yaml

# In another terminal, send test data
cd ../../examples
./send-test-logs.sh
./send-test-traces.sh
```

### Using telemetrygen

[telemetrygen](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen):

```bash
# Generate test traces
telemetrygen traces --otlp-endpoint localhost:4317 --otlp-insecure --traces 100

# Generate test logs
telemetrygen logs --otlp-endpoint localhost:4317 --otlp-insecure --logs 100
```

## Performance Considerations

- **Batch Size**: Configure the `batch` processor with appropriate `send_batch_size` (recommended: 512-2048)
- **Queue Workers**: Adjust `sending_queue.num_consumers` based on upload throughput requirements (recommended: 5-20)
- **Concurrent Batches**: The exporter uploads multiple batches concurrently for optimal throughput

## Troubleshooting

### CGO Not Enabled

If you see an error about CGO requirements, ensure `CGO_ENABLED=1` during build:

```bash
CGO_ENABLED=1 go build
```

### File Storage Errors

If persistent queue fails, check:
- Storage directory exists and is writable
- Sufficient disk space available
- `file_storage` extension is properly configured

### Authentication Errors

For MSI authentication:
- Ensure the service has appropriate managed identity configured
- Verify the identity has permissions to write to Geneva

For certificate authentication:
- Verify certificate path is accessible
- Check certificate password is correct
- Ensure certificate has not expired

## Known Limitations

- **CGO Dependency**: Requires CGO enabled, which may complicate cross-compilation
- **Rust Toolchain**: Building requires Rust toolchain installed
- **Alpha Stability**: This exporter is in alpha stage and APIs may change

## Documentation

- **[Root README](../../README.md)** - Complete module documentation
- **[Quick Start Guide](../../docs/QUICK_START.md)** - Get started in 5 minutes
- **[Testing Guide](../../docs/TESTING.md)** - Comprehensive testing documentation
- **[Example Configurations](../../examples/)** - Sample configs and test scripts

## References

- [Geneva OpenTelemetry Documentation](https://eng.ms/docs/products/geneva/collect/instrument/opentelemetryotlp)
- [OpenTelemetry Collector Architecture](https://opentelemetry.io/docs/collector/)
- [Component Stability Definitions](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md)
