package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"worker-project/internal/domain"
)

// Scanner implements ports.JourneyScanner using Redis.
type Scanner struct {
	client    *Client
	scanCount int64
	logger    *slog.Logger
}

// NewScanner creates a new Redis scanner.
func NewScanner(client *Client, scanCount int64, logger *slog.Logger) *Scanner {
	return &Scanner{
		client:    client,
		scanCount: scanCount,
		logger:    logger,
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

	for {
		keys, nextCursor, err := s.client.Native().Scan(ctx, cursor, pattern, s.scanCount).Result()
		if err != nil {
			return nil, fmt.Errorf("scan redis keys: %w", err)
		}

		for _, key := range keys {
			data, err := s.client.Get(ctx, key)
			if err != nil {
				s.logger.Warn("failed to get key", "key", key, "error", err)
				continue
			}

			var journey domain.JourneyState
			if err := json.Unmarshal([]byte(data), &journey); err != nil {
				s.logger.Warn("failed to unmarshal journey state", "key", key, "error", err)
				continue
			}

			journeys = append(journeys, &journey)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	s.logger.Debug("scan completed", "pattern", pattern, "count", len(journeys))
	return journeys, nil
}
