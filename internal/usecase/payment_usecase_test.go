package usecase

import (
	"context"
	"testing"
	"time"

	"yuno_assesment/internal/domain"
	"yuno_assesment/internal/domain/repository"
)

type mockPaymentRepository struct {
	payments map[string]*domain.Payment
	errors   map[string]*domain.PaymentError
}

func newMockPaymentRepository() *mockPaymentRepository {
	return &mockPaymentRepository{
		payments: make(map[string]*domain.Payment),
		errors:   make(map[string]*domain.PaymentError),
	}
}

func (m *mockPaymentRepository) ProcessPayment(ctx context.Context, provider string, amount float64, currency string) (*domain.Payment, *domain.PaymentError) {
	if err, exists := m.errors[provider]; exists {
		return nil, err
	}

	if payment, exists := m.payments[provider]; exists {
		return payment, nil
	}

	return nil, &domain.PaymentError{
		Code:    domain.ErrProviderNotFound,
		Message: "Provider not found",
	}
}

func (m *mockPaymentRepository) GetProviderMetadata(providerName string) map[string]interface{} {
	return map[string]interface{}{
		"name":    providerName,
		"timeout": "5s",
	}
}

func (m *mockPaymentRepository) ListProviders() []string {
	providers := make([]string, 0, len(m.payments))
	for provider := range m.payments {
		providers = append(providers, provider)
	}
	return providers
}

func (m *mockPaymentRepository) BatchProcessPayments(ctx context.Context, requests []repository.PaymentRequest) []repository.PaymentResult {
	results := make([]repository.PaymentResult, len(requests))
	for i, req := range requests {
		payment, err := m.ProcessPayment(ctx, req.Provider, req.Amount, req.Currency)
		results[i] = repository.PaymentResult{
			Request: req,
			Payment: payment,
			Error:   err,
		}
	}
	return results
}

func TestPaymentUseCase_ProcessPayment(t *testing.T) {
	// Setup test data
	now := time.Now()
	successfulPayment := &domain.Payment{
		ID:        "TXN-123",
		Amount:    100.00,
		Currency:  domain.USD,
		Status:    domain.StatusApproved,
		Provider:  "ProviderA",
		Timestamp: now,
	}

	declinedError := &domain.PaymentError{
		Code:    domain.ErrCardDeclined,
		Message: "Payment was declined",
	}

	tests := []struct {
		name           string
		provider       string
		amount         float64
		currency       string
		setupMock      func(*mockPaymentRepository)
		expectedError  bool
		expectedStatus domain.PaymentStatus
	}{
		{
			name:     "successful payment",
			provider: "ProviderA",
			amount:   100.00,
			currency: "USD",
			setupMock: func(m *mockPaymentRepository) {
				m.payments["ProviderA"] = successfulPayment
			},
			expectedError:  false,
			expectedStatus: domain.StatusApproved,
		},
		{
			name:     "declined payment",
			provider: "ProviderA",
			amount:   999.00,
			currency: "USD",
			setupMock: func(m *mockPaymentRepository) {
				m.errors["ProviderA"] = declinedError
			},
			expectedError: true,
		},
		{
			name:     "invalid amount",
			provider: "ProviderA",
			amount:   -100.00,
			currency: "USD",
			setupMock: func(m *mockPaymentRepository) {
				m.errors["ProviderA"] = &domain.PaymentError{
					Code:    domain.ErrInvalidAmount,
					Message: "Amount must be positive",
				}
			},
			expectedError: true,
		},
		{
			name:          "provider not found",
			provider:      "NonExistentProvider",
			amount:        100.00,
			currency:      "USD",
			setupMock:     func(m *mockPaymentRepository) {},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := newMockPaymentRepository()
			tt.setupMock(mockRepo)

			// Create use case
			useCase := NewPaymentUseCase(mockRepo)

			// Process payment
			payment, err := useCase.ProcessPayment(context.Background(), tt.provider, tt.amount, tt.currency)

			// Check error expectation
			if tt.expectedError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// For successful cases, verify payment details
			if err == nil {
				if payment.Amount != tt.amount {
					t.Errorf("expected amount %v, got %v", tt.amount, payment.Amount)
				}
				if string(payment.Currency) != tt.currency {
					t.Errorf("expected currency %v, got %v", tt.currency, payment.Currency)
				}
				if payment.Status != tt.expectedStatus {
					t.Errorf("expected status %v, got %v", tt.expectedStatus, payment.Status)
				}
				if payment.Provider != tt.provider {
					t.Errorf("expected provider %v, got %v", tt.provider, payment.Provider)
				}
				if payment.ID == "" {
					t.Error("expected non-empty payment ID")
				}
				if payment.Timestamp.IsZero() {
					t.Error("expected non-zero timestamp")
				}
			}
		})
	}
}
