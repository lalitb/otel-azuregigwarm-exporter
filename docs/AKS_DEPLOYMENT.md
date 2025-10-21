# Deploying to Azure Kubernetes Service (AKS) with Workload Identity

This guide walks through deploying the OpenTelemetry Collector with Azure GigWarm exporter to AKS using Azure Workload Identity for authentication.

## Prerequisites

- Azure subscription
- AKS cluster (or ability to create one)
- Azure CLI installed
- kubectl configured to access your AKS cluster
- Azure Container Registry (ACR) for Docker images
- User-assigned Managed Identity with Geneva permissions

## Step 1: Enable Workload Identity on AKS

If your AKS cluster doesn't have Workload Identity enabled:

```bash
# Enable OIDC issuer and Workload Identity
az aks update \
  --resource-group <resource-group> \
  --name <aks-cluster-name> \
  --enable-oidc-issuer \
  --enable-workload-identity
```

Get the OIDC issuer URL (you'll need this later):

```bash
export AKS_OIDC_ISSUER=$(az aks show \
  --resource-group <resource-group> \
  --name <aks-cluster-name> \
  --query "oidcIssuerProfile.issuerUrl" \
  -o tsv)

echo "OIDC Issuer: $AKS_OIDC_ISSUER"
```

## Step 2: Create User-Assigned Managed Identity

```bash
# Create managed identity
az identity create \
  --resource-group <resource-group> \
  --name otel-collector-identity

# Get identity details
export USER_ASSIGNED_CLIENT_ID=$(az identity show \
  --resource-group <resource-group> \
  --name otel-collector-identity \
  --query 'clientId' \
  -o tsv)

export USER_ASSIGNED_OBJECT_ID=$(az identity show \
  --resource-group <resource-group> \
  --name otel-collector-identity \
  --query 'principalId' \
  -o tsv)

echo "Client ID: $USER_ASSIGNED_CLIENT_ID"
echo "Object (Principal) ID: $USER_ASSIGNED_OBJECT_ID"
```

## Step 3: Register Managed Identity with Geneva

Register the managed identity with your Geneva account through the Geneva/Jarvis portal or via support ticket.

**Information needed:**
- **Object ID** (Principal ID): `$USER_ASSIGNED_OBJECT_ID`
- **Geneva Account** name (e.g., `PipelineAgent2Demo`)
- **Geneva Namespace** name (e.g., `PAdemo2`)
- **Permissions** required: Write access for telemetry ingestion

**Important:** Wait for Geneva team confirmation before proceeding. The identity must be registered in Geneva's authorization system.

## Step 4: Create Kubernetes Service Account

```bash
# Create namespace (if not exists)
kubectl create namespace otel-system

# Create service account with Workload Identity annotation
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: otel-collector-sa
  namespace: otel-system
  annotations:
    azure.workload.identity/client-id: "$USER_ASSIGNED_CLIENT_ID"
EOF
```

Verify the service account was created:

```bash
kubectl get sa -n otel-system otel-collector-sa -o yaml
```

## Step 5: Create Federated Identity Credential

Link the Kubernetes service account to the Azure managed identity:

```bash
az identity federated-credential create \
  --name otel-collector-federated-identity \
  --identity-name otel-collector-identity \
  --resource-group <resource-group> \
  --issuer "$AKS_OIDC_ISSUER" \
  --subject system:serviceaccount:otel-system:otel-collector-sa \
  --audience api://AzureADTokenExchange
```

Verify the federated credential:

```bash
az identity federated-credential list \
  --identity-name otel-collector-identity \
  --resource-group <resource-group> \
  -o table
```

**Important fields to verify:**
- **Issuer**: Must match `$AKS_OIDC_ISSUER`
- **Subject**: Must be `system:serviceaccount:otel-system:otel-collector-sa`
- **Audience**: Must be `api://AzureADTokenExchange`

## Step 6: Build and Push Docker Image

### 6.1 Build on AMD64 Linux (Required)

The collector must be built for Linux AMD64. If you're on macOS/Windows, use a Linux VM:

```bash
# On Linux AMD64 VM or machine

# Clone your exporter repository
git clone https://github.com/your-org/otel-azuregigwarm-exporter.git
cd otel-azuregigwarm-exporter/examples

# Build using the Makefile (handles Rust FFI from crates.io automatically)
make build

# The binary is now at: ./bin/otelcol-azuregigwarm
cp ./bin/otelcol-azuregigwarm ~/otelcontribcol_linux_amd64
```

**Note:** The Rust FFI bridge is now built from **crates.io** (geneva-uploader-ffi v0.3.0) - no need to clone opentelemetry-rust-contrib!

### 6.2 Create Dockerfile

With static linking, the Dockerfile is much simpler:

```dockerfile
FROM ubuntu:22.04

# Install CA certificates
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy the statically-linked collector binary (includes Rust FFI)
COPY otelcontribcol_linux_amd64 /otelcontribcol

# Make executable
RUN chmod +x /otelcontribcol

# Set entrypoint
ENTRYPOINT ["/otelcontribcol"]
CMD ["--config=/etc/otel/config.yaml"]
```

**Note:** No need to copy separate Rust library or run `ldconfig` - the binary is statically linked!

### 6.3 Build and Push

```bash
# Build Docker image
docker build -t <your-acr>.azurecr.io/otel-collector-geneva:v1.0.0 .

# Login to ACR
az acr login --name <your-acr>

# Push image
docker push <your-acr>.azurecr.io/otel-collector-geneva:v1.0.0
```

## Step 7: Create ConfigMap with Collector Configuration

```bash
export GENEVA_ENDPOINT="https://abc.monitoring.core.windows.net"
export GENEVA_ACCOUNT="YourGenevaAccount"
export GENEVA_NAMESPACE="YourGenevaNamespace"
export AZURE_TENANT_ID="your-tenant-id"

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-collector-config
  namespace: otel-system
data:
  config.yaml: |
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
        endpoint: "$GENEVA_ENDPOINT"
        environment: "Test"
        account: "$GENEVA_ACCOUNT"
        namespace: "$GENEVA_NAMESPACE"
        region: "eastus"
        config_major_version: 2
        auth_method: 2  # WorkloadIdentity
        workload_identity_resource: "$GENEVA_ENDPOINT"
        tenant: "$AZURE_TENANT_ID"
        role_name: "otel-collector"
        role_instance: "aks-instance-001"

        # Retry configuration
        retry_on_failure:
          enabled: true
          initial_interval: 5s
          max_interval: 30s
          max_elapsed_time: 300s

        # Batch retry configuration
        batch_retry:
          enabled: true
          max_retries: 3
          initial_interval: 100ms
          max_interval: 5s
          multiplier: 2.0

    extensions:
      health_check:
        endpoint: 0.0.0.0:13133

    service:
      extensions: [health_check]
      pipelines:
        logs:
          receivers: [otlp]
          processors: [batch]
          exporters: [azuregigwarm]
        traces:
          receivers: [otlp]
          processors: [batch]
          exporters: [azuregigwarm]
EOF
```

**Important Configuration Notes:**

| Parameter | Description | Example |
|-----------|-------------|---------|
| `endpoint` | Geneva GCS endpoint | `https://gcs.ppe.monitoring.core.windows.net` (PPE)<br>`https://gcs.prod.monitoring.core.windows.net` (Prod) |
| `workload_identity_resource` | Token audience | `https://monitor.core.windows.net/` |
| `auth_method` | Authentication type | `2` for Workload Identity |
| `account` | Geneva account name | `PipelineAgent2Demo` |
| `namespace` | Geneva namespace | `PAdemo2` |
| `tenant` | Azure AD tenant ID | `72f988bf-86f1-41af-91ab-2d7cd011db47` |

## Step 8: Deploy to Kubernetes

```bash
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otel-collector
  namespace: otel-system
  labels:
    app: otel-collector
spec:
  replicas: 1
  selector:
    matchLabels:
      app: otel-collector
  template:
    metadata:
      labels:
        app: otel-collector
        azure.workload.identity/use: "true"  # CRITICAL: Enable Workload Identity injection
    spec:
      serviceAccountName: otel-collector-sa
      containers:
      - name: otel-collector
        image: <your-acr>.azurecr.io/otel-collector-geneva:v1.0.0
        ports:
        - containerPort: 4317
          name: otlp-grpc
          protocol: TCP
        - containerPort: 4318
          name: otlp-http
          protocol: TCP
        env:
        - name: AZURE_CLIENT_ID
          value: "$USER_ASSIGNED_CLIENT_ID"
        - name: AZURE_TENANT_ID
          value: "$AZURE_TENANT_ID"
        volumeMounts:
        - name: config
          mountPath: /etc/otel
          readOnly: true
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /
            port: 13133
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 13133
          initialDelaySeconds: 10
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: otel-collector-config
---
apiVersion: v1
kind: Service
metadata:
  name: otel-collector
  namespace: otel-system
spec:
  selector:
    app: otel-collector
  ports:
  - name: otlp-grpc
    port: 4317
    targetPort: 4317
    protocol: TCP
  - name: otlp-http
    port: 4318
    targetPort: 4318
    protocol: TCP
  type: ClusterIP
EOF
```

**Critical Deployment Requirements:**

1. ✅ Label `azure.workload.identity/use: "true"` on pod template
2. ✅ Service account annotation with managed identity client ID
3. ✅ Environment variables `AZURE_CLIENT_ID` and `AZURE_TENANT_ID`
4. ✅ Service account name matches federated credential subject

## Step 9: Verify Deployment

### Check Pod Status

```bash
kubectl get pods -n otel-system
```

Expected output:
```
NAME                              READY   STATUS    RESTARTS   AGE
otel-collector-xxxxxxxxxx-xxxxx   1/1     Running   0          30s
```

### Check Logs

```bash
kubectl logs -n otel-system deployment/otel-collector --tail=50
```

Expected output (success):
```
Starting otelcontribcol...
Starting extensions...
Starting AzureGigWarm exporter endpoint=https://gcs.ppe.monitoring.core.windows.net
Starting GRPC server endpoint=[::]:4317
Starting HTTP server endpoint=[::]:4318
Everything is ready. Begin running and processing data.
```

### Verify Workload Identity Injection

```bash
kubectl exec -n otel-system deployment/otel-collector -- env | grep AZURE
```

Expected output:
```
AZURE_CLIENT_ID=9f2d364a-3eb8-4f68-bc93-6f564cdb66a8
AZURE_TENANT_ID=72f988bf-86f1-41af-91ab-2d7cd011db47
AZURE_FEDERATED_TOKEN_FILE=/var/run/secrets/azure/tokens/azure-identity-token
AZURE_AUTHORITY_HOST=https://login.microsoftonline.com/
```

If `AZURE_FEDERATED_TOKEN_FILE` is missing, Workload Identity injection is not working. Check:
- Pod label `azure.workload.identity/use: "true"` is present
- Service account annotation is correct
- AKS has Workload Identity enabled

## Step 10: Test the Deployment

### Send Test Log

```bash
kubectl run test-logs --image=curlimages/curl --rm -it --restart=Never -- sh -c 'curl -X POST http://otel-collector.otel-system:4318/v1/logs -H "Content-Type: application/json" -d @- << "EOFDATA"
{
  "resourceLogs": [{
    "scopeLogs": [{
      "logRecords": [{
        "timeUnixNano": "'$(date +%s)000000000'",
        "severityNumber": 9,
        "severityText": "INFO",
        "body": {"stringValue": "Test log from AKS"}
      }]
    }]
  }]
}
EOFDATA
'
```

### Send Test Trace

```bash
kubectl run test-traces --image=curlimages/curl --rm -it --restart=Never -- sh -c 'curl -X POST http://otel-collector.otel-system:4318/v1/traces -H "Content-Type: application/json" -d @- << "EOFDATA"
{
  "resourceSpans": [{
    "scopeSpans": [{
      "spans": [{
        "traceId": "'$(openssl rand -hex 16)'",
        "spanId": "'$(openssl rand -hex 8)'",
        "name": "test-span-aks",
        "kind": 1,
        "startTimeUnixNano": "'$(date +%s)000000000'",
        "endTimeUnixNano": "'$(expr $(date +%s) + 1)000000000'",
        "attributes": [{
          "key": "test.source",
          "value": {"stringValue": "aks-manual-test"}
        }]
      }]
    }]
  }]
}
EOFDATA
'
```

### Check Upload Success

```bash
kubectl logs -n otel-system deployment/otel-collector --tail=20
```

Expected output for logs (success):
```
pushLogs called log_record_count=1
Marshaled logs to protobuf data_size=58
Encoded logs into batches batch_count=1
Sending upload request to URL: "https://eastus-shared.ppe.warm.ingest.monitor.core.windows.net/..."
Upload Successful
```

Expected output for traces (success):
```
pushTraces called span_count=1
Marshaled traces to protobuf data_size=125
Encoded traces into batches batch_count=1
Sending upload request to URL: "https://eastus-shared.ppe.warm.ingest.monitor.core.windows.net/..."
Upload Successful
```

If you see `Upload Successful`, the data has been sent to Geneva! 🎉

## Troubleshooting

### Issue: No logs appearing in Geneva

**Symptoms:** Upload shows success but data not in Geneva

**Possible causes:**
1. **Old timestamp:** Test data using old timestamps (e.g., 2023) won't appear in current queries
2. **Wrong environment:** Querying Production tables but sending to PPE
3. **Geneva registration pending:** Managed identity not yet authorized

**Solutions:**
- Always use current timestamps: `$(date +%s%N)`
- Verify you're querying the correct Geneva environment (PPE vs Prod)
- Check with Geneva team that managed identity is registered

### Issue: Authentication errors (AADSTS70011)

**Symptoms:**
```
geneva upload failed: AADSTS70011: The scope https://monitor.azure.com/ is not valid
```

**Possible causes:**
1. Wrong `workload_identity_resource` value
2. Federated credential misconfigured
3. Wrong OIDC issuer

**Solutions:**
- Set `workload_identity_resource` value (e.g., `https://monitor.core.windows.net`) corresponding to `endpoint` (e.g., `https://gcs.ppe.monitoring.core.windows.net`)
- Verify federated credential subject: `az identity federated-credential list --identity-name otel-collector-identity --resource-group <rg>`
- Ensure federated credential issuer matches AKS OIDC issuer

### Issue: Pod fails to start

**Symptoms:** Pod in `CrashLoopBackOff` or `Error` state

**Possible causes:**
1. Missing Rust library
2. Image pull errors
3. Configuration errors

**Solutions:**
```bash
# Check pod events
kubectl describe pod -n otel-system -l app=otel-collector

# Check logs
kubectl logs -n otel-system -l app=otel-collector --previous

# Common fixes:
# - Verify libgeneva_uploader_ffi.so is in Docker image at /usr/local/lib/
# - Run ldconfig in Dockerfile
# - Check ACR authentication: az acr login --name <acr-name>
```

### Issue: Workload Identity not injected

**Symptoms:** Missing `AZURE_FEDERATED_TOKEN_FILE` environment variable

**Solutions:**
1. Verify pod label exists:
   ```bash
   kubectl get pod -n otel-system -l app=otel-collector -o jsonpath='{.items[0].metadata.labels}'
   ```
   Must include: `azure.workload.identity/use: "true"`

2. Check service account annotation:
   ```bash
   kubectl get sa -n otel-system otel-collector-sa -o yaml
   ```
   Must include: `azure.workload.identity/client-id: "<client-id>"`

3. Verify AKS has Workload Identity enabled:
   ```bash
   az aks show --resource-group <rg> --name <cluster> --query "oidcIssuerProfile.enabled"
   az aks show --resource-group <rg> --name <cluster> --query "securityProfile.workloadIdentity.enabled"
   ```

### Issue: Upload hangs or times out

**Possible causes:**
1. Network connectivity issues
2. Geneva endpoint unreachable
3. Firewall/NSG blocking traffic

**Solutions:**
```bash
# Test network connectivity from pod
kubectl exec -n otel-system deployment/otel-collector -- curl -v https://gcs.ppe.monitoring.core.windows.net

# Check NSG rules allow outbound HTTPS
# Check if private endpoint is required for Geneva
```

## Monitoring and Observability

### Add Health Check Extension

Update ConfigMap to include health check:

```yaml
extensions:
  health_check:
    endpoint: 0.0.0.0:13133
  zpages:
    endpoint: 0.0.0.0:55679

service:
  extensions: [health_check, zpages]
  pipelines:
    # ... existing pipelines
```

### View Metrics

```bash
# Port forward to access zpages
kubectl port-forward -n otel-system deployment/otel-collector 55679:55679

# Open in browser: http://localhost:55679/debug/tracez
```

### Scaling

To scale the collector for higher throughput:

```bash
kubectl scale deployment otel-collector -n otel-system --replicas=3
```

## Production Checklist

Before going to production:

- [ ] Managed identity registered with Geneva production account
- [ ] Using production Geneva endpoint (`gcs.prod.monitoring.core.windows.net`)
- [ ] Resource requests/limits tuned for your workload
- [ ] Health checks configured
- [ ] Monitoring and alerting set up
- [ ] ACR image tagged with version (not `:latest`)
- [ ] Backup/DR strategy documented
- [ ] Security review completed
- [ ] Network policies applied (if required)
- [ ] Resource quotas defined for namespace

## Next Steps

- Configure log collection from your applications
- Set up trace collection with instrumentation
- Create Geneva dashboards for your telemetry
- Set up alerts based on log/trace data
- Optimize batch and retry configuration for your workload

## References

- [Azure Workload Identity Documentation](https://azure.github.io/azure-workload-identity/)
- [OpenTelemetry Collector Configuration](https://opentelemetry.io/docs/collector/configuration/)
- [Geneva Documentation](https://eng.ms/docs/products/geneva)
- [AKS OIDC Issuer](https://learn.microsoft.com/en-us/azure/aks/use-oidc-issuer)
