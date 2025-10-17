module github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuregigwarmexporter/testbed

go 1.24

require (
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/common v0.137.0
	go.opentelemetry.io/collector/component v1.41.0
	go.opentelemetry.io/collector/consumer v1.41.0
	go.opentelemetry.io/collector/pdata v1.41.0
)

// Use local contrib testbed if needed
replace github.com/open-telemetry/opentelemetry-collector-contrib/internal/common => ../../../internal/common
