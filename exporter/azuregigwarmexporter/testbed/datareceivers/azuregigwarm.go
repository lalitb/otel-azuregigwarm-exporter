// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package datareceivers // import "go.opentelemetry.io/collector/testbed/datareceivers"

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/testbed"
)

// AzureGigWarmDataReceiver implements a mock Azure Geneva receiver for load testing.
// This is a simplified HTTP endpoint that mimics the Geneva backend to enable
// load testing of the azuregigwarmexporter without requiring actual Azure credentials
// or connectivity.
type AzureGigWarmDataReceiver struct {
	testbed.DataReceiverBase
	server         *http.Server
	tracesReceived atomic.Uint64
	logsReceived   atomic.Uint64
	bytesReceived  atomic.Uint64
}

// NewAzureGigWarmDataReceiver creates a new mock Azure GigWarm receiver
func NewAzureGigWarmDataReceiver(port int) testbed.DataReceiver {
	return &AzureGigWarmDataReceiver{
		DataReceiverBase: testbed.DataReceiverBase{Port: port},
	}
}

// Start starts the mock receiver HTTP server
func (r *AzureGigWarmDataReceiver) Start(tc consumer.Traces, _ consumer.Metrics, lc consumer.Logs) error {
	mux := http.NewServeMux()

	// Mock endpoint for receiving telemetry from GigWarm exporter
	// In reality, Geneva uses a more complex protocol, but for load testing
	// we just need to accept the data and track metrics
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Read and count bytes
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		r.bytesReceived.Add(uint64(len(bodyBytes)))

		// Try to determine if it's traces or logs based on content-type or path
		contentType := req.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// For simplicity, we'll just count requests as either traces or logs
		// In a real implementation, you would parse the Geneva protocol
		if req.URL.Path == "/traces" || req.Header.Get("X-Telemetry-Type") == "traces" {
			r.tracesReceived.Add(1)
		} else {
			r.logsReceived.Add(1)
		}

		// Return success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", r.Port),
		Handler: mux,
	}

	go func() {
		if err := r.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Mock receiver server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the mock receiver
func (r *AzureGigWarmDataReceiver) Stop() error {
	if r.server != nil {
		return r.server.Shutdown(context.Background())
	}
	return nil
}

// GenConfigYAMLStr returns collector config for the azuregigwarm exporter
func (r *AzureGigWarmDataReceiver) GenConfigYAMLStr() string {
	// Note: This creates a config that points to our mock endpoint
	// For real Geneva testing, you would use actual credentials and endpoints
	return fmt.Sprintf(`
  azuregigwarm:
    endpoint: http://127.0.0.1:%d
    environment: loadtest
    account: testbed
    namespace: perftest
    region: local
    config_major_version: 1
    auth_method: 0
    tenant: test-tenant
    role_name: testbed-role
    role_instance: instance-01
`, r.Port)
}

// ProtocolName returns the protocol name
func (r *AzureGigWarmDataReceiver) ProtocolName() string {
	return "azuregigwarm"
}

// ReceivedTraces returns the received traces count
func (r *AzureGigWarmDataReceiver) ReceivedTraces() uint64 {
	return r.tracesReceived.Load()
}

// ReceivedLogs returns the received logs count
func (r *AzureGigWarmDataReceiver) ReceivedLogs() uint64 {
	return r.logsReceived.Load()
}

// ReceivedBytes returns the total bytes received
func (r *AzureGigWarmDataReceiver) ReceivedBytes() uint64 {
	return r.bytesReceived.Load()
}

// Ensure interface implementation
var _ testbed.DataReceiver = (*AzureGigWarmDataReceiver)(nil)
