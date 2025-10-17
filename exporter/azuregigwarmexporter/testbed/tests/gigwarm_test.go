// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package tests contains test cases for Azure GigWarm exporter load testing
package tests

import (
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/common/testutil"
	"github.com/open-telemetry/opentelemetry-collector-contrib/testbed/testbed"

	"go.opentelemetry.io/collector/testbed/datareceivers"
)

var performanceResultsSummary testbed.TestResultsSummary = &testbed.PerformanceResults{}

// TestMain is used to initiate setup, execution and tear down of testbed.
func TestMain(m *testing.M) {
	testbed.DoTestMain(m, performanceResultsSummary)
}

// TestGigWarmTrace10kSPS tests the GigWarm exporter with 10k spans per second
func TestGigWarmTrace10kSPS(t *testing.T) {
	tests := []struct {
		name         string
		sender       testbed.DataSender
		receiver     testbed.DataReceiver
		resourceSpec testbed.ResourceSpec
	}{
		{
			"OTLP-to-GigWarm",
			testbed.NewOTLPTraceDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t)),
			datareceivers.NewAzureGigWarmDataReceiver(testutil.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 50,
				ExpectedMaxRAM: 150,
			},
		},
		{
			"OTLP-HTTP-to-GigWarm",
			testbed.NewOTLPHTTPTraceDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t), ""),
			datareceivers.NewAzureGigWarmDataReceiver(testutil.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 50,
				ExpectedMaxRAM: 150,
			},
		},
	}

	processors := []testbed.ProcessorNameAndConfigBody{
		{
			Name: "batch",
			Body: `
  batch:
    send_batch_size: 1024
    timeout: 1s
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testbed.Scenario10kItemsPerSecond(
				t,
				test.sender,
				test.receiver,
				test.resourceSpec,
				performanceResultsSummary,
				processors,
				nil,
				nil,
			)
		})
	}
}

// TestGigWarmLog10kSPS tests the GigWarm exporter with 10k log records per second
func TestGigWarmLog10kSPS(t *testing.T) {
	tests := []struct {
		name         string
		sender       testbed.DataSender
		receiver     testbed.DataReceiver
		resourceSpec testbed.ResourceSpec
	}{
		{
			"OTLP-Logs-to-GigWarm",
			testbed.NewOTLPLogsDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t)),
			datareceivers.NewAzureGigWarmDataReceiver(testutil.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 50,
				ExpectedMaxRAM: 150,
			},
		},
		{
			"OTLP-HTTP-Logs-to-GigWarm",
			testbed.NewOTLPHTTPLogsDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t), ""),
			datareceivers.NewAzureGigWarmDataReceiver(testutil.GetAvailablePort(t)),
			testbed.ResourceSpec{
				ExpectedMaxCPU: 50,
				ExpectedMaxRAM: 150,
			},
		},
	}

	processors := []testbed.ProcessorNameAndConfigBody{
		{
			Name: "batch",
			Body: `
  batch:
    send_batch_size: 1024
    timeout: 1s
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testbed.Scenario10kItemsPerSecond(
				t,
				test.sender,
				test.receiver,
				test.resourceSpec,
				performanceResultsSummary,
				processors,
				nil,
				nil,
			)
		})
	}
}

// TestGigWarmTrace1kSPSWithAttributes tests GigWarm with attributes (more realistic scenario)
func TestGigWarmTrace1kSPSWithAttributes(t *testing.T) {
	sender := testbed.NewOTLPTraceDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t))
	receiver := datareceivers.NewAzureGigWarmDataReceiver(testutil.GetAvailablePort(t))

	processors := []testbed.ProcessorNameAndConfigBody{
		{
			Name: "batch",
			Body: `
  batch:
    send_batch_size: 512
    timeout: 2s
`,
		},
	}

	// Test with realistic attributes
	loadOptions := testbed.LoadOptions{
		DataItemsPerSecond: 1000,
		ItemsPerBatch:      10,
		Parallel:           1,
		Attributes: map[string]string{
			"service.name":        "gigwarm-loadtest",
			"service.version":     "1.0.0",
			"deployment.environment": "loadtest",
			"host.name":           "testbed-host",
		},
	}

	testbed.Scenario1kSPSWithAttrs(
		t,
		sender,
		receiver,
		testbed.ResourceSpec{
			ExpectedMaxCPU: 40,
			ExpectedMaxRAM: 120,
		},
		performanceResultsSummary,
		processors,
		nil,
		loadOptions,
	)
}

// TestGigWarmHighThroughput tests maximum throughput capabilities
func TestGigWarmHighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high throughput test in short mode")
	}

	sender := testbed.NewOTLPTraceDataSender(testbed.DefaultHost, testutil.GetAvailablePort(t))
	receiver := datareceivers.NewAzureGigWarmDataReceiver(testutil.GetAvailablePort(t))

	processors := []testbed.ProcessorNameAndConfigBody{
		{
			Name: "batch",
			Body: `
  batch:
    send_batch_size: 2048
    timeout: 500ms
`,
		},
	}

	loadOptions := testbed.LoadOptions{
		DataItemsPerSecond: 50000,
		ItemsPerBatch:      100,
		Parallel:           4,
	}

	testbed.ScenarioTestWithLoadOptions(
		t,
		sender,
		receiver,
		testbed.ResourceSpec{
			ExpectedMaxCPU: 200,
			ExpectedMaxRAM: 400,
		},
		performanceResultsSummary,
		processors,
		nil,
		loadOptions,
	)
}
