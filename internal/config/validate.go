package config

import (
	"errors"
	"fmt"
)

// Validate validates the application configuration.
func (c *AppConfig) Validate() error {
	var errs []error

	if c.Redis.Addr == "" {
		errs = append(errs, errors.New("redis address is required"))
	}

	if c.Redis.DialTimeout <= 0 {
		errs = append(errs, errors.New("redis dial timeout must be positive"))
	}

	if c.Worker.ScanCount <= 0 {
		errs = append(errs, errors.New("worker scan count must be positive"))
	}

	if c.Worker.DefaultStateTTL <= 0 {
		errs = append(errs, errors.New("worker default state TTL must be positive"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %w", errors.Join(errs...))
	}

	return nil
}

// ValidateJourneyConfig validates a journey configuration.
func ValidateJourneyConfig(cfg *JourneyConfig) error {
	var errs []error

	if cfg.Journey.ID == "" {
		errs = append(errs, errors.New("journey.id is required"))
	}

	if cfg.Settings.MaxInactiveTime.Minutes <= 0 {
		errs = append(errs, errors.New("settings.max_inactive_time.minutes must be positive"))
	}

	for i, step := range cfg.Steps {
		if step.ID == "" {
			errs = append(errs, fmt.Errorf("steps[%d].id is required", i))
		}

		for j, repique := range step.Repiques {
			if repique.ID == "" {
				errs = append(errs, fmt.Errorf("steps[%d].repiques[%d].id is required", i, j))
			}
			if repique.MaxAttempts <= 0 {
				errs = append(errs, fmt.Errorf("steps[%d].repiques[%d].max_attempts must be positive", i, j))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("journey config validation failed: %w", errors.Join(errs...))
	}

	return nil
}
