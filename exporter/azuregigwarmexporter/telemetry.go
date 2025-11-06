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
	batchesExported     metric.Int64Counter
	batchesExportErrors metric.Int64Counter
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

	return &telemetry{
		spansExported:       spansExported,
		spansExportErrors:   spansExportErrors,
		batchesExported:     batchesExported,
		batchesExportErrors: batchesExportErrors,
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

// recordBatchExported records a successful batch export
func (t *telemetry) recordBatchExported(ctx context.Context, attributes ...attribute.KeyValue) {
	t.batchesExported.Add(ctx, 1, metric.WithAttributes(attributes...))
}

// recordBatchExportError records a failed batch export
func (t *telemetry) recordBatchExportError(ctx context.Context, attributes ...attribute.KeyValue) {
	t.batchesExportErrors.Add(ctx, 1, metric.WithAttributes(attributes...))
}
