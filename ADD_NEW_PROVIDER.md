## Adding a New Provider

1. Create Provider Implementation

   Create a new file `internal/infrastructure/providers/provider_c.go`:
   ```go
   package providers

   type ProviderC struct {
       config     config.PaymentProviderConfig
       httpClient *http.Client
   }

   func NewProviderC(config config.PaymentProviderConfig, client *http.Client) *ProviderC {
       return &ProviderC{
           config:     config,
           httpClient: client,
       }
   }

   func (p *ProviderC) ProcessPayment(ctx context.Context, amount float64, currency string) (*domain.Payment, *domain.PaymentError) {
       // Implementation here
   }

   // impl other methods ...
   ```

2. Add Provider Configuration

   In `config/config.go`:
   ```go
   "ProviderC": {
		Name:        "ProviderB",
		Endpoint:    endpoints.ProviderB,
		Timeout:     30 * time.Second,
		RetryCount:  3,
		MaxAmount:   10000.0,
		Description: "Payment Provider B",
		RetryPolicy: defaultRetryPolicy,
		RateLimit: RateLimit{
			RequestsPerSecond: 50,
			BurstSize:         5,
		},
	}
   ```

   In `config/service-endpoints.go`:
   ```go
	ProviderC: "http://localhost:8083/payments",
   ```

3. Register in Factory

   In `internal/infrastructure/providers/factory.go`:
   ```go
   case "ProviderC":
       provider = NewProviderC(cfg, f.httpClient)
   ```
