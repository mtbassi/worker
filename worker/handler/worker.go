package handler

import (
	"context"
	"log/slog"

	"worker-project/shared/domain"
	"worker-project/worker/appconfig"
	"worker-project/worker/service"
)

// Run executes the worker's main processing loop.
func Run(
	ctx context.Context,
	scanner *service.Scanner,
	configLoader *appconfig.Loader,
	processor *service.Processor,
	logger *slog.Logger,
) error {
	logger.Info("starting worker run")

	// Scan all active journeys
	journeys, err := scanner.ScanAllJourneys(ctx)
	if err != nil {
		logger.Error("failed to scan journeys", "error", err)
		return err
	}

	logger.Info("scanned journeys", "count", len(journeys))

	// Group journeys by journey_id for batch processing
	grouped := groupByJourneyID(journeys)

	// Process each journey group
	processed := 0
	skipped := 0
	errors := 0

	for journeyID, states := range grouped {
		// Load journey configuration
		cfg, err := configLoader.LoadJourneyConfig(journeyID)
		if err != nil {
			logger.Error("failed to load config",
				"journey_id", journeyID,
				"error", err)
			errors++
			continue
		}

		// Skip disabled journeys
		if !cfg.Global.Enabled {
			logger.Debug("journey disabled, skipping all states",
				"journey_id", journeyID,
				"state_count", len(states))
			skipped += len(states)
			continue
		}

		// Process each customer state
		for _, state := range states {
			if err := processor.ProcessJourney(ctx, cfg, state); err != nil {
				logger.Error("failed to process journey",
					"journey_id", state.JourneyID,
					"customer", state.CustomerNumber,
					"error", err)
				errors++
			} else {
				processed++
			}
		}
	}

	logger.Info("worker run completed",
		"total_scanned", len(journeys),
		"processed", processed,
		"skipped", skipped,
		"errors", errors,
		"journey_types", len(grouped))

	return nil
}

// groupByJourneyID groups journey states by their journey_id.
func groupByJourneyID(journeys []*domain.JourneyState) map[string][]*domain.JourneyState {
	grouped := make(map[string][]*domain.JourneyState)
	for _, journey := range journeys {
		grouped[journey.JourneyID] = append(grouped[journey.JourneyID], journey)
	}
	return grouped
}
