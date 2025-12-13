// Package main Payment Serviceのエントリーポイント
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"microservice-architecture-sample/pkg/database"
	"microservice-architecture-sample/pkg/messaging"
	"microservice-architecture-sample/pkg/outbox"
	"microservice-architecture-sample/services/payment/internal/application"
	"microservice-architecture-sample/services/payment/internal/infrastructure"
	"microservice-architecture-sample/services/payment/internal/interfaces"
)

func main() {
	// ロガー初期化
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Payment Service")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// データベース接続初期化
	dbConfig := database.ConfigFromEnv()
	pool, err := database.NewPool(ctx, dbConfig)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("Connected to database")

	// NATSクライアント初期化
	natsClient, err := messaging.NewClient(ctx, logger)
	if err != nil {
		logger.Error("Failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer natsClient.Close()

	// コンポーネント初期化（DI）
	repo := infrastructure.NewPostgresPaymentRepository(pool)
	outboxPublisher := outbox.NewPublisher(pool, logger)
	idempotencyChecker := outbox.NewIdempotencyChecker(pool)
	useCase := application.NewPaymentUseCase(pool, repo, outboxPublisher, idempotencyChecker, logger)

	// Outbox Relay初期化
	relay := outbox.NewRelay(pool, logger, natsClient.Publish)
	go relay.Start(ctx)

	// イベントコンシューマ初期化
	consumer := interfaces.NewEventConsumer(natsClient, useCase, logger)
	if err := consumer.Start(ctx); err != nil {
		logger.Error("Failed to start event consumer", "error", err)
		os.Exit(1)
	}
	defer consumer.Stop()

	// HTTPサーバー初期化
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info("request",
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency", v.Latency,
				"error", v.Error,
			)
			return nil
		},
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	handler := interfaces.NewHTTPHandler(useCase)
	handler.RegisterRoutes(e)

	// HTTPサーバー起動
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		if err := e.Start(":" + port); err != nil {
			logger.Info("HTTP server stopped", "error", err)
		}
	}()
	logger.Info("HTTP server started", "port", port)

	// シャットダウンシグナル待機
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Payment Service...")
	cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown HTTP server", "error", err)
	}

	logger.Info("Payment Service stopped")
}
