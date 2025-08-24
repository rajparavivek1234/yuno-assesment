package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"yuno_assesment/config"
	"yuno_assesment/internal/domain"
	"yuno_assesment/pkg/httpclient"
)

func TestProviderB_ProcessPayment_Extended(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		currency       string
		mockResponse   interface{}
		mockStatus     int
		delay          time.Duration
		expectedError  bool
		errorCode      string
		expectedStatus domain.PaymentStatus
	}{
		{
			name:     "successful payment",
			amount:   100.00,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"paymentId": "PAY-TEST-100",
				"state":     "SUCCESS",
				"value": map[string]interface{}{
					"amount":       "100.00",
					"currencyCode": "USD",
				},
				"processedAt": 1705318200000,
			},
			mockStatus:     http.StatusOK,
			expectedError:  false,
			expectedStatus: domain.StatusApproved,
		},
		{
			name:     "payment with decimal amount",
			amount:   100.50,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"paymentId": "PAY-TEST-101",
				"state":     "SUCCESS",
				"value": map[string]interface{}{
					"amount":       "100.50",
					"currencyCode": "USD",
				},
				"processedAt": time.Now().UnixNano() / int64(time.Millisecond),
			},
			mockStatus:     http.StatusOK,
			expectedError:  false,
			expectedStatus: domain.StatusApproved,
		},
		{
			name:     "payment with different currency",
			amount:   100.00,
			currency: "EUR",
			mockResponse: map[string]interface{}{
				"paymentId": "PAY-TEST-102",
				"state":     "SUCCESS",
				"value": map[string]interface{}{
					"amount":       "100.00",
					"currencyCode": "EUR",
				},
				"processedAt": time.Now().UnixNano() / int64(time.Millisecond),
			},
			mockStatus:     http.StatusOK,
			expectedError:  false,
			expectedStatus: domain.StatusApproved,
		},
		{
			name:     "failed payment",
			amount:   999.00,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"paymentId": "PAY-TEST-999",
				"state":     "FAILED",
				"value": map[string]interface{}{
					"amount":       "999.00",
					"currencyCode": "USD",
				},
				"processedAt": time.Now().UnixNano() / int64(time.Millisecond),
			},
			mockStatus:     http.StatusOK,
			expectedError:  true,
			errorCode:      domain.ErrCardDeclined,
			expectedStatus: domain.StatusDeclined,
		},
		{
			name:          "provider timeout",
			amount:        100.00,
			currency:      "USD",
			delay:         6 * time.Second, // Greater than client timeout
			mockStatus:    http.StatusOK,
			expectedError: true,
			errorCode:     domain.ErrNetworkError,
		},
		{
			name:     "invalid amount in response",
			amount:   100.00,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"paymentId": "PAY-TEST-103",
				"state":     "SUCCESS",
				"value": map[string]interface{}{
					"amount":       "invalid",
					"currencyCode": "USD",
				},
				"processedAt": time.Now().UnixNano() / int64(time.Millisecond),
			},
			mockStatus:    http.StatusOK,
			expectedError: true,
			errorCode:     domain.ErrProviderInvalidResp,
		},
		{
			name:     "missing currency",
			amount:   100.00,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"paymentId": "PAY-TEST-104",
				"state":     "SUCCESS",
				"value": map[string]interface{}{
					"amount": "100.00",
				},
				"processedAt": time.Now().UnixNano() / int64(time.Millisecond),
			},
			mockStatus:    http.StatusOK,
			expectedError: true,
			errorCode:     domain.ErrProviderInvalidResp,
		},
		{
			name:     "invalid state",
			amount:   100.00,
			currency: "USD",
			mockResponse: map[string]interface{}{
				"paymentId": "PAY-TEST-105",
				"state":     "INVALID",
				"value": map[string]interface{}{
					"amount":       "100.00",
					"currencyCode": "USD",
				},
				"processedAt": time.Now().UnixNano() / int64(time.Millisecond),
			},
			mockStatus:    http.StatusOK,
			expectedError: true,
			errorCode:     domain.ErrProviderInvalidResp,
		},
		{
			name:          "amount exceeds maximum",
			amount:        20000.00,
			currency:      "USD",
			mockStatus:    http.StatusBadRequest,
			expectedError: true,
			errorCode:     domain.ErrInvalidAmount,
		},
		{
			name:          "malformed response",
			amount:        100.00,
			currency:      "USD",
			mockResponse:  "invalid json",
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
					if str, ok := tt.mockResponse.(string); ok {
						respBody = []byte(str)
					} else {
						respBody, _ = json.Marshal(tt.mockResponse)
					}
				}

				return httpclient.NewMockResponse(tt.mockStatus, respBody), nil
			})

			// Create provider configuration
			cfg := config.PaymentProviderConfig{
				Name:        "ProviderB",
				Endpoint:    "http://test-provider-b.com",
				Timeout:     5 * time.Second,
				RetryCount:  3,
				MaxAmount:   10000,
				Description: "Test Provider B",
			}

			// Create provider
			provider := NewProviderB(cfg, client)

			// Process payment
			payment, err := provider.ProcessPayment(context.Background(), tt.amount, tt.currency)

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

			// Verify payment details
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
