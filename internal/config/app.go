package config

import (
	"os"
	"time"
)

// AppConfig holds application-level configuration.
type AppConfig struct {
	Redis     RedisConfig
	AppConfig AppConfigSettings
	Worker    WorkerConfig
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
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

// LoadFromEnv loads configuration from environment variables with sensible defaults.
func LoadFromEnv() (*AppConfig, error) {
	cfg := &AppConfig{
		Redis: RedisConfig{
			Addr:         getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
			Password:     os.Getenv("REDIS_PASSWORD"),
			DB:           0,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolSize:     10,
			MinIdleConns: 2,
		},
		AppConfig: AppConfigSettings{
			Endpoint:      getEnvOrDefault("APPCONFIG_ENDPOINT", "http://localhost:2772"),
			ApplicationID: os.Getenv("APPCONFIG_APP_ID"),
			EnvironmentID: os.Getenv("APPCONFIG_ENV_ID"),
		},
		Worker: WorkerConfig{
			ScanCount:       100,
			DefaultStateTTL: 24 * time.Hour,
		},
	}

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
