# Azure GigWarm Exporter Examples

This directory contains examples and helper scripts to get started with the Azure GigWarm (Geneva Warm) exporter for OpenTelemetry Collector.

## Contents

| File | Description |
|------|-------------|
| `Makefile` | Automated build and test targets |
| `build.sh` | Build script with static linking |
| `builder-config.yaml` | OpenTelemetry Collector Builder configuration |
| `config.yaml` | Example collector configuration with Azure GigWarm exporter |
| `send-test-logs.sh` | Bash script to send test logs to the collector |
| `send-test-traces.sh` | Bash script to send test traces to the collector |
| `test-logs.json` | Sample OTLP log payload |
| `test-traces.json` | Sample OTLP trace payload |

## Quick Start (Using Makefile - Recommended)

### Prerequisites

- Go 1.21+ installed
- Rust toolchain installed
- Make utility
- `$HOME/go/bin` in your PATH (for the `builder` tool)

**Note:** After installing Go tools, ensure `$HOME/go/bin` is in your PATH:
```bash
export PATH=$HOME/go/bin:$PATH
# Or add permanently to ~/.bashrc:
echo 'export PATH=$HOME/go/bin:$PATH' >> ~/.bashrc
```

### Build and Run

```bash
# From the examples directory
cd examples

# Check prerequisites
make check-prereqs

# Configure your Geneva account (IMPORTANT!)
# Edit config.yaml and replace these values:
#   - endpoint: "https://abc.monitoring.core.windows.net"
#   - account: "YourGenevaAccount"
#   - namespace: "YourGenevaNamespace"
#   - tenant: "your-tenant-id"
#   - region: "eastus" (or your region)
#   - auth_method, cert_path, workload_identity_resource, etc.
vim config.yaml  # or use your preferred editor

# Build everything (Rust FFI + Collector with static linking)
make build

# Run the collector
make run

# In another terminal, send test data
make test
```

**Important:** Before running, edit `config.yaml` to configure:
- Geneva endpoint (PPE or Production)
- Account and namespace
- Tenant ID and region
- Authentication method (MSI/Certificate/Workload Identity)

**That's it!** The Makefile handles:
- ✅ Building Rust FFI bridge with static linking
- ✅ Building collector with proper CGO flags
- ✅ No manual library copying needed
- ✅ No LD_LIBRARY_PATH configuration

### Available Make Targets

```bash
make build            # Build everything
make run              # Build and run collector
make test             # Send test logs and traces
make test-logs        # Send test logs only
make test-traces      # Send test traces only
make clean            # Clean build artifacts
make rebuild          # Clean and rebuild from scratch
make install-tools    # Install builder if missing
make help             # Show all available targets
```

## Quick Start (Using Build Script)

Alternatively, use the build script:

```bash
cd examples

# Edit config.yaml first (IMPORTANT!)
vim config.yaml  # Configure your Geneva account details

# Build
./build.sh

# Run
./bin/otelcol-azuregigwarm --config config.yaml
```

## Manual Build (Advanced)

If you prefer to build manually:

### Step 1: Build the Rust FFI Bridge

The exporter uses a Rust FFI bridge with **static linking** (no runtime dependencies).

```bash
# Navigate to the FFI bridge directory
cd ../exporter/azuregigwarmexporter/geneva_ffi_bridge

# Build the Rust library (generates both .a and .so)
# Uses geneva-uploader-ffi 0.3.0 from crates.io
cargo build --release
```

This generates:
- `libgeneva_ffi_bridge.a` (static library - recommended)
- `libgeneva_ffi_bridge.so` (dynamic library)

### Step 2: Build the Collector with Static Linking

```bash
# From the examples directory
cd examples

# Install builder if not already installed
go install go.opentelemetry.io/collector/cmd/builder@latest

# Build with static linking (no library copying needed!)
CGO_ENABLED=1 \
CGO_LDFLAGS="-L../exporter/azuregigwarmexporter/geneva_ffi_bridge/target/release -lgeneva_ffi_bridge" \
builder --config builder-config.yaml
```

The collector binary now includes the Rust library **statically** - no runtime dependencies!

### Step 3: Configure the Exporter

Edit `config.yaml` and update the following fields with your actual values:

```yaml
exporters:
  azuregigwarm:
    # Geneva endpoint - REPLACE with your environment
    endpoint: "https://gcs.ppe.monitoring.core.windows.net"  # PPE
    # endpoint: "https://gcs.prod.monitoring.core.windows.net"  # Production

    # Environment - REPLACE as needed
    environment: "Test"  # or "Production", "Staging", etc.

    # Your Geneva account details - REPLACE these
    account: "YourGenevaAccount"           # e.g., "PipelineAgent2Demo"
    namespace: "YourGenevaNamespace"       # e.g., "PAdemo2"
    region: "eastus"                       # e.g., "eastus", "westus2", etc.

    # Azure tenant ID - REPLACE with your tenant
    tenant: "your-tenant-id"               # e.g., "72f988bf-86f1-41af-91ab-2d7cd011db47"

    # Authentication method
    auth_method: 2  # 0=MSI, 1=Certificate, 2=WorkloadIdentity

    # For Workload Identity (auth_method=2)
    workload_identity_resource: "https://gcs.ppe.monitoring.core.windows.net"

    # For Certificate (auth_method=1)
    # cert_path: "/path/to/certificate.p12"
    # cert_password: "your-password"
```

**Configuration Options:**

| Parameter | Required | Description | Default |
|-----------|----------|-------------|---------|
| `endpoint` | Yes | Geneva GCS endpoint | - |
| `environment` | Yes | Environment name (e.g., "Test", "Production") | - |
| `account` | Yes | Geneva account name | - |
| `namespace` | Yes | Geneva namespace name | - |
| `region` | Yes | Azure region (e.g., "eastus") | - |
| `config_major_version` | Yes | Geneva config version | `2` |
| `tenant` | Yes | Azure AD tenant ID | - |
| `role_name` | Yes | Role name for telemetry | - |
| `role_instance` | Yes | Role instance identifier | - |
| `auth_method` | Yes | Authentication method (0/1/2) | `0` |
| `workload_identity_resource` | Conditional | Required when auth_method=2 | - |
| `cert_path` | Conditional | Required when auth_method=1 | - |
| `cert_password` | Conditional | Required when auth_method=1 | - |

### Step 3: Run the Collector

With static linking, just run the binary directly:

```bash
# Run the collector (no environment variables needed!)
./bin/otelcol-azuregigwarm --config config.yaml
```

You should see:
```
Starting otelcontribcol...
Starting extensions...
Starting AzureGigWarm exporter endpoint=https://gcs.ppe.monitoring.core.windows.net
Starting GRPC server endpoint=[::]:4317
Starting HTTP server endpoint=[::]:4318
Everything is ready. Begin running and processing data.
```

### Step 4: Send Test Data

In a new terminal, send test logs:

```bash
# Send a test log
./send-test-logs.sh

# Output:
# Sending test log to http://localhost:4318/v1/logs...
# ✓ Log sent successfully!
```

Send test traces:

```bash
# Send a test trace
./send-test-traces.sh

# Output:
# Sending test trace to http://localhost:4318/v1/traces...
# ✓ Trace sent successfully!
```

### Step 5: Verify in Collector Logs

Check the collector output:

```
pushLogs called log_record_count=1
Marshaled logs to protobuf data_size=XXX
Encoded logs into batches batch_count=1
Sending upload request to URL: "https://..."
Upload Successful
```

## Testing Different Authentication Methods

### System Managed Identity (MSI)

For Azure VMs with managed identity:

```yaml
exporters:
  azuregigwarm:
    auth_method: 0  # System MSI
    # No additional configuration needed
```

### Certificate Authentication

For certificate-based authentication:

```yaml
exporters:
  azuregigwarm:
    auth_method: 1  # Certificate
    cert_path: "/path/to/certificate.p12"
    cert_password: "${CERT_PASSWORD}"  # Use env var for security
```

### Workload Identity (Kubernetes)

For AKS with Workload Identity:

```yaml
exporters:
  azuregigwarm:
    auth_method: 2  # Workload Identity
    workload_identity_resource: "https://gcs.ppe.monitoring.core.windows.net"
```

See [AKS_DEPLOYMENT.md](../docs/AKS_DEPLOYMENT.md) for complete Kubernetes setup.

## Customizing Test Data

### Modify Test Logs

Edit `test-logs.json` to customize the log payload:

```json
{
  "resourceLogs": [{
    "resource": {
      "attributes": [
        {
          "key": "service.name",
          "value": {"stringValue": "my-service"}
        }
      ]
    },
    "scopeLogs": [{
      "logRecords": [{
        "timeUnixNano": "TIMESTAMP_PLACEHOLDER",  // Auto-replaced by script
        "severityText": "INFO",
        "body": {"stringValue": "Custom log message"}
      }]
    }]
  }]
}
```

The script `send-test-logs.sh` automatically replaces `TIMESTAMP_PLACEHOLDER` with the current timestamp.

### Modify Test Traces

Edit `test-traces.json` to customize the trace payload:

```json
{
  "resourceSpans": [{
    "resource": {
      "attributes": [
        {
          "key": "service.name",
          "value": {"stringValue": "my-service"}
        }
      ]
    },
    "scopeSpans": [{
      "spans": [{
        "traceId": "5b8aa5a2d2c872e8321cf37308d69df2",
        "spanId": "051581bf3cb55c13",
        "name": "my-span",
        "startTimeUnixNano": "START_TIME_PLACEHOLDER",
        "endTimeUnixNano": "END_TIME_PLACEHOLDER"
      }]
    }]
  }]
}
```

## Advanced Configuration Examples

### High-Throughput Configuration

For high-volume telemetry:

```yaml
exporters:
  azuregigwarm:
    endpoint: "https://gcs.prod.monitoring.core.windows.net"
    # ... other config ...

    sending_queue:
      enabled: true
      num_consumers: 20        # Increase for higher throughput
      queue_size: 5000         # Larger queue

    batch_retry:
      enabled: true
      max_retries: 5
      initial_interval: 100ms
      max_interval: 10s
      multiplier: 2.0

processors:
  batch:
    timeout: 5s
    send_batch_size: 2048      # Larger batches
    send_batch_max_size: 4096
```

### Production Configuration with Memory Limiter

```yaml
processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 1024            # Adjust based on available memory
    spike_limit_mib: 256

  batch:
    timeout: 10s
    send_batch_size: 1024

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [azuregigwarm]
```

### Dual Export (Debug + Geneva)

For testing and debugging:

```yaml
exporters:
  debug:
    verbosity: detailed

  azuregigwarm:
    # ... Geneva config ...

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug, azuregigwarm]  # Export to both
```

## Environment-Specific Configurations

### PPE (Pre-Production Environment)

```yaml
exporters:
  azuregigwarm:
    endpoint: "https://gcs.ppe.monitoring.core.windows.net"
    environment: "Test"
    workload_identity_resource: "https://gcs.ppe.monitoring.core.windows.net"
```

### Production Environment

```yaml
exporters:
  azuregigwarm:
    endpoint: "https://gcs.prod.monitoring.core.windows.net"
    environment: "Production"
    workload_identity_resource: "https://gcs.prod.monitoring.core.windows.net"
```

## Troubleshooting

### Collector fails to start

**Error:** `error while loading shared libraries: libgeneva_ffi_bridge.so`

**Cause:** You're using dynamic linking instead of static linking.

**Solution:** Rebuild with static linking (recommended):
```bash
make rebuild
```

Or manually:
```bash
CGO_ENABLED=1 \
CGO_LDFLAGS="-L../exporter/azuregigwarmexporter/geneva_ffi_bridge/target/release -lgeneva_ffi_bridge" \
builder --config builder-config.yaml
```

**Note:** Using the Makefile or build.sh script automatically handles static linking, eliminating this issue.

### Authentication errors

**Error:** `AADSTS70011: The scope is not valid`

**Solutions:**
1. Ensure `workload_identity_resource` matches the `endpoint`
2. Verify managed identity is registered with Geneva
3. Check federated credential configuration (for Workload Identity)

### No data in Geneva

**Common causes:**
1. **Old timestamps** - Test data using timestamps from 2023 won't appear in current queries
   - Always use current timestamps: `$(date +%s%N)`
2. **Wrong environment** - Querying Production but sending to PPE
3. **Geneva registration pending** - Managed identity not authorized yet

### Upload failures

Check collector logs for detailed errors:
```bash
# If using debug exporter
grep -i "error" collector.log

# Look for upload messages
grep -i "upload" collector.log
```

### Kubernetes health probe failures

**Error:** Pod in CrashLoopBackOff with health check failures

**Cause:** Kubernetes liveness/readiness probes configured but `health_check` extension not enabled

**Solution:** Add the health_check extension to your configuration:
```yaml
extensions:
  health_check:
    endpoint: 0.0.0.0:13133

service:
  extensions: [health_check]
  pipelines:
    # ... your pipelines
```

This is already included in the example `config.yaml` and is required when deploying to Kubernetes with health probes.

## Performance Tips

1. **Batch Configuration**: Tune batch processor for your workload
   - Smaller batches = lower latency, higher overhead
   - Larger batches = higher latency, better throughput

2. **Queue Configuration**: Adjust queue size based on traffic patterns
   - Larger queues handle bursts better
   - More consumers = higher concurrency

3. **Resource Limits**: Set appropriate memory limits
   - Monitor memory usage under load
   - Adjust `memory_limiter` accordingly

4. **Retry Configuration**: Balance between reliability and resource usage
   - More retries = better reliability, more resources
   - Shorter intervals = faster recovery, more network traffic

## Integration Examples

### With Kubernetes Applications

Deploy the collector as a DaemonSet or Sidecar and configure your applications to send telemetry:

```yaml
# In your application deployment
env:
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "http://otel-collector:4318"
  - name: OTEL_SERVICE_NAME
    value: "my-app"
```

### With Docker Compose

```yaml
services:
  otel-collector:
    image: otelcol-azuregigwarm:latest
    ports:
      - "4317:4317"  # GRPC
      - "4318:4318"  # HTTP
    volumes:
      - ./config.yaml:/etc/otel/config.yaml
      - /tmp/geneva-libs:/usr/local/lib
    environment:
      - LD_LIBRARY_PATH=/usr/local/lib
```

### With Systemd

Create a systemd service:

```ini
[Unit]
Description=OpenTelemetry Collector with Azure GigWarm
After=network.target

[Service]
Type=simple
User=otel
Environment="LD_LIBRARY_PATH=/usr/local/lib"
ExecStart=/usr/local/bin/otelcol-azuregigwarm --config=/etc/otel/config.yaml
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## Next Steps

- **Production Deployment**: See [AKS_DEPLOYMENT.md](../docs/AKS_DEPLOYMENT.md) for Kubernetes setup
- **Custom Instrumentation**: Instrument your applications with OpenTelemetry SDKs
- **Geneva Dashboards**: Create dashboards to visualize your telemetry data
- **Alerting**: Set up alerts based on log/trace patterns

## References

- [OpenTelemetry Collector Documentation](https://opentelemetry.io/docs/collector/)
- [OTLP Protocol Specification](https://github.com/open-telemetry/opentelemetry-proto)
- [OpenTelemetry Collector Builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder)
- [Geneva Documentation](https://eng.ms/docs/products/geneva) (Microsoft internal)
