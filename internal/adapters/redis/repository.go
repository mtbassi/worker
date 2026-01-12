package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"worker-project/internal/domain"
)

// Repository implements ports.StateRepository using Redis.
type Repository struct {
	client *Client
	ttl    time.Duration
}

// NewRepository creates a new Redis repository.
func NewRepository(client *Client, ttl time.Duration) *Repository {
	return &Repository{
		client: client,
		ttl:    ttl,
	}
}

// GetJourneyState retrieves the current state of a customer's journey.
func (r *Repository) GetJourneyState(ctx context.Context, journeyID, customerNumber string) (*domain.JourneyState, error) {
	key := fmt.Sprintf(KeyPatternJourneyState, journeyID, customerNumber)

	data, err := r.client.Get(ctx, key)
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

// GetRepiqueAttempts retrieves repique attempt counts for a customer's journey.
func (r *Repository) GetRepiqueAttempts(ctx context.Context, journeyID, customerNumber string) (*domain.RepiqueAttempts, error) {
	key := fmt.Sprintf(KeyPatternJourneyRepiques, journeyID, customerNumber)

	data, err := r.client.Get(ctx, key)
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
func (r *Repository) IncrementRepiqueAttempt(ctx context.Context, journeyID, customerNumber, repiqueID string) error {
	attempts, err := r.GetRepiqueAttempts(ctx, journeyID, customerNumber)
	if err != nil {
		return err
	}

	attempts.Attempts[repiqueID]++

	data, err := json.Marshal(attempts)
	if err != nil {
		return fmt.Errorf("marshal repique attempts: %w", err)
	}

	key := fmt.Sprintf(KeyPatternJourneyRepiques, journeyID, customerNumber)
	if err := r.client.Set(ctx, key, string(data), r.ttl); err != nil {
		return fmt.Errorf("save repique attempts: %w", err)
	}

	return nil
}

// DeleteJourneyState removes a journey state.
func (r *Repository) DeleteJourneyState(ctx context.Context, journeyID, customerNumber string) error {
	key := fmt.Sprintf(KeyPatternJourneyState, journeyID, customerNumber)
	if err := r.client.Del(ctx, key); err != nil {
		return fmt.Errorf("delete journey state: %w", err)
	}
	return nil
}
