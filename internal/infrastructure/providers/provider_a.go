package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"yuno_assesment/config"
	"yuno_assesment/internal/domain"
	"yuno_assesment/pkg/logger"
)

// ProviderA implements the payment provider interface for Provider A
type ProviderA struct {
	config     config.PaymentProviderConfig
	httpClient *http.Client
}

// NewProviderA creates a new instance of Provider A
func NewProviderA(config config.PaymentProviderConfig, client *http.Client) *ProviderA {
	return &ProviderA{
		config:     config,
		httpClient: client,
	}
}

// Name returns the provider name
func (p *ProviderA) Name() string {
	return p.config.Name
}

// GetMetadata returns provider metadata
func (p *ProviderA) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"name":        p.config.Name,
		"endpoint":    p.config.Endpoint,
		"timeout":     p.config.Timeout.String(),
		"retryCount":  p.config.RetryCount,
		"maxAmount":   p.config.MaxAmount,
		"description": p.config.Description,
	}
}

// ProcessPayment processes a payment through Provider A
func (p *ProviderA) ProcessPayment(ctx context.Context, amount float64, currency string) (*domain.Payment, *domain.PaymentError) {
	logger.Debug("[ProviderA] Processing payment request: amount=%.2f, currency=%s", amount, currency)

	// Validate input
	if amount <= 0 {
		logger.Error("[ProviderA] Invalid amount: %.2f", amount)
		return nil, &domain.PaymentError{
			Code:      domain.ErrInvalidAmount,
			Message:   "Amount must be greater than 0",
			Provider:  p.Name(),
			Retryable: false,
		}
	}
	if amount > p.config.MaxAmount {
		logger.Error("[ProviderA] Amount %.2f exceeds maximum limit of %.2f", amount, p.config.MaxAmount)
		return nil, &domain.PaymentError{
			Code:      domain.ErrInvalidAmount,
			Message:   fmt.Sprintf("Amount exceeds maximum limit of %v", p.config.MaxAmount),
			Provider:  p.Name(),
			Retryable: false,
		}
	}
	if currency == "" || (currency != string(domain.USD) && currency != string(domain.EUR) && currency != string(domain.GBP)) {
		logger.Error("[ProviderA] Invalid or unsupported currency: %s", currency)
		return nil, &domain.PaymentError{
			Code:      domain.ErrInvalidCurrency,
			Message:   "Invalid or unsupported currency",
			Provider:  p.Name(),
			Retryable: false,
		}
	}

	logger.Debug("[ProviderA] Preparing request payload")
	body, err := json.Marshal(map[string]interface{}{
		"amount":   amount,
		"currency": currency,
	})
	if err != nil {
		logger.Error("[ProviderA] Failed to marshal request body: %v", err)
		return nil, &domain.PaymentError{
			Code:      domain.ErrInternalError,
			Message:   "Failed to marshal request body: " + err.Error(),
			Provider:  p.Name(),
			Retryable: false,
			Details:   err.Error(),
		}
	}

	logger.Debug("[ProviderA] Creating HTTP request to endpoint: %s", p.config.Endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		logger.Error("[ProviderA] Failed to create request: %v", err)
		return nil, &domain.PaymentError{
			Code:      domain.ErrInternalError,
			Message:   "Failed to create request: " + err.Error(),
			Provider:  p.Name(),
			Retryable: false,
		}
	}
	req.Header.Set("Content-Type", "application/json")

	logger.Debug("[ProviderA] Sending payment request")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		logger.Error("[ProviderA] Failed to send request: %v", err)
		return nil, &domain.PaymentError{
			Code:      domain.ErrNetworkError,
			Message:   "Failed to send request: " + err.Error(),
			Provider:  p.Name(),
			Retryable: true,
			Details:   err.Error(),
		}
	}
	defer resp.Body.Close()

	logger.Debug("[ProviderA] Received response with status code: %d", resp.StatusCode)

	// Check HTTP status code
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		logger.Error("[ProviderA] Rate limit exceeded")
		return nil, &domain.PaymentError{
			Code:       domain.ErrRateLimitExceeded,
			Message:    "Rate limit exceeded",
			Provider:   p.Name(),
			Retryable:  true,
			HTTPStatus: resp.StatusCode,
		}
	case http.StatusInternalServerError:
		logger.Error("[ProviderA] Provider internal error occurred")
		return nil, &domain.PaymentError{
			Code:       domain.ErrInternalError,
			Message:    "Provider internal error",
			Provider:   p.Name(),
			Retryable:  true,
			HTTPStatus: resp.StatusCode,
		}
	case http.StatusBadRequest:
		logger.Error("[ProviderA] Invalid request parameters received")
		// This might be due to invalid amount, currency or other validation failures
		return nil, &domain.PaymentError{
			Code:       domain.ErrInvalidAmount,
			Message:    "Invalid request parameters",
			Provider:   p.Name(),
			Retryable:  false,
			HTTPStatus: resp.StatusCode,
		}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &domain.PaymentError{
			Code:      domain.ErrInternalError,
			Message:   "Failed to read response body: " + err.Error(),
			Provider:  p.Name(),
			Retryable: true,
			Details:   err.Error(),
		}
	}

	var response struct {
		TransactionID string    `json:"transaction_id"`
		Status        string    `json:"status"`
		Amount        float64   `json:"amount"`
		Currency      string    `json:"currency"`
		Timestamp     time.Time `json:"timestamp"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, &domain.PaymentError{
			Code:      domain.ErrProviderInvalidResp,
			Message:   "Failed to parse response: " + err.Error(),
			Provider:  p.Name(),
			Retryable: false,
			Details:   string(respBody),
		}
	}

	// Validate response fields
	if response.TransactionID == "" || response.Status == "" || response.Currency == "" {
		return nil, &domain.PaymentError{
			Code:      domain.ErrProviderInvalidResp,
			Message:   "Missing required fields in response",
			Provider:  p.Name(),
			Retryable: false,
			Details:   string(respBody),
		}
	}

	if response.Timestamp.IsZero() {
		return nil, &domain.PaymentError{
			Code:      domain.ErrProviderInvalidResp,
			Message:   "Invalid timestamp in response",
			Provider:  p.Name(),
			Retryable: false,
			Details:   string(respBody),
		}
	}

	switch response.Status {
	case "APPROVED":
		return &domain.Payment{
			ID:        response.TransactionID,
			Amount:    response.Amount,
			Currency:  domain.Currency(response.Currency),
			Status:    domain.PaymentStatus(response.Status),
			Provider:  p.Name(),
			Timestamp: response.Timestamp,
		}, nil
	case "DECLINED":
		return nil, &domain.PaymentError{
			Code:      domain.ErrCardDeclined,
			Message:   "Payment was declined",
			Provider:  p.Name(),
			Retryable: false,
		}
	default:
		return nil, &domain.PaymentError{
			Code:      domain.ErrProviderInvalidResp,
			Message:   "Invalid payment status: " + response.Status,
			Provider:  p.Name(),
			Retryable: false,
			Details:   string(respBody),
		}
	}
}
