package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides a configurable HTTP client with common functionality
type Client struct {
	httpClient *http.Client
	timeout    time.Duration
}

// New creates a new HTTP client with the specified timeout
func New(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		timeout:    timeout,
	}
}

// NewWithTransport creates a new HTTP client with custom transport
func NewWithTransport(timeout time.Duration, transport *http.Transport) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout, Transport: transport},
		timeout:    timeout,
	}
}

// Get performs a GET request with proper context and headers
func (c *Client) Get(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return c.httpClient.Do(req)
}

// Post performs a POST request with proper context and headers
func (c *Client) Post(ctx context.Context, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if headers == nil || headers["Content-Type"] == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// GetTimeout returns the client timeout
func (c *Client) GetTimeout() time.Duration {
	return c.timeout
}
