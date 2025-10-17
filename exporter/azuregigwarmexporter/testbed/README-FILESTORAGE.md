# File Storage (Persistent Queue) Setup Guide

This guide explains how to enable persistent storage (Write-Ahead Log) for the GigWarm exporter to prevent data loss during collector crashes.

## What is File Storage?

File storage provides a **persistent queue** that writes telemetry data to disk before attempting to export it. This ensures:

✅ **Data survives collector crashes** - Queue state persisted to disk
✅ **Data survives collector restarts** - Resumes from where it left off
✅ **Protection against data loss** - Even during unexpected shutdowns

## When to Use File Storage

### ✅ **Use File Storage When:**

- Running in **production environments**
- Data loss during crashes is **unacceptable**
- Collecting **critical telemetry** (security logs, compliance data)
- Deploying as a **Gateway collector** (aggregation tier)
- Geneva endpoint experiences **frequent outages**

### ⚠️ **Skip File Storage When:**

- Running in **development/testing** environments
- Data loss during crashes is **acceptable** (non-critical metrics)
- Disk I/O latency would **harm performance**
- Using **ephemeral storage** (containers without persistent volumes)

## Quick Start

### Option 1: Use the High-Reliability Config (Recommended)

```bash
# 1. Copy the high-reliability template
cp config-real-backend-persistent.yaml.template config-real-backend-persistent.yaml

# 2. Edit with your Geneva credentials
vi config-real-backend-persistent.yaml

# 3. Create storage directory
mkdir -p ./storage
chmod 755 ./storage

# 4. Set environment variable (optional)
export OTEL_FILE_STORAGE_DIR=/var/lib/otelcol/storage

# 5. Run the collector
../../../../bin/otelcontribcol_darwin_arm64 --config config-real-backend-persistent.yaml
```

### Option 2: Enable in Standard Config

Edit `config-real-backend.yaml` and uncomment these sections:

```yaml
# 1. Add extensions section
extensions:
  file_storage:
    directory: ./storage  # Or /var/lib/otelcol/storage in production
    timeout: 10s

# 2. Update sending_queue
exporters:
  azuregigwarm:
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 5000
      storage: file_storage  # *** ADD THIS LINE ***

# 3. Enable extension in service
service:
  extensions: [file_storage]  # *** ADD THIS LINE ***
```

## Configuration Details

### File Storage Extension

```yaml
extensions:
  file_storage:
    # Directory where queue data is persisted
    directory: /var/lib/otelcol/storage

    # Timeout for file operations
    timeout: 10s

    # Optional: Compaction settings (cleans up old data)
    compaction:
      directory: /var/lib/otelcol/storage
      on_start: true                        # Clean up on startup
      on_rebound: true                      # Compact when queue shrinks
      rebound_needed_threshold_mib: 5       # Trigger at 5MB
      rebound_trigger_threshold_mib: 10     # Must have 10MB to compact
```

### Storage Directory Requirements

| Environment | Recommended Path | Notes |
|-------------|------------------|-------|
| **Production (Linux)** | `/var/lib/otelcol/storage` | Standard location, needs setup |
| **Production (Windows)** | `C:\ProgramData\otelcol\storage` | Standard location |
| **Docker/K8s** | `/var/lib/otelcol/storage` | Mount as persistent volume |
| **Development** | `./storage` | Local directory (relative path) |

**Requirements:**
- Directory must exist and be writable by collector process
- Sufficient disk space (monitor with alerts)
- Fast disk recommended (SSD preferred)
- Persistent storage (not ephemeral for containers)

### Connecting to Sending Queue

```yaml
exporters:
  azuregigwarm:
    sending_queue:
      enabled: true
      num_consumers: 20      # Increase for production
      queue_size: 10000      # Larger queue for high reliability
      storage: file_storage  # *** Reference extension ID ***
```

## Docker/Kubernetes Setup

### Docker Compose

```yaml
version: '3.8'
services:
  otel-collector:
    image: your-otelcol-image
    volumes:
      - otel-storage:/var/lib/otelcol/storage  # Persistent volume
      - ./config-real-backend-persistent.yaml:/etc/otelcol/config.yaml
    environment:
      - OTEL_FILE_STORAGE_DIR=/var/lib/otelcol/storage

volumes:
  otel-storage:  # Named volume for persistence
```

### Kubernetes StatefulSet

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: otel-collector
spec:
  serviceName: otel-collector
  replicas: 1
  template:
    spec:
      containers:
      - name: otel-collector
        image: your-otelcol-image
        volumeMounts:
        - name: storage
          mountPath: /var/lib/otelcol/storage
  volumeClaimTemplates:
  - metadata:
      name: storage
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 10Gi  # Adjust based on needs
```

## Monitoring File Storage

### Key Metrics

```promql
# Storage directory size (bytes)
otelcol_file_storage_size_bytes

# Items in persistent queue
otelcol_file_storage_items

# Compaction operations
rate(otelcol_file_storage_compactions_total[5m])

# File operation errors
rate(otelcol_file_storage_errors_total[5m])
```

### Disk Space Alerts

```promql
# Alert when storage exceeds 80% of available disk
(otelcol_file_storage_size_bytes / disk_total_bytes) > 0.8
```

### Log Messages to Watch

```
# Successful persistence
"persisted batch to file storage"

# Queue restored after restart
"restored queue from file storage" items=1234

# Disk space warning
"file storage directory running low on space"

# Compaction completed
"file storage compaction completed" reclaimed_bytes=52428800
```

## Testing File Storage

### Test 1: Verify Persistence on Shutdown

```bash
# 1. Start collector
../../../../bin/otelcontribcol_$(go env GOOS)_$(go env GOARCH) --config config-real-backend-persistent.yaml

# 2. Send some data
telemetrygen logs --otlp-endpoint 127.0.0.1:4317 --otlp-insecure --logs 1000

# 3. Gracefully shutdown (Ctrl+C)

# 4. Check storage directory
ls -lh ./storage/
# Should see files with queue data

# 5. Restart collector
../../../../bin/otelcontribcol_$(go env GOOS)_$(go env GOARCH) --config config-real-backend-persistent.yaml

# 6. Check logs for restoration
grep "restored queue" collector.log
```

### Test 2: Verify Crash Recovery

```bash
# 1. Start collector
../../../../bin/otelcontribcol_$(go env GOOS)_$(go env GOARCH) --config config-real-backend-persistent.yaml &
COLLECTOR_PID=$!

# 2. Send data continuously
telemetrygen logs --otlp-endpoint 127.0.0.1:4317 --duration 60s --rate 100 &

# 3. Kill collector abruptly (simulates crash)
kill -9 $COLLECTOR_PID

# 4. Wait a few seconds
sleep 3

# 5. Restart collector
../../../../bin/otelcontribcol_$(go env GOOS)_$(go env GOARCH) --config config-real-backend-persistent.yaml

# 6. Verify queue restored
grep "restored queue from file storage" collector.log
```

## Troubleshooting

### Permission Errors

```
Error: failed to create file storage: open ./storage: permission denied
```

**Solution:**
```bash
# Create directory with proper permissions
mkdir -p /var/lib/otelcol/storage
chmod 755 /var/lib/otelcol/storage
chown otelcol:otelcol /var/lib/otelcol/storage  # If running as otelcol user
```

### Disk Space Issues

```
Error: file storage directory full
```

**Solutions:**
1. Increase disk space allocation
2. Reduce `queue_size` in config
3. Enable compaction (should be automatic)
4. Monitor and alert on disk usage

### Data Not Persisting

**Check:**
1. `storage: file_storage` is set in `sending_queue`
2. `extensions: [file_storage]` is in `service` section
3. Directory exists and is writable
4. No errors in collector logs

### Performance Degradation

If file I/O is slowing down the collector:

1. **Use faster disk** (SSD instead of HDD)
2. **Reduce queue size** to limit disk writes
3. **Disable compaction** during peak hours
4. **Consider in-memory queue** if acceptable

## Production Recommendations

### Small Deployment (1-10 agents)
```yaml
file_storage:
  directory: /var/lib/otelcol/storage
sending_queue:
  queue_size: 5000
  storage: file_storage
```
**Disk:** 5-10 GB

### Medium Deployment (10-100 agents)
```yaml
file_storage:
  directory: /var/lib/otelcol/storage
sending_queue:
  queue_size: 10000
  num_consumers: 20
  storage: file_storage
```
**Disk:** 20-50 GB

### Large Deployment (100+ agents or Gateway)
```yaml
file_storage:
  directory: /mnt/fast-ssd/otelcol/storage
sending_queue:
  queue_size: 50000
  num_consumers: 50
  storage: file_storage
```
**Disk:** 100+ GB on dedicated SSD

## Summary

✅ File storage provides **zero data loss** during collector crashes
✅ Easy to enable - just 3 configuration changes
✅ Works with all deployment types (bare metal, Docker, K8s)
✅ Recommended for **production deployments**

For questions or issues, see [CONFIG_EXPLANATION.md](CONFIG_EXPLANATION.md) for detailed resilience documentation.
