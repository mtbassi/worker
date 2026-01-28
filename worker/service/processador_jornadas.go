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

	// Evaluate recovery rules and get the single rule to execute
	// (when multiple rules trigger, selects the one with highest inactivity threshold)
	triggeredRule := FindTriggeredRule(
		stepCfg.RecoveryRules,
		&cfg.Global,
		state,
		history,
		logger,
	)

	if triggeredRule == nil {
		logger.Debug("no rules triggered")
		return nil
	}

	// Send message for the triggered rule
	if err := p.sendRecoveryMessage(ctx, cfg, state, *triggeredRule, history, logger); err != nil {
		logger.Error("failed to send recovery message",
			"rule", triggeredRule.Rule.Name,
			"error", err)
	}

	return nil
}

// sendRecoveryMessage envia uma mensagem de recuperação para o cliente.
// Fluxo seguro para evitar duplicatas:
//  1. Adquire lock de idempotência (se falhar, outro worker já está processando)
//  2. Grava no histórico ANTES de enviar (se crashar depois, não enviará de novo)
//  3. Envia a mensagem via WhatsApp
//  4. Atualiza LastInteractionAt para evitar flood
func (p *Processor) sendRecoveryMessage(
	ctx context.Context,
	cfg *config.JourneyConfig,
	state *domain.JourneyState,
	result EvaluationResult,
	history *domain.RepiqueHistory,
	logger *slog.Logger,
) error {
	rule := result.Rule
	attemptNumber := history.GetRuleAttemptCount(rule.Name) + 1

	logger.Info("regra de recuperação disparada",
		"rule", rule.Name,
		"reason", result.Reason,
		"time_since_interaction", state.TimeSinceLastInteraction(),
		"attempt_number", attemptNumber,
	)

	// PASSO 1: Adquire lock de idempotência
	// Se outro worker já pegou este lock, não envia duplicata
	acquired, err := p.store.AcquireMessageLock(ctx, state.JourneyID, state.CustomerNumber, rule.Name, attemptNumber)
	if err != nil {
		logger.Error("falha ao adquirir lock", "rule", rule.Name, "error", err)
		return err
	}
	if !acquired {
		// Outro worker já está processando esta mensagem
		logger.Warn("lock já adquirido por outro worker, ignorando duplicata",
			"rule", rule.Name,
			"attempt_number", attemptNumber)
		return nil
	}

	// PASSO 2: Grava no histórico ANTES de enviar
	// Isso garante que mesmo se crashar após enviar, não tentará de novo
	entry := domain.RepiqueEntry{
		Step:          state.Step,
		Rule:          rule.Name,
		SentAt:        time.Now(),
		TemplateUsed:  rule.Template,
		AttemptNumber: attemptNumber,
	}

	if err := p.store.AppendRepiqueHistory(ctx, state.JourneyID, state.CustomerNumber, entry); err != nil {
		logger.Error("falha ao gravar histórico", "rule", rule.Name, "error", err)
		return err
	}

	// PASSO 3: Monta e envia a mensagem
	templateRef := fmt.Sprintf("journey.%s.templates:%s:%s", cfg.Journey, state.Step, rule.Template)

	msg := domain.Message{
		CustomerNumber: state.CustomerNumber,
		TenantID:       state.TenantID,
		ContactID:      state.ContactID,
		Template:       templateRef,
		RepiqueID:      rule.Name,
		Step:           state.Step,
		Metadata:       state.Metadata,
	}

	if err := p.messenger.Send(ctx, msg); err != nil {
		// NOTA: Não desfazemos o histórico - melhor perder uma msg do que enviar duplicada
		// O lock expira em 5min e permite retry na próxima execução se necessário
		logger.Error("falha ao enviar mensagem (histórico já gravado)",
			"rule", rule.Name,
			"error", err,
			"attempt_number", attemptNumber)
		return err
	}

	// PASSO 4: Atualiza LastInteractionAt para evitar flood de mensagens
	if err := p.store.UpdateLastInteractionAt(ctx, state.JourneyID, state.CustomerNumber, time.Now()); err != nil {
		// Apenas loga - a mensagem já foi enviada com sucesso
		logger.Warn("falha ao atualizar LastInteractionAt",
			"rule", rule.Name,
			"error", err)
	}

	logger.Info("mensagem de recuperação enviada", "rule", rule.Name, "attempt", attemptNumber)
	return nil
}
