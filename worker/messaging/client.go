package messaging

import (
	"context"
	"log/slog"

	"worker-project/shared/domain"
)

// TemplateRenderer is the interface for rendering templates.
type TemplateRenderer interface {
	LoadTemplate(templateRef string) (*Template, error)
	Render(template *Template, metadata map[string]any) (string, error)
}

// Template represents a loaded template.
type Template struct {
	Channel string
	Content TemplateContent
}

// TemplateContent holds template content information.
type TemplateContent struct {
	Type string
	Body string
}

// Client sends messages to customers via WhatsApp Business API.
type Client struct {
	templateRenderer TemplateRenderer
	whatsappClient   *WhatsAppClient
	logger           *slog.Logger
}

// NewClient creates a new messaging client.
func NewClient(templateRenderer TemplateRenderer, whatsappConfig WhatsAppConfig, logger *slog.Logger) *Client {
	return &Client{
		templateRenderer: templateRenderer,
		whatsappClient:   NewWhatsAppClient(whatsappConfig),
		logger:           logger,
	}
}

// Send sends a message to a customer via WhatsApp Business API.
func (c *Client) Send(ctx context.Context, msg domain.Message) error {
	// Load and render template
	template, err := c.templateRenderer.LoadTemplate(msg.Template)
	if err != nil {
		return &domain.MessagingError{
			CustomerNumber: msg.CustomerNumber,
			TemplateRef:    msg.Template,
			Err:            err,
		}
	}

	renderedBody, err := c.templateRenderer.Render(template, msg.Metadata)
	if err != nil {
		return &domain.MessagingError{
			CustomerNumber: msg.CustomerNumber,
			TemplateRef:    msg.Template,
			Err:            err,
		}
	}

	c.logger.Info("sending whatsapp message",
		"customer_number", msg.CustomerNumber,
		"repique_id", msg.RepiqueID,
		"step", msg.Step,
	)

	// Send via WhatsApp Business API
	resp, err := c.whatsappClient.Send(ctx, msg.CustomerNumber, renderedBody)
	if err != nil {
		c.logger.Error("whatsapp api error",
			"customer_number", msg.CustomerNumber,
			"error", err,
		)
		return &domain.MessagingError{
			CustomerNumber: msg.CustomerNumber,
			TemplateRef:    msg.Template,
			Err:            err,
		}
	}

	c.logger.Info("whatsapp message sent successfully",
		"customer_number", msg.CustomerNumber,
		"message_id", resp.Messages[0].ID,
		"wa_id", resp.Contacts[0].WaID,
	)

	return nil
}
