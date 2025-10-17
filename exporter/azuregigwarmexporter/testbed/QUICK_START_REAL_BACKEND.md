# Quick Start: Real Backend Testing

Fast guide to get started with real Azure Geneva backend testing.

## 1. Setup (One-time)

```bash
cd gigwarmexporter/opentelemetry-collector/testbed

# Copy template and edit with your credentials
cp .env.template .env
nano .env  # or vim, code, etc.
```

## 2. Configure Your .env

Fill in these values in `.env`:

```bash
# Your Geneva settings
GENEVA_ENDPOINT="https://gcs.ppe.monitoring.core.windows.net"  # Or prod endpoint
GENEVA_ACCOUNT="YourAccountName"
GENEVA_NAMESPACE="YourNamespace"
GENEVA_REGION="eastus"
GENEVA_TENANT="your-tenant-id"
GENEVA_ROLE_NAME="your-role"
GENEVA_ROLE_INSTANCE="instance-01"

# Certificate auth (recommended)
GENEVA_AUTH_METHOD="1"
GENEVA_CERT_PATH="/path/to/your/cert.p12"
GENEVA_CERT_PASSWORD="your-password"

# Or MSI auth
# GENEVA_AUTH_METHOD="0"
```

## 3. Run Your First Test

```bash
# Light test - 1k spans/sec for 1 minute
./run-real-backend-test.sh trace light

# Moderate test - 5k spans/sec for 5 minutes
./run-real-backend-test.sh trace moderate
```

## 4. Verify in Geneva

Look for data with these filters:
- `test.scenario = "real-backend-loadtest"`
- `test.timestamp` = (timestamp shown in test output)

## Quick Commands

```bash
# Traces
./run-real-backend-test.sh trace light     # 1k/sec, 1min
./run-real-backend-test.sh trace moderate  # 5k/sec, 5min
./run-real-backend-test.sh trace heavy     # 10k/sec, 10min
./run-real-backend-test.sh trace stress    # 50k/sec, 10min

# Logs
./run-real-backend-test.sh logs light      # 1k/sec, 1min
./run-real-backend-test.sh logs moderate   # 5k/sec, 5min
./run-real-backend-test.sh logs heavy      # 10k/sec, 10min
./run-real-backend-test.sh logs stress     # 50k/sec, 10min
```

## Load Levels

| Level | Rate | Duration | Total Items |
|-------|------|----------|-------------|
| light | 1k/sec | 1 min | ~60k |
| moderate | 5k/sec | 5 min | ~1.5M |
| heavy | 10k/sec | 10 min | ~6M |
| stress | 50k/sec | 10 min | ~30M |

## Troubleshooting

**Auth errors?**
- Check `GENEVA_CERT_PATH` exists
- Verify `GENEVA_CERT_PASSWORD` is correct
- Confirm `GENEVA_TENANT` matches your Azure tenant

**No data in Geneva?**
- Wait 5-10 minutes for ingestion
- Check collector.log: `tail -f collector.log`
- Verify account/namespace are correct

**Connection errors?**
- Test endpoint: `curl $GENEVA_ENDPOINT`
- Check firewall/network rules
- Verify endpoint URL is correct

## File Locations

- Config template: `config-real-backend.yaml.template`
- Env template: `.env.template`
- Your env file: `.env` (DO NOT COMMIT!)
- Test script: `run-real-backend-test.sh`
- Collector logs: `collector.log`

## Full Documentation

See `testbed/REAL_BACKEND_TESTING.md` for complete guide including:
- Detailed troubleshooting
- Performance benchmarks
- CI/CD integration
- Security best practices

## Example Session

```bash
$ cd gigwarmexporter/opentelemetry-collector/testbed
$ cp .env.template .env
$ nano .env  # Fill in credentials
$ ./run-real-backend-test.sh trace moderate

=== GigWarm Real Backend Load Test ===

Loading environment variables from .env...
✓ Using certificate authentication: /path/to/cert.p12
✓ Collector built

=== Configuration Summary ===
Endpoint: https://gcs.ppe.monitoring.core.windows.net
Account: MyAccount
Namespace: MyNamespace
Region: eastus
Auth Method: Certificate
Test Scenario: real-backend-loadtest
Test Timestamp: 2025-10-08T15:30:00Z

Test Type: trace
Load Level: moderate (5000 items/sec for 300 seconds)

✓ Generated config: config-real-backend.yaml
✓ Collector started (PID: 12345)
✓ Collector is ready

=== Starting Load Test ===
Duration: 300s
Rate: 5000 items/sec

[... telemetrygen output ...]

=== Test Complete ===
Total Duration: 300s
Items Sent: 1500000
Average Rate: 5000 items/sec

Check your Azure Geneva dashboard to verify data arrival
Test identifiers:
  - test.scenario: real-backend-loadtest
  - test.timestamp: 2025-10-08T15:30:00Z
```

## What's Next?

1. ✅ Run light test to validate
2. ✅ Check Geneva for your data
3. ✅ Scale up to moderate/heavy tests
4. ✅ Compare with mock receiver performance
5. ✅ Integrate into your CI/CD pipeline

Need help? See full docs in `testbed/REAL_BACKEND_TESTING.md`
