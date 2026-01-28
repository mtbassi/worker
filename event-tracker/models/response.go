package models

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// SuccessResponse represents a success response.
type SuccessResponse struct {
	Data interface{} `json:"data"`
}

// NewErrorResponse creates an API Gateway error response.
func NewErrorResponse(statusCode int, message string) events.APIGatewayProxyResponse {
	body := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		slog.Error("failed to marshal error response",
			"error", err,
			"status_code", statusCode,
			"message", message)
		// Return a fallback error response
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"error":"Internal Server Error","message":"failed to build error response"}`,
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(bodyJSON),
	}
}

// NewSuccessResponse creates an API Gateway success response.
func NewSuccessResponse(statusCode int, data interface{}) events.APIGatewayProxyResponse {
	body := SuccessResponse{Data: data}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		slog.Error("failed to marshal success response",
			"error", err,
			"status_code", statusCode)
		// Return an error response instead
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: `{"error":"Internal Server Error","message":"failed to build response"}`,
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(bodyJSON),
	}
}
