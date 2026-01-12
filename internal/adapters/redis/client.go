package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"worker-project/internal/config"
)

// Key patterns for Redis keys.
const (
	KeyPatternJourneyState    = "journey:%s:%s:state"
	KeyPatternJourneyRepiques = "journey:%s:%s:repiques"
)

// Client wraps a Redis client with configuration.
type Client struct {
	native *redis.Client
}

// NewClient creates a new Redis client with the given configuration.
func NewClient(cfg config.RedisConfig) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Client{native: rdb}, nil
}

// Native returns the underlying redis.Client for advanced operations.
func (c *Client) Native() *redis.Client {
	return c.native
}

// Get retrieves a value by key.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.native.Get(ctx, key).Result()
}

// Set stores a value with an expiration.
func (c *Client) Set(ctx context.Context, key, value string, expiration time.Duration) error {
	return c.native.Set(ctx, key, value, expiration).Err()
}

// Del deletes keys.
func (c *Client) Del(ctx context.Context, keys ...string) error {
	return c.native.Del(ctx, keys...).Err()
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.native.Close()
}
