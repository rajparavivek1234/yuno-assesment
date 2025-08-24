package httpclient

import (
	"net/http"
	"time"
)

// New returns a configured http.Client with sensible timeouts
func New() *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
	}
}
