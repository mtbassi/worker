package service

import (
	"time"

	"worker-project/shared/domain"
	"worker-project/worker/config"
)

// EvaluationResult represents the result of evaluating a recovery rule.
type EvaluationResult struct {
	ShouldTrigger bool
	Rule          *config.RecoveryRule
	Reason        string
}

// EvaluateRecoveryRule checks if a recovery rule should trigger.
func EvaluateRecoveryRule(
	rule *config.RecoveryRule,
	globalCfg *config.GlobalConfig,
	state *domain.JourneyState,
	history *domain.RepiqueHistory,
) EvaluationResult {
	// Check if rule is enabled
	if !rule.Enabled {
		return EvaluationResult{
			ShouldTrigger: false,
			Rule:          rule,
			Reason:        "rule disabled",
		}
	}

	// Check if global max total attempts exceeded
	if history.GetTotalAttemptCount() >= globalCfg.MaxTotalAttempts {
		return EvaluationResult{
			ShouldTrigger: false,
			Rule:          rule,
			Reason:        "global max total attempts exceeded",
		}
	}

	// Check if rule-specific max attempts exceeded
	ruleAttempts := history.GetRuleAttemptCount(rule.Name)
	if ruleAttempts >= rule.MaxAttempts {
		return EvaluationResult{
			ShouldTrigger: false,
			Rule:          rule,
			Reason:        "rule max attempts exceeded",
		}
	}

	// Check minimum interval between attempts
	lastAttemptTime := history.GetLastAttemptTime(rule.Name)
	if lastAttemptTime != nil {
		minInterval := time.Duration(globalCfg.MinIntervalBetweenAttemptsMinutes) * time.Minute
		timeSinceLastAttempt := time.Since(*lastAttemptTime)

		if timeSinceLastAttempt < minInterval {
			return EvaluationResult{
				ShouldTrigger: false,
				Rule:          rule,
				Reason:        "min interval not reached",
			}
		}
	}

	// Check inactivity threshold
	inactivityThreshold := time.Duration(rule.InactiveMinutes) * time.Minute
	timeSinceLastInteraction := state.TimeSinceLastInteraction()

	if timeSinceLastInteraction < inactivityThreshold {
		return EvaluationResult{
			ShouldTrigger: false,
			Rule:          rule,
			Reason:        "inactivity threshold not reached",
		}
	}

	// All checks passed - should trigger!
	return EvaluationResult{
		ShouldTrigger: true,
		Rule:          rule,
		Reason:        "all conditions met",
	}
}

// FindTriggeredRules returns all recovery rules that should trigger for a step.
func FindTriggeredRules(
	rules []config.RecoveryRule,
	globalCfg *config.GlobalConfig,
	state *domain.JourneyState,
	history *domain.RepiqueHistory,
) []EvaluationResult {
	var results []EvaluationResult

	for i := range rules {
		result := EvaluateRecoveryRule(&rules[i], globalCfg, state, history)
		if result.ShouldTrigger {
			results = append(results, result)
		}
	}

	return results
}
