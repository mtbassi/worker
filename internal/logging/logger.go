package logging

import (
	"io"
	"log/slog"
	"os"
)

// Config holds logger configuration.
type Config struct {
	Level  slog.Level
	Format string // "json" or "text"
	Output io.Writer
}

// DefaultConfig returns sensible defaults for the logger.
// Uses JSON format in Lambda environment, text format locally.
// Defaults to Info level unless DEBUG env var is set.
func DefaultConfig() Config {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	}

	format := "json"
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
		format = "text"
	}

	return Config{
		Level:  level,
		Format: format,
		Output: os.Stdout,
	}
}

// New creates a configured slog.Logger.
func New(cfg Config) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: cfg.Level,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	} else {
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	return slog.New(handler)
}

// WithComponent returns a logger with a component attribute.
func WithComponent(logger *slog.Logger, component string) *slog.Logger {
	return logger.With("component", component)
}
