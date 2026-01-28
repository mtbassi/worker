package appconfig

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"worker-project/worker/config"
)

// Loader implements ports.JourneyConfigLoader using AWS AppConfig.
type Loader struct {
	httpClient *http.Client
	endpoint   string
	logger     *slog.Logger
	cache      map[string]*config.JourneyConfig
	mu         sync.RWMutex
}

// NewLoader creates a new AppConfig loader.
func NewLoader(cfg config.AppConfigSettings, logger *slog.Logger) *Loader {
	return &Loader{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		endpoint: cfg.Endpoint,
		logger:   logger,
		cache:    make(map[string]*config.JourneyConfig),
	}
}

// LoadJourneyConfig loads configuration for a specific journey.
func (l *Loader) LoadJourneyConfig(journeyID string) (*config.JourneyConfig, error) {
	// Check cache with read lock
	l.mu.RLock()
	if cached, ok := l.cache[journeyID]; ok {
		l.mu.RUnlock()
		return cached, nil
	}
	l.mu.RUnlock()

	// Acquire write lock for loading
	l.mu.Lock()
	defer l.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, ok := l.cache[journeyID]; ok {
		return cached, nil
	}

	configName := fmt.Sprintf("journey.%s", journeyID)
	data, err := l.loadProfile(configName)
	if err != nil {
		return nil, fmt.Errorf("load journey config %s: %w", journeyID, err)
	}

	var cfg config.JourneyConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse journey config %s: %w", journeyID, err)
	}

	if err := config.ValidateJourneyConfig(&cfg); err != nil {
		return nil, err
	}

	l.cache[journeyID] = &cfg
	l.logger.Debug("loaded journey config", "journey_id", journeyID)

	return &cfg, nil
}

// loadProfile fetches a configuration profile from AppConfig.
func (l *Loader) loadProfile(profile string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s.yaml", l.endpoint, profile)

	resp, err := l.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch config: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			l.logger.Warn("failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config not found: %s (status %d)", profile, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// ClearCache clears the configuration cache.
func (l *Loader) ClearCache() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache = make(map[string]*config.JourneyConfig)
}
