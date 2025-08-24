package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"yuno_assesment/config"
	"yuno_assesment/internal/domain/repository"
	"yuno_assesment/internal/infrastructure/providers"
	"yuno_assesment/internal/usecase"
	"yuno_assesment/pkg/logger"
)

func main() {
	// Create mock servers for demonstration
	serverA := createMockProviderAServer()
	defer serverA.Close()

	serverB := createMockProviderBServer()
	defer serverB.Close()

	// Initialize configuration
	cfg := config.DefaultConfig()

	// Map mock servers to providers
	mockServers := map[string]*httptest.Server{
		"ProviderA": serverA,
		"ProviderB": serverB,
	}

	// Update provider endpoints in config
	for providerName, server := range mockServers {
		if providerConfig, exists := cfg.Providers[providerName]; exists {
			providerConfig.Endpoint = server.URL
			cfg.Providers[providerName] = providerConfig
		}
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Create provider factory which implements PaymentRepository
	paymentRepo := providers.NewFactory(cfg, client)
	logger.Info("Initializing payment processing system")

	// Create payment use case with the payment repository
	paymentUseCase := usecase.NewPaymentUseCase(paymentRepo)

	// Process payments from CSV file
	// for debugging purposes, replace the following line with:
	// logger.Info("Starting batch payment processing from CSV file")
	// exePath, err := os.Executable()
	// if err != nil {
	// 	panic(err)
	// }

	// exeDir := filepath.Dir(exePath)
	// filePath := filepath.Join(, "..", "test_data", "payment_requests.csv")

	results, err := paymentUseCase.ProcessPaymentRequestsFromCSV(context.Background(), "test_data/payment_requests.csv")
	if err != nil {
		logger.Error("Failed to process CSV file: %v", err)
		os.Exit(1)
	}

	// Create results directory if it doesn't exist
	err = os.MkdirAll("test_data", 0755)
	if err != nil {
		logger.Error("Failed to create test_data directory: %v", err)
		os.Exit(1)
	}

	// Write results to output file
	makeResultOutPutFile(results)

	logger.Info("Payment processing completed. Results written to test_data/payment_results.txt")
}

// createMockProviderAServer creates a test server that simulates Provider A's API
func createMockProviderAServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		amount := requestBody["amount"].(float64)
		status := "APPROVED"
		if amount == 999 {
			status = "DECLINED"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"transaction_id": fmt.Sprintf("TXN-DEMO-%d", int(amount)),
			"status":         status,
			"amount":         amount,
			"currency":       requestBody["currency"],
			"timestamp":      time.Now().Format(time.RFC3339),
		})
	}))
}

// createMockProviderBServer creates a test server that simulates Provider B's API
func createMockProviderBServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		amount := requestBody["amount"].(float64)
		state := "SUCCESS"
		if amount == 999 {
			state = "FAILED"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"paymentId": fmt.Sprintf("PAY-DEMO-%d", int(amount)),
			"state":     state,
			"value": map[string]interface{}{
				"amount":       fmt.Sprintf("%.2f", amount),
				"currencyCode": requestBody["currency"],
			},
			"processedAt": time.Now().UnixMilli(),
		})
	}))
}

func makeResultOutPutFile(results []repository.PaymentResult) {
	// Write results to output file
	// for debugging purposes, replace the following line with:
	// logger.Info("Starting batch payment processing from CSV file")
	// exePath, err := os.Executable()
	// if err != nil {
	// 	panic(err)
	// }

	// exeDir := filepath.Dir(exePath)
	// filePath := filepath.Join(, "..", "test_data", "payment_results.txt")
	outputFile, err := os.Create("test_data/payment_results.txt")
	if err != nil {
		logger.Error("Failed to create output file: %v", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	// Write results header
	fmt.Fprintln(outputFile, "Payment Processing Results")
	fmt.Fprintln(outputFile, "------------------------")
	fmt.Fprintln(outputFile)

	// Process and write each result
	for i, result := range results {
		fmt.Fprintf(outputFile, "Payment Request #%d:\n", i+1)

		if result.Request.Amount != 0 {
			fmt.Fprintf(outputFile, "  Amount: %.2f %s\n", result.Request.Amount, result.Request.Currency)
			fmt.Fprintf(outputFile, "  Provider: %s\n", result.Request.Provider)

			if result.Error != nil {
				fmt.Fprintf(outputFile, "  Status: Failed\n")
				fmt.Fprintf(outputFile, "  Error: %s (%s)\n", result.Error.Message, result.Error.Code)
			} else if result.Payment != nil {
				fmt.Fprintf(outputFile, "  Status: Success\n")
				fmt.Fprintf(outputFile, "  Payment ID: %s\n", result.Payment.ID)
				fmt.Fprintf(outputFile, "  Payment Status: %s\n", result.Payment.Status)
			} else {
				fmt.Fprintf(outputFile, "  Status: Unknown\n")
			}
		} else {
			fmt.Fprintf(outputFile, "  Status: Invalid Request\n")
		}
		fmt.Fprintln(outputFile)
	}
}
