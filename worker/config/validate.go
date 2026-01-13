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

	if cfg.Journey == "" {
		errs = append(errs, errors.New("journey is required"))
	}

	if cfg.Global.MaxTotalAttempts <= 0 {
		errs = append(errs, errors.New("global.max_total_attempts must be positive"))
	}

	if cfg.Global.MinIntervalBetweenAttemptsMinutes <= 0 {
		errs = append(errs, errors.New("global.min_interval_between_attempts_minutes must be positive"))
	}

	for i, step := range cfg.Steps {
		if step.Name == "" {
			errs = append(errs, fmt.Errorf("steps[%d].name is required", i))
		}

		for j, rule := range step.RecoveryRules {
			if rule.Name == "" {
				errs = append(errs, fmt.Errorf("steps[%d].recovery_rules[%d].name is required", i, j))
			}
			if rule.InactiveMinutes <= 0 {
				errs = append(errs, fmt.Errorf("steps[%d].recovery_rules[%d].inactive_minutes must be positive", i, j))
			}
			if rule.MaxAttempts <= 0 {
				errs = append(errs, fmt.Errorf("steps[%d].recovery_rules[%d].max_attempts must be positive", i, j))
			}
			if rule.Template == "" {
				errs = append(errs, fmt.Errorf("steps[%d].recovery_rules[%d].template is required", i, j))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("journey config validation failed: %w", errors.Join(errs...))
	}

	return nil
}
