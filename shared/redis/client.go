package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis connection settings.
type Config struct {
	Addr         string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int

	// ElastiCache-specific settings
	ClusterMode    bool
	SentinelAddrs  []string
	MasterName     string
	RouteByLatency bool
	RouteRandomly  bool
}

// Client wraps a Redis client with configuration.
type Client struct {
	native redis.UniversalClient
}

// NewClient creates a new Redis client with the given configuration.
// Supports single-node, cluster mode, and sentinel (failover) configurations.
func NewClient(cfg Config) (*Client, error) {
	var rdb redis.UniversalClient

	if cfg.ClusterMode {
		// ElastiCache cluster mode enabled
		rdb = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:          []string{cfg.Addr},
			Password:       cfg.Password,
			DialTimeout:    cfg.DialTimeout,
			ReadTimeout:    cfg.ReadTimeout,
			WriteTimeout:   cfg.WriteTimeout,
			PoolSize:       cfg.PoolSize,
			MinIdleConns:   cfg.MinIdleConns,
			RouteByLatency: cfg.RouteByLatency,
			RouteRandomly:  cfg.RouteRandomly,
		})
	} else if len(cfg.SentinelAddrs) > 0 {
		// ElastiCache with replication groups (Sentinel/Failover)
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.MasterName,
			SentinelAddrs: cfg.SentinelAddrs,
			Password:      cfg.Password,
			DB:            cfg.DB,
			DialTimeout:   cfg.DialTimeout,
			ReadTimeout:   cfg.ReadTimeout,
			WriteTimeout:  cfg.WriteTimeout,
			PoolSize:      cfg.PoolSize,
			MinIdleConns:  cfg.MinIdleConns,
		})
	} else {
		// Standard single-node client (for local dev or simple deployments)
		rdb = redis.NewClient(&redis.Options{
			Addr:         cfg.Addr,
			Password:     cfg.Password,
			DB:           cfg.DB,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
		})
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &Client{native: rdb}, nil
}

// Native returns the underlying redis.UniversalClient for advanced operations.
func (c *Client) Native() redis.UniversalClient {
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

// Close fecha a conexão com o Redis.
func (c *Client) Close() error {
	return c.native.Close()
}

// SetNX define um valor APENAS se a chave NÃO existir.
// Usado para locks de idempotência - garante que apenas um worker processa cada mensagem.
// Retorna true se criou a chave (você é o primeiro), false se já existia.
func (c *Client) SetNX(ctx context.Context, key, value string, expiration time.Duration) (bool, error) {
	return c.native.SetNX(ctx, key, value, expiration).Result()
}
