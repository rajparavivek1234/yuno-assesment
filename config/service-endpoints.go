package config

// ServiceEndpoints defines the endpoints for external services
type ServiceEndpoints struct {
	ProviderA string
	ProviderB string
}

// DefaultServiceEndpoints returns the default service endpoints
func DefaultServiceEndpoints() ServiceEndpoints {
	return ServiceEndpoints{
		ProviderA: "http://localhost:8081/process",
		ProviderB: "http://localhost:8082/payments",
	}
}
