package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"worker-project/event-tracker/config"
	"worker-project/event-tracker/handler"
	"worker-project/event-tracker/service"
	"worker-project/shared/logging"
	"worker-project/shared/redis"
)

func main() {
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(handleRequest)
	} else {
		if err := runLocal(); err != nil {
			log.Fatal(err)
		}
	}
}

func runLocal() error {
	// Initialize components
	logger := logging.New(logging.DefaultConfig())

	// Load configuration
	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		return err
	}

	// Create Redis client
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		return err
	}
	defer redisClient.Close()

	logger.Info("connected to redis", "addr", cfg.Redis.Addr)

	// Create state store with TTL
	stateStore := redis.NewStateStore(redisClient, cfg.DefaultStateTTL)

	// Create tracker service
	trackerService := service.NewTracker(stateStore, logger.With("component", "tracker"))

	// Create API handler
	apiHandler := handler.NewAPIHandler(trackerService, logger.With("component", "api"))

	// HTTP handler function
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		// Convert HTTP request to API Gateway format
		body := []byte{}
		if r.Body != nil {
			body, _ = io.ReadAll(r.Body)
		}

		req := events.APIGatewayProxyRequest{
			HTTPMethod: r.Method,
			Path:       r.URL.Path,
			Body:       string(body),
			Headers:    make(map[string]string),
		}

		// Copy headers
		for k, v := range r.Header {
			if len(v) > 0 {
				req.Headers[k] = v[0]
			}
		}

		// Handle request
		resp, err := apiHandler.Handle(context.Background(), req)
		if err != nil {
			logger.Error("handler error", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"Internal Server Error"}`))
			return
		}

		// Write response
		for k, v := range resp.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(resp.StatusCode)
		w.Write([]byte(resp.Body))
	}

	// Start HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	logger.Info("Starting Event Tracker HTTP server", "addr", addr)

	return http.ListenAndServe(addr, http.HandlerFunc(httpHandler))
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
