package handler

import (
	"context"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"

	"worker-project/event-tracker/service"
)

// APIHandler handles HTTP requests from API Gateway.
type APIHandler struct {
	tracker *service.Tracker
	logger  *slog.Logger
}

// NewAPIHandler creates a new API handler.
func NewAPIHandler(tracker *service.Tracker, logger *slog.Logger) *APIHandler {
	return &APIHandler{
		tracker: tracker,
		logger:  logger,
	}
}

// Handle routes API Gateway requests to the appropriate handler.
func (h *APIHandler) Handle(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	h.logger.Info("request received",
		"path", req.Path,
		"method", req.HTTPMethod)

	switch {
	case req.Path == "/journey/event" && req.HTTPMethod == "POST":
		return h.handleEvent(ctx, req)
	case req.Path == "/journey/finish" && req.HTTPMethod == "POST":
		return h.handleFinish(ctx, req)
	default:
		h.logger.Warn("route not found",
			"path", req.Path,
			"method", req.HTTPMethod)
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"error":"Not Found","message":"route not found"}`,
		}, nil
	}
}
