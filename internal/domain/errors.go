package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for common conditions.
var (
	ErrNotFound       = errors.New("not found")
	ErrJourneyExpired = errors.New("journey expired")
	ErrInvalidConfig  = errors.New("invalid configuration")
)

// JourneyError represents an error related to journey processing.
type JourneyError struct {
	JourneyID      string
	CustomerNumber string
	Op             string // operation that failed
	Err            error  // underlying error
}

func (e *JourneyError) Error() string {
	if e.CustomerNumber != "" {
		return fmt.Sprintf("%s: journey=%s customer=%s: %v", e.Op, e.JourneyID, e.CustomerNumber, e.Err)
	}
	return fmt.Sprintf("%s: journey=%s: %v", e.Op, e.JourneyID, e.Err)
}

func (e *JourneyError) Unwrap() error {
	return e.Err
}

// ConfigError represents a configuration-related error.
type ConfigError struct {
	ConfigName string
	Field      string
	Err        error
}

func (e *ConfigError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("config %s: field %s: %v", e.ConfigName, e.Field, e.Err)
	}
	return fmt.Sprintf("config %s: %v", e.ConfigName, e.Err)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// MessagingError represents a message sending error.
type MessagingError struct {
	CustomerNumber string
	TemplateRef    string
	Err            error
}

func (e *MessagingError) Error() string {
	return fmt.Sprintf("messaging: customer=%s template=%s: %v", e.CustomerNumber, e.TemplateRef, e.Err)
}

func (e *MessagingError) Unwrap() error {
	return e.Err
}
