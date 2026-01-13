package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"worker-project/shared/domain"
)

// StateStore handles persistence of journey states and repique attempts.
type StateStore struct {
	client *Client
	ttl    time.Duration
}

// NewStateStore creates a new state store with the given TTL.
func NewStateStore(client *Client, ttl time.Duration) *StateStore {
	return &StateStore{
		client: client,
		ttl:    ttl,
	}
}

// SaveJourneyState stores a journey state in Redis with TTL.
// Used by Lambda 1 (Event Tracker) to persist customer journey state.
func (s *StateStore) SaveJourneyState(ctx context.Context, state *domain.JourneyState) error {
	key := fmt.Sprintf(KeyPatternJourneyState, state.JourneyID, state.CustomerNumber)

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := s.client.Set(ctx, key, string(data), s.ttl); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	return nil
}

// GetJourneyState retrieves the current state of a customer's journey.
// Used by both Lambda 1 and Lambda 2.
func (s *StateStore) GetJourneyState(ctx context.Context, journeyID, customerNumber string) (*domain.JourneyState, error) {
	key := fmt.Sprintf(KeyPatternJourneyState, journeyID, customerNumber)

	data, err := s.client.Get(ctx, key)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get journey state: %w", err)
	}

	var state domain.JourneyState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, fmt.Errorf("unmarshal journey state: %w", err)
	}

	return &state, nil
}

// DeleteJourneyState removes a journey state from Redis.
// Used by Lambda 1 when a journey is finished.
func (s *StateStore) DeleteJourneyState(ctx context.Context, journeyID, customerNumber string) error {
	key := fmt.Sprintf(KeyPatternJourneyState, journeyID, customerNumber)
	if err := s.client.Del(ctx, key); err != nil {
		return fmt.Errorf("delete journey state: %w", err)
	}
	return nil
}

// GetRepiqueAttempts retrieves repique attempt counts for a customer's journey.
// Deprecated: Use GetRepiqueHistory for detailed tracking.
// Used by Lambda 2 (Recovery Message Sender).
func (s *StateStore) GetRepiqueAttempts(ctx context.Context, journeyID, customerNumber string) (*domain.RepiqueAttempts, error) {
	key := fmt.Sprintf(KeyPatternJourneyRepiques, journeyID, customerNumber)

	data, err := s.client.Get(ctx, key)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return domain.NewRepiqueAttempts(), nil
		}
		return nil, fmt.Errorf("get repique attempts: %w", err)
	}

	var attempts domain.RepiqueAttempts
	if err := json.Unmarshal([]byte(data), &attempts); err != nil {
		return nil, fmt.Errorf("unmarshal repique attempts: %w", err)
	}

	if attempts.Attempts == nil {
		attempts.Attempts = make(map[string]int)
	}

	return &attempts, nil
}

// IncrementRepiqueAttempt increments the attempt count for a specific repique.
// Deprecated: Use AppendRepiqueHistory for detailed tracking.
// Used by Lambda 2 after sending a recovery message.
func (s *StateStore) IncrementRepiqueAttempt(ctx context.Context, journeyID, customerNumber, repiqueID string) error {
	attempts, err := s.GetRepiqueAttempts(ctx, journeyID, customerNumber)
	if err != nil {
		return err
	}

	attempts.Attempts[repiqueID]++

	data, err := json.Marshal(attempts)
	if err != nil {
		return fmt.Errorf("marshal repique attempts: %w", err)
	}

	key := fmt.Sprintf(KeyPatternJourneyRepiques, journeyID, customerNumber)
	if err := s.client.Set(ctx, key, string(data), s.ttl); err != nil {
		return fmt.Errorf("save repique attempts: %w", err)
	}

	return nil
}

// GetRepiqueHistory retrieves the recovery history for a customer's journey.
// Returns an empty history if no entries exist.
// Used by Lambda 2 (Recovery Message Sender).
func (s *StateStore) GetRepiqueHistory(ctx context.Context, journeyID, customerNumber string) (*domain.RepiqueHistory, error) {
	key := fmt.Sprintf(KeyPatternJourneyRepiques, journeyID, customerNumber)

	data, err := s.client.Get(ctx, key)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &domain.RepiqueHistory{Entries: []domain.RepiqueEntry{}}, nil
		}
		return nil, fmt.Errorf("get repique history: %w", err)
	}

	var history domain.RepiqueHistory
	if err := json.Unmarshal([]byte(data), &history); err != nil {
		return nil, fmt.Errorf("unmarshal repique history: %w", err)
	}

	if history.Entries == nil {
		history.Entries = []domain.RepiqueEntry{}
	}

	return &history, nil
}

// AppendRepiqueHistory appends a new entry to the recovery history.
// Used by Lambda 2 after sending a recovery message.
func (s *StateStore) AppendRepiqueHistory(ctx context.Context, journeyID, customerNumber string, entry domain.RepiqueEntry) error {
	history, err := s.GetRepiqueHistory(ctx, journeyID, customerNumber)
	if err != nil {
		return err
	}

	history.Entries = append(history.Entries, entry)

	data, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("marshal repique history: %w", err)
	}

	key := fmt.Sprintf(KeyPatternJourneyRepiques, journeyID, customerNumber)
	if err := s.client.Set(ctx, key, string(data), s.ttl); err != nil {
		return fmt.Errorf("save repique history: %w", err)
	}

	return nil
}
