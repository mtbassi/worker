package ports

import (
	"context"

	"worker-project/internal/domain"
)

// JourneyScanner scans for active journeys in the data store.
type JourneyScanner interface {
	// ScanAllJourneys returns all active journey states.
	ScanAllJourneys(ctx context.Context) ([]*domain.JourneyState, error)

	// ScanJourneys returns active journey states for a specific journey ID.
	ScanJourneys(ctx context.Context, journeyID string) ([]*domain.JourneyState, error)
}
