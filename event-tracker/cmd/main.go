package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"worker-project/event-tracker/config"
	"worker-project/event-tracker/handler"
	"worker-project/event-tracker/service"
	"worker-project/shared/logging"
	"worker-project/shared/redis"
)

func main() {
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Initialize logger
	logger := logging.New(logging.DefaultConfig())

	// Load configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"Internal Server Error","message":"configuration error"}`,
		}, nil
	}

	// Create Redis client
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error":"Internal Server Error","message":"database connection error"}`,
		}, nil
	}
	defer redisClient.Close()

	// Create state store with TTL
	stateStore := redis.NewStateStore(redisClient, cfg.DefaultStateTTL)

	// Create tracker service
	trackerService := service.NewTracker(stateStore, logger.With("component", "tracker"))

	// Create API handler
	apiHandler := handler.NewAPIHandler(trackerService, logger.With("component", "api"))

	// Handle request
	return apiHandler.Handle(ctx, request)
}
