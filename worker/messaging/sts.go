package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// STSConfig holds STS authentication configuration.
type STSConfig struct {
	Endpoint     string
	ClientID     string
	ClientSecret string
	Timeout      time.Duration
}

// STSTokenResponse represents the OAuth2 token response.
type STSTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// STSTokenRequest represents the OAuth2 client credentials request.
type STSTokenRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	GrantType    string `json:"grant_type"`
}

// STSClient manages OAuth2 token acquisition and caching.
type STSClient struct {
	config     STSConfig
	httpClient *http.Client
	token      string
	expiresAt  time.Time
	mu         sync.RWMutex
}

// NewSTSClient creates a new STS client with token caching.
func NewSTSClient(config STSConfig) *STSClient {
	return &STSClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// GetToken returns a valid access token, fetching a new one if necessary.
// Tokens are cached and refreshed when they're within 60 seconds of expiration.
func (c *STSClient) GetToken(ctx context.Context) (string, error) {
	// Check if we have a valid cached token (with 60 second buffer)
	c.mu.RLock()
	if time.Now().Before(c.expiresAt.Add(-60 * time.Second)) {
		token := c.token
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	// Need to fetch a new token
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have fetched)
	if time.Now().Before(c.expiresAt.Add(-60 * time.Second)) {
		return c.token, nil
	}

	// Fetch new token from STS
	token, expiresIn, err := c.fetchToken(ctx)
	if err != nil {
		return "", fmt.Errorf("fetch token from STS: %w", err)
	}

	// Update cache
	c.token = token
	c.expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)

	return token, nil
}

// fetchToken makes the HTTP request to the STS endpoint.
func (c *STSClient) fetchToken(ctx context.Context) (string, int, error) {
	// Build request body
	reqBody := STSTokenRequest{
		ClientID:     c.config.ClientID,
		ClientSecret: c.config.ClientSecret,
		GrantType:    "client_credentials",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("STS error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var tokenResp STSTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", 0, fmt.Errorf("unmarshal response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", 0, fmt.Errorf("empty access token in response")
	}

	return tokenResp.AccessToken, tokenResp.ExpiresIn, nil
}
