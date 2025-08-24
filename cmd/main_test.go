package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"yuno_assesment/config"
	"yuno_assesment/internal/domain"
	"yuno_assesment/internal/domain/repository"
	"yuno_assesment/internal/infrastructure/providers"
	"yuno_assesment/internal/usecase"
)

func TestMain(t *testing.T) {
	// Test the main components without actually running the main function
	t.Run("Initialize Configuration", func(t *testing.T) {
		cfg := config.DefaultConfig()
		if cfg == nil {
			t.Error("Expected non-nil configuration")
		}
	})

	t.Run("Create Mock Servers", func(t *testing.T) {
		serverA := createMockProviderAServer()
		defer serverA.Close()

		serverB := createMockProviderBServer()
		defer serverB.Close()

		if serverA == nil || serverB == nil {
			t.Error("Expected non-nil mock servers")
		}
	})

	t.Run("Create Provider Factory", func(t *testing.T) {
		cfg := config.DefaultConfig()
		client := &http.Client{}
		factory := providers.NewFactory(cfg, client)
		if factory == nil {
			t.Error("Expected non-nil provider factory")
		}
	})
}

func TestCreateMockProviderAServer(t *testing.T) {
	server := createMockProviderAServer()
	defer server.Close()

	t.Run("Successful Payment", func(t *testing.T) {
		payload := map[string]interface{}{
			"amount":   100.00,
			"currency": "USD",
		}
		jsonPayload, _ := json.Marshal(payload)
		resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["status"] != "APPROVED" {
			t.Errorf("Expected status APPROVED, got %v", result["status"])
		}
	})

	t.Run("Declined Payment", func(t *testing.T) {
		payload := map[string]interface{}{
			"amount":   999.00,
			"currency": "USD",
		}
		jsonPayload, _ := json.Marshal(payload)
		resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["status"] != "DECLINED" {
			t.Errorf("Expected status DECLINED, got %v", result["status"])
		}
	})
}

func TestCreateMockProviderBServer(t *testing.T) {
	server := createMockProviderBServer()
	defer server.Close()

	t.Run("Successful Payment", func(t *testing.T) {
		payload := map[string]interface{}{
			"amount":   100.00,
			"currency": "USD",
		}
		jsonPayload, _ := json.Marshal(payload)
		resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["state"] != "SUCCESS" {
			t.Errorf("Expected state SUCCESS, got %v", result["state"])
		}
	})

	t.Run("Failed Payment", func(t *testing.T) {
		payload := map[string]interface{}{
			"amount":   999.00,
			"currency": "USD",
		}
		jsonPayload, _ := json.Marshal(payload)
		resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result["state"] != "FAILED" {
			t.Errorf("Expected state FAILED, got %v", result["state"])
		}
	})
}

func TestMakeResultOutputFile(t *testing.T) {
	// Create a temporary directory for test output
	tempDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change working directory to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create test_data directory
	err = os.Mkdir("test_data", 0755)
	if err != nil {
		t.Fatalf("Failed to create test_data directory: %v", err)
	}

	// Create test data
	results := []repository.PaymentResult{
		{
			Request: repository.PaymentRequest{Amount: 100.00, Currency: "USD", Provider: "ProviderA"},
			Payment: &domain.Payment{ID: "PAY-001", Status: domain.StatusApproved},
		},
		{
			Request: repository.PaymentRequest{Amount: 999.00, Currency: "USD", Provider: "ProviderB"},
			Error:   &domain.PaymentError{Code: "DECLINED", Message: "Payment declined"},
		},
	}

	// Call the function
	makeResultOutPutFile(results)

	// Check if the file was created
	if _, err := os.Stat("test_data/payment_results.txt"); os.IsNotExist(err) {
		t.Error("Expected payment_results.txt to be created, but it doesn't exist")
	}

	// TODO: Add more detailed checks on the content of the file
}

func TestIntegration(t *testing.T) {
	// Create mock servers
	serverA := createMockProviderAServer()
	defer serverA.Close()

	serverB := createMockProviderBServer()
	defer serverB.Close()

	// Initialize configuration
	cfg := config.DefaultConfig()

	// Update provider endpoints and MaxAmount in config
	cfg.Providers["ProviderA"] = config.PaymentProviderConfig{
		Endpoint:  serverA.URL,
		MaxAmount: 1000.00, // Set an appropriate maximum amount
	}
	cfg.Providers["ProviderB"] = config.PaymentProviderConfig{
		Endpoint:  serverB.URL,
		MaxAmount: 1000.00, // Set an appropriate maximum amount
	}

	// Create HTTP client
	client := &http.Client{}

	// Create provider factory
	paymentRepo := providers.NewFactory(cfg, client)

	// Create payment use case
	paymentUseCase := usecase.NewPaymentUseCase(paymentRepo)

	// Create a temporary CSV file for testing
	tempFile, err := os.CreateTemp("", "test_payments_*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test data to the CSV file
	_, err = tempFile.WriteString("amount,currency,provider\n100.00,USD,ProviderA\n999.00,USD,ProviderB\n")
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Process payments from CSV file
	results, err := paymentUseCase.ProcessPaymentRequestsFromCSV(context.Background(), tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to process CSV file: %v", err)
	}

	// Add detailed logging
	t.Logf("Results: %+v", results)

	// Check results
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if results[0].Payment == nil {
		t.Errorf("Expected first payment to be non-nil, got nil")
	} else if results[0].Payment.Status != domain.StatusApproved {
		t.Errorf("Expected first payment status to be %s, got %s", domain.StatusApproved, results[0].Payment.Status)
	}

	if results[1].Error == nil {
		t.Errorf("Expected second payment to have an error, got nil")
	} else if results[1].Error.Code != domain.ErrCardDeclined {
		t.Errorf("Expected second payment error code to be %s, got %s", domain.ErrCardDeclined, results[1].Error.Code)
	}
}
