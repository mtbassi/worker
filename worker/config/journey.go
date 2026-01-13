package config

// JourneyConfig represents the configuration for a journey.
type JourneyConfig struct {
	Journey string       `yaml:"journey"`
	Global  GlobalConfig `yaml:"global"`
	Steps   []StepConfig `yaml:"steps"`
}

// GlobalConfig holds global journey settings.
type GlobalConfig struct {
	Enabled                           bool `yaml:"enabled"`
	MaxTotalAttempts                  int  `yaml:"max_total_attempts"`
	MinIntervalBetweenAttemptsMinutes int  `yaml:"min_interval_between_attempts_minutes"`
}

// StepConfig represents a step within a journey.
type StepConfig struct {
	Name          string         `yaml:"name"`
	RecoveryRules []RecoveryRule `yaml:"recovery_rules"`
}

// RecoveryRule represents a recovery message rule.
type RecoveryRule struct {
	Name            string `yaml:"name"`
	Enabled         bool   `yaml:"enabled"`
	InactiveMinutes int    `yaml:"inactive_minutes"`
	MaxAttempts     int    `yaml:"max_attempts"`
	Template        string `yaml:"template"`
}

// FindStepByName finds a step by name, returns nil if not found.
func (c *JourneyConfig) FindStepByName(name string) *StepConfig {
	for i := range c.Steps {
		if c.Steps[i].Name == name {
			return &c.Steps[i]
		}
	}
	return nil
}
