package app

import (
	"context"
	"log/slog"

	"worker-project/internal/config"
	"worker-project/internal/domain"
	"worker-project/internal/ports"
	"worker-project/internal/service"
)

// Stats holds processing statistics.
type Stats struct {
	JourneyTypes int
	TotalSessions int
	Processed    int
	Errors       int
}

// App is the main application container.
type App struct {
	cfg          *config.AppConfig
	logger       *slog.Logger
	scanner      ports.JourneyScanner
	repository   ports.StateRepository
	configLoader ports.JourneyConfigLoader
	messenger    ports.Messenger
	processor    *service.Processor
}

// Options configures the App.
type Options struct {
	Config       *config.AppConfig
	Logger       *slog.Logger
	Scanner      ports.JourneyScanner
	Repository   ports.StateRepository
	ConfigLoader ports.JourneyConfigLoader
	Messenger    ports.Messenger
}

// New creates a new App with all dependencies injected.
func New(opts Options) *App {
	processor := service.NewProcessor(
		opts.Repository,
		opts.Messenger,
		opts.Logger.With("component", "processor"),
	)

	return &App{
		cfg:          opts.Config,
		logger:       opts.Logger,
		scanner:      opts.Scanner,
		repository:   opts.Repository,
		configLoader: opts.ConfigLoader,
		messenger:    opts.Messenger,
		processor:    processor,
	}
}

// Run executes the worker.
func (a *App) Run(ctx context.Context) error {
	a.logger.Info("starting worker")

	journeys, err := a.scanner.ScanAllJourneys(ctx)
	if err != nil {
		return &domain.JourneyError{
			Op:  "ScanAllJourneys",
			Err: err,
		}
	}

	if len(journeys) == 0 {
		a.logger.Info("no active journeys found")
		return nil
	}

	grouped := groupByJourneyID(journeys)

	a.logger.Info("found active journeys",
		"journey_types", len(grouped),
		"total_sessions", len(journeys),
	)

	stats := a.processJourneyGroups(ctx, grouped)

	a.logger.Info("worker completed",
		"journey_types", stats.JourneyTypes,
		"total_sessions", stats.TotalSessions,
		"processed", stats.Processed,
		"errors", stats.Errors,
	)

	return nil
}

func (a *App) processJourneyGroups(ctx context.Context, groups map[string][]*domain.JourneyState) Stats {
	stats := Stats{
		JourneyTypes: len(groups),
	}

	for journeyID, states := range groups {
		stats.TotalSessions += len(states)

		logger := a.logger.With("journey_id", journeyID, "session_count", len(states))
		logger.Info("processing journey type")

		cfg, err := a.configLoader.LoadJourneyConfig(journeyID)
		if err != nil {
			logger.Error("failed to load config", "error", err)
			stats.Errors += len(states)
			continue
		}

		logger.Debug("loaded config",
			"journey_name", cfg.Journey.Name,
			"max_inactive_minutes", cfg.Settings.MaxInactiveTime.Minutes,
			"lifecycle_repiques", len(cfg.Settings.LifecycleRepiques),
			"steps", len(cfg.Steps),
		)

		for _, state := range states {
			select {
			case <-ctx.Done():
				a.logger.Warn("context cancelled, stopping processing")
				return stats
			default:
				if err := a.processor.ProcessJourney(ctx, cfg, state); err != nil {
					a.logger.Error("failed to process customer",
						"customer_number", state.CustomerNumber,
						"error", err,
					)
					stats.Errors++
				} else {
					stats.Processed++
				}
			}
		}
	}

	return stats
}

func groupByJourneyID(journeys []*domain.JourneyState) map[string][]*domain.JourneyState {
	groups := make(map[string][]*domain.JourneyState)
	for _, j := range journeys {
		groups[j.JourneyID] = append(groups[j.JourneyID], j)
	}
	return groups
}
