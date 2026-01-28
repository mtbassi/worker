package service

import (
	"log/slog"
	"time"

	"worker-project/shared/domain"
	"worker-project/worker/config"
)

// EvaluationResult representa o resultado da avaliação de uma regra de recuperação.
type EvaluationResult struct {
	ShouldTrigger bool                 // true se a regra deve disparar
	Rule          *config.RecoveryRule // a regra avaliada
	Reason        string               // motivo do resultado
}

// EvaluateRecoveryRule verifica se uma regra de recuperação deve disparar.
// Verifica em ordem:
//  1. Regra está habilitada?
//  2. Máximo global de tentativas excedido?
//  3. Máximo de tentativas desta regra excedido?
//  4. Intervalo mínimo entre tentativas respeitado?
//  5. Tempo de inatividade atingido?
func EvaluateRecoveryRule(
	rule *config.RecoveryRule,
	globalCfg *config.GlobalConfig,
	state *domain.JourneyState,
	history *domain.RepiqueHistory,
) EvaluationResult {
	// 1. Verifica se regra está habilitada
	if !rule.Enabled {
		return EvaluationResult{
			ShouldTrigger: false,
			Rule:          rule,
			Reason:        "regra desabilitada",
		}
	}

	// 2. Verifica se máximo global de tentativas foi excedido
	if history.GetTotalAttemptCount() >= globalCfg.MaxTotalAttempts {
		return EvaluationResult{
			ShouldTrigger: false,
			Rule:          rule,
			Reason:        "máximo global de tentativas excedido",
		}
	}

	// 3. Verifica se máximo de tentativas desta regra foi excedido
	ruleAttempts := history.GetRuleAttemptCount(rule.Name)
	if ruleAttempts >= rule.MaxAttempts {
		return EvaluationResult{
			ShouldTrigger: false,
			Rule:          rule,
			Reason:        "máximo de tentativas da regra excedido",
		}
	}

	// 4. Verifica intervalo mínimo entre tentativas
	lastAttemptTime := history.GetLastAttemptTime(rule.Name)
	if lastAttemptTime != nil {
		minInterval := time.Duration(globalCfg.MinIntervalBetweenAttemptsMinutes) * time.Minute
		timeSinceLastAttempt := time.Since(*lastAttemptTime)

		if timeSinceLastAttempt < minInterval {
			return EvaluationResult{
				ShouldTrigger: false,
				Rule:          rule,
				Reason:        "intervalo mínimo não atingido",
			}
		}
	}

	// 5. Verifica tempo de inatividade
	inactivityThreshold := time.Duration(rule.InactiveMinutes) * time.Minute
	timeSinceLastInteraction := state.TimeSinceLastInteraction()

	if timeSinceLastInteraction < inactivityThreshold {
		return EvaluationResult{
			ShouldTrigger: false,
			Rule:          rule,
			Reason:        "tempo de inatividade não atingido",
		}
	}

	// Todas as verificações passaram - deve disparar!
	return EvaluationResult{
		ShouldTrigger: true,
		Rule:          rule,
		Reason:        "todas as condições atendidas",
	}
}

// FindTriggeredRule retorna a ÚNICA regra de recuperação que deve disparar.
// Quando múltiplas regras são elegíveis, retorna a com MAIOR InactiveMinutes
// (ou seja, a que esperou mais tempo) e loga um warning sobre concorrência.
//
// Exemplo: Se regras de 10min, 20min e 30min disparam ao mesmo tempo,
// retorna apenas a de 30min e loga warning para ajustar configuração.
//
// Retorna nil se nenhuma regra deve disparar.
func FindTriggeredRule(
	rules []config.RecoveryRule,
	globalCfg *config.GlobalConfig,
	state *domain.JourneyState,
	history *domain.RepiqueHistory,
	logger *slog.Logger,
) *EvaluationResult {
	var triggeredResults []EvaluationResult

	// Avalia todas as regras
	for i := range rules {
		result := EvaluateRecoveryRule(&rules[i], globalCfg, state, history)
		if result.ShouldTrigger {
			triggeredResults = append(triggeredResults, result)
		}
	}

	if len(triggeredResults) == 0 {
		return nil
	}

	// Se múltiplas regras dispararam, loga warning e seleciona a de maior inatividade
	if len(triggeredResults) > 1 {
		ruleNames := make([]string, len(triggeredResults))
		for i, r := range triggeredResults {
			ruleNames[i] = r.Rule.Name
		}
		logger.Warn("múltiplas regras dispararam simultaneamente - selecionando a de maior inatividade",
			"triggered_rules", ruleNames,
			"customer_number", state.CustomerNumber,
			"journey_id", state.JourneyID,
			"step", state.Step,
			"recomendacao", "considere ajustar timing das regras na configuração")
	}

	// Encontra a regra com maior InactiveMinutes
	selected := triggeredResults[0]
	for _, r := range triggeredResults[1:] {
		if r.Rule.InactiveMinutes > selected.Rule.InactiveMinutes {
			selected = r
		}
	}

	return &selected
}
