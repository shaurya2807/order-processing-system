package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/shaurya2807/order-processing-system/configs"
	"github.com/shaurya2807/order-processing-system/internal/inventory"
	"github.com/shaurya2807/order-processing-system/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load()

	cfg, err := configs.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(os.Getenv("APP_ENV"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger init error: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer, err := inventory.NewConsumer(ctx,
		cfg.SQS.Region,
		cfg.SQS.EndpointURL,
		cfg.SQS.QueueURL,
		cfg.SQS.AccessKey,
		cfg.SQS.SecretKey,
		log,
	)
	if err != nil {
		log.Fatal("failed to create inventory consumer", zap.Error(err))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Info("shutting down inventory service...")
		cancel()
	}()

	consumer.Start(ctx)
	log.Info("inventory service stopped")
}
