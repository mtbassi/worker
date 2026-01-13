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
// Deprecated: Use RepiqueHistory for detailed tracking.
type RepiqueAttempts struct {
	Attempts map[string]int `json:"attempts"` // key: repique_id, value: attempt count
}

// NewRepiqueAttempts creates a new RepiqueAttempts with an initialized map.
// Deprecated: Use RepiqueHistory for detailed tracking.
func NewRepiqueAttempts() *RepiqueAttempts {
	return &RepiqueAttempts{
		Attempts: make(map[string]int),
	}
}

// RepiqueHistory tracks execution history for recovery rules.
type RepiqueHistory struct {
	Entries []RepiqueEntry `json:"entries"`
}

// RepiqueEntry represents a single recovery message execution.
type RepiqueEntry struct {
	Step          string    `json:"step"`
	Rule          string    `json:"rule"`
	SentAt        time.Time `json:"sent_at"`
	TemplateUsed  string    `json:"template_used"`
	AttemptNumber int       `json:"attempt_number"`
}

// GetRuleAttemptCount returns the number of attempts for a specific rule.
func (h *RepiqueHistory) GetRuleAttemptCount(ruleName string) int {
	count := 0
	for _, entry := range h.Entries {
		if entry.Rule == ruleName {
			count++
		}
	}
	return count
}

// GetLastAttemptTime returns the timestamp of the last attempt for a specific rule.
// Returns nil if no attempts have been made for this rule.
func (h *RepiqueHistory) GetLastAttemptTime(ruleName string) *time.Time {
	var lastTime *time.Time
	for _, entry := range h.Entries {
		if entry.Rule == ruleName {
			if lastTime == nil || entry.SentAt.After(*lastTime) {
				lastTime = &entry.SentAt
			}
		}
	}
	return lastTime
}

// GetTotalAttemptCount returns the total number of recovery attempts across all rules.
func (h *RepiqueHistory) GetTotalAttemptCount() int {
	return len(h.Entries)
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

// Message represents a message to be sent to a customer.
type Message struct {
	CustomerNumber string         `json:"customer_number"`
	TenantID       string         `json:"tenant_id"`
	ContactID      string         `json:"contact_id"`
	Template       string         `json:"template"`
	RepiqueID      string         `json:"repique_id"`
	Step           string         `json:"step,omitempty"`
	Metadata       map[string]any `json:"metadata"`
}

// NewMessage creates a new Message from journey state and repique info.
func NewMessage(state *JourneyState, repiqueID, template, step string) Message {
	return Message{
		CustomerNumber: state.CustomerNumber,
		TenantID:       state.TenantID,
		ContactID:      state.ContactID,
		Template:       template,
		RepiqueID:      repiqueID,
		Step:           step,
		Metadata:       state.Metadata,
	}
}
