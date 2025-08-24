package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"yuno_assesment/config"
	"yuno_assesment/internal/domain"
	"yuno_assesment/internal/domain/repository"
	"yuno_assesment/pkg/httpclient"
)

func TestProviderA_ProcessPayment_Extended(t *testing.T) {
	var (
		provider repository.PaymentProvider
		payment  *domain.Payment
		err      *domain.PaymentError
	)
	tests := []struct {
		name           string
		amount         float64
		currency       string
		mockResponse   interface{}
		mockStatus     int
		delay          time.Duration // Add delay for timeout tests
		expectedError  bool
		errorCode      string // Expected error code
		expectedStatus domain.PaymentStatus
	}{
		{
			name:     "successful payment",
			amount:   100.00,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"transaction_id": "TXN-TEST-100",
				"status":         "APPROVED",
				"amount":         100.00,
				"currency":       "USD",
				"timestamp":      "2024-01-15T10:30:00Z",
			},
			mockStatus:     http.StatusOK,
			expectedError:  false,
			errorCode:      "",
			expectedStatus: domain.StatusApproved,
		},
		{
			name:     "payment with decimal amount",
			amount:   100.50,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"transaction_id": "TXN-TEST-101",
				"status":         "APPROVED",
				"amount":         100.50,
				"currency":       "USD",
				"timestamp":      "2024-01-15T10:30:00Z",
			},
			mockStatus:     http.StatusOK,
			expectedError:  false,
			errorCode:      "",
			expectedStatus: domain.StatusApproved,
		},
		{
			name:     "payment with different currency",
			amount:   100.00,
			currency: "EUR",
			mockResponse: map[string]interface{}{
				"transaction_id": "TXN-TEST-102",
				"status":         "APPROVED",
				"amount":         100.00,
				"currency":       "EUR",
				"timestamp":      "2024-01-15T10:30:00Z",
			},
			mockStatus:     http.StatusOK,
			expectedError:  false,
			errorCode:      "",
			expectedStatus: domain.StatusApproved,
		},
		{
			name:     "declined payment",
			amount:   999.00,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"transaction_id": "TXN-TEST-999",
				"status":         "DECLINED",
				"amount":         999.00,
				"currency":       "USD",
				"timestamp":      "2024-01-15T10:30:00Z",
			},
			mockStatus:     http.StatusOK,
			expectedError:  true,
			errorCode:      domain.ErrCardDeclined,
			expectedStatus: domain.StatusDeclined,
		},
		{
			name:     "payment timeout",
			amount:   100.00,
			currency: "USD",
			delay:    6 * time.Second, // Greater than client timeout
			mockResponse: map[string]interface{}{
				"transaction_id": "TXN-TEST-103",
				"status":         "APPROVED",
				"amount":         100.00,
				"currency":       "USD",
				"timestamp":      "2024-01-15T10:30:00Z",
			},
			mockStatus:    http.StatusOK,
			expectedError: true,
			errorCode:     domain.ErrNetworkError,
		},
		{
			name:     "malformed timestamp",
			amount:   100.00,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"transaction_id": "TXN-TEST-104",
				"status":         "APPROVED",
				"amount":         100.00,
				"currency":       "USD",
				"timestamp":      "invalid-time",
			},
			mockStatus:    http.StatusOK,
			expectedError: true,
			errorCode:     domain.ErrProviderInvalidResp,
		},
		{
			name:     "missing required fields",
			amount:   100.00,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"transaction_id": "TXN-TEST-105",
				// Missing status field
				"amount":    100.00,
				"currency":  "USD",
				"timestamp": "2024-01-15T10:30:00Z",
			},
			mockStatus:    http.StatusOK,
			expectedError: true,
			errorCode:     domain.ErrProviderInvalidResp,
		},
		{
			name:     "type mismatch in response",
			amount:   100.00,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"transaction_id": "TXN-TEST-106",
				"status":         "APPROVED",
				"amount":         "not-a-number", // String instead of number
				"currency":       "USD",
				"timestamp":      "2024-01-15T10:30:00Z",
			},
			mockStatus:    http.StatusOK,
			expectedError: true,
			errorCode:     domain.ErrProviderInvalidResp,
		},
		{
			name:          "invalid amount",
			amount:        -100.00,
			currency:      "USD",
			mockStatus:    http.StatusBadRequest,
			expectedError: true,
			errorCode:     "INVALID_AMOUNT",
		},
		{
			name:          "amount exceeds maximum",
			amount:        20000.00, // Exceeds config.MaxAmount
			currency:      "USD",
			mockStatus:    http.StatusBadRequest,
			expectedError: true,
			errorCode:     domain.ErrInvalidAmount,
		},
		{
			name:          "unsupported currency",
			amount:        100.00,
			currency:      "XXX", // Invalid currency
			mockStatus:    http.StatusBadRequest,
			expectedError: true,
			errorCode:     domain.ErrInvalidCurrency,
		},
		{
			name:          "provider error",
			amount:        100.00,
			currency:      "USD",
			mockStatus:    http.StatusInternalServerError,
			expectedError: true,
			errorCode:     domain.ErrInternalError,
		},
		{
			name:          "rate limit exceeded",
			amount:        100.00,
			currency:      "USD",
			mockStatus:    http.StatusTooManyRequests,
			expectedError: true,
			errorCode:     domain.ErrRateLimitExceeded,
		},
		{
			name:     "invalid response format",
			amount:   100.00,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"invalid": "response",
			},
			mockStatus:    http.StatusOK,
			expectedError: true,
			errorCode:     domain.ErrProviderInvalidResp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP client
			client := httpclient.NewMockClient(func(req *http.Request) (*http.Response, error) {
				// Verify request method
				if req.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", req.Method)
				}

				// Verify content type
				if req.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
				}

				// For timeout test case
				if tt.delay > 0 {
					return nil, &httpclient.TimeoutError{}
				}

				// For successful response
				var respBody []byte
				if tt.mockResponse != nil {
					respBody, _ = json.Marshal(tt.mockResponse)
				}

				return httpclient.NewMockResponse(tt.mockStatus, respBody), nil
			})

			// Create provider configuration
			cfg := config.PaymentProviderConfig{
				Name:        "ProviderA",
				Endpoint:    "http://test-provider-a.com",
				Timeout:     5 * time.Second,
				RetryCount:  3,
				MaxAmount:   10000,
				Description: "Test Provider A",
			}

			provider = NewProviderA(cfg, client)

			// Process payment
			payment, err = provider.ProcessPayment(context.Background(), tt.amount, tt.currency)

			// Check error expectations
			if tt.expectedError {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errorCode != "" && err.Code != tt.errorCode {
					t.Errorf("expected error code %v, got %v", tt.errorCode, err.Code)
				}
				return
			}

			// Handle successful cases
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if payment.Amount != tt.amount {
				t.Errorf("expected amount %v, got %v", tt.amount, payment.Amount)
			}
			if payment.Currency != domain.Currency(tt.currency) {
				t.Errorf("expected currency %v, got %v", tt.currency, payment.Currency)
			}
			if payment.Status != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, payment.Status)
			}
			if payment.Provider != cfg.Name {
				t.Errorf("expected provider %v, got %v", cfg.Name, payment.Provider)
			}
			if payment.ID == "" {
				t.Error("expected non-empty payment ID")
			}
			if payment.Timestamp.IsZero() {
				t.Error("expected non-zero timestamp")
			}
		})
	}
}
