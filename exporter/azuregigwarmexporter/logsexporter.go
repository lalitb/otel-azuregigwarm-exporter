// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build cgo

package azuregigwarmexporter

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	cgogeneva "github.com/open-telemetry/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter/internal/cgo"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.uber.org/zap"
)

// logsExporter implements the logs exporter for Azure Geneva Warm (GigWarm) via Rust FFI.
type logsExporter struct {
	params exporter.Settings
	cfg    *Config
	client *cgogeneva.GenevaClient
	logger *zap.Logger
}

// logsExporter no longer needs to implement consumer.Logs or component.Component
// because exporterhelper handles those interfaces

// newLogsExporter creates a new GigWarm logs exporter.
func newLogsExporter(_ context.Context, set exporter.Settings, cfg *Config) (*logsExporter, error) {
	// Validate early to fail fast
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid azuregigwarm config: %w", err)
	}

	// Build CGO config
	cgoCfg := cgogeneva.GenevaConfig{
		Endpoint:           cfg.Endpoint,
		Environment:        cfg.Environment,
		Account:            cfg.Account,
		Namespace:          cfg.Namespace,
		Region:             cfg.Region,
		ConfigMajorVersion: cfg.ConfigMajorVersion,
		AuthMethod:         int32(cfg.AuthMethod), // 0 = MSI, 1 = Certificate
		Tenant:             cfg.Tenant,
		RoleName:           cfg.RoleName,
		RoleInstance:       cfg.RoleInstance,
	}

	// Add certificate options if needed
	if cfg.AuthMethod == Certificate {
		cgoCfg.CertPath = cfg.CertPath
		cgoCfg.CertPassword = cfg.CertPassword
	}
    // Add workload identity resource if needed
    if cfg.AuthMethod == WorkloadIdentity {
        cgoCfg.WorkloadIdentityResource = cfg.WorkloadIdentityResource
    }

	client, err := cgogeneva.NewGenevaClient(cgoCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Geneva FFI client: %w", err)
	}

	return &logsExporter{
		params: set,
		cfg:    cfg,
		client: client,
		logger: set.Logger,
	}, nil
}

// start is called by the Collector when the exporter is starting.
func (e *logsExporter) start(_ context.Context, _ component.Host) error {
	e.logger.Info("Starting AzureGigWarm exporter",
		zap.String("endpoint", e.cfg.Endpoint),
		zap.String("environment", e.cfg.Environment),
		zap.String("account", e.cfg.Account),
		zap.String("namespace", e.cfg.Namespace),
		zap.String("region", e.cfg.Region),
	)
	return nil
}

// shutdown is called by the Collector when the exporter is shutting down.
func (e *logsExporter) shutdown(_ context.Context) error {
	e.logger.Info("Shutting down AzureGigWarm exporter")
	if e.client != nil {
		e.client.Close()
	}
	return nil
}

// pushLogs implements the push function for exporterhelper and sends logs via Rust FFI.
func (e *logsExporter) pushLogs(ctx context.Context, ld plog.Logs) error {
	// Marshal to OTLP ExportLogsServiceRequest protobuf bytes
	req := plogotlp.NewExportRequestFromLogs(ld)
	data, err := req.MarshalProto()
	if err != nil {
		return fmt.Errorf("failed to marshal logs to protobuf: %w", err)
	}

	// Encode once, then upload each batch synchronously via FFI.
	batches, err := e.client.EncodeAndCompressLogs(data)
	if err != nil {
		e.logger.Error("Failed to encode logs for Geneva Warm", zap.Error(err))
		return fmt.Errorf("failed to encode logs for Geneva Warm: %w", err)
	}
	defer batches.Close()

	n := batches.Len()

	// Upload batches with retry logic
	if err := e.uploadBatchesWithRetry(ctx, batches, n); err != nil {
		return err
	}

	e.logger.Debug("Successfully uploaded logs to Geneva Warm",
		zap.Int("log_records", ld.LogRecordCount()),
		zap.Int("batches", n),
	)
	return nil
}

// uploadBatchesWithRetry uploads batches concurrently and retries failed batches
func (e *logsExporter) uploadBatchesWithRetry(ctx context.Context, batches *cgogeneva.EncodedBatches, n int) error {
	type batchResult struct {
		index int
		err   error
	}

	resultChan := make(chan batchResult, n)
	var wg sync.WaitGroup

	// First attempt: upload all batches concurrently
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			if err := e.uploadBatchWithRetry(ctx, batches, index); err != nil {
				resultChan <- batchResult{index: index, err: err}
			}
		}(i)
	}

	wg.Wait()
	close(resultChan)

	// Collect failed batch indices
	var failedBatches []batchResult
	for result := range resultChan {
		failedBatches = append(failedBatches, result)
	}

	// If any batches failed after retries, return error
	if len(failedBatches) > 0 {
		e.logger.Error("Failed to upload batches after retries",
			zap.Int("failed_count", len(failedBatches)),
			zap.Int("total_batches", n),
		)
		// Return the first error
		return failedBatches[0].err
	}

	return nil
}

// uploadBatchWithRetry uploads a single batch with exponential backoff retry
func (e *logsExporter) uploadBatchWithRetry(ctx context.Context, batches *cgogeneva.EncodedBatches, index int) error {
	if !e.cfg.BatchRetryConfig.Enabled {
		// Batch retry disabled, upload once
		if err := e.client.UploadBatch(batches, index); err != nil {
			e.logger.Error("Failed to upload batch to Geneva Warm",
				zap.Int("batch_index", index),
				zap.Error(err),
			)
			return fmt.Errorf("failed to upload logs batch %d to Geneva Warm: %w", index, err)
		}
		return nil
	}

	// Batch retry enabled
	maxRetries := e.cfg.BatchRetryConfig.MaxRetries
	if maxRetries < 0 {
		maxRetries = 3 // Default
	}

	initialInterval := e.cfg.BatchRetryConfig.GetInitialInterval()
	maxInterval := e.cfg.BatchRetryConfig.GetMaxInterval()
	multiplier := e.cfg.BatchRetryConfig.Multiplier
	if multiplier <= 0 {
		multiplier = 2.0 // Default
	}

	var lastErr error
	backoff := initialInterval

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Attempt upload
		err := e.client.UploadBatch(batches, index)
		if err == nil {
			// Success
			if attempt > 0 {
				e.logger.Info("Batch upload succeeded after retry",
					zap.Int("batch_index", index),
					zap.Int("attempt", attempt+1),
				)
			}
			return nil
		}

		lastErr = err
		e.logger.Warn("Batch upload failed, will retry",
			zap.Int("batch_index", index),
			zap.Int("attempt", attempt+1),
			zap.Int("max_attempts", maxRetries+1),
			zap.Duration("backoff", backoff),
			zap.Error(err),
		)

		// If this was the last attempt, don't sleep
		if attempt == maxRetries {
			break
		}

		// Sleep with backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		// Calculate next backoff with exponential increase
		backoff = time.Duration(float64(backoff) * multiplier)
		if backoff > maxInterval {
			backoff = maxInterval
		}
		// Add jitter (±10%)
		jitter := float64(backoff) * 0.1 * (2*math.Float64frombits(uint64(time.Now().UnixNano())) - 1)
		backoff += time.Duration(jitter)
	}

	e.logger.Error("Failed to upload batch after all retries",
		zap.Int("batch_index", index),
		zap.Int("attempts", maxRetries+1),
		zap.Error(lastErr),
	)
	return fmt.Errorf("failed to upload logs batch %d after %d attempts: %w", index, maxRetries+1, lastErr)
}

// These interface methods are no longer needed because exporterhelper wraps the exporter
// and handles the consumer.Logs and component.Component interfaces
