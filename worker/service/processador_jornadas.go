package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"worker-project/shared/domain"
	"worker-project/shared/redis"
	"worker-project/worker/config"
)

// Messenger is the interface for sending messages.
type Messenger interface {
	Send(ctx context.Context, msg domain.Message) error
}

// Processor handles journey processing and message sending.
type Processor struct {
	store     *redis.StateStore
	messenger Messenger
	logger    *slog.Logger
}

// NewProcessor creates a new processor with injected dependencies.
func NewProcessor(
	store *redis.StateStore,
	messenger Messenger,
	logger *slog.Logger,
) *Processor {
	return &Processor{
		store:     store,
		messenger: messenger,
		logger:    logger,
	}
}

// ProcessJourney checks a single customer journey and sends messages if needed.
func (p *Processor) ProcessJourney(ctx context.Context, cfg *config.JourneyConfig, state *domain.JourneyState) error {
	logger := p.logger.With(
		"journey_id", state.JourneyID,
		"customer_number", state.CustomerNumber,
		"step", state.Step,
	)

	// Check if journey is globally enabled
	if !cfg.Global.Enabled {
		logger.Debug("journey disabled, skipping")
		return nil
	}

	logger.Debug("processing journey")

	// Get recovery history
	history, err := p.store.GetRepiqueHistory(ctx, state.JourneyID, state.CustomerNumber)
	if err != nil {
		return &domain.JourneyError{
			JourneyID:      state.JourneyID,
			CustomerNumber: state.CustomerNumber,
			Op:             "GetRepiqueHistory",
			Err:            err,
		}
	}

	// Find the step configuration
	stepCfg := cfg.FindStepByName(state.Step)
	if stepCfg == nil {
		logger.Warn("step not found in config", "step", state.Step)
		return nil
	}

	// Evaluate all recovery rules for this step
	triggeredRules := FindTriggeredRules(
		stepCfg.RecoveryRules,
		&cfg.Global,
		state,
		history,
	)

	if len(triggeredRules) == 0 {
		logger.Debug("no rules triggered")
		return nil
	}

	// Send messages for triggered rules
	for _, result := range triggeredRules {
		if err := p.sendRecoveryMessage(ctx, cfg, state, result, history, logger); err != nil {
			logger.Error("failed to send recovery message",
				"rule", result.Rule.Name,
				"error", err)
		}
	}

	return nil
}

func (p *Processor) sendRecoveryMessage(
	ctx context.Context,
	cfg *config.JourneyConfig,
	state *domain.JourneyState,
	result EvaluationResult,
	history *domain.RepiqueHistory,
	logger *slog.Logger,
) error {
	rule := result.Rule

	logger.Info("recovery rule triggered",
		"rule", rule.Name,
		"reason", result.Reason,
		"time_since_interaction", state.TimeSinceLastInteraction(),
	)

	// Build template reference: journey.<journey-id>.templates:<step>:<template-name>
	templateRef := fmt.Sprintf("journey.%s.templates:%s:%s", cfg.Journey, state.Step, rule.Template)

	// Create message
	msg := domain.Message{
		CustomerNumber: state.CustomerNumber,
		TenantID:       state.TenantID,
		ContactID:      state.ContactID,
		Template:       templateRef,
		RepiqueID:      rule.Name, // Use rule name as repique ID
		Step:           state.Step,
		Metadata:       state.Metadata,
	}

	// Send message
	if err := p.messenger.Send(ctx, msg); err != nil {
		logger.Error("failed to send message", "rule", rule.Name, "error", err)
		return err
	}

	// Record in history
	entry := domain.RepiqueEntry{
		Step:          state.Step,
		Rule:          rule.Name,
		SentAt:        time.Now(),
		TemplateUsed:  rule.Template,
		AttemptNumber: history.GetRuleAttemptCount(rule.Name) + 1,
	}

	if err := p.store.AppendRepiqueHistory(ctx, state.JourneyID, state.CustomerNumber, entry); err != nil {
		logger.Error("failed to record history", "rule", rule.Name, "error", err)
		return err
	}

	logger.Info("recovery message sent", "rule", rule.Name, "attempt", entry.AttemptNumber)
	return nil
}
