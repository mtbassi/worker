package domain

import "time"

// JourneyState represents the current state of a customer's journey.
type JourneyState struct {
	JourneyID         string         `json:"journey_id"`
	Step              string         `json:"step"`
	CustomerNumber    string         `json:"customer_number"`
	TenantID          string         `json:"tenant_id"`
	ContactID         string         `json:"contact_id"`
	LastInteractionAt time.Time      `json:"last_interaction_at"`
	StepStartedAt     time.Time      `json:"step_started_at"`
	JourneyStartedAt  time.Time      `json:"journey_started_at"`
	Metadata          map[string]any `json:"metadata"`
}

// RepiqueAttempts tracks how many times each repique has been sent.
type RepiqueAttempts struct {
	Attempts map[string]int `json:"attempts"` // key: repique_id, value: attempt count
}

// NewRepiqueAttempts creates a new RepiqueAttempts with an initialized map.
func NewRepiqueAttempts() *RepiqueAttempts {
	return &RepiqueAttempts{
		Attempts: make(map[string]int),
	}
}

// IsExpired checks if the journey has expired based on max inactive time.
func (s *JourneyState) IsExpired(maxInactiveTime time.Duration) bool {
	return time.Since(s.LastInteractionAt) >= maxInactiveTime
}

// TimeInStep returns how long the customer has been in the current step.
func (s *JourneyState) TimeInStep() time.Duration {
	return time.Since(s.StepStartedAt)
}

// TimeUntilExpiry returns how much time is left before the journey expires.
func (s *JourneyState) TimeUntilExpiry(maxInactiveTime time.Duration) time.Duration {
	elapsed := time.Since(s.LastInteractionAt)
	remaining := maxInactiveTime - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// TimeSinceLastInteraction returns time elapsed since the last interaction.
func (s *JourneyState) TimeSinceLastInteraction() time.Duration {
	return time.Since(s.LastInteractionAt)
}
