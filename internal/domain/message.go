package domain

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
