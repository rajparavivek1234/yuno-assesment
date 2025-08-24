package httpclient

import (
	"bytes"
	"io"
	"net/http"
)

// MockTransport is a mock implementation of http.RoundTripper for testing
type MockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

// RoundTrip implements the http.RoundTripper interface
func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

// NewMockClient creates a new http.Client with a mock transport
func NewMockClient(roundTripFunc func(req *http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{
		Transport: &MockTransport{RoundTripFunc: roundTripFunc},
	}
}

// NewMockResponse creates a mock HTTP response
func NewMockResponse(statusCode int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}
}

// NewTimeoutMockClient creates a mock client that simulates a timeout
func NewTimeoutMockClient() *http.Client {
	return NewMockClient(func(req *http.Request) (*http.Response, error) {
		return nil, &TimeoutError{}
	})
}

// timeoutError simulates a timeout error
type TimeoutError struct{}

func (e *TimeoutError) Error() string   { return "mock timeout error" }
func (e *TimeoutError) Timeout() bool   { return true }
func (e *TimeoutError) Temporary() bool { return true }
