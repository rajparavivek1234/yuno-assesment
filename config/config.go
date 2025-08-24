package config

import (
	"fmt"
	"os"
	"time"
)

// PaymentProviderConfig represents the configuration for a payment provider
type PaymentProviderConfig struct {
	Name        string        `json:"name"`
	Endpoint    string        `json:"endpoint"`
	Timeout     time.Duration `json:"timeout"`
	RetryCount  int           `json:"retry_count"`
	MaxAmount   float64       `json:"max_amount"`
	Description string        `json:"description"`
	RetryPolicy RetryPolicy   `json:"retry_policy"`
	RateLimit   RateLimit     `json:"rate_limit"`
}

// RetryPolicy defines retry behavior configuration
type RetryPolicy struct {
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	MaxAttempts     int           `json:"max_attempts"`
	RetryableErrors []string      `json:"retryable_errors"`
	RetryableCodes  []int         `json:"retryable_codes"`
}

// RateLimit defines rate limiting configuration
type RateLimit struct {
	RequestsPerSecond int `json:"requests_per_second"`
	BurstSize         int `json:"burst_size"`
}

// GlobalConfig defines global application settings
type GlobalConfig struct {
	DefaultCurrency     string               `json:"default_currency"`
	SupportedCurrencies []string             `json:"supported_currencies"`
	DefaultTimeout      time.Duration        `json:"default_timeout"`
	MaxRequestSize      string               `json:"max_request_size"`
	Metrics             MetricsConfig        `json:"metrics"`
	Logging             LoggingConfig        `json:"logging"`
	CircuitBreaker      CircuitBreakerConfig `json:"circuit_breaker"`
}

// MetricsConfig defines metrics collection settings
type MetricsConfig struct {
	Enabled           bool             `json:"enabled"`
	ReportingInterval time.Duration    `json:"reporting_interval"`
	Exporters         []ExporterConfig `json:"exporters"`
}

// ExporterConfig defines a metrics exporter
type ExporterConfig struct {
	Type    string `json:"type"`
	Port    int    `json:"port,omitempty"`
	Address string `json:"address,omitempty"`
}

// LoggingConfig defines logging settings
type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}

// CircuitBreakerConfig defines circuit breaker settings
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold"`
	ResetTimeout     time.Duration `json:"reset_timeout"`
	HalfOpenRequests int           `json:"half_open_requests"`
}

// Config holds all configuration for the application
type Config struct {
	Providers  map[string]PaymentProviderConfig `json:"providers"`
	Endpoints  ServiceEndpoints                 `json:"endpoints"`
	Global     GlobalConfig                     `json:"global"`
	Monitoring MonitoringConfig                 `json:"monitoring"`
}

// MonitoringConfig holds monitoring-related configuration
type MonitoringConfig struct {
	Metrics     MetricsConfig     `json:"metrics"`
	Tracing     TracingConfig     `json:"tracing"`
	HealthCheck HealthCheckConfig `json:"health_check"`
}

// TracingConfig defines tracing settings
type TracingConfig struct {
	Enabled  bool    `json:"enabled"`
	Sampler  float64 `json:"sampler"`
	Exporter string  `json:"exporter"`
}

// HealthCheckConfig defines health check settings
type HealthCheckConfig struct {
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port"`
	Path    string `json:"path"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	endpoints := DefaultServiceEndpoints()

	defaultRetryPolicy := RetryPolicy{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     2 * time.Second,
		MaxAttempts:  3,
		RetryableErrors: []string{
			"NETWORK_ERROR",
			"TIMEOUT_ERROR",
			"RATE_LIMIT_EXCEEDED",
		},
		RetryableCodes: []int{
			408, // Request Timeout
			429, // Too Many Requests
			500, // Internal Server Error
			502, // Bad Gateway
			503, // Service Unavailable
			504, // Gateway Timeout
		},
	}

	defaultRateLimit := RateLimit{
		RequestsPerSecond: 100,
		BurstSize:         10,
	}

	return &Config{
		Endpoints: endpoints,
		Providers: map[string]PaymentProviderConfig{
			"ProviderA": {
				Name:        "ProviderA",
				Endpoint:    endpoints.ProviderA,
				Timeout:     30 * time.Second,
				RetryCount:  3,
				MaxAmount:   10000.0,
				Description: "Payment Provider A",
				RetryPolicy: defaultRetryPolicy,
				RateLimit:   defaultRateLimit,
			},
			"ProviderB": {
				Name:        "ProviderB",
				Endpoint:    endpoints.ProviderB,
				Timeout:     30 * time.Second,
				RetryCount:  3,
				MaxAmount:   10000.0,
				Description: "Payment Provider B",
				RetryPolicy: defaultRetryPolicy,
				RateLimit: RateLimit{
					RequestsPerSecond: 50,
					BurstSize:         5,
				},
			},
		},
		Global: GlobalConfig{
			DefaultCurrency: "USD",
			SupportedCurrencies: []string{
				"USD",
				"EUR",
				"GBP",
			},
			DefaultTimeout: 30 * time.Second,
			MaxRequestSize: "1MB",
			Metrics: MetricsConfig{
				Enabled:           true,
				ReportingInterval: time.Minute,
			},
			Logging: LoggingConfig{
				Level:  "info",
				Format: "json",
			},
			CircuitBreaker: CircuitBreakerConfig{
				FailureThreshold: 5,
				ResetTimeout:     time.Minute,
				HalfOpenRequests: 3,
			},
		},
		Monitoring: MonitoringConfig{
			Metrics: MetricsConfig{
				Enabled:           true,
				ReportingInterval: time.Minute,
				Exporters: []ExporterConfig{
					{
						Type: "prometheus",
						Port: 9090,
					},
					{
						Type:    "statsd",
						Address: "localhost:8125",
					},
				},
			},
			Tracing: TracingConfig{
				Enabled:  true,
				Sampler:  0.1,
				Exporter: "jaeger",
			},
			HealthCheck: HealthCheckConfig{
				Enabled: true,
				Port:    8080,
				Path:    "/health",
			},
		},
	}
}

// LoadEnvironment loads configuration from environment variables
func (c *Config) LoadEnvironment() {
	if endpoint := os.Getenv("PROVIDER_A_ENDPOINT"); endpoint != "" {
		if provider, ok := c.Providers["ProviderA"]; ok {
			provider.Endpoint = endpoint
			c.Providers["ProviderA"] = provider
		}
	}

	if endpoint := os.Getenv("PROVIDER_B_ENDPOINT"); endpoint != "" {
		if provider, ok := c.Providers["ProviderB"]; ok {
			provider.Endpoint = endpoint
			c.Providers["ProviderB"] = provider
		}
	}

	if timeout := os.Getenv("DEFAULT_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			c.Global.DefaultTimeout = duration
		}
	}

	if currency := os.Getenv("DEFAULT_CURRENCY"); currency != "" {
		c.Global.DefaultCurrency = currency
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.Global.Logging.Level = level
	}

	if metricsEnabled := os.Getenv("METRICS_ENABLED"); metricsEnabled != "" {
		c.Global.Metrics.Enabled = metricsEnabled == "true"
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if !contains(c.Global.SupportedCurrencies, c.Global.DefaultCurrency) {
		return fmt.Errorf("default currency %s is not in supported currencies", c.Global.DefaultCurrency)
	}

	return nil
}

// helper function to check if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
