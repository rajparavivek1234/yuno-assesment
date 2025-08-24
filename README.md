## Implementation

- Support for multiple payment providers (Provider A and Provider B)
- Batch processing from CSV files
- Parallel payment processing with worker pool
- Basic validation for payment requests:
  - Amount validation (must be > 0)
  - Currency validation
  - Provider validation
- CSV input processing
- Detailed result reporting

## Folder Structure

```
.
├── cmd/
│   └── main.go            # Application entry point
├── config/
│   ├── config.go          # Configuration types and loading
│   └── endpoints.go       # Provider endpoint configurations
├── internal/
│   ├── domain/
│   │   ├── payment.go     # Core domain types and interfaces
│   │   └── repository/    # Repository interfaces
│   ├── infrastructure/
│   │   └── providers/     # Provider implementations
│   └── usecase/           # Business logic
├── pkg/
│   ├── logger/            # Loggers
│   └── httpclient/        # HTTP client utilities
└── go.mod
```

### TO Run

1. Prepare input CSV file in `test_data/payment_requests.csv`:
   ```csv
   amount,currency,provider
   100.00,USD,ProviderA
   50.75,EUR,ProviderB
   ```

   => CSV File Format
      - amount: Decimal number (> 0)
      - currency: Currency code (e.g., USD, EUR)
      - provider: ProviderA or ProviderB

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Run the tests:
   ```bash
   go test ./...
   ```

4. Running the Application
   ```bash
   go run cmd/main.go
   ```

5. Check results in `test_data/payment_results.txt`:
   ```text
   Payment Processing Results
   ------------------------

   Payment Request #1:
     Amount: 100.00 USD
     Provider: ProviderA
     Status: Success
     Payment ID: TXN-DEMO-100
     Payment Status: APPROVED

   Payment Request #2:
     Amount: -50.00 USD
     Provider: ProviderB
     Status: Failed
     Error: Amount must be greater than 0 (INVALID_AMOUNT)
   ```

## Error Handling

The system handles various types of errors:
   - Validation Errors (Invalid amount, currency)
   - Provider Errors (Provider unavailable, timeout)

Each error includes:
   - Error Code
   - message


## Testing

1. Unit Tests
   ```bash
   go test ./...
   ```
   
   Tests individual components in isolation:
   - Provider implementations
   - Use case business logic
   - Error handling and normalization
   - Edge cases for each component

## Test Scenarios

1. Payment Processing
   - Successful payments
   - Declined payments
   - Invalid amounts
   - Invalid currencies

2. Validation
   - Amount range validation
   - Currency code validation
   - Provider availability
   - Request format
   - Response parsing

3. Edge Cases
   - Zero amount
   - Maximum amount
   - Invalid timestamps
   - Provider downtime