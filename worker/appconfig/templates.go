package appconfig

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"

	"worker-project/worker/config"
	"worker-project/worker/messaging"
)

// TemplateConfig holds the templates configuration from AppConfig.
type TemplateConfig struct {
	Templates map[string]TemplateDefinition `yaml:"templates"`
}

// TemplateDefinition represents a single template definition.
type TemplateDefinition struct {
	Channel string                 `yaml:"channel"`
	Content TemplateContentDef     `yaml:"content"`
}

// TemplateContentDef holds the content type and body.
type TemplateContentDef struct {
	Type string `yaml:"type"`
	Body string `yaml:"body"`
}

// TemplateRenderer implements messaging.TemplateRenderer using AppConfig.
type TemplateRenderer struct {
	httpClient *http.Client
	endpoint   string
	logger     *slog.Logger
	cache      map[string]*TemplateConfig
}

// NewTemplateRenderer creates a new template renderer.
func NewTemplateRenderer(cfg config.AppConfigSettings, logger *slog.Logger) *TemplateRenderer {
	return &TemplateRenderer{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		endpoint: cfg.Endpoint,
		logger:   logger,
		cache:    make(map[string]*TemplateConfig),
	}
}

// LoadTemplate loads a template by reference.
// Format: "config_name:template_key" (e.g., "journey.account_creation.templates:reminder_10_min")
func (r *TemplateRenderer) LoadTemplate(templateRef string) (*messaging.Template, error) {
	configName, templateKey, err := parseTemplateRef(templateRef)
	if err != nil {
		return nil, err
	}

	templateConfig, err := r.loadTemplateConfig(configName)
	if err != nil {
		return nil, err
	}

	def, ok := templateConfig.Templates[templateKey]
	if !ok {
		return nil, fmt.Errorf("template key %s not found in config %s", templateKey, configName)
	}

	return &messaging.Template{
		Channel: def.Channel,
		Content: messaging.TemplateContent{
			Type: def.Content.Type,
			Body: def.Content.Body,
		},
	}, nil
}

// Render applies metadata to a template and returns the rendered content.
func (r *TemplateRenderer) Render(tmpl *messaging.Template, metadata map[string]any) (string, error) {
	t, err := template.New("message").Parse(tmpl.Content.Body)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, metadata); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

// loadTemplateConfig fetches and caches a template configuration.
func (r *TemplateRenderer) loadTemplateConfig(configName string) (*TemplateConfig, error) {
	if cached, ok := r.cache[configName]; ok {
		return cached, nil
	}

	url := fmt.Sprintf("%s/%s.yaml", r.endpoint, configName)

	resp, err := r.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch template config: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			r.logger.Warn("failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("template config not found: %s (status %d)", configName, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read template config: %w", err)
	}

	var cfg TemplateConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse template config: %w", err)
	}

	r.cache[configName] = &cfg
	r.logger.Debug("loaded template config", "config_name", configName)

	return &cfg, nil
}

// parseTemplateRef parses a template reference into config name and template key.
func parseTemplateRef(ref string) (configName, templateKey string, err error) {
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == ':' {
			return ref[:i], ref[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid template reference format: %s (expected 'config_name:template_key')", ref)
}

// ClearCache clears the template configuration cache.
func (r *TemplateRenderer) ClearCache() {
	r.cache = make(map[string]*TemplateConfig)
}
