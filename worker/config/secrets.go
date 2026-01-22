package config

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// WhatsAppSecret represents the structure of the secret stored in AWS Secrets Manager.
type WhatsAppSecret struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// SecretsManagerClient wraps AWS Secrets Manager operations.
type SecretsManagerClient struct {
	client *secretsmanager.Client
}

// NewSecretsManagerClient creates a new Secrets Manager client.
// It automatically loads AWS credentials from the Lambda execution role.
func NewSecretsManagerClient(ctx context.Context) (*SecretsManagerClient, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	return &SecretsManagerClient{
		client: secretsmanager.NewFromConfig(cfg),
	}, nil
}

// GetWhatsAppSecret fetches and parses WhatsApp OAuth2 credentials from Secrets Manager.
func (c *SecretsManagerClient) GetWhatsAppSecret(ctx context.Context, secretName string) (*WhatsAppSecret, error) {
	if secretName == "" {
		return nil, fmt.Errorf("secret name is empty")
	}

	// Fetch secret from Secrets Manager
	output, err := c.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, fmt.Errorf("fetch secret %q from secrets manager: %w", secretName, err)
	}

	if output.SecretString == nil {
		return nil, fmt.Errorf("secret %q has no string value (binary secrets not supported)", secretName)
	}

	// Parse JSON secret
	var secret WhatsAppSecret
	if err := json.Unmarshal([]byte(*output.SecretString), &secret); err != nil {
		return nil, fmt.Errorf("parse secret %q as JSON: %w", secretName, err)
	}

	// Validate required fields
	if secret.ClientID == "" {
		return nil, fmt.Errorf("secret %q missing required field: client_id", secretName)
	}
	if secret.ClientSecret == "" {
		return nil, fmt.Errorf("secret %q missing required field: client_secret", secretName)
	}

	return &secret, nil
}
