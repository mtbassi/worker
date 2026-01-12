package service

import (
	"time"

	"worker-project/internal/config"
	"worker-project/internal/domain"
)

// EvaluationResult represents the result of evaluating a repique rule.
type EvaluationResult struct {
	ShouldTrigger bool
	Repique       *config.Repique
	Reason        string
}

// EvaluateLifecycleRepique checks if a lifecycle repique should trigger.
func EvaluateLifecycleRepique(
	repique *config.Repique,
	attempts *domain.RepiqueAttempts,
	state *domain.JourneyState,
	maxInactiveTime time.Duration,
) EvaluationResult {
	// Check if max attempts reached
	if attempts.Attempts[repique.ID] >= repique.MaxAttempts {
		return EvaluationResult{
			ShouldTrigger: false,
			Repique:       repique,
			Reason:        "max attempts reached",
		}
	}

	// Check on_expire trigger
	if repique.Trigger.OnExpire && state.IsExpired(maxInactiveTime) {
		return EvaluationResult{
			ShouldTrigger: true,
			Repique:       repique,
			Reason:        "journey expired",
		}
	}

	// Check before_expire trigger
	if repique.Trigger.BeforeExpire != nil {
		triggerTime := repique.Trigger.BeforeExpire.ToDuration()
		timeUntilExpiry := state.TimeUntilExpiry(maxInactiveTime)

		if timeUntilExpiry <= triggerTime && timeUntilExpiry > 0 {
			return EvaluationResult{
				ShouldTrigger: true,
				Repique:       repique,
				Reason:        "before expiry window reached",
			}
		}
	}

	return EvaluationResult{
		ShouldTrigger: false,
		Repique:       repique,
		Reason:        "conditions not met",
	}
}

// EvaluateStepRepique checks if a step repique should trigger.
func EvaluateStepRepique(
	repique *config.Repique,
	attempts *domain.RepiqueAttempts,
	state *domain.JourneyState,
) EvaluationResult {
	// Check if max attempts reached
	if attempts.Attempts[repique.ID] >= repique.MaxAttempts {
		return EvaluationResult{
			ShouldTrigger: false,
			Repique:       repique,
			Reason:        "max attempts reached",
		}
	}

	// Check time_in_step condition
	if repique.Condition.TimeInStep != nil {
		requiredTime := time.Duration(repique.Condition.TimeInStep.GteMinutes) * time.Minute
		timeInStep := state.TimeInStep()

		if timeInStep >= requiredTime {
			return EvaluationResult{
				ShouldTrigger: true,
				Repique:       repique,
				Reason:        "time in step threshold reached",
			}
		}
	}

	return EvaluationResult{
		ShouldTrigger: false,
		Repique:       repique,
		Reason:        "conditions not met",
	}
}

// FindTriggeredLifecycleRepiques returns all lifecycle repiques that should trigger.
func FindTriggeredLifecycleRepiques(
	repiques []config.Repique,
	attempts *domain.RepiqueAttempts,
	state *domain.JourneyState,
	maxInactiveTime time.Duration,
) []EvaluationResult {
	var results []EvaluationResult
	for i := range repiques {
		result := EvaluateLifecycleRepique(&repiques[i], attempts, state, maxInactiveTime)
		if result.ShouldTrigger {
			results = append(results, result)
		}
	}
	return results
}

// FindTriggeredStepRepiques returns all step repiques that should trigger.
func FindTriggeredStepRepiques(
	repiques []config.Repique,
	attempts *domain.RepiqueAttempts,
	state *domain.JourneyState,
) []EvaluationResult {
	var results []EvaluationResult
	for i := range repiques {
		result := EvaluateStepRepique(&repiques[i], attempts, state)
		if result.ShouldTrigger {
			results = append(results, result)
		}
	}
	return results
}
