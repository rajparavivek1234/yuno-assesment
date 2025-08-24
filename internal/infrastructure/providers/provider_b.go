package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"yuno_assesment/config"
	"yuno_assesment/internal/domain"
	"yuno_assesment/pkg/logger"
)

// ProviderB implements the payment provider interface for Provider B
type ProviderB struct {
	config     config.PaymentProviderConfig
	httpClient *http.Client
}

// NewProviderB creates a new instance of Provider B
func NewProviderB(config config.PaymentProviderConfig, client *http.Client) *ProviderB {
	return &ProviderB{
		config:     config,
		httpClient: client,
	}
}

// Name returns the provider name
func (p *ProviderB) Name() string {
	return p.config.Name
}

// GetMetadata returns provider metadata
func (p *ProviderB) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"name":        p.config.Name,
		"endpoint":    p.config.Endpoint,
		"timeout":     p.config.Timeout.String(),
		"retryCount":  p.config.RetryCount,
		"maxAmount":   p.config.MaxAmount,
		"description": p.config.Description,
	}
}

// ProcessPayment processes a payment through Provider B
func (p *ProviderB) ProcessPayment(ctx context.Context, amount float64, currency string) (*domain.Payment, *domain.PaymentError) {
	logger.Debug("[ProviderB] Processing payment request: amount=%.2f, currency=%s", amount, currency)

	// Validate amount and currency
	if amount <= 0 {
		logger.Error("[ProviderB] Invalid amount: %.2f", amount)
		return nil, &domain.PaymentError{
			Code:    domain.ErrInvalidAmount,
			Message: "Amount must be greater than 0",
		}
	}

	if amount > p.config.MaxAmount {
		logger.Error("[ProviderB] Amount %.2f exceeds maximum limit of %.2f", amount, p.config.MaxAmount)
		return nil, &domain.PaymentError{
			Code:    domain.ErrInvalidAmount,
			Message: fmt.Sprintf("Amount exceeds maximum limit of %v", p.config.MaxAmount),
		}
	}

	if currency == "" {
		logger.Error("[ProviderB] Currency is required")
		return nil, &domain.PaymentError{
			Code:    domain.ErrInvalidCurrency,
			Message: "Currency is required",
		}
	}

	// Prepare request body
	logger.Debug("[ProviderB] Preparing request payload")
	body, err := json.Marshal(map[string]interface{}{
		"amount":   amount,
		"currency": currency,
	})
	if err != nil {
		logger.Error("[ProviderB] Failed to marshal request body: %v", err)
		return nil, &domain.PaymentError{
			Code:      domain.ErrInternalError,
			Message:   "Failed to marshal request body",
			Provider:  p.Name(),
			Details:   err.Error(),
			Retryable: false,
		}
	}

	logger.Debug("[ProviderB] Creating HTTP request to endpoint: %s", p.config.Endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		logger.Error("[ProviderB] Failed to create request: %v", err)
		return nil, &domain.PaymentError{
			Code:    domain.ErrInternalError,
			Message: "Failed to create request",
		}
	}
	req.Header.Set("Content-Type", "application/json")

	logger.Debug("[ProviderB] Sending payment request")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		logger.Error("[ProviderB] Request failed: %v", err)
		errCode := domain.ErrNetworkError
		if err.Error() == "context deadline exceeded" {
			errCode = domain.ErrProviderTimeout
		}
		return nil, &domain.PaymentError{
			Code:      errCode,
			Message:   "Failed to send request: " + err.Error(),
			Provider:  p.Name(),
			Retryable: true,
		}
	}
	defer resp.Body.Close()

	logger.Debug("[ProviderB] Received response with status code: %d", resp.StatusCode)

	// Check response status
	if resp.StatusCode >= 500 {
		logger.Error("[ProviderB] Provider server error: %d", resp.StatusCode)
		return nil, &domain.PaymentError{
			Code:      domain.ErrProviderUnavailable,
			Message:   fmt.Sprintf("Provider error: %d", resp.StatusCode),
			Provider:  p.Name(),
			Retryable: true,
		}
	} else if resp.StatusCode == http.StatusTooManyRequests {
		logger.Error("[ProviderB] Rate limit exceeded")
		return nil, &domain.PaymentError{
			Code:      domain.ErrRateLimitExceeded,
			Message:   "Rate limit exceeded",
			Provider:  p.Name(),
			Retryable: true,
		}
	} else if resp.StatusCode >= 400 {
		logger.Error("[ProviderB] Invalid request error: %d", resp.StatusCode)
		return nil, &domain.PaymentError{
			Code:      domain.ErrProviderInvalidResp,
			Message:   fmt.Sprintf("Invalid request: %d", resp.StatusCode),
			Provider:  p.Name(),
			Retryable: false,
		}
	}

	logger.Debug("[ProviderB] Reading response body")
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("[ProviderB] Failed to read response body: %v", err)
		return nil, &domain.PaymentError{
			Code:      domain.ErrInternalError,
			Message:   "Failed to read response body: " + err.Error(),
			Provider:  p.Name(),
			Retryable: true,
		}
	}

	var response struct {
		PaymentID string `json:"paymentId"`
		State     string `json:"state"`
		Value     struct {
			Amount       string `json:"amount"`
			CurrencyCode string `json:"currencyCode"`
		} `json:"value"`
		ProcessedAt int64 `json:"processedAt"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, &domain.PaymentError{
			Code:      domain.ErrProviderInvalidResp,
			Message:   "Failed to parse provider response: " + err.Error(),
			Provider:  p.Name(),
			Retryable: false,
			Details:   string(respBody),
		}
	}

	// Map provider status to domain status
	var status domain.PaymentStatus
	switch response.State {
	case "SUCCESS":
		status = domain.StatusApproved
	case "FAILED":
		return nil, &domain.PaymentError{
			Code:      domain.ErrCardDeclined,
			Message:   "Payment was declined by provider",
			Provider:  p.Name(),
			Retryable: false,
		}
	default:
		return nil, &domain.PaymentError{
			Code:      domain.ErrProviderInvalidResp,
			Message:   "Invalid payment status from provider: " + response.State,
			Provider:  p.Name(),
			Retryable: false,
		}
	}

	// Validate and parse amount
	amount, err = strconv.ParseFloat(response.Value.Amount, 64)
	if err != nil {
		return nil, &domain.PaymentError{
			Code:      domain.ErrProviderInvalidResp,
			Message:   "Invalid amount format in response: " + err.Error(),
			Provider:  p.Name(),
			Retryable: false,
		}
	}

	// Validate currency
	if response.Value.CurrencyCode == "" {
		return nil, &domain.PaymentError{
			Code:      domain.ErrProviderInvalidResp,
			Message:   "Missing currency in provider response",
			Provider:  p.Name(),
			Retryable: false,
		}
	}

	return &domain.Payment{
		ID:        response.PaymentID,
		Amount:    amount,
		Currency:  domain.Currency(response.Value.CurrencyCode),
		Status:    status,
		Provider:  p.Name(),
		Timestamp: time.Unix(response.ProcessedAt/1000, 0),
	}, nil
}
