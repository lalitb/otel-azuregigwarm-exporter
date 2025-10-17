// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !cgo

package azuregigwarmexporter

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
)

var Type = component.MustNewType("azuregigwarm")

const stability = component.StabilityLevelAlpha

// NewFactory returns a factory that reports a clear error when built without cgo.
func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		Type,
		func() component.Config { return &Config{} },
		exporter.WithLogs(
			func(context.Context, exporter.Settings, component.Config) (exporter.Logs, error) {
				return nil, errors.New("azuregigwarm exporter requires CGO (build with CGO_ENABLED=1 and make AZUREGIGWARM=1)")
			},
			stability,
		),
	)
}
