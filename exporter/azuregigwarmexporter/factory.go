// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build cgo

package azuregigwarmexporter

import (
	"context"
	"errors"
	"os"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.uber.org/zap"
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
func overrideConfigFromEnv(cfg *Config, logger *zap.Logger) {
	originalRoleName := cfg.RoleName
	originalRoleInstance := cfg.RoleInstance

	if role := os.Getenv("GENEVA_ROLE_NAME"); role != "" {
		cfg.RoleName = role
		if originalRoleName != role {
			logger.Info("Config override from environment variable",
				zap.String("field", "role_name"),
				zap.String("original_value", originalRoleName),
				zap.String("override_value", role),
				zap.String("env_var", "GENEVA_ROLE_NAME"),
			)
		}
	}

	if roleInstance := os.Getenv("GENEVA_ROLE_INSTANCE"); roleInstance != "" {
		cfg.RoleInstance = roleInstance
		if originalRoleInstance != roleInstance {
			logger.Info("Config override from environment variable",
				zap.String("field", "role_instance"),
				zap.String("original_value", originalRoleInstance),
				zap.String("override_value", roleInstance),
				zap.String("env_var", "GENEVA_ROLE_INSTANCE"),
			)
		}
	}

	// Log final values for debugging
	logger.Debug("Final role configuration after environment variable processing",
		zap.String("role_name", cfg.RoleName),
		zap.String("role_instance", cfg.RoleInstance),
		zap.Bool("role_name_from_env", os.Getenv("GENEVA_ROLE_NAME") != ""),
		zap.Bool("role_instance_from_env", os.Getenv("GENEVA_ROLE_INSTANCE") != ""),
	)
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

	// Override config from environment variables with logging
	overrideConfigFromEnv(cfg, set.Logger)

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

	// Override config from environment variables with logging
	overrideConfigFromEnv(cfg, set.Logger)

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
