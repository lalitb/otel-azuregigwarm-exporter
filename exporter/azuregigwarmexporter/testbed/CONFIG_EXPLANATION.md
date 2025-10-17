# Azure GigWarm Exporter Configuration Guide

This document explains the configuration options for the OpenTelemetry Collector with Azure GigWarm exporter.

## Pipeline Architecture

Data flows through the collector in this order:

```
OTLP Receiver → Batch Processor → Persistent Queue (Disk) → Retry Logic → GigWarm Exporter → Batch Retry → Azure Geneva
                                          ↓
                                   WAL to ./storage/
                                   (survives crashes)
                                          ↓
                                  Encode & Split into Batches
                                          ↓
                                  Upload Batches Concurrently
                                          ↓
                                  Retry Failed Batches Only
```

**Key Features:**
- ✅ **Persistent Queue** enabled by default (prevents data loss on crashes)
- ✅ **Two-level retry** (batch-level + export-level)
- ✅ **Concurrent uploads** for high throughput
- ✅ **Automatic compaction** to manage disk usage

## Configuration Components

### 1. OTLP Receiver

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317  # Binds to all interfaces (IPv4 and IPv6)
      http:
        endpoint: 0.0.0.0:4318
```

**Purpose:** Receives telemetry data via OTLP protocol (gRPC or HTTP)

**Key Settings:**
- `0.0.0.0` - Listens on all network interfaces
- Port `4317` - Standard OTLP gRPC port
- Port `4318` - Standard OTLP HTTP port

**Note:** Use `127.0.0.1:4317` when connecting from localhost to avoid IPv4/IPv6 resolution issues on macOS.

---

### 2. Batch Processor

```yaml
processors:
  batch:
    send_batch_size: 1024  # Send when batch reaches this size
    timeout: 2s            # Send when timeout expires
```

**Purpose:** Batches telemetry data for efficient export

**How it works:**
- Collects incoming telemetry records
- Sends a batch when **either** condition is met:
  - Batch size reaches `send_batch_size` (1024 records), OR
  - `timeout` expires (2 seconds)

**Default Values:**
- `send_batch_size: 8192`
- `timeout: 200ms`

**Tuning Guide:**
- **Higher throughput**: Increase `send_batch_size`, decrease `timeout`
- **Lower latency**: Decrease `send_batch_size`, decrease `timeout`
- **Lower memory**: Decrease `send_batch_size`

---

### 3. File Storage Extension (Persistent Queue)

```yaml
extensions:
  file_storage:
    directory: ./storage  # Directory for persistent queue data
    timeout: 10s          # Timeout for file operations
    compaction:
      directory: ./storage
      on_start: true      # Clean up old data on startup
      on_rebound: true    # Compact when queue shrinks
```

**Purpose:** Provides persistent Write-Ahead Log (WAL) to survive collector crashes

**How it works:**
- Queue data is written to disk before export attempts
- If collector crashes, queue state is restored on restart
- Automatic compaction prevents disk space growth
- Falls back to memory if disk is unavailable

**Default Values:**
- `directory: ./storage` (production: `/var/lib/otelcol/storage`)
- `timeout: 10s`
- Compaction enabled with 5MB/10MB thresholds

**Benefits:**
- ✅ **Zero data loss** during collector crashes
- ✅ **Automatic recovery** on restart
- ✅ **Disk space management** via compaction
- ✅ **Production ready** out of the box

---

### 4. Sending Queue

```yaml
sending_queue:
  enabled: true
  num_consumers: 10   # Number of concurrent export workers
  queue_size: 5000    # Max batches in queue
  storage: file_storage  # *** PERSIST TO DISK ***
```

**Purpose:** Persistent queue that decouples receiving from exporting

**How it works:**
1. Batches from the batch processor are added to the queue
2. Queue is persisted to disk via `file_storage` extension
3. `num_consumers` workers pull batches from the queue and export them concurrently
4. If queue is full, data overflows to disk (up to disk capacity)

**Default Values:**
- `enabled: true`
- `num_consumers: 10`
- `queue_size: 5000`
- `storage: file_storage` (**persistent queue enabled**)

**Tuning Guide:**
- **Higher throughput**: Increase `num_consumers` and `queue_size`
- **Lower memory**: Decrease `queue_size` (data overflows to disk)
- **More disk usage**: Increase `queue_size`

**Behavior when queue full:**
- Queue overflows to disk (via `file_storage`)
- Data persists until successfully exported or disk fills
- No data loss during collector crashes

---

### 5. Export-Level Retry Configuration

```yaml
retry_on_failure:
  enabled: true
  initial_interval: 5s    # First retry after 5 seconds
  max_interval: 30s       # Cap retry interval at 30 seconds
  max_elapsed_time: 300s  # Give up after 5 minutes total
```

**Purpose:** Automatically retries failed exports with exponential backoff

**How it works:**
1. If export fails, wait `initial_interval` (5s) and retry
2. Each retry increases wait time by `multiplier` (1.5x default)
3. Wait time capped at `max_interval` (30s)
4. Stops retrying after `max_elapsed_time` (5 minutes)
5. Randomization factor (0.5) prevents thundering herd

**Default Values:**
- `enabled: true`
- `initial_interval: 5s`
- `max_interval: 30s`
- `max_elapsed_time: 300s` (5 minutes)
- `multiplier: 1.5`
- `randomization_factor: 0.5`

**Retry Schedule Example:**
```
Attempt 1: Immediate
Attempt 2: ~5s   (5s ± 50% randomization)
Attempt 3: ~7.5s (7.5s ± 50%)
Attempt 4: ~11.25s
Attempt 5: ~16.875s
Attempt 6: ~25.3s
Attempt 7+: ~30s (max_interval)
```

**Tuning Guide:**
- **Transient failures**: Keep defaults
- **Longer retry period**: Increase `max_elapsed_time`
- **Faster retry**: Decrease `initial_interval`
- **Aggressive retry**: Increase `multiplier`

---

### 6. Batch-Level Retry

```yaml
batch_retry:
  enabled: true          # Enable batch-level retry
  max_retries: 3         # Max retry attempts per batch
  initial_interval: 100ms # Initial backoff interval
  max_interval: 5s       # Max backoff interval
  multiplier: 2.0        # Backoff multiplier
```

**Purpose:** Retries individual failed batches without re-encoding successful batches

**How it works:**
1. After encoding, telemetry is split into multiple compressed batches
2. All batches are uploaded concurrently
3. If a batch fails, it's retried individually with exponential backoff
4. Successful batches are NOT re-uploaded
5. Only if all retries for a batch fail, the error propagates to exporterhelper

**Default Values:**
- `enabled: true`
- `max_retries: 3`
- `initial_interval: 100ms`
- `max_interval: 5s`
- `multiplier: 2.0`

**Retry Schedule Example (per batch):**
```
Attempt 1: Immediate
Attempt 2: ~100ms backoff
Attempt 3: ~200ms backoff
Attempt 4: ~400ms backoff
```

**Benefits:**
- **Efficiency**: Avoids re-encoding and re-uploading successful batches
- **Fast recovery**: Short backoff intervals (100ms-5s) for transient errors
- **Reduced load**: Only failed batches are retried, not the entire export

**Difference from retry_on_failure:**
- `batch_retry` operates **within** a single pushLogs/pushTraces call
- `retry_on_failure` retries the **entire** pushLogs/pushTraces call
- Use both for maximum resilience

**Tuning Guide:**
- **Transient network errors**: Keep defaults (3 retries, 100ms-5s)
- **Longer retry window**: Increase `max_retries` to 5-10
- **Faster failure detection**: Decrease `max_retries` to 1-2
- **More aggressive backoff**: Increase `multiplier` to 3.0

---

### 7. Azure GigWarm Exporter

```yaml
azuregigwarm:
  endpoint: "https://gcs.ppe.monitoring.core.windows.net"
  environment: "Test"
  account: "PipelineAgent2Demo"
  namespace: "PAdemo2"
  region: "southeastasia"
  config_major_version: 2
  auth_method: 1  # 0=MSI, 1=Certificate
  tenant: "your-tenant-id"
  role_name: "your-role"
  role_instance: "instance-01"
  cert_path: "/path/to/cert.p12"
  cert_password: "password"
```

**Purpose:** Exports telemetry to Azure Geneva Warm storage

**Authentication Methods:**
- `auth_method: 0` - Managed Service Identity (MSI)
- `auth_method: 1` - Certificate-based (requires `cert_path` and `cert_password`)

**Key Settings:**
- `endpoint` - Geneva Config Service endpoint
- `environment` - Geneva environment name
- `account` - Geneva account name
- `namespace` - Geneva namespace
- `region` - Azure region (e.g., "southeastasia", "eastus")
- `config_major_version` - Geneva config version (1 or 2)

---

## Complete Pipeline Flow

### Normal Operation (Success)

```
1. OTLP Receiver receives 10,000 logs
2. Batch Processor:
   - Waits for 1024 logs OR 2 seconds
   - Creates ~10 batches of 1024 logs each
3. Sending Queue:
   - Adds 10 batches to queue
   - 10 workers pull batches concurrently
4. GigWarm Exporter:
   - Encodes and compresses each batch
   - Uploads to Geneva
5. Geneva responds: 202 Accepted
6. ✅ Success - data delivered
```

### Failure Scenario (Transient Error)

```
1. OTLP Receiver receives 1000 logs
2. Batch Processor creates 1 batch
3. Sending Queue dispatches to worker
4. GigWarm Exporter uploads to Geneva
5. Geneva responds: 503 Service Unavailable
6. Retry Logic:
   - Wait 5 seconds
   - Retry upload
7. Geneva responds: 202 Accepted
8. ✅ Success after retry
```

### Failure Scenario (Queue Full)

```
1. OTLP Receiver receives 100,000 logs/sec
2. Batch Processor creates many batches
3. Sending Queue:
   - Queue has 5000 batches (full)
   - New batches are DROPPED
4. ❌ Data loss (queue overflow)

Solution: Increase queue_size or num_consumers, or enable persistent queue
```

### Batch Retry Scenario (Partial Failure)

```
1. OTLP Receiver receives 5000 logs
2. Batch Processor creates 5 batches
3. Sending Queue dispatches to worker
4. GigWarm Exporter:
   - Encodes into 10 compressed batches
   - Uploads all 10 batches concurrently
5. Upload Results:
   - Batches 0-7: SUCCESS (202 Accepted)
   - Batch 8: FAILED (503 Service Unavailable)
   - Batch 9: FAILED (503 Service Unavailable)
6. Batch Retry Logic:
   - Batch 8: Retry after 100ms → SUCCESS
   - Batch 9: Retry after 100ms → SUCCESS
7. ✅ All batches uploaded successfully

Without batch retry: Would retry entire export (re-encode + re-upload all 10 batches)
With batch retry: Only retried 2 failed batches (80% reduction in retry work)
```

---

## Performance Tuning

### Low Latency (Real-time)
```yaml
batch:
  send_batch_size: 100
  timeout: 100ms
sending_queue:
  num_consumers: 20
```

### High Throughput
```yaml
batch:
  send_batch_size: 8192
  timeout: 5s
sending_queue:
  num_consumers: 50
  queue_size: 10000
```

### Low Memory
```yaml
batch:
  send_batch_size: 512
  timeout: 1s
sending_queue:
  num_consumers: 5
  queue_size: 1000
```

### Reliable (No Data Loss)
```yaml
# Add persistent queue with file_storage extension
extensions:
  file_storage:
    directory: /var/lib/otelcol/file_storage
    timeout: 10s

exporters:
  azuregigwarm:
    # ... existing config ...
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 5000
      storage: file_storage  # Persist to disk on overflow or shutdown

service:
  extensions: [file_storage]  # Enable the extension
  pipelines:
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [azuregigwarm]
```

**Benefits:**
- **Survives Collector crashes**: Data in queue persists to disk
- **Prevents data loss on restart**: Queue state restored after restart
- **Disk overflow protection**: Falls back to memory if disk full
- **Production ready**: Recommended for critical deployments

**Important Notes:**
- Ensure `directory` has sufficient disk space (monitor with alerts)
- Use fast disk (SSD recommended) to avoid performance degradation
- Directory must be writable by collector process
- Data durability depends on filesystem reliability

---

## Resilience and Data Loss Prevention

The GigWarm exporter implements multiple layers of resilience to prevent data loss:

### Resilience Layers (All Enabled by Default)

```
Layer 1: Batch-Level Retry (100ms-5s, 3 attempts)
   ↓ If all batch retries fail
Layer 2: Export-Level Retry (5s-30s, up to 5 minutes)
   ↓ If export retries timeout
Layer 3: In-Memory Queue (buffers 5000 batches)
   ↓ Simultaneously persisted to disk
Layer 4: Persistent Storage (WAL to ./storage/)
   ↓ Survives collector crashes and restarts
```

**All layers are enabled by default** - no additional configuration required for zero data loss!

### Data Loss Scenarios and Mitigations

#### Scenario 1: Transient Network Error (Geneva unavailable for <5 seconds)
**Mitigation:** Batch-level retry handles this automatically
```
Batch fails → Retry after 100ms → SUCCESS (no data loss)
```

#### Scenario 2: Geneva Service Restart (unavailable for 30 seconds)
**Mitigation:** Export-level retry with exponential backoff
```
Export fails → Wait 5s → Retry → Wait 7.5s → Retry → SUCCESS
Total time: ~45s including batch retries (no data loss)
```

#### Scenario 3: Geneva Outage (unavailable for 10 minutes)
**With persistent storage (enabled by default):**
```
Queue persists to disk → Retries continue → Geneva recovers → All data sent (✅ NO DATA LOSS)
```

#### Scenario 4: Collector Crash
**With persistent storage (enabled by default):**
```
Collector crashes → Queue persisted to disk
Collector restarts → Loads queue from disk → Resumes upload (✅ NO DATA LOSS)
```

#### Scenario 5: Partial Batch Failure (2 out of 10 batches fail)
**Mitigation:** Batch-level retry prevents re-uploading successful batches
```
Batches 0-7: SUCCESS (not retried)
Batch 8: FAIL → Retry → SUCCESS
Batch 9: FAIL → Retry → SUCCESS
Result: Only 2 batches retried instead of all 10 (efficient recovery)
```

### Recommended Configuration by Deployment Type

**Note:** Persistent storage is **enabled by default** in all configurations below.

#### Development/Testing
```yaml
extensions:
  file_storage:
    directory: ./storage  # Local directory

sending_queue:
  enabled: true
  num_consumers: 5
  queue_size: 1000  # Smaller queue
  storage: file_storage  # ✅ Still persistent

retry_on_failure:
  enabled: true
  max_elapsed_time: 60s  # Fail faster

batch_retry:
  enabled: true
  max_retries: 1
```

#### Production (Standard) - **DEFAULT**
```yaml
extensions:
  file_storage:
    directory: /var/lib/otelcol/storage

sending_queue:
  enabled: true
  num_consumers: 10
  queue_size: 5000
  storage: file_storage  # ✅ Persistent by default

retry_on_failure:
  enabled: true
  max_elapsed_time: 300s

batch_retry:
  enabled: true
  max_retries: 3
```

#### Production (High Throughput)
```yaml
extensions:
  file_storage:
    directory: /var/lib/otelcol/storage

sending_queue:
  enabled: true
  num_consumers: 50      # More workers
  queue_size: 20000      # Larger queue
  storage: file_storage  # ✅ Persistent

retry_on_failure:
  enabled: true
  max_elapsed_time: 600s

batch_retry:
  enabled: true
  max_retries: 5
```

### Persistent Storage - Always Enabled

**Persistent storage is enabled by default** for zero data loss.

**Storage Directory:**
- Development: `./storage` (auto-created)
- Production: `/var/lib/otelcol/storage` (must exist)
- Docker/K8s: Mount persistent volume

**Benefits:**
- ✅ Survives collector crashes
- ✅ No data loss on restart
- ✅ Automatic disk management
- ✅ Production-ready out of the box

### Monitoring for Resilience

Monitor these metrics to detect resilience issues:

```
# Queue depth (healthy: < 50% capacity)
otelcol_exporter_queue_size / otelcol_exporter_queue_capacity > 0.5

# Failed exports (healthy: 0)
rate(otelcol_exporter_send_failed_requests[5m]) > 0

# Batch retry attempts (healthy: low rate)
rate(azuregigwarm_batch_retry_attempts[5m])

# Data dropped due to queue overflow (healthy: 0)
rate(otelcol_exporter_enqueue_failed_spans[5m]) > 0
```

### Comparison to OpenTelemetry Recommendations

| Feature | OTel Recommendation | GigWarm Exporter | Status |
|---------|-------------------|------------------|--------|
| **Sending Queue** | Required for network exporters | ✅ Enabled by default | Implemented |
| **Retry with Backoff** | Required | ✅ Two-level retry (batch + export) | Enhanced |
| **Persistent Storage (WAL)** | Recommended for critical collectors | ✅ Supported via file_storage | Supported |
| **Message Queue Integration** | Optional for high durability | ⚠️ Use Kafka receiver/exporter | Not exporter-specific |
| **Queue Metrics** | Required for monitoring | ✅ Standard OTel metrics | Implemented |
| **Configurable Queue Size** | Required | ✅ Fully configurable | Implemented |

---

## Monitoring

Key metrics to monitor:
- `otelcol_exporter_queue_size` - Current queue depth
- `otelcol_exporter_queue_capacity` - Max queue size
- `otelcol_exporter_send_failed_requests` - Failed exports
- `otelcol_exporter_send_failed_retries` - Failed after retries
- `otelcol_processor_batch_batch_send_size` - Batch sizes
- `otelcol_processor_batch_timeout_trigger_send` - Batches sent by timeout

---

## Troubleshooting

### Logs not appearing in Geneva
1. Check collector logs for errors
2. Verify Geneva credentials and endpoint
3. Check network connectivity to Geneva
4. Look for `[DEBUG] upload_batch: SUCCESS` in logs

### High memory usage
1. Reduce `queue_size`
2. Reduce `send_batch_size`
3. Increase `num_consumers` to drain queue faster

### Data loss
1. Enable persistent queue with file_storage
2. Increase `queue_size`
3. Increase `num_consumers`
4. Check for `queue full` errors in logs

### Slow performance
1. Increase `num_consumers`
2. Increase `send_batch_size`
3. Check Geneva endpoint latency
4. Look for retry logs indicating failures

---

## Testing Configuration

To test your configuration:

```bash
# Send 100 test logs
./run-real-backend-test.sh logs 100

# Check collector logs for success
grep "upload_batch: SUCCESS" collector.log

# Send higher load (10,000 logs)
./run-real-backend-test.sh logs 10000
```

Expected output in `collector.log`:
```
[DEBUG] upload_batch: Starting upload for event=Log1
[DEBUG] uploader.upload: Getting ingestion info for event=Log1
[DEBUG] uploader.upload: Sending POST to https://...
[DEBUG] uploader.upload: Upload ACCEPTED (202)
[DEBUG] upload_batch: SUCCESS for event=Log1
```
