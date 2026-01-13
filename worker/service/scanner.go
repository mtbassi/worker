package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	goredis "github.com/redis/go-redis/v9"

	"worker-project/shared/domain"
	"worker-project/shared/redis"
)

// Scanner scans Redis for active journey states.
type Scanner struct {
	client       *redis.Client
	scanCount    int64
	pipelineSize int // How many keys to fetch in one pipeline
	logger       *slog.Logger
}

// NewScanner creates a new Redis scanner.
func NewScanner(client *redis.Client, scanCount int64, logger *slog.Logger) *Scanner {
	return &Scanner{
		client:       client,
		scanCount:    scanCount,
		pipelineSize: 100, // Fetch 100 keys at a time for optimal performance
		logger:       logger,
	}
}

// ScanAllJourneys returns all active journey states.
func (s *Scanner) ScanAllJourneys(ctx context.Context) ([]*domain.JourneyState, error) {
	return s.scan(ctx, "journey:*:*:state")
}

// ScanJourneys returns active journey states for a specific journey ID.
func (s *Scanner) ScanJourneys(ctx context.Context, journeyID string) ([]*domain.JourneyState, error) {
	pattern := fmt.Sprintf("journey:%s:*:state", journeyID)
	return s.scan(ctx, pattern)
}

// scan is a helper that performs the actual Redis SCAN operation.
func (s *Scanner) scan(ctx context.Context, pattern string) ([]*domain.JourneyState, error) {
	var journeys []*domain.JourneyState
	var cursor uint64
	keyBatch := make([]string, 0, s.pipelineSize)

	for {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("context cancelled during scan: %w", err)
		}

		// SCAN for next batch of keys
		keys, nextCursor, err := s.client.Native().Scan(ctx, cursor, pattern, s.scanCount).Result()
		if err != nil {
			return nil, fmt.Errorf("scan redis keys: %w", err)
		}

		// Accumulate keys for pipelined fetch
		keyBatch = append(keyBatch, keys...)

		// Fetch batch if we've accumulated enough keys or this is the last iteration
		if len(keyBatch) >= s.pipelineSize || nextCursor == 0 {
			batchJourneys, err := s.fetchBatch(ctx, keyBatch)
			if err != nil {
				s.logger.Warn("failed to fetch key batch", "error", err, "batch_size", len(keyBatch))
			} else {
				journeys = append(journeys, batchJourneys...)
			}
			keyBatch = keyBatch[:0] // Reset batch
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	s.logger.Debug("scan completed", "pattern", pattern, "count", len(journeys))
	return journeys, nil
}

// fetchBatch fetches multiple keys using pipeline for efficiency.
// Reduces network round trips from N+1 to Ceil(N/pipelineSize).
func (s *Scanner) fetchBatch(ctx context.Context, keys []string) ([]*domain.JourneyState, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	pipe := s.client.Native().Pipeline()

	// Queue all GET commands
	cmds := make([]*goredis.StringCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}

	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		// Don't fail entire batch on pipeline error, process individual results
		s.logger.Warn("pipeline exec encountered errors", "error", err)
	}

	// Process results
	var journeys []*domain.JourneyState
	for i, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			s.logger.Warn("failed to get key", "key", keys[i], "error", err)
			continue
		}

		var journey domain.JourneyState
		if err := json.Unmarshal([]byte(data), &journey); err != nil {
			s.logger.Warn("failed to unmarshal journey state", "key", keys[i], "error", err)
			continue
		}

		journeys = append(journeys, &journey)
	}

	return journeys, nil
}
