// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build cgo

package azuregigwarmexporter

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// telemetry holds the metrics for the Azure GigWarm exporter
type telemetry struct {
	spansExported       metric.Int64Counter
	spansExportErrors   metric.Int64Counter
	spansReceived       metric.Int64Counter
	batchesExported     metric.Int64Counter
	batchesExportErrors metric.Int64Counter
	tracesExported      metric.Int64Counter
	tracesExportErrors  metric.Int64Counter
	tracesReceived      metric.Int64Counter
}

// newTelemetry creates a new telemetry instance with Prometheus metrics
func newTelemetry(set component.TelemetrySettings) (*telemetry, error) {
	meter := set.MeterProvider.Meter("azuregigwarmexporter")

	spansExported, err := meter.Int64Counter(
		"otelcol_exporter_sent_spans_total",
		metric.WithDescription("Number of spans successfully exported to Azure GigWarm"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	spansExportErrors, err := meter.Int64Counter(
		"otelcol_exporter_send_failed_spans_total",
		metric.WithDescription("Number of spans that failed to export to Azure GigWarm"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	spansReceived, err := meter.Int64Counter(
		"otelcol_exporter_received_spans_total",
		metric.WithDescription("Number of spans received by the Azure GigWarm exporter"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	batchesExported, err := meter.Int64Counter(
		"otelcol_exporter_sent_batches_total",
		metric.WithDescription("Number of batches successfully exported to Azure GigWarm"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	batchesExportErrors, err := meter.Int64Counter(
		"otelcol_exporter_send_failed_batches_total",
		metric.WithDescription("Number of batches that failed to export to Azure GigWarm"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	tracesExported, err := meter.Int64Counter(
		"otelcol_exporter_sent_traces_total",
		metric.WithDescription("Number of trace requests successfully exported to Azure GigWarm"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	tracesExportErrors, err := meter.Int64Counter(
		"otelcol_exporter_send_failed_traces_total",
		metric.WithDescription("Number of trace requests that failed to export to Azure GigWarm"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	tracesReceived, err := meter.Int64Counter(
		"otelcol_exporter_received_traces_total",
		metric.WithDescription("Number of trace requests received by the Azure GigWarm exporter"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	return &telemetry{
		spansExported:       spansExported,
		spansExportErrors:   spansExportErrors,
		spansReceived:       spansReceived,
		batchesExported:     batchesExported,
		batchesExportErrors: batchesExportErrors,
		tracesExported:      tracesExported,
		tracesExportErrors:  tracesExportErrors,
		tracesReceived:      tracesReceived,
	}, nil
}

// recordSpansExported records the number of spans successfully exported
func (t *telemetry) recordSpansExported(ctx context.Context, count int64, attributes ...attribute.KeyValue) {
	t.spansExported.Add(ctx, count, metric.WithAttributes(attributes...))
}

// recordSpansExportError records the number of spans that failed to export
func (t *telemetry) recordSpansExportError(ctx context.Context, count int64, attributes ...attribute.KeyValue) {
	t.spansExportErrors.Add(ctx, count, metric.WithAttributes(attributes...))
}

// recordSpansReceived records the number of spans received
func (t *telemetry) recordSpansReceived(ctx context.Context, count int64, attributes ...attribute.KeyValue) {
	t.spansReceived.Add(ctx, count, metric.WithAttributes(attributes...))
}

// recordBatchExported records a successful batch export
func (t *telemetry) recordBatchExported(ctx context.Context, attributes ...attribute.KeyValue) {
	t.batchesExported.Add(ctx, 1, metric.WithAttributes(attributes...))
}

// recordBatchExportError records a failed batch export
func (t *telemetry) recordBatchExportError(ctx context.Context, attributes ...attribute.KeyValue) {
	t.batchesExportErrors.Add(ctx, 1, metric.WithAttributes(attributes...))
}

// recordTracesReceived records the number of trace requests received
func (t *telemetry) recordTracesReceived(ctx context.Context, attributes ...attribute.KeyValue) {
	t.tracesReceived.Add(ctx, 1, metric.WithAttributes(attributes...))
}

// recordTracesExported records a successful trace export
func (t *telemetry) recordTracesExported(ctx context.Context, attributes ...attribute.KeyValue) {
	t.tracesExported.Add(ctx, 1, metric.WithAttributes(attributes...))
}

// recordTracesExportError records a failed trace export
func (t *telemetry) recordTracesExportError(ctx context.Context, attributes ...attribute.KeyValue) {
	t.tracesExportErrors.Add(ctx, 1, metric.WithAttributes(attributes...))
}
