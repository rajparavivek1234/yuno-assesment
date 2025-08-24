package providers

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"yuno_assesment/config"
	"yuno_assesment/internal/domain"
	"yuno_assesment/internal/domain/repository"
	"yuno_assesment/pkg/logger"
)

// ProviderState tracks the health and status of a provider
type ProviderState struct {
	IsAvailable     bool
	LastChecked     time.Time
	ConsecutiveErrs int
	ErrorCount      int64
	SuccessCount    int64
	LastError       error
	mutex           sync.RWMutex
}

// Factory is responsible for creating and managing payment providers
type Factory struct {
	config         *config.Config
	httpClient     *http.Client
	providers      map[string]repository.PaymentProvider
	providerStates map[string]*ProviderState
	mutex          sync.RWMutex
}

// BatchProcessPayments processes multiple payment requests in parallel
func (f *Factory) BatchProcessPayments(ctx context.Context, requests []repository.PaymentRequest) []repository.PaymentResult {
	results := make([]repository.PaymentResult, len(requests))
	var wg sync.WaitGroup

	// Process payments in parallel with a worker pool
	workerCount := 5 // Number of concurrent workers
	requestCh := make(chan int, len(requests))

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range requestCh {
				req := requests[idx]
				payment, err := f.ProcessPayment(ctx, req.Provider, req.Amount, req.Currency)
				results[idx] = repository.PaymentResult{
					Request: req,
					Payment: payment,
					Error:   err,
				}
			}
		}()
	}

	// Send requests to workers
	for i := range requests {
		requestCh <- i
	}
	close(requestCh)

	// Wait for all requests to complete
	wg.Wait()
	return results
}

// NewFactory creates a new provider factory
func NewFactory(cfg *config.Config, client *http.Client) *Factory {
	return &Factory{
		config:         cfg,
		httpClient:     client,
		providers:      make(map[string]repository.PaymentProvider),
		providerStates: make(map[string]*ProviderState),
	}
}

// GetProviderMetadata returns metadata for a specific provider
func (f *Factory) GetProviderMetadata(providerName string) map[string]interface{} {
	provider, err := f.getOrCreateProvider(providerName)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}
	return provider.GetMetadata()
}

// ListProviders returns a list of all available providers
func (f *Factory) ListProviders() []string {
	providers := make([]string, 0, len(f.config.Providers))
	for providerName := range f.config.Providers {
		providers = append(providers, providerName)
	}
	return providers
}

// getOrCreateProvider gets an existing provider or creates a new one
func (f *Factory) getOrCreateProvider(providerName string) (repository.PaymentProvider, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// Check if provider already exists
	if provider, exists := f.providers[providerName]; exists {
		return provider, nil
	}

	// Get provider config
	providerConfig, exists := f.config.Providers[providerName]
	if !exists {
		return nil, &domain.PaymentError{
			Code:    domain.ErrProviderNotFound,
			Message: fmt.Sprintf("Provider %s not found", providerName),
		}
	}

	// Create new provider
	var provider repository.PaymentProvider
	switch providerName {
	case "ProviderA":
		provider = NewProviderA(providerConfig, f.httpClient)
	case "ProviderB":
		provider = NewProviderB(providerConfig, f.httpClient)
	default:
		return nil, &domain.PaymentError{
			Code:    domain.ErrProviderNotFound,
			Message: fmt.Sprintf("Provider %s not supported", providerName),
		}
	}

	// Initialize provider state
	f.providerStates[providerName] = &ProviderState{
		IsAvailable: true,
		LastChecked: time.Now(),
	}

	f.providers[providerName] = provider
	return provider, nil
}

// updateProviderState updates the state of a provider
func (f *Factory) updateProviderState(providerName string, success bool, err error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	state, exists := f.providerStates[providerName]
	if !exists {
		state = &ProviderState{}
		f.providerStates[providerName] = state
	}

	state.mutex.Lock()
	defer state.mutex.Unlock()

	state.LastChecked = time.Now()

	if success {
		state.IsAvailable = true
		state.ConsecutiveErrs = 0
		state.SuccessCount++
	} else {
		state.ConsecutiveErrs++
		state.ErrorCount++
		state.LastError = err
		if state.ConsecutiveErrs >= 3 {
			state.IsAvailable = false
		}
	}
}

// ProcessPayment processes a payment through the specified provider
func (f *Factory) ProcessPayment(ctx context.Context, providerName string, amount float64, currency string) (*domain.Payment, *domain.PaymentError) {
	provider, err := f.getOrCreateProvider(providerName)
	if err != nil {
		return nil, err.(*domain.PaymentError)
	}

	payment, paymentErr := provider.ProcessPayment(ctx, amount, currency)

	if paymentErr != nil {
		f.updateProviderState(providerName, false, paymentErr)
		return nil, paymentErr
	}

	f.updateProviderState(providerName, true, nil)
	return payment, nil
}

// validateProviderConfig checks if the provider configuration is valid
func (f *Factory) validateProviderConfig(cfg config.PaymentProviderConfig) error {
	if cfg.Name == "" {
		return &domain.PaymentError{
			Code:    domain.ErrInvalidConfiguration,
			Message: "Provider name is required",
		}
	}

	if cfg.Endpoint == "" {
		return &domain.PaymentError{
			Code:    domain.ErrInvalidConfiguration,
			Message: fmt.Sprintf("Endpoint is required for provider %s", cfg.Name),
		}
	}

	if cfg.MaxAmount <= 0 {
		return &domain.PaymentError{
			Code:    domain.ErrInvalidConfiguration,
			Message: fmt.Sprintf("Invalid max amount for provider %s", cfg.Name),
		}
	}

	return nil
}

// CreateProvider creates a new provider instance with validation and state tracking
func (f *Factory) CreateProvider(name string) (repository.PaymentProvider, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	logger.Debug("Creating provider: %s", name)

	// Check if provider already exists
	if provider, exists := f.providers[name]; exists {
		logger.Debug("Provider %s already exists, returning existing instance", name)
		return provider, nil
	}

	cfg, exists := f.config.Providers[name]
	if !exists {
		logger.Error("No configuration found for provider: %s", name)
		return nil, &domain.PaymentError{
			Code:    domain.ErrProviderNotFound,
			Message: fmt.Sprintf("No configuration found for provider: %s", name),
		}
	}

	// Validate provider configuration
	if err := f.validateProviderConfig(cfg); err != nil {
		logger.Error("Invalid configuration for provider %s: %v", name, err)
		return nil, err
	}

	logger.Info("Creating new instance of provider: %s", name)
	var provider repository.PaymentProvider
	switch name {
	case "ProviderA":
		provider = NewProviderA(cfg, f.httpClient)
	case "ProviderB":
		provider = NewProviderB(cfg, f.httpClient)
	default:
		return nil, &domain.PaymentError{
			Code:    domain.ErrProviderNotFound,
			Message: fmt.Sprintf("Unknown provider type: %s", name),
		}
	}

	// Initialize provider state
	f.providerStates[name] = &ProviderState{
		IsAvailable: true,
		LastChecked: time.Now(),
	}

	f.providers[name] = provider
	return provider, nil
}

// UpdateProviderState updates the state of a provider based on operation results
func (f *Factory) UpdateProviderState(name string, err error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	state, exists := f.providerStates[name]
	if !exists {
		return
	}

	state.mutex.Lock()
	defer state.mutex.Unlock()

	state.LastChecked = time.Now()
	if err != nil {
		state.ConsecutiveErrs++
		state.ErrorCount++
		state.LastError = err

		// Disable provider if too many consecutive errors
		if state.ConsecutiveErrs >= 3 {
			state.IsAvailable = false
		}
	} else {
		state.ConsecutiveErrs = 0
		state.SuccessCount++
		state.IsAvailable = true
	}
}

// GetProviderState returns the current state of a provider
func (f *Factory) GetProviderState(name string) *ProviderState {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	return f.providerStates[name]
}

// GetAllProviders returns all registered providers
func (f *Factory) GetAllProviders() []repository.PaymentProvider {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	var result []repository.PaymentProvider
	for _, provider := range f.providers {
		result = append(result, provider)
	}
	return result
}
