package models

import "errors"

// EventRequest represents a journey event from the client.
type EventRequest struct {
	JourneyID      string                 `json:"journey_id"`
	Step           string                 `json:"step"`
	CustomerNumber string                 `json:"customer_number"`
	TenantID       string                 `json:"tenant_id"`
	ContactID      string                 `json:"contact_id"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// Validate checks if the event request is valid.
func (r *EventRequest) Validate() error {
	if r.JourneyID == "" {
		return errors.New("journey_id is required")
	}
	if r.Step == "" {
		return errors.New("step is required")
	}
	if r.CustomerNumber == "" {
		return errors.New("customer_number is required")
	}
	if r.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if r.ContactID == "" {
		return errors.New("contact_id is required")
	}
	return nil
}

// FinishRequest represents a request to finish a journey.
type FinishRequest struct {
	JourneyID      string `json:"journey_id"`
	CustomerNumber string `json:"customer_number"`
}

// Validate checks if the finish request is valid.
func (r *FinishRequest) Validate() error {
	if r.JourneyID == "" {
		return errors.New("journey_id is required")
	}
	if r.CustomerNumber == "" {
		return errors.New("customer_number is required")
	}
	return nil
}
