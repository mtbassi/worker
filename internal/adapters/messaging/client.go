package messaging

import (
	"context"
	"encoding/json"
	"log/slog"

	"worker-project/internal/domain"
	"worker-project/internal/ports"
)

// Client implements ports.Messenger.
// This is a stub implementation that logs messages instead of sending them.
type Client struct {
	templateRenderer ports.TemplateRenderer
	logger           *slog.Logger
}

// NewClient creates a new messaging client.
func NewClient(templateRenderer ports.TemplateRenderer, logger *slog.Logger) *Client {
	return &Client{
		templateRenderer: templateRenderer,
		logger:           logger,
	}
}

// Send sends a message to a customer.
// TODO: Implement actual message sending via WhatsApp Business API.
// Options include:
// - Publish to SNS topic
// - Send to SQS queue
// - Call external notification API
func (c *Client) Send(ctx context.Context, msg domain.Message) error {
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

	finalMessage := map[string]any{
		"customer_number": msg.CustomerNumber,
		"tenant_id":       msg.TenantID,
		"contact_id":      msg.ContactID,
		"repique_id":      msg.RepiqueID,
		"step":            msg.Step,
		"channel":         template.Channel,
		"content": map[string]any{
			"type": template.Content.Type,
			"body": renderedBody,
		},
	}

	data, err := json.MarshalIndent(finalMessage, "", "  ")
	if err != nil {
		return &domain.MessagingError{
			CustomerNumber: msg.CustomerNumber,
			TemplateRef:    msg.Template,
			Err:            err,
		}
	}

	c.logger.Info("sending message",
		"customer_number", msg.CustomerNumber,
		"repique_id", msg.RepiqueID,
		"channel", template.Channel,
	)
	c.logger.Debug("message payload", "payload", string(data))

	// TODO: Implement actual message sending here
	// Example implementations:
	//
	// SNS:
	//   snsClient.Publish(ctx, &sns.PublishInput{
	//       TopicArn: aws.String(topicArn),
	//       Message:  aws.String(string(data)),
	//   })
	//
	// SQS:
	//   sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
	//       QueueUrl:    aws.String(queueUrl),
	//       MessageBody: aws.String(string(data)),
	//   })
	//
	// HTTP:
	//   httpClient.Post(apiURL, "application/json", bytes.NewReader(data))

	return nil
}
