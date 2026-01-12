package ports

import (
	"context"

	"worker-project/internal/domain"
)

// StateRepository handles journey state persistence.
type StateRepository interface {
	// GetJourneyState retrieves the current state of a customer's journey.
	GetJourneyState(ctx context.Context, journeyID, customerNumber string) (*domain.JourneyState, error)

	// GetRepiqueAttempts retrieves repique attempt counts for a customer's journey.
	GetRepiqueAttempts(ctx context.Context, journeyID, customerNumber string) (*domain.RepiqueAttempts, error)

	// IncrementRepiqueAttempt increments the attempt count for a specific repique.
	IncrementRepiqueAttempt(ctx context.Context, journeyID, customerNumber, repiqueID string) error

	// DeleteJourneyState removes a journey state.
	DeleteJourneyState(ctx context.Context, journeyID, customerNumber string) error
}
