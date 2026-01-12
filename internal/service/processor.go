package service

import (
	"context"
	"log/slog"

	"worker-project/internal/config"
	"worker-project/internal/domain"
	"worker-project/internal/ports"
)

// Processor handles journey processing and message sending.
type Processor struct {
	repository ports.StateRepository
	messenger  ports.Messenger
	logger     *slog.Logger
}

// NewProcessor creates a new processor with injected dependencies.
func NewProcessor(
	repository ports.StateRepository,
	messenger ports.Messenger,
	logger *slog.Logger,
) *Processor {
	return &Processor{
		repository: repository,
		messenger:  messenger,
		logger:     logger,
	}
}

// ProcessJourney checks a single customer journey and sends messages if needed.
func (p *Processor) ProcessJourney(ctx context.Context, cfg *config.JourneyConfig, state *domain.JourneyState) error {
	logger := p.logger.With(
		"journey_id", state.JourneyID,
		"customer_number", state.CustomerNumber,
		"step", state.Step,
	)

	logger.Debug("processing journey")

	attempts, err := p.repository.GetRepiqueAttempts(ctx, state.JourneyID, state.CustomerNumber)
	if err != nil {
		return &domain.JourneyError{
			JourneyID:      state.JourneyID,
			CustomerNumber: state.CustomerNumber,
			Op:             "GetRepiqueAttempts",
			Err:            err,
		}
	}

	maxInactiveTime := cfg.Settings.MaxInactiveTime.ToDuration()

	// Check if journey has expired
	if state.IsExpired(maxInactiveTime) {
		return p.handleExpiredJourney(ctx, cfg, state, attempts, logger)
	}

	// Process lifecycle repiques
	if err := p.processLifecycleRepiques(ctx, cfg, state, attempts, logger); err != nil {
		logger.Error("error processing lifecycle repiques", "error", err)
	}

	// Process step repiques
	if err := p.processStepRepiques(ctx, cfg, state, attempts, logger); err != nil {
		logger.Error("error processing step repiques", "error", err)
	}

	return nil
}

func (p *Processor) handleExpiredJourney(
	ctx context.Context,
	cfg *config.JourneyConfig,
	state *domain.JourneyState,
	attempts *domain.RepiqueAttempts,
	logger *slog.Logger,
) error {
	logger.Info("journey expired")

	maxInactiveTime := cfg.Settings.MaxInactiveTime.ToDuration()

	for i := range cfg.Settings.LifecycleRepiques {
		repique := &cfg.Settings.LifecycleRepiques[i]

		result := EvaluateLifecycleRepique(repique, attempts, state, maxInactiveTime)
		if !result.ShouldTrigger {
			continue
		}

		if repique.Action.Template != "" {
			msg := domain.NewMessage(state, repique.ID, repique.Action.Template, "")

			if err := p.messenger.Send(ctx, msg); err != nil {
				logger.Error("failed to send on_expire message", "repique_id", repique.ID, "error", err)
				continue
			}

			if err := p.repository.IncrementRepiqueAttempt(ctx, state.JourneyID, state.CustomerNumber, repique.ID); err != nil {
				logger.Error("failed to increment repique attempt", "repique_id", repique.ID, "error", err)
			}

			logger.Info("sent on_expire message", "repique_id", repique.ID)
		}

		if repique.Action.EndJourney {
			logger.Info("ending journey")
		}
	}

	return nil
}

func (p *Processor) processLifecycleRepiques(
	ctx context.Context,
	cfg *config.JourneyConfig,
	state *domain.JourneyState,
	attempts *domain.RepiqueAttempts,
	logger *slog.Logger,
) error {
	maxInactiveTime := cfg.Settings.MaxInactiveTime.ToDuration()

	triggered := FindTriggeredLifecycleRepiques(
		cfg.Settings.LifecycleRepiques,
		attempts,
		state,
		maxInactiveTime,
	)

	for _, result := range triggered {
		repique := result.Repique

		if repique.Action.Template == "" {
			continue
		}

		logger.Info("lifecycle repique triggered",
			"repique_id", repique.ID,
			"reason", result.Reason,
			"time_until_expiry", state.TimeUntilExpiry(maxInactiveTime),
		)

		msg := domain.NewMessage(state, repique.ID, repique.Action.Template, "")

		if err := p.messenger.Send(ctx, msg); err != nil {
			logger.Error("failed to send lifecycle message", "repique_id", repique.ID, "error", err)
			continue
		}

		if err := p.repository.IncrementRepiqueAttempt(ctx, state.JourneyID, state.CustomerNumber, repique.ID); err != nil {
			logger.Error("failed to increment repique attempt", "repique_id", repique.ID, "error", err)
		}
	}

	return nil
}

func (p *Processor) processStepRepiques(
	ctx context.Context,
	cfg *config.JourneyConfig,
	state *domain.JourneyState,
	attempts *domain.RepiqueAttempts,
	logger *slog.Logger,
) error {
	step := cfg.FindStep(state.Step)
	if step == nil {
		logger.Warn("step not found in config", "step", state.Step)
		return nil
	}

	triggered := FindTriggeredStepRepiques(step.Repiques, attempts, state)

	for _, result := range triggered {
		repique := result.Repique

		if repique.Action.Template == "" {
			continue
		}

		logger.Info("step repique triggered",
			"repique_id", repique.ID,
			"reason", result.Reason,
			"time_in_step", state.TimeInStep(),
		)

		msg := domain.NewMessage(state, repique.ID, repique.Action.Template, state.Step)

		if err := p.messenger.Send(ctx, msg); err != nil {
			logger.Error("failed to send step message", "repique_id", repique.ID, "error", err)
			continue
		}

		if err := p.repository.IncrementRepiqueAttempt(ctx, state.JourneyID, state.CustomerNumber, repique.ID); err != nil {
			logger.Error("failed to increment repique attempt", "repique_id", repique.ID, "error", err)
		}
	}

	return nil
}
