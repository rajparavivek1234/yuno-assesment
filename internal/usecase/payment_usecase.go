package usecase

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	"yuno_assesment/internal/domain"
	"yuno_assesment/internal/domain/repository"
	"yuno_assesment/pkg/logger"
)

// PaymentUseCase implements payment business logic
type PaymentUseCase struct {
	paymentRepo repository.PaymentRepository
}

// NewPaymentUseCase creates a new payment use case
func NewPaymentUseCase(repo repository.PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{
		paymentRepo: repo,
	}
}

// ProcessPayment processes a payment through the specified provider
func (uc *PaymentUseCase) ProcessPayment(ctx context.Context, provider string, amount float64, currency string) (*domain.Payment, *domain.PaymentError) {
	logger.Debug("Processing payment request: provider=%s, amount=%.2f, currency=%s", provider, amount, currency)

	if amount <= 0 {
		logger.Error("Invalid payment amount: %.2f", amount)
		return nil, &domain.PaymentError{
			Code:    domain.ErrInvalidAmount,
			Message: "Amount must be greater than zero",
		}
	}

	if currency == "" {
		logger.Error("Missing currency in payment request")
		return nil, &domain.PaymentError{
			Code:    domain.ErrInvalidCurrency,
			Message: "Currency is required",
		}
	}

	if provider == "" {
		logger.Error("Missing provider in payment request")
		return nil, &domain.PaymentError{
			Code:    domain.ErrProviderNotFound,
			Message: "Provider is required",
		}
	}

	payment, err := uc.paymentRepo.ProcessPayment(ctx, provider, amount, currency)
	if err != nil {
		logger.Error("Payment processing failed: %v", err)
		return nil, err
	}

	logger.Info("Payment processed successfully: ID=%s, Status=%s", payment.ID, payment.Status)
	return payment, nil
}

// GetProviderMetadata returns metadata for a specific provider
func (uc *PaymentUseCase) GetProviderMetadata(providerName string) map[string]interface{} {
	return uc.paymentRepo.GetProviderMetadata(providerName)
}

// ListProviders returns a list of all available providers
func (uc *PaymentUseCase) ListProviders() []string {
	return uc.paymentRepo.ListProviders()
}

// BatchProcessPayments processes multiple payments in batch
func (uc *PaymentUseCase) BatchProcessPayments(ctx context.Context, requests []repository.PaymentRequest) []repository.PaymentResult {
	logger.Info("Starting batch processing of %d payment requests", len(requests))
	return uc.paymentRepo.BatchProcessPayments(ctx, requests)
}

// ProcessPaymentRequestsFromCSV reads payment requests from a CSV file and processes them
func (uc *PaymentUseCase) ProcessPaymentRequestsFromCSV(ctx context.Context, filePath string) ([]repository.PaymentResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Skip header row
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	var requests []repository.PaymentRequest
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV record: %w", err)
		}

		amount, err := strconv.ParseFloat(record[0], 64)
		if err != nil {
			logger.Error("Invalid amount in CSV: %s", record[0])
			continue
		}

		request := repository.PaymentRequest{
			Amount:   amount,
			Currency: record[1],
			Provider: record[2],
		}
		requests = append(requests, request)
	}

	results := uc.BatchProcessPayments(ctx, requests)
	return results, nil
}
