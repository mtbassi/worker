package handler

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"

	"worker-project/event-tracker/models"
)

// handleEvent handles POST /journey/event requests.
func (h *APIHandler) handleEvent(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var eventReq models.EventRequest

	// Parse request body
	if err := json.Unmarshal([]byte(req.Body), &eventReq); err != nil {
		h.logger.Warn("invalid request body", "error", err)
		return models.NewErrorResponse(400, "invalid request body"), nil
	}

	// Validate request
	if err := eventReq.Validate(); err != nil {
		h.logger.Warn("validation failed", "error", err)
		return models.NewErrorResponse(400, err.Error()), nil
	}

	// Call service layer
	if err := h.tracker.RecordEvent(ctx, &eventReq); err != nil {
		h.logger.Error("failed to record event", "error", err)
		return models.NewErrorResponse(500, "internal error"), nil
	}

	return models.NewSuccessResponse(200, map[string]string{"status": "ok"}), nil
}
