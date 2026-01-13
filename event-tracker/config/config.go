package config

import (
	"os"
	"time"

	"worker-project/shared/redis"
)

// Config holds Lambda 1 configuration.
type Config struct {
	Redis           redis.Config
	DefaultStateTTL time.Duration
}

// LoadFromEnv loads configuration from environment variables with sensible defaults.
func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		Redis: redis.Config{
			Addr:         getEnvOrDefault("REDIS_ADDR", "localhost:6379"),
			Password:     os.Getenv("REDIS_PASSWORD"),
			DB:           0,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolSize:     10,
			MinIdleConns: 2,
		},
		DefaultStateTTL: parseDuration(getEnvOrDefault("STATE_TTL", "24h")),
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}
