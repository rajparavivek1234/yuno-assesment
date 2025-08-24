package repository

import (
	"context"

	"yuno_assesment/internal/domain"
)

// PaymentProvider defines the interface that all payment providers must implement
type PaymentProvider interface {
	Name() string
	ProcessPayment(ctx context.Context, amount float64, currency string) (*domain.Payment, *domain.PaymentError)
	GetMetadata() map[string]interface{}
}

// PaymentRepository defines the interface for payment processing
type PaymentRepository interface {
	ProcessPayment(ctx context.Context, provider string, amount float64, currency string) (*domain.Payment, *domain.PaymentError)
	BatchProcessPayments(ctx context.Context, requests []PaymentRequest) []PaymentResult
	GetProviderMetadata(providerName string) map[string]interface{}
	ListProviders() []string
}

// PaymentRequest represents a single payment request for batch processing
type PaymentRequest struct {
	Amount   float64
	Currency string
	Provider string
}

// PaymentResult represents the result of a batch payment request
type PaymentResult struct {
	Request PaymentRequest
	Payment *domain.Payment
	Error   *domain.PaymentError
}
