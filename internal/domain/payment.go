package domain

import (
	"fmt"
	"time"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	// StatusPending represents a payment that is being processed
	StatusPending PaymentStatus = "PENDING"
	// StatusApproved represents a successful payment
	StatusApproved PaymentStatus = "APPROVED"
	// StatusDeclined represents a declined payment
	StatusDeclined PaymentStatus = "DECLINED"
	// StatusError represents a payment that failed due to an error
	StatusError PaymentStatus = "ERROR"
	// StatusCancelled represents a payment that was cancelled
	StatusCancelled PaymentStatus = "CANCELLED"
	// StatusRefunded represents a payment that was refunded
	StatusRefunded PaymentStatus = "REFUNDED"
)

// Currency represents a supported currency code
type Currency string

const (
	// USD represents US Dollars
	USD Currency = "USD"
	// EUR represents Euros
	EUR Currency = "EUR"
	// GBP represents British Pounds
	GBP Currency = "GBP"
)

// Payment represents a payment entity in our domain
type Payment struct {
	ID              string        `json:"id"`
	Amount          float64       `json:"amount"`
	Currency        Currency      `json:"currency"`
	Status          PaymentStatus `json:"status"`
	Provider        string        `json:"provider"`
	Timestamp       time.Time     `json:"timestamp"`
	TransactionID   string        `json:"transaction_id,omitempty"`
	ReferenceID     string        `json:"reference_id,omitempty"`
	ErrorCode       string        `json:"error_code,omitempty"`
	ErrorMessage    string        `json:"error_message,omitempty"`
	Metadata        interface{}   `json:"metadata,omitempty"`
	RetryCount      int           `json:"retry_count,omitempty"`
	LastRetryTime   *time.Time    `json:"last_retry_time,omitempty"`
	ProviderRawData interface{}   `json:"provider_raw_data,omitempty"`
}

// Validate checks if the payment data is valid
func (p *Payment) Validate() error {
	if p.Amount <= 0 {
		return &PaymentError{Code: ErrInvalidAmount, Message: "Amount must be greater than 0"}
	}

	if p.Currency == "" {
		return &PaymentError{Code: ErrInvalidCurrency, Message: "Currency is required"}
	}

	if p.Provider == "" {
		return &PaymentError{Code: ErrProviderNotFound, Message: "Provider is required"}
	}

	return nil
}

// PaymentError represents a domain error
type PaymentError struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Provider   string      `json:"provider,omitempty"`
	Details    interface{} `json:"details,omitempty"`
	Retryable  bool        `json:"retryable"`
	HTTPStatus int         `json:"http_status,omitempty"`
}

// Error implements the error interface for PaymentError
func (e *PaymentError) Error() string {
	if e.Provider != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Provider, e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common error codes
const (
	// Payment validation errors
	ErrInsufficientFunds = "INSUFFICIENT_FUNDS"
	ErrCardDeclined      = "CARD_DECLINED"
	ErrInvalidAmount     = "INVALID_AMOUNT"
	ErrInvalidCurrency   = "INVALID_CURRENCY"

	// Provider errors
	ErrProviderNotFound     = "PROVIDER_NOT_FOUND"
	ErrProviderUnavailable  = "PROVIDER_UNAVAILABLE"
	ErrProviderTimeout      = "PROVIDER_TIMEOUT"
	ErrProviderInvalidResp  = "PROVIDER_INVALID_RESPONSE"
	ErrInvalidConfiguration = "INVALID_CONFIGURATION"

	// System errors
	ErrInvalidTimestamp = "INVALID_TIMESTAMP"
	ErrNetworkError     = "NETWORK_ERROR"
	ErrInternalError    = "INTERNAL_ERROR"

	// Rate limiting errors
	ErrRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	ErrTooManyRetries    = "TOO_MANY_RETRIES"

	// Transaction errors
	ErrDuplicateTransaction = "DUPLICATE_TRANSACTION"
	ErrTransactionNotFound  = "TRANSACTION_NOT_FOUND"
)
