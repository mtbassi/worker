package config

import "time"

// JourneyConfig represents the configuration for a journey.
type JourneyConfig struct {
	Journey  Journey  `yaml:"journey"`
	Settings Settings `yaml:"settings"`
	Steps    []Step   `yaml:"steps"`
}

// Journey holds journey identification.
type Journey struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// Settings holds journey-level settings.
type Settings struct {
	MaxInactiveTime   Duration        `yaml:"max_inactive_time"`
	Session           SessionSettings `yaml:"session"`
	LifecycleRepiques []Repique       `yaml:"lifecycle_repiques"`
}

// SessionSettings controls session behavior.
type SessionSettings struct {
	ResetOnInteraction bool          `yaml:"reset_on_interaction"`
	ResetRepiques      ResetRepiques `yaml:"reset_repiques"`
}

// ResetRepiques controls which repiques are reset on interaction.
type ResetRepiques struct {
	Lifecycle bool `yaml:"lifecycle"`
	Step      bool `yaml:"step"`
}

// Duration represents a duration in minutes for YAML configuration.
type Duration struct {
	Minutes int `yaml:"minutes"`
}

// ToDuration converts to a standard time.Duration.
func (d Duration) ToDuration() time.Duration {
	return time.Duration(d.Minutes) * time.Minute
}

// Step represents a step within a journey.
type Step struct {
	ID       string    `yaml:"id"`
	Name     string    `yaml:"name"`
	Repiques []Repique `yaml:"repiques"`
}

// Repique represents a recovery message rule.
type Repique struct {
	ID          string    `yaml:"id"`
	MaxAttempts int       `yaml:"max_attempts"`
	Condition   Condition `yaml:"condition,omitempty"`
	Trigger     Trigger   `yaml:"trigger,omitempty"`
	Action      Action    `yaml:"action"`
}

// Condition defines when a repique should trigger.
type Condition struct {
	TimeInStep *TimeCondition `yaml:"time_in_step,omitempty"`
}

// TimeCondition defines a time-based condition.
type TimeCondition struct {
	GteMinutes int `yaml:"gte_minutes"`
}

// Trigger defines lifecycle-based triggers.
type Trigger struct {
	BeforeExpire *Duration `yaml:"before_expire,omitempty"`
	OnExpire     bool      `yaml:"on_expire,omitempty"`
}

// Action defines what happens when a repique triggers.
type Action struct {
	Template   string `yaml:"template,omitempty"`
	EndJourney bool   `yaml:"end_journey,omitempty"`
}

// FindStep finds a step by ID, returns nil if not found.
func (c *JourneyConfig) FindStep(stepID string) *Step {
	for i := range c.Steps {
		if c.Steps[i].ID == stepID {
			return &c.Steps[i]
		}
	}
	return nil
}
