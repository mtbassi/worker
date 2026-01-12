package ports

import (
	"context"

	"worker-project/internal/domain"
)

// Messenger sends recovery messages to customers.
type Messenger interface {
	// Send sends a single message.
	Send(ctx context.Context, msg domain.Message) error
}

// Template represents a message template.
type Template struct {
	Channel string
	Content TemplateContent
}

// TemplateContent holds the template content details.
type TemplateContent struct {
	Type string
	Body string
}

// TemplateRenderer loads and renders message templates.
type TemplateRenderer interface {
	// LoadTemplate loads a template by reference.
	LoadTemplate(templateRef string) (*Template, error)

	// Render applies metadata to a template and returns the rendered content.
	Render(template *Template, metadata map[string]any) (string, error)
}
