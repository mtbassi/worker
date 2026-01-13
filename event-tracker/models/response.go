package models

import (
	"encoding/json"
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

	bodyJSON, _ := json.Marshal(body)

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
	bodyJSON, _ := json.Marshal(body)

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(bodyJSON),
	}
}
