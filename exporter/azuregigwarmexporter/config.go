// Copyright OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azuregigwarmexporter // import "github.com/open-telemetry/otel-azuregigwarm-exporter/exporter/azuregigwarmexporter"

import (
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// AuthMethod represents the authentication method for Geneva Warm
type AuthMethod int

const (
	// MSI uses Managed Service Identity
	MSI AuthMethod = iota
	// Certificate uses certificate-based authentication
	Certificate
	// WorkloadIdentity uses Workload Identity authentication
	WorkloadIdentity
)

// String returns the string representation of AuthMethod
func (a AuthMethod) String() string {
	switch a {
	case MSI:
		return "msi"
	case Certificate:
		return "certificate"
	case WorkloadIdentity:
		return "workload_identity"
	default:
		return "unknown"
	}
}

// Config defines configuration for the Azure Geneva Warm exporter.
//
// This exporter sends OTLP Log data to Azure Geneva (Warm path) using a Rust FFI uploader.
type Config struct {
	// Geneva-specific configuration (required)
	Endpoint           string     `mapstructure:"endpoint"`
	Environment        string     `mapstructure:"environment"`
	Account            string     `mapstructure:"account"`
	Namespace          string     `mapstructure:"namespace"`
	Region             string     `mapstructure:"region"`
	ConfigMajorVersion uint32     `mapstructure:"config_major_version"`
	AuthMethod         AuthMethod `mapstructure:"auth_method"`
	Tenant             string     `mapstructure:"tenant"`
	RoleName           string     `mapstructure:"role_name"`
	RoleInstance       string     `mapstructure:"role_instance"`

	// Certificate auth parameters (optional; required only when AuthMethod == Certificate)
	CertPath     string `mapstructure:"cert_path"`
	CertPassword string `mapstructure:"cert_password"`

	// Workload Identity auth parameters (optional; required only when AuthMethod == WorkloadIdentity)
	WorkloadIdentityResource string `mapstructure:"workload_identity_resource"`

	// QueueConfig configures the sending queue for the exporter
	QueueConfig exporterhelper.QueueBatchConfig `mapstructure:"sending_queue"`

	// RetryConfig configures retry behavior for failed exports
	RetryConfig configretry.BackOffConfig `mapstructure:"retry_on_failure"`

	// BatchRetryConfig configures retry behavior for individual batch uploads
	BatchRetryConfig BatchRetryConfig `mapstructure:"batch_retry"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// BatchRetryConfig configures retry behavior for individual batch uploads within a single export request.
// This provides fine-grained retry for failed batches without re-encoding and re-uploading successful batches.
type BatchRetryConfig struct {
	// Enabled indicates whether batch-level retry is enabled (default: true)
	Enabled bool `mapstructure:"enabled"`
	// MaxRetries is the maximum number of retry attempts per batch (default: 3)
	MaxRetries int `mapstructure:"max_retries"`
	// InitialInterval is the initial backoff interval (default: 100ms)
	InitialInterval string `mapstructure:"initial_interval"`
	// MaxInterval is the maximum backoff interval (default: 5s)
	MaxInterval string `mapstructure:"max_interval"`
	// Multiplier is the backoff multiplier (default: 2.0)
	Multiplier float64 `mapstructure:"multiplier"`
}

// NewDefaultBatchRetryConfig creates a BatchRetryConfig with default values
func NewDefaultBatchRetryConfig() BatchRetryConfig {
	return BatchRetryConfig{
		Enabled:         true,
		MaxRetries:      3,
		InitialInterval: "100ms",
		MaxInterval:     "5s",
		Multiplier:      2.0,
	}
}

// GetInitialInterval parses and returns the initial interval duration
func (c *BatchRetryConfig) GetInitialInterval() time.Duration {
	if c.InitialInterval == "" {
		return 100 * time.Millisecond
	}
	d, err := time.ParseDuration(c.InitialInterval)
	if err != nil {
		return 100 * time.Millisecond
	}
	return d
}

// GetMaxInterval parses and returns the max interval duration
func (c *BatchRetryConfig) GetMaxInterval() time.Duration {
	if c.MaxInterval == "" {
		return 5 * time.Second
	}
	d, err := time.ParseDuration(c.MaxInterval)
	if err != nil {
		return 5 * time.Second
	}
	return d
}

var _ component.Config = (*Config)(nil)

// Validate checks if the exporter configuration is valid
func (cfg *Config) Validate() error {
	if cfg.Endpoint == "" {
		return errors.New(`requires a non-empty "endpoint"`)
	}
	if cfg.Environment == "" {
		return errors.New(`requires a non-empty "environment"`)
	}
	if cfg.Account == "" {
		return errors.New(`requires a non-empty "account"`)
	}
	if cfg.Namespace == "" {
		return errors.New(`requires a non-empty "namespace"`)
	}
	if cfg.Region == "" {
		return errors.New(`requires a non-empty "region"`)
	}
	if cfg.Tenant == "" {
		return errors.New(`requires a non-empty "tenant"`)
	}
	if cfg.RoleName == "" {
		return errors.New(`requires a non-empty "role_name"`)
	}
	if cfg.RoleInstance == "" {
		return errors.New(`requires a non-empty "role_instance"`)
	}
	if cfg.AuthMethod != MSI && cfg.AuthMethod != Certificate && cfg.AuthMethod != WorkloadIdentity {
		return fmt.Errorf(`invalid auth_method: %d (must be 0 for MSI, 1 for Certificate, or 2 for WorkloadIdentity)`, cfg.AuthMethod)
	}
	if cfg.AuthMethod == Certificate {
		if cfg.CertPath == "" {
			return errors.New(`requires a non-empty "cert_path" when auth_method == certificate`)
		}
		// cert_password can be empty if the cert is not password protected, so no hard check here.
	}
	if cfg.AuthMethod == WorkloadIdentity {
		if cfg.WorkloadIdentityResource == "" {
			return errors.New(`requires a non-empty "workload_identity_resource" when auth_method == workload_identity`)
		}
	}
	return nil
}
