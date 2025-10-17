# GigWarm Exporter Load Testing

This directory contains load testing infrastructure for the Azure GigWarm exporter using the OpenTelemetry Collector testbed framework.

## Quick Links

- 📖 **[Configuration Guide](CONFIG_EXPLANATION.md)** - Complete configuration documentation
- 💾 **[File Storage Guide](README-FILESTORAGE.md)** - Persistent queue setup for zero data loss
- 🧪 **Load Testing** - Performance and stress tests (below)

## Overview

The testbed provides a controlled environment for conducting end-to-end load tests, including:
- Performance benchmarks (throughput, latency, resource usage)
- Stress tests (maximum load capacity)
- Stability tests (long-running scenarios)
- Resilience testing (file storage, crash recovery)

## Architecture

```
Load Generator → OTLP Receiver → Batch Processor → GigWarm Exporter → Mock Geneva Backend
     (testbed)                                      (azuregigwarmexporter)    (test receiver)
```

## Prerequisites

1. **Build the contrib collector with GigWarm exporter**:
   ```bash
   cd ../../../..
   make otelcontribcol
   ```

2. **Ensure dependencies are available**:
   ```bash
   go mod download
   ```

## Running Load Tests

### List Available Tests

```bash
make list-tests
```

### Run All GigWarm Tests

```bash
make run-gigwarm-tests
```

### Run Specific Test Categories

**Trace Load Tests** (10k spans/sec):
```bash
make run-trace-tests
```

**Log Load Tests** (10k logs/sec):
```bash
make run-log-tests
```

**High Throughput Test** (50k items/sec):
```bash
make run-high-throughput-test
```

### Run Individual Tests

```bash
RUN_TESTBED=1 go test -v ./tests -run TestGigWarmTrace10kSPS -timeout 30m
RUN_TESTBED=1 go test -v ./tests -run TestGigWarmLog10kSPS -timeout 30m
```

## Test Scenarios

### 1. TestGigWarmTrace10kSPS
- **Throughput**: 10,000 spans/second
- **Protocols**: OTLP gRPC and OTLP HTTP
- **Duration**: ~5 minutes
- **Purpose**: Validate trace handling at moderate load

### 2. TestGigWarmLog10kSPS
- **Throughput**: 10,000 log records/second
- **Protocols**: OTLP gRPC and OTLP HTTP
- **Duration**: ~5 minutes
- **Purpose**: Validate log handling at moderate load

### 3. TestGigWarmTrace1kSPSWithAttributes
- **Throughput**: 1,000 spans/second with realistic attributes
- **Purpose**: Test with production-like span attributes
- **Attributes**: service.name, service.version, deployment.environment, host.name

### 4. TestGigWarmHighThroughput
- **Throughput**: 50,000 items/second
- **Purpose**: Stress test to determine maximum capacity
- **Note**: Skipped in short test mode (`go test -short`)

## Resource Expectations

| Test Scenario | Expected CPU | Expected RAM | Notes |
|--------------|--------------|--------------|-------|
| 10k SPS Traces | < 50% | < 150 MB | With batch processor |
| 10k SPS Logs | < 50% | < 150 MB | With batch processor |
| 1k SPS w/ Attrs | < 40% | < 120 MB | Realistic attributes |
| 50k SPS High Load | < 200% | < 400 MB | Stress test |

## Mock Receiver

The `datareceivers.NewAzureGigWarmDataReceiver()` provides a mock Geneva backend that:
- Accepts HTTP POST requests
- Tracks received data (traces, logs, bytes)
- Returns success responses
- Does NOT require Azure credentials or connectivity

This allows load testing without external dependencies.

## Test Results

Test results are output to:
- Console (real-time metrics)
- `./tests/results/` directory (detailed reports)

Metrics tracked:
- **Throughput**: Items/second sent and received
- **Latency**: End-to-end processing time
- **CPU Usage**: Collector CPU consumption
- **Memory Usage**: Collector RAM consumption
- **Error Rate**: Failed exports

## Troubleshooting

### Test Fails with "connection refused"

Ensure the contrib collector binary is built:
```bash
cd ../../../..
make otelcontribcol
```

### High CPU/Memory Usage

This may indicate:
- Insufficient batch processor tuning
- Resource constraints on test machine
- Exporter bottleneck

Adjust batch processor settings in test scenarios:
```go
{
    Name: "batch",
    Body: `
  batch:
    send_batch_size: 2048
    timeout: 500ms
`,
}
```

### Tests Timeout

Increase test timeout:
```bash
RUN_TESTBED=1 go test -v ./tests -timeout 60m
```

## Integration with CI/CD

To run in CI/CD pipelines:

```yaml
- name: Build Contrib Collector
  run: cd opentelemetry-collector-contrib && make otelcontribcol

- name: Run Load Tests
  run: cd opentelemetry-collector-contrib/exporter/azuregigwarmexporter/testbed && make run-gigwarm-tests
```

## Adding New Tests

To add new load test scenarios:

1. Create a test function in `tests/gigwarm_test.go`:
   ```go
   func TestGigWarmMyScenario(t *testing.T) {
       sender := testbed.NewOTLPTraceDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t))
       receiver := datareceivers.NewAzureGigWarmDataReceiver(testutil.GetAvailablePort(t))

       // Configure your test scenario...
   }
   ```

2. Run your new test:
   ```bash
   RUN_TESTBED=1 go test -v ./tests -run TestGigWarmMyScenario
   ```

## Real Backend Testing

For testing against a real Azure Geneva backend (instead of the mock receiver), see:
- **[REAL_BACKEND_TESTING.md](REAL_BACKEND_TESTING.md)** - Complete guide for real backend testing

Quick start:
```bash
# Setup credentials
cp .env.template .env
# Edit .env with your Geneva credentials

# Run test against real backend
./run-real-backend-test.sh trace moderate
```

## References

- [OpenTelemetry Collector Testbed](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/testbed)
- [Azure GigWarm Exporter](../exporter/azuregigwarmexporter/README.md)
- [Real Backend Testing Guide](REAL_BACKEND_TESTING.md)
