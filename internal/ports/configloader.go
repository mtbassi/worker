package ports

import (
	"worker-project/internal/config"
)

// JourneyConfigLoader loads journey configurations.
type JourneyConfigLoader interface {
	// LoadJourneyConfig loads configuration for a specific journey.
	LoadJourneyConfig(journeyID string) (*config.JourneyConfig, error)
}
