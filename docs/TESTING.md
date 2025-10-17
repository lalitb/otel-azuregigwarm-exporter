# Testing Guide for Azure GigWarm Exporter

This guide explains how to test the Azure GigWarm exporter locally.

## Prerequisites

1. **Rust toolchain** installed
2. **Go 1.21+** installed
3. **OpenTelemetry Collector Builder** (`ocb`) installed
4. **curl** for sending test data
5. **openssl** for generating test IDs

## Quick Start

### Step 1: Build the Rust FFI Bridge

```bash
cd exporter/azuregigwarmexporter/geneva_ffi_bridge
cargo build --release
cd ../../..
```

### Step 2: Test Local Build

Test that the exporter compiles:

```bash
cd exporter/azuregigwarmexporter
CGO_ENABLED=1 go build ./...
cd ../..
```

### Step 3: Build a Test Collector

#### Option A: Using Local Module (for development)

Create a test directory:

```bash
mkdir -p test-collector
cd test-collector
```

Create `builder-config.yaml`:

```yaml
dist:
  name: test-collector
  description: Test collector
  output_path: ./bin
  otelcol_version: 0.137.0

exporters:
  - gomod: github.com/open-telemetry/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter v0.1.0
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.137.0

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.137.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.137.0
```

Build with local replacement:

```bash
# Generate the collector
ocb --config builder-config.yaml

# Add local replacement
cd test-collector
go mod edit -replace github.com/open-telemetry/otel-azuregigwarm-exporter=/path/to/otel-azuregigwarm-exporter

# Build with CGO
CGO_ENABLED=1 go build -o ../bin/test-collector .
cd ..
```

#### Option B: After Publishing Module

```bash
# Just build normally
ocb --config builder-config.yaml
cd test-collector
CGO_ENABLED=1 go build -o ../bin/test-collector .
cd ..
```

### Step 4: Configure the Collector

Copy and edit the example configuration:

```bash
cp examples/config.yaml config.yaml
# Edit config.yaml with your Geneva credentials
```

Update these fields:
- `endpoint`: Your Geneva endpoint
- `account`: Your Geneva account
- `namespace`: Your Geneva namespace
- `tenant`: Your Azure tenant ID
- `role_name`: Your role name
- `role_instance`: Your instance name
- `cert_path` and `cert_password`: Your certificate details (if using cert auth)

### Step 5: Run the Collector

```bash
./bin/test-collector --config config.yaml
```

You should see:

```
2025-10-16T16:07:14.688-0700    info    service@v0.137.0/service.go:222    Starting otelcol...
2025-10-16T16:07:14.688-0700    info    azuregigwarmexporter/logsexporter.go:80    Starting AzureGigWarm exporter
2025-10-16T16:07:14.688-0700    info    service@v0.137.0/service.go:245    Everything is ready. Begin running and processing data.
```

### Step 6: Send Test Data

In a separate terminal:

#### Send Test Logs

```bash
cd examples
./send-test-logs.sh
```

You should see:
```
Sending test log to http://localhost:4318/v1/logs...
✓ Log sent successfully!
Response: {"partialSuccess":{}}
```

#### Send Test Traces

```bash
./send-test-traces.sh
```

You should see:
```
Sending test trace to http://localhost:4318/v1/traces...
Trace ID: 5b8aa5a2d2c872e8321cf37308d69df2
Span ID: 051581bf3cb55c13
✓ Trace sent successfully!
Response: {"partialSuccess":{}}
```

## Manual Testing with curl

### Send a Log

```bash
curl -X POST http://localhost:4318/v1/logs \
  -H "Content-Type: application/json" \
  -d '{
    "resourceLogs": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": {"stringValue": "manual-test"}
        }]
      },
      "scopeLogs": [{
        "scope": {"name": "test"},
        "logRecords": [{
          "timeUnixNano": "'$(date +%s%N)'",
          "severityText": "INFO",
          "body": {"stringValue": "Manual test log"}
        }]
      }]
    }]
  }'
```

### Send a Trace

```bash
curl -X POST http://localhost:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{
    "resourceSpans": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": {"stringValue": "manual-test"}
        }]
      },
      "scopeSpans": [{
        "scope": {"name": "test"},
        "spans": [{
          "traceId": "'$(openssl rand -hex 16)'",
          "spanId": "'$(openssl rand -hex 8)'",
          "name": "manual-test-span",
          "kind": 1,
          "startTimeUnixNano": "'$(date +%s%N)'",
          "endTimeUnixNano": "'$(date +%s%N)'"
        }]
      }]
    }]
  }'
```

## Verifying Exports

### Check Collector Logs

The collector will show debug output for each export:

```
2025-10-16T16:13:14.767-0700    info    Logs    {"resource logs": 1, "log records": 1}
2025-10-16T16:13:14.768-0700    info    ResourceLog #0
Resource SchemaURL:
Resource attributes:
     -> service.name: Str(test-service)
ScopeLogs #0
...
```

### Check for Errors

If there are errors, you'll see detailed messages:

```
Error: geneva client initialization failed: Certificate error: No such file or directory (os error 2)
```

This means your certificate path is incorrect. Update `config.yaml` with the correct path.

## Common Issues

### Issue: "library 'geneva_ffi_bridge' not found"

**Solution**: Build the Rust FFI bridge first:

```bash
cd exporter/azuregigwarmexporter/geneva_ffi_bridge
cargo build --release
```

### Issue: "CGO is disabled"

**Solution**: Enable CGO before building:

```bash
export CGO_ENABLED=1
go build ./...
```

### Issue: Certificate errors

**Solution**: Ensure:
1. Certificate file exists at the specified path
2. Certificate password is correct
3. Certificate is in PKCS12 (.p12) format

Or switch to MSI authentication:

```yaml
auth_method: 0  # System MSI
```

### Issue: Connection refused

**Solution**: Check that:
1. Geneva endpoint is correct
2. Network connectivity to Geneva endpoint
3. Firewall rules allow outbound HTTPS

## Integration Testing

### Using Python OpenTelemetry SDK

```python
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter

# Set up the tracer
trace.set_tracer_provider(TracerProvider())
tracer = trace.get_tracer(__name__)

# Configure OTLP exporter to send to collector
otlp_exporter = OTLPSpanExporter(endpoint="http://localhost:4318/v1/traces")
span_processor = BatchSpanProcessor(otlp_exporter)
trace.get_tracer_provider().add_span_processor(span_processor)

# Create a test span
with tracer.start_as_current_span("test-operation"):
    print("Sending test span to collector...")
```

### Using Go OpenTelemetry SDK

```go
package main

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	ctx := context.Background()

	// Create OTLP exporter
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("localhost:4318"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Create tracer provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)

	// Create test span
	tracer := tp.Tracer("test-tracer")
	_, span := tracer.Start(ctx, "test-operation")
	span.End()

	tp.Shutdown(ctx)
}
```

## Performance Testing

### Load Testing with Multiple Requests

```bash
# Send 100 logs
for i in {1..100}; do
  ./examples/send-test-logs.sh &
done
wait

# Send 100 traces
for i in {1..100}; do
  ./examples/send-test-traces.sh &
done
wait
```

### Monitor Performance

Watch collector metrics and resource usage:

```bash
# In one terminal
watch -n 1 'ps aux | grep test-collector'

# Check logs for throughput
tail -f collector.log
```

## Troubleshooting

### Enable Verbose Logging

Update `config.yaml`:

```yaml
exporters:
  debug:
    verbosity: detailed  # Shows full payload
    sampling_initial: 100
    sampling_thereafter: 100
```

### Check FFI Bridge Logs

The Rust FFI bridge logs errors to stderr. Redirect to a file:

```bash
./bin/test-collector --config config.yaml 2> ffi-errors.log
```

### Validate Configuration

Test configuration without running:

```bash
./bin/test-collector validate --config config.yaml
```

## Next Steps

Once local testing works:

1. **Deploy to test environment** with real Geneva credentials
2. **Monitor Geneva ingestion** to verify data arrival
3. **Test failover scenarios** (network issues, auth failures)
4. **Load test** with production-like volumes
5. **Set up monitoring** for the collector itself

## Resources

- [OpenTelemetry Collector Documentation](https://opentelemetry.io/docs/collector/)
- [OTLP Specification](https://github.com/open-telemetry/opentelemetry-proto)
- [Geneva Documentation](https://eng.ms/docs/products/geneva)
