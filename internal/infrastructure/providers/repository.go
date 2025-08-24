package providers

import (
	"context"
	"fmt"

	"yuno_assesment/internal/domain"
	"yuno_assesment/internal/domain/repository"
	"yuno_assesment/pkg/logger"
)

// Repository implements the payment repository interface
type Repository struct {
	providers map[string]repository.PaymentProvider
}

// NewRepository creates a new payment repository with the given providers
func NewRepository(providers ...repository.PaymentProvider) *Repository {
	providerMap := make(map[string]repository.PaymentProvider)
	for _, p := range providers {
		providerName := p.Name()
		providerMap[providerName] = p
		logger.Info("Registered payment provider: %s", providerName)
	}
	logger.Info("Payment repository initialized with %d providers", len(providers))
	return &Repository{
		providers: providerMap,
	}
}

// ProcessPayment processes a payment using the specified provider
func (r *Repository) ProcessPayment(ctx context.Context, providerName string, amount float64, currency string) (*domain.Payment, *domain.PaymentError) {
	logger.Debug("Repository: Processing payment request for provider %s: amount=%.2f, currency=%s", providerName, amount, currency)

	provider, exists := r.providers[providerName]
	if !exists {
		logger.Error("Repository: Provider %s not found", providerName)
		return nil, &domain.PaymentError{
			Code:    domain.ErrProviderNotFound,
			Message: fmt.Sprintf("Provider %s not found", providerName),
		}
	}

	payment, err := provider.ProcessPayment(ctx, amount, currency)
	if err != nil {
		logger.Error("Repository: Payment processing failed with provider %s: %v", providerName, err)
		return nil, err
	}

	logger.Info("Repository: Payment processed successfully with provider %s: ID=%s", providerName, payment.ID)
	return payment, nil
}

// GetProviderMetadata returns metadata for a specific provider
func (r *Repository) GetProviderMetadata(providerName string) map[string]interface{} {
	logger.Debug("Repository: Fetching metadata for provider: %s", providerName)
	if provider, exists := r.providers[providerName]; exists {
		metadata := provider.GetMetadata()
		logger.Debug("Repository: Retrieved metadata for provider %s", providerName)
		return metadata
	}
	logger.Error("Repository: Provider %s not found for metadata request", providerName)
	return nil
}

// ListProviders returns a list of all available provider names
func (r *Repository) ListProviders() []string {
	logger.Debug("Repository: Listing all available providers")
	var providers []string
	for name := range r.providers {
		providers = append(providers, name)
	}
	return providers
}
