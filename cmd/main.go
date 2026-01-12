package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-lambda-go/lambda"

	"worker-project/internal/adapters/appconfig"
	"worker-project/internal/adapters/messaging"
	"worker-project/internal/adapters/redis"
	"worker-project/internal/app"
	"worker-project/internal/config"
	"worker-project/internal/logging"
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

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		return err
	}

	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		return err
	}
	defer redisClient.Close()

	logger.Info("connected to redis", "addr", cfg.Redis.Addr)

	templateRenderer := appconfig.NewTemplateRenderer(cfg.AppConfig, logger.With("component", "templates"))
	configLoader := appconfig.NewLoader(cfg.AppConfig, logger.With("component", "config_loader"))
	messengerClient := messaging.NewClient(templateRenderer, logger.With("component", "messenger"))

	application := app.New(app.Options{
		Config:       cfg,
		Logger:       logger,
		Scanner:      redis.NewScanner(redisClient, cfg.Worker.ScanCount, logger.With("component", "scanner")),
		Repository:   redis.NewRepository(redisClient, cfg.Worker.DefaultStateTTL),
		ConfigLoader: configLoader,
		Messenger:    messengerClient,
	})

	return application.Run(ctx)
}
