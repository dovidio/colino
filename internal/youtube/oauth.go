package youtube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golino/internal/httpclient"
)

const (
	// DefaultOAuthBaseURL is the default base URL for OAuth operations
	DefaultOAuthBaseURL = "https://colino.umberto.xyz"
)

// OAuthConfig contains configuration for YouTube OAuth operations
type OAuthConfig struct {
	BaseURL          string
	InitiatePath     string
	PollPath         string
	PollParam        string
	RequestTimeout   time.Duration
}

// DefaultOAuthConfig returns a default OAuth configuration
func DefaultOAuthConfig() *OAuthConfig {
	return &OAuthConfig{
		BaseURL:        DefaultOAuthBaseURL,
		RequestTimeout: 180 * time.Second,
	}
}

// OAuthInitiateResponse represents the response from OAuth initiation
type OAuthInitiateResponse struct {
	AuthURL string `json:"auth_url"`
	FlowID  string `json:"flow_id"`
}

// OAuthPollResponse represents the response from OAuth polling
type OAuthPollResponse struct {
	Status       string `json:"status"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    string `json:"expires_in"`
	Error        string `json:"error"`
}

// Channel represents a YouTube channel
type Channel struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// OAuthService handles YouTube OAuth operations
type OAuthService struct {
	client *httpclient.Client
	config *OAuthConfig
}

// NewOAuthService creates a new OAuth service
func NewOAuthService(config *OAuthConfig) *OAuthService {
	if config == nil {
		config = DefaultOAuthConfig()
	}

	client := httpclient.New(config.RequestTimeout)

	return &OAuthService{
		client: client,
		config: config,
	}
}

// InitiateOAuth starts the OAuth flow by requesting an authorization URL
func (s *OAuthService) InitiateOAuth(ctx context.Context) (*OAuthInitiateResponse, error) {
	url := strings.TrimRight(s.config.BaseURL, "/") + "/auth/initiate"

	resp, err := s.client.Get(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate OAuth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("OAuth initiate failed with status %d", resp.StatusCode)
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode OAuth response: %w", err)
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return s
				}
			}
		}
		return ""
	}

	authURL := getStr("auth_url", "authorization_url", "url", "authorize_url")
	sessionID := getStr("session_id", "flow_id", "flow", "id", "state")

	if strings.TrimSpace(authURL) == "" || strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("missing auth_url or session_id in OAuth response")
	}

	return &OAuthInitiateResponse{
		AuthURL: authURL,
		FlowID:  sessionID,
	}, nil
}

// PollOAuth polls for OAuth completion with the given flow ID
func (s *OAuthService) PollOAuth(ctx context.Context, flowID string, timeout time.Duration) (*OAuthPollResponse, error) {
	deadline := time.Now().Add(timeout)
	pollURL := strings.TrimRight(s.config.BaseURL, "/") + "/auth/poll/" + url.PathEscape(flowID)

	for time.Now().Before(deadline) {
		resp, err := s.client.Get(ctx, pollURL, nil)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			time.Sleep(2 * time.Second)
			continue
		}

		var pollResp OAuthPollResponse
		if err := json.NewDecoder(resp.Body).Decode(&pollResp); err != nil {
			resp.Body.Close()
			time.Sleep(2 * time.Second)
			continue
		}

		// Success: access_token present
		if strings.TrimSpace(pollResp.AccessToken) != "" {
			return &pollResp, nil
		}

		// Error: surface server-provided error if present
		if strings.TrimSpace(pollResp.Error) != "" {
			return nil, fmt.Errorf("OAuth error: %s", pollResp.Error)
		}

		// Still pending, continue retrying
		if strings.EqualFold(strings.TrimSpace(pollResp.Status), "pending") {
			time.Sleep(2 * time.Second)
			continue
		}

		time.Sleep(2 * time.Second)
	}

	return nil, errors.New("OAuth polling timed out")
}