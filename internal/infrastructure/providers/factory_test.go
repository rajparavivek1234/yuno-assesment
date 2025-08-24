package providers

import (
	"net/http"
	"testing"
	"time"

	"yuno_assesment/config"
	"yuno_assesment/internal/domain"
)

func TestFactory_CreateProvider(t *testing.T) {
	// Create base configuration
	cfg := &config.Config{
		Providers: map[string]config.PaymentProviderConfig{
			"ProviderA": {
				Name:        "ProviderA",
				Endpoint:    "http://provider-a.test",
				Timeout:     5 * time.Second,
				RetryCount:  3,
				MaxAmount:   10000,
				Description: "Test Provider A",
			},
			"ProviderB": {
				Name:        "ProviderB",
				Endpoint:    "http://provider-b.test",
				Timeout:     5 * time.Second,
				RetryCount:  3,
				MaxAmount:   10000,
				Description: "Test Provider B",
			},
		},
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	tests := []struct {
		name          string
		providerName  string
		modifyConfig  func(*config.Config)
		expectedError bool
		errorCode     string
	}{
		{
			name:          "create provider A successfully",
			providerName:  "ProviderA",
			expectedError: false,
		},
		{
			name:          "create provider B successfully",
			providerName:  "ProviderB",
			expectedError: false,
		},
		{
			name:          "provider not found",
			providerName:  "NonExistentProvider",
			expectedError: true,
			errorCode:     domain.ErrProviderNotFound,
		},
		{
			name:         "invalid provider config - missing endpoint",
			providerName: "ProviderA",
			modifyConfig: func(c *config.Config) {
				providerConfig := c.Providers["ProviderA"]
				providerConfig.Endpoint = ""
				c.Providers["ProviderA"] = providerConfig
			},
			expectedError: true,
			errorCode:     domain.ErrInvalidConfiguration,
		},
		{
			name:         "invalid provider config - invalid max amount",
			providerName: "ProviderA",
			modifyConfig: func(c *config.Config) {
				providerConfig := c.Providers["ProviderA"]
				providerConfig.MaxAmount = -1
				c.Providers["ProviderA"] = providerConfig
			},
			expectedError: true,
			errorCode:     domain.ErrInvalidConfiguration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the config for each test
			testConfig := &config.Config{
				Providers: make(map[string]config.PaymentProviderConfig),
			}
			for k, v := range cfg.Providers {
				testConfig.Providers[k] = v
			}

			// Apply config modifications if any
			if tt.modifyConfig != nil {
				tt.modifyConfig(testConfig)
			}

			// Create factory
			factory := NewFactory(testConfig, client)

			// Create provider
			provider, err := factory.CreateProvider(tt.providerName)

			// Check error expectations
			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}

				// Check if it's a PaymentError with the expected code
				if paymentErr, ok := err.(*domain.PaymentError); ok {
					if paymentErr.Code != tt.errorCode {
						t.Errorf("expected error code %s, got %s", tt.errorCode, paymentErr.Code)
					}
				} else {
					t.Errorf("expected PaymentError, got %T", err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify provider was created correctly
			if provider == nil {
				t.Error("expected non-nil provider")
				return
			}

			// Verify provider name
			if provider.Name() != tt.providerName {
				t.Errorf("expected provider name %s, got %s", tt.providerName, provider.Name())
			}

			// Verify provider was stored in factory
			if len(factory.providers) != 1 {
				t.Errorf("expected 1 provider in factory, got %d", len(factory.providers))
			}

			// Verify provider state was initialized
			state := factory.GetProviderState(tt.providerName)
			if state == nil {
				t.Error("expected non-nil provider state")
				return
			}

			if !state.IsAvailable {
				t.Error("expected provider to be available")
			}

			if state.ConsecutiveErrs != 0 {
				t.Errorf("expected 0 consecutive errors, got %d", state.ConsecutiveErrs)
			}

			if state.LastChecked.IsZero() {
				t.Error("expected non-zero last checked time")
			}
		})
	}
}

func TestFactory_UpdateProviderState(t *testing.T) {
	// Create factory with a provider
	cfg := &config.Config{
		Providers: map[string]config.PaymentProviderConfig{
			"ProviderA": {
				Name:        "ProviderA",
				Endpoint:    "http://provider-a.test",
				Timeout:     5 * time.Second,
				RetryCount:  3,
				MaxAmount:   10000,
				Description: "Test Provider A",
			},
		},
	}

	client := &http.Client{Timeout: 5 * time.Second}
	factory := NewFactory(cfg, client)

	// Create the provider to initialize its state
	_, err := factory.CreateProvider("ProviderA")
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	tests := []struct {
		name             string
		provider         string
		errors           []error
		expectedState    bool
		expectedConsErrs int
	}{
		{
			name:             "handle successful operation",
			provider:         "ProviderA",
			errors:           []error{nil},
			expectedState:    true,
			expectedConsErrs: 0,
		},
		{
			name:     "handle consecutive errors",
			provider: "ProviderA",
			errors: []error{
				&domain.PaymentError{Code: domain.ErrNetworkError},
				&domain.PaymentError{Code: domain.ErrNetworkError},
				&domain.PaymentError{Code: domain.ErrNetworkError},
			},
			expectedState:    false,
			expectedConsErrs: 3,
		},
		{
			name:     "recover after errors",
			provider: "ProviderA",
			errors: []error{
				&domain.PaymentError{Code: domain.ErrNetworkError},
				&domain.PaymentError{Code: domain.ErrNetworkError},
				nil,
			},
			expectedState:    true,
			expectedConsErrs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, err := range tt.errors {
				factory.UpdateProviderState(tt.provider, err)
			}

			state := factory.GetProviderState(tt.provider)
			if state == nil {
				t.Fatal("expected non-nil provider state")
			}

			if state.IsAvailable != tt.expectedState {
				t.Errorf("expected IsAvailable to be %v, got %v", tt.expectedState, state.IsAvailable)
			}

			if state.ConsecutiveErrs != tt.expectedConsErrs {
				t.Errorf("expected ConsecutiveErrs to be %d, got %d", tt.expectedConsErrs, state.ConsecutiveErrs)
			}
		})
	}
}
