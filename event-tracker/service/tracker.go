package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"worker-project/event-tracker/models"
	"worker-project/shared/domain"
	"worker-project/shared/redis"
)

// Tracker handles journey event tracking business logic.
type Tracker struct {
	store  *redis.StateStore
	logger *slog.Logger
}

// NewTracker creates a new tracker service.
func NewTracker(store *redis.StateStore, logger *slog.Logger) *Tracker {
	return &Tracker{
		store:  store,
		logger: logger,
	}
}

// RecordEvent records a customer event in their journey.
// Server-side timestamp logic:
// - Always sets LastInteractionAt to current time
// - Preserves JourneyStartedAt if journey already exists
// - Preserves StepStartedAt if customer remains in same step
// - Updates StepStartedAt if customer moves to new step
func (t *Tracker) RecordEvent(ctx context.Context, req *models.EventRequest) error {
	now := time.Now()

	// Try to get existing state
	existing, err := t.store.GetJourneyState(ctx, req.JourneyID, req.CustomerNumber)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		t.logger.Error("failed to get existing state",
			"journey_id", req.JourneyID,
			"customer", req.CustomerNumber,
			"error", err)
		return err
	}

	// Build new state
	state := &domain.JourneyState{
		JourneyID:         req.JourneyID,
		Step:              req.Step,
		CustomerNumber:    req.CustomerNumber,
		TenantID:          req.TenantID,
		ContactID:         req.ContactID,
		LastInteractionAt: now, // ALWAYS set server timestamp
		StepStartedAt:     now,
		JourneyStartedAt:  now,
		Metadata:          req.Metadata,
	}

	// Preserve timestamps from existing state
	if existing != nil {
		// Journey started time never changes
		state.JourneyStartedAt = existing.JourneyStartedAt

		// If same step, preserve StepStartedAt
		if existing.Step == req.Step {
			state.StepStartedAt = existing.StepStartedAt
		}
		// If different step, StepStartedAt is already set to now
	}

	// Save state with TTL
	if err := t.store.SaveJourneyState(ctx, state); err != nil {
		t.logger.Error("failed to save state",
			"journey_id", req.JourneyID,
			"customer", req.CustomerNumber,
			"error", err)
		return err
	}

	t.logger.Info("event recorded",
		"journey_id", req.JourneyID,
		"step", req.Step,
		"customer", req.CustomerNumber,
		"is_new_journey", existing == nil,
		"step_changed", existing != nil && existing.Step != req.Step)

	return nil
}

// FinishJourney marks a journey as complete by deleting its state.
// This prevents Lambda 2 from sending any further recovery messages.
func (t *Tracker) FinishJourney(ctx context.Context, req *models.FinishRequest) error {
	if err := t.store.DeleteJourneyState(ctx, req.JourneyID, req.CustomerNumber); err != nil {
		t.logger.Error("failed to delete state",
			"journey_id", req.JourneyID,
			"customer", req.CustomerNumber,
			"error", err)
		return err
	}

	t.logger.Info("journey finished",
		"journey_id", req.JourneyID,
		"customer", req.CustomerNumber)

	return nil
}
