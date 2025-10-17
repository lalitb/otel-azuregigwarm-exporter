# Real Azure Geneva Backend Testing

This guide explains how to run load tests against a real Azure Geneva backend instead of the mock receiver.

## Overview

While the mock receiver is great for isolated testing, you'll want to test against the real Azure Geneva backend to:
- Validate end-to-end connectivity
- Test with actual authentication (MSI or certificate)
- Measure real-world performance
- Verify data appears correctly in Geneva

## Prerequisites

1. **Azure Geneva Access**
   - Valid Geneva account and namespace
   - Authentication credentials (MSI or certificate)
   - Network access to Geneva endpoints

2. **Collector Built**
   ```bash
   cd gigwarmexporter/opentelemetry-collector
   make otelcontribcol
   ```

3. **telemetrygen Installed**
   ```bash
   go install github.com/open-telemetry/opentelemetry-collector-contrib/cmd/telemetrygen@latest
   ```

## Setup

### 1. Create Environment Configuration

Copy the template and fill in your credentials:

```bash
cd gigwarmexporter/opentelemetry-collector/testbed
cp .env.template .env
```

Edit `.env` with your actual values:

```bash
# Required Geneva Configuration
GENEVA_ENDPOINT="https://gcs.ppe.monitoring.core.windows.net"
GENEVA_ENVIRONMENT="prod"
GENEVA_ACCOUNT="YourAccountName"
GENEVA_NAMESPACE="YourNamespace"
GENEVA_REGION="eastus"
GENEVA_CONFIG_VERSION="1"
GENEVA_TENANT="your-tenant-id"
GENEVA_ROLE_NAME="your-role-name"
GENEVA_ROLE_INSTANCE="your-role-instance"

# Authentication Method: 0=MSI, 1=Certificate
GENEVA_AUTH_METHOD="1"

# Certificate Authentication (required if GENEVA_AUTH_METHOD=1)
GENEVA_CERT_PATH="/path/to/your/cert.p12"
GENEVA_CERT_PASSWORD="your-cert-password"
```

**⚠️ IMPORTANT**: Never commit `.env` file - it contains sensitive credentials!

### 2. Add .env to .gitignore

Ensure your `.env` file is ignored by git:

```bash
# Add to .gitignore if not already present
echo ".env" >> .gitignore
echo "config-real-backend.yaml" >> .gitignore
echo "collector.log" >> .gitignore
```

## Running Tests

### Quick Start

```bash
cd gigwarmexporter/opentelemetry-collector/testbed

# Run moderate trace load test (5k spans/sec for 5 minutes)
./run-real-backend-test.sh trace moderate

# Run light log load test (1k logs/sec for 1 minute)
./run-real-backend-test.sh logs light
```

### Test Levels

The script supports different load levels:

| Level | Rate | Duration | Use Case |
|-------|------|----------|----------|
| `light` | 1,000/sec | 60s | Quick validation |
| `moderate` | 5,000/sec | 300s | Standard testing |
| `heavy` | 10,000/sec | 600s | Performance testing |
| `stress` | 50,000/sec | 600s | Stress testing |

### Examples

**Quick Validation** (1k traces/sec for 1 min):
```bash
./run-real-backend-test.sh trace light
```

**Standard Load Test** (5k logs/sec for 5 min):
```bash
./run-real-backend-test.sh logs moderate
```

**Performance Test** (10k traces/sec for 10 min):
```bash
./run-real-backend-test.sh trace heavy
```

**Stress Test** (50k traces/sec for 10 min):
```bash
./run-real-backend-test.sh trace stress
```

## What the Script Does

1. **Validates** environment variables and credentials
2. **Builds** collector if not already built
3. **Generates** config from template with your credentials
4. **Starts** collector in background
5. **Runs** telemetrygen load test
6. **Captures** collector logs
7. **Cleans up** on exit

## Verifying Results in Geneva

After running tests, verify data in your Geneva dashboard:

### 1. Find Your Test Data

Filter by test identifiers that are automatically added:
- `test.scenario`: Identifies this as a load test
- `test.timestamp`: Unique timestamp for this test run

### 2. Check Metrics

Look for:
- **Data arrival**: Traces/logs appearing in Geneva
- **Timestamps**: Data freshness
- **Volume**: Expected number of items
- **No errors**: Check for any error messages

### 3. Example Geneva Query

```kusto
traces
| where customDimensions["test.scenario"] == "real-backend-loadtest"
| where timestamp > ago(1h)
| summarize count() by bin(timestamp, 1m)
```

## Manual Testing (Without Script)

If you prefer manual control:

### 1. Set Environment Variables

```bash
source .env
export TEST_TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

### 2. Generate Config

```bash
envsubst < config-real-backend.yaml.template > config-real-backend.yaml
```

### 3. Start Collector

```bash
../../../../bin/otelcontribcol_$(go env GOOS)_$(go env GOARCH) --config config-real-backend.yaml
```

### 4. Run Load Generator (in another terminal)

```bash
# Traces
telemetrygen traces \
  --otlp-endpoint localhost:4317 \
  --otlp-insecure \
  --rate 5000 \
  --duration 300s \
  --telemetry-attributes "test.scenario=manual-test"

# Logs
telemetrygen logs \
  --otlp-endpoint localhost:4317 \
  --otlp-insecure \
  --rate 5000 \
  --duration 300s \
  --telemetry-attributes "test.scenario=manual-test"
```

## Monitoring During Tests

### Watch Collector Logs

```bash
tail -f collector.log
```

Look for:
- ✅ Successful exports: `"successfully exported"`
- ⚠️ Warnings: Connection issues, throttling
- ❌ Errors: Authentication failures, network errors

### Monitor System Resources

```bash
# CPU and memory usage
top -pid $(pgrep otelcontribcol)

# Or with ps
ps aux | grep otelcontribcol
```

## Troubleshooting

### Authentication Errors

**Error**: `authentication failed`

**Solutions**:
- **MSI**: Ensure running on Azure VM with correct managed identity
- **Certificate**: Verify cert path and password are correct
- Check `GENEVA_TENANT` matches your Azure tenant

### Connection Errors

**Error**: `connection refused` or `timeout`

**Solutions**:
- Verify `GENEVA_ENDPOINT` is correct and reachable
- Check network/firewall rules allow outbound HTTPS
- Test with `curl` to verify connectivity:
  ```bash
  curl -v $GENEVA_ENDPOINT
  ```

### Certificate Errors

**Error**: `failed to load certificate`

**Solutions**:
- Verify certificate file exists: `ls -la $GENEVA_CERT_PATH`
- Check password is correct
- Ensure certificate is in PFX/P12 format
- Verify certificate hasn't expired

### Rate Limiting / Throttling

**Error**: `too many requests` or `throttled`

**Solutions**:
- Reduce load level (use `light` or `moderate`)
- Check Geneva account quotas
- Increase batch timeout to reduce request rate
- Contact Geneva support for quota increase

### Data Not Appearing in Geneva

**Possible Causes**:
1. **Authentication failed** - Check collector logs for auth errors
2. **Wrong account/namespace** - Verify `GENEVA_ACCOUNT` and `GENEVA_NAMESPACE`
3. **Geneva ingestion delay** - Wait 5-10 minutes for data to appear
4. **Test identifiers** - Use correct filters in Geneva query

## Performance Benchmarks

Based on testing, typical performance characteristics:

| Metric | MSI Auth | Cert Auth |
|--------|----------|-----------|
| Throughput | ~40k items/sec | ~35k items/sec |
| Latency (p50) | ~50ms | ~60ms |
| Latency (p99) | ~200ms | ~250ms |
| CPU @ 10k/sec | ~30% | ~35% |
| RAM @ 10k/sec | ~120MB | ~130MB |

*Actual performance depends on network latency, payload size, and Geneva backend capacity.*

## Best Practices

1. **Start small**: Begin with `light` load, then scale up
2. **Monitor first**: Watch logs and metrics before increasing load
3. **Tag your tests**: Use unique `test.scenario` values
4. **Clean up**: Remove test data from Geneva after validation
5. **Respect quotas**: Don't exceed your Geneva account limits
6. **Secure credentials**: Never commit `.env` file
7. **Rotate certificates**: Update before expiration

## CI/CD Integration

Example for GitHub Actions:

```yaml
name: Real Backend Load Test

on:
  workflow_dispatch:
    inputs:
      load_level:
        description: 'Load level'
        required: true
        default: 'moderate'
        type: choice
        options:
          - light
          - moderate
          - heavy

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup credentials
        run: |
          cd gigwarmexporter/opentelemetry-collector/testbed
          cat > .env << EOF
          GENEVA_ENDPOINT="${{ secrets.GENEVA_ENDPOINT }}"
          GENEVA_ACCOUNT="${{ secrets.GENEVA_ACCOUNT }}"
          GENEVA_NAMESPACE="${{ secrets.GENEVA_NAMESPACE }}"
          # ... other secrets ...
          EOF

      - name: Run load test
        run: |
          cd gigwarmexporter/opentelemetry-collector/testbed
          ./run-real-backend-test.sh trace ${{ inputs.load_level }}

      - name: Upload logs
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: collector-logs
          path: gigwarmexporter/opentelemetry-collector/testbed/collector.log
```

## Security Notes

⚠️ **Never commit sensitive information**:
- `.env` file
- Generated `config-real-backend.yaml`
- Certificate files
- Passwords or keys

✅ **Always**:
- Use `.gitignore` to exclude sensitive files
- Store credentials in secure vaults (Azure Key Vault, GitHub Secrets)
- Rotate certificates regularly
- Use MSI authentication when possible (more secure than certificates)
- Restrict Geneva account permissions to minimum required

## Next Steps

1. **Run your first test**:
   ```bash
   ./run-real-backend-test.sh trace light
   ```

2. **Verify data in Geneva** using test identifiers

3. **Scale up gradually** to test higher loads

4. **Compare performance** with mock receiver tests

5. **Integrate into CI/CD** for automated testing

## Support

For issues:
1. Check collector logs: `tail -f collector.log`
2. Verify environment variables are set correctly
3. Test Geneva connectivity separately
4. Review troubleshooting section above
5. Contact Azure Geneva support for backend issues
