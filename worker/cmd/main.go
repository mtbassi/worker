package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"worker-project/shared/logging"
	"worker-project/shared/redis"
	"worker-project/worker/appconfig"
	"worker-project/worker/config"
	"worker-project/worker/handler"
	"worker-project/worker/messaging"
	"worker-project/worker/service"
)

func main() {
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(handleLambda)
	} else {
		if err := runLocal(); err != nil {
			os.Exit(1)
		}
	}
}

func handleLambda(ctx context.Context) error {
	return run(ctx)
}

func runLocal() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	return run(ctx)
}

func run(ctx context.Context) error {
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

	// Create shared state store
	stateStore := redis.NewStateStore(redisClient, cfg.Worker.DefaultStateTTL)

	// Create worker-specific components
	scanner := service.NewScanner(redisClient, cfg.Worker.ScanCount, logger.With("component", "scanner"))
	configLoader := appconfig.NewLoader(cfg.AppConfig, logger.With("component", "config_loader"))
	templateRenderer := appconfig.NewTemplateRenderer(cfg.AppConfig, logger.With("component", "templates"))

	// Create WhatsApp messaging client
	whatsappCfg := messaging.WhatsAppConfig{
		APIEndpoint:   cfg.WhatsApp.APIEndpoint,
		PhoneNumberID: cfg.WhatsApp.PhoneNumberID,
		AccessToken:   cfg.WhatsApp.AccessToken,
		Timeout:       10 * time.Second,
		MaxRetries:    3,
		RetryDelay:    2 * time.Second,
	}
	messengerClient := messaging.NewClient(templateRenderer, whatsappCfg, logger.With("component", "messenger"))

	// Create processor
	processor := service.NewProcessor(stateStore, messengerClient, logger.With("component", "processor"))

	// Run worker
	return handler.Run(ctx, scanner, configLoader, processor, logger)
}
