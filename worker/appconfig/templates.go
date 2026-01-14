package appconfig

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"

	"worker-project/worker/config"
	"worker-project/worker/messaging"
)

// TemplateConfig holds the templates configuration from AppConfig.
// Templates are organized by step name, then template name.
// Each template is a simple string containing the message body.
type TemplateConfig struct {
	Templates map[string]map[string]string `yaml:"templates"`
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
// Format: "config_name:step_name:template_key" (e.g., "journey.onboarding-v2.templates:personal-data:personal-data-soft")
func (r *TemplateRenderer) LoadTemplate(templateRef string) (*messaging.Template, error) {
	configName, stepName, templateKey, err := parseTemplateRef(templateRef)
	if err != nil {
		return nil, err
	}

	templateConfig, err := r.loadTemplateConfig(configName)
	if err != nil {
		return nil, err
	}

	// Navigate nested structure: step -> template
	stepTemplates, ok := templateConfig.Templates[stepName]
	if !ok {
		return nil, fmt.Errorf("step '%s' not found in config %s", stepName, configName)
	}

	body, ok := stepTemplates[templateKey]
	if !ok {
		return nil, fmt.Errorf("template '%s' not found in step '%s' for config %s", templateKey, stepName, configName)
	}

	return &messaging.Template{
		Channel: "whatsapp",
		Content: messaging.TemplateContent{
			Type: "text",
			Body: body,
		},
	}, nil
}

// Render applies metadata to a template and returns the rendered content.
// Supports both {{field}} (direct access) and {{metadata.field}} (dot notation) syntax.
func (r *TemplateRenderer) Render(tmpl *messaging.Template, metadata map[string]any) (string, error) {
	t, err := template.New("message").Parse(tmpl.Content.Body)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	// Create data structure supporting both access patterns
	data := map[string]any{
		"metadata": metadata,
	}

	// Merge metadata fields at root level for direct access ({{field}})
	for k, v := range metadata {
		data[k] = v
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
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

// parseTemplateRef parses a template reference into config name, step name, and template key.
// Expected format: "journey.{journey-id}.templates:{step}:{template}"
// Example: "journey.onboarding-v2.templates:personal-data:personal-data-soft"
func parseTemplateRef(ref string) (configName, stepName, templateKey string, err error) {
	// Find last colon (separates step from template)
	lastColon := strings.LastIndex(ref, ":")
	if lastColon == -1 {
		return "", "", "", fmt.Errorf("invalid template reference format: %s (expected 'config:step:template')", ref)
	}

	// Find second-to-last colon (separates config from step)
	secondLastColon := strings.LastIndex(ref[:lastColon], ":")
	if secondLastColon == -1 {
		return "", "", "", fmt.Errorf("invalid template reference format: %s (expected 'config:step:template')", ref)
	}

	return ref[:secondLastColon], ref[secondLastColon+1:lastColon], ref[lastColon+1:], nil
}

// ClearCache clears the template configuration cache.
func (r *TemplateRenderer) ClearCache() {
	r.cache = make(map[string]*TemplateConfig)
}
