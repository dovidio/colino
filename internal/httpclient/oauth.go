package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuthResponse represents a generic OAuth response with flexible field mapping
type OAuthResponse struct {
	Data map[string]any `json:"-"`
}

// NewOAuthResponse creates a new OAuth response from raw data
func NewOAuthResponse(data map[string]any) *OAuthResponse {
	return &OAuthResponse{Data: data}
}

// GetString extracts a string value from the response using multiple possible field names
func (r *OAuthResponse) GetString(keys ...string) string {
	for _, key := range keys {
		if v, ok := r.Data[key]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}

// DoOAuthRequest performs an HTTP request with common OAuth error handling
func (c *Client) DoOAuthRequest(ctx context.Context, method, url string, headers map[string]string) (*OAuthResponse, error) {
	var resp *http.Response
	var err error

	switch strings.ToUpper(method) {
	case "GET":
		resp, err = c.Get(ctx, url, headers)
	case "POST":
		resp, err = c.Post(ctx, url, nil, headers)
	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return NewOAuthResponse(data), nil
}

// TryMultipleURLs attempts the same request on multiple URLs until one succeeds
func (c *Client) TryMultipleURLs(ctx context.Context, method string, urls []string, headers map[string]string) (*OAuthResponse, error) {
	var lastErr error

	for i, requestURL := range urls {
		resp, err := c.DoOAuthRequest(ctx, method, requestURL, headers)
		if err != nil {
			lastErr = err
			if i < len(urls)-1 {
				continue // Try next URL
			}
		}
		return resp, err
	}

	return nil, lastErr
}

// BuildOAuthURLs generates multiple URL candidates for OAuth endpoints
func BuildOAuthURLs(baseURL string, pathVariants []string, paramVariants []string, paramValue string) []string {
	baseURL = strings.TrimRight(baseURL, "/")
	var urls []string

	for _, path := range pathVariants {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		// Path parameter style: /path/{value}
		if paramValue != "" {
			urls = append(urls, baseURL+path+"/"+url.PathEscape(paramValue))
		}

		// Query parameter styles
		for _, param := range paramVariants {
			if paramValue != "" {
				urls = append(urls, baseURL+path+"?"+param+"="+url.QueryEscape(paramValue))
			}
		}
	}

	return urls
}

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff(ctx context.Context, maxAttempts int, baseDelay time.Duration, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		if err := fn(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}

	return lastErr
}

// ValidateOAuthResponse checks if an OAuth response contains required fields
func ValidateOAuthResponse(resp *OAuthResponse, requiredFields map[string][]string) error {
	if resp == nil || resp.Data == nil {
		return errors.New("empty OAuth response")
	}

	for fieldName, possibleKeys := range requiredFields {
		value := resp.GetString(possibleKeys...)
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("missing required field: %s", fieldName)
		}
	}

	return nil
}
