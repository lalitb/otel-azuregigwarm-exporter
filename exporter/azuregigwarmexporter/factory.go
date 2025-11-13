// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build cgo

package azuregigwarmexporter

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

import (
	"os"
)

var (
	Type = component.MustNewType("azuregigwarm")
)

const (
	stability = component.StabilityLevelAlpha
)

var (
	errUnexpectedConfigurationType = errors.New("failed to cast configuration to AzureGigWarm Config")
)

type factory struct{}

// NewFactory creates an exporter factory for Azure Geneva Warm.
func NewFactory() exporter.Factory {
	f := &factory{}
	return exporter.NewFactory(
		Type,
		f.createDefaultConfig,
		exporter.WithLogs(f.createLogsExporter, stability),
		exporter.WithTraces(f.createTracesExporter, stability),
	)
}

// overrideConfigFromEnv overrides config values - Role and RoleInstance from environment variables if set.
func overrideConfigFromEnv(cfg *Config) {
	if role := os.Getenv("GENEVA_ROLE_NAME"); role != "" {
		cfg.Role = role
	}
	if roleInstance := os.Getenv("GENEVA_ROLE_INSTANCE"); roleInstance != "" {
		cfg.RoleInstance = roleInstance
	}
}

// createDefaultConfig creates the default exporter configuration.
func (f *factory) createDefaultConfig() component.Config {
	return &Config{
		QueueConfig:      exporterhelper.NewDefaultQueueConfig(),
		RetryConfig:      configretry.NewDefaultBackOffConfig(),
		BatchRetryConfig: NewDefaultBatchRetryConfig(),
	}
}

// createLogsExporter creates a logs exporter based on the config.
func (f *factory) createLogsExporter(ctx context.Context, set exporter.Settings, c component.Config) (exporter.Logs, error) {
	cfg, ok := c.(*Config)
	if !ok {
		return nil, errUnexpectedConfigurationType
	}
	overrideConfigFromEnv(cfg)

	exp, err := newLogsExporter(ctx, set, cfg)
	if err != nil {
		return nil, err
	}

	// Wrap with exporterhelper to enable queuing and retries
	return exporterhelper.NewLogs(
		ctx,
		set,
		cfg,
		exp.pushLogs,
		exporterhelper.WithRetry(cfg.RetryConfig),
		exporterhelper.WithQueue(cfg.QueueConfig),
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithShutdown(exp.shutdown),
	)
}

// createTracesExporter creates a traces exporter based on the config.
func (f *factory) createTracesExporter(ctx context.Context, set exporter.Settings, c component.Config) (exporter.Traces, error) {
	cfg, ok := c.(*Config)
	if !ok {
		return nil, errUnexpectedConfigurationType
	}

    overrideConfigFromEnv(cfg)

	exp, err := newTracesExporter(ctx, set, cfg)
	if err != nil {
		return nil, err
	}

	// Wrap with exporterhelper to enable queuing and retries
	return exporterhelper.NewTraces(
		ctx,
		set,
		cfg,
		exp.pushTraces,
		exporterhelper.WithRetry(cfg.RetryConfig),
		exporterhelper.WithQueue(cfg.QueueConfig),
		exporterhelper.WithStart(exp.start),
		exporterhelper.WithShutdown(exp.shutdown),
	)
}
