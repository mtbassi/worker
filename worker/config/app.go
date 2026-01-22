package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"worker-project/shared/redis"
)

// AppConfig holds application-level configuration.
type AppConfig struct {
	Redis     redis.Config
	AppConfig AppConfigSettings
	Worker    WorkerConfig
	WhatsApp  WhatsAppConfig
}

// AppConfigSettings holds AWS AppConfig settings.
type AppConfigSettings struct {
	Endpoint      string
	ApplicationID string
	EnvironmentID string
}

// WorkerConfig holds worker-specific settings.
type WorkerConfig struct {
	ScanCount       int64
	DefaultStateTTL time.Duration
}

// WhatsAppConfig holds WhatsApp Business API configuration.
type WhatsAppConfig struct {
	APIEndpoint   string
	PhoneNumberID string
	ClientID      string
	ClientSecret  string
	STSEndpoint   string
	Timeout       time.Duration
	MaxRetries    int
	RetryDelay    time.Duration
}

// LoadFromEnv loads configuration from environment variables with sensible defaults.
func LoadFromEnv(ctx context.Context) (*AppConfig, error) {
	// Build Redis config with ElastiCache support
	redisAddr := getEnvOrDefault("REDIS_ADDR", "localhost:6379")

	// Check for ElastiCache configuration
	if elasticacheEndpoint := os.Getenv("ELASTICACHE_ENDPOINT"); elasticacheEndpoint != "" {
		redisAddr = elasticacheEndpoint
	}

	redisCfg := redis.Config{
		Addr:         redisAddr,
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	}

	// ElastiCache cluster mode
	if os.Getenv("ELASTICACHE_CLUSTER_MODE") == "true" {
		redisCfg.ClusterMode = true
	}

	// ElastiCache Sentinel configuration
	if sentinelAddrs := os.Getenv("ELASTICACHE_SENTINEL_ADDRS"); sentinelAddrs != "" {
		redisCfg.SentinelAddrs = strings.Split(sentinelAddrs, ",")
		redisCfg.MasterName = os.Getenv("ELASTICACHE_MASTER_NAME")
	}

	cfg := &AppConfig{
		Redis: redisCfg,
		AppConfig: AppConfigSettings{
			Endpoint:      getEnvOrDefault("APPCONFIG_ENDPOINT", "http://localhost:2772"),
			ApplicationID: os.Getenv("APPCONFIG_APP_ID"),
			EnvironmentID: os.Getenv("APPCONFIG_ENV_ID"),
		},
		Worker: WorkerConfig{
			ScanCount:       100,
			DefaultStateTTL: 24 * time.Hour,
		},
		WhatsApp: WhatsAppConfig{
			APIEndpoint:   getEnvOrDefault("WHATSAPP_API_ENDPOINT", "https://graph.facebook.com/v18.0"),
			PhoneNumberID: os.Getenv("WHATSAPP_PHONE_NUMBER_ID"),
			STSEndpoint:   os.Getenv("WHATSAPP_STS_ENDPOINT"),
			Timeout:       parseDurationOrDefault("WHATSAPP_TIMEOUT", 10*time.Second),
			MaxRetries:    parseIntOrDefault("WHATSAPP_MAX_RETRIES", 3),
			RetryDelay:    parseDurationOrDefault("WHATSAPP_RETRY_DELAY", 2*time.Second),
		},
	}

	// Load WhatsApp OAuth2 credentials from Secrets Manager
	secretName := os.Getenv("WHATSAPP_SECRET_NAME")
	if secretName == "" {
		return nil, fmt.Errorf("WHATSAPP_SECRET_NAME environment variable is required")
	}

	// Create Secrets Manager client
	smClient, err := NewSecretsManagerClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create secrets manager client: %w", err)
	}

	// Fetch WhatsApp secret
	whatsappSecret, err := smClient.GetWhatsAppSecret(ctx, secretName)
	if err != nil {
		return nil, fmt.Errorf("load whatsapp credentials from secrets manager: %w", err)
	}

	// Populate credentials from secret
	cfg.WhatsApp.ClientID = whatsappSecret.ClientID
	cfg.WhatsApp.ClientSecret = whatsappSecret.ClientSecret

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func parseDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultValue
}

func parseIntOrDefault(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}
