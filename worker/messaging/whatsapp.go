package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WhatsAppConfig holds WhatsApp Business API configuration.
type WhatsAppConfig struct {
	APIEndpoint   string        // e.g., "https://graph.facebook.com/v18.0"
	PhoneNumberID string        // Your WhatsApp Business phone number ID
	ClientID      string        // OAuth2 client ID for STS
	ClientSecret  string        // OAuth2 client secret for STS
	STSEndpoint   string        // STS token endpoint URL
	Timeout       time.Duration // HTTP timeout
	MaxRetries    int           // Maximum retry attempts
	RetryDelay    time.Duration // Delay between retries
}

// WhatsAppClient sends messages via WhatsApp Business API.
type WhatsAppClient struct {
	config     WhatsAppConfig
	httpClient *http.Client
	stsClient  *STSClient
}

// NewWhatsAppClient creates a new WhatsApp API client.
func NewWhatsAppClient(config WhatsAppConfig) *WhatsAppClient {
	stsConfig := STSConfig{
		Endpoint:     config.STSEndpoint,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Timeout:      config.Timeout,
	}

	return &WhatsAppClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		stsClient: NewSTSClient(stsConfig),
	}
}

// WhatsAppMessage represents a WhatsApp message request.
type WhatsAppMessage struct {
	MessagingProduct string       `json:"messaging_product"`
	RecipientType    string       `json:"recipient_type"`
	To               string       `json:"to"`
	Type             string       `json:"type"`
	Text             *TextContent `json:"text,omitempty"`
}

// TextContent represents text message content.
type TextContent struct {
	PreviewURL bool   `json:"preview_url"`
	Body       string `json:"body"`
}

// WhatsAppResponse represents the API response.
type WhatsAppResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// WhatsAppError represents an API error response.
type WhatsAppError struct {
	ErrorInfo struct {
		Message      string `json:"message"`
		Type         string `json:"type"`
		Code         int    `json:"code"`
		ErrorSubcode int    `json:"error_subcode"`
		FBTraceID    string `json:"fbtrace_id"`
	} `json:"error"`
}

func (e *WhatsAppError) Error() string {
	return fmt.Sprintf("whatsapp api error: %s (code: %d, type: %s, trace: %s)",
		e.ErrorInfo.Message, e.ErrorInfo.Code, e.ErrorInfo.Type, e.ErrorInfo.FBTraceID)
}

// Send sends a text message via WhatsApp Business API with retries.
func (c *WhatsAppClient) Send(ctx context.Context, to, body string) (*WhatsAppResponse, error) {
	message := WhatsAppMessage{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "text",
		Text: &TextContent{
			PreviewURL: false,
			Body:       body,
		},
	}

	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retrying
			select {
			case <-time.After(c.config.RetryDelay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := c.sendRequest(ctx, message)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry on client errors (4xx)
		if whatsappErr, ok := err.(*WhatsAppError); ok {
			if whatsappErr.ErrorInfo.Code >= 400 && whatsappErr.ErrorInfo.Code < 500 {
				return nil, err
			}
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

func (c *WhatsAppClient) sendRequest(ctx context.Context, message WhatsAppMessage) (*WhatsAppResponse, error) {
	// Fetch access token from STS
	accessToken, err := c.stsClient.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	// Marshal request body
	body, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/%s/messages", c.config.APIEndpoint, c.config.PhoneNumberID)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var whatsappErr WhatsAppError
		if err := json.Unmarshal(respBody, &whatsappErr); err != nil {
			return nil, fmt.Errorf("whatsapp api error (status %d): %s", resp.StatusCode, string(respBody))
		}
		return nil, &whatsappErr
	}

	// Parse success response
	var whatsappResp WhatsAppResponse
	if err := json.Unmarshal(respBody, &whatsappResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &whatsappResp, nil
}
