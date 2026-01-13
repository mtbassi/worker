package handler

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"

	"worker-project/event-tracker/models"
)

// handleFinish handles POST /journey/finish requests.
func (h *APIHandler) handleFinish(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var finishReq models.FinishRequest

	// Parse request body
	if err := json.Unmarshal([]byte(req.Body), &finishReq); err != nil {
		h.logger.Warn("invalid request body", "error", err)
		return models.NewErrorResponse(400, "invalid request body"), nil
	}

	// Validate request
	if err := finishReq.Validate(); err != nil {
		h.logger.Warn("validation failed", "error", err)
		return models.NewErrorResponse(400, err.Error()), nil
	}

	// Call service layer
	if err := h.tracker.FinishJourney(ctx, &finishReq); err != nil {
		h.logger.Error("failed to finish journey", "error", err)
		return models.NewErrorResponse(500, "internal error"), nil
	}

	return models.NewSuccessResponse(200, map[string]string{"status": "ok"}), nil
}
