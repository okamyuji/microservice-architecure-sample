// Package main Inventory Serviceのエントリーポイント
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"microservice-architecture-sample/pkg/database"
	"microservice-architecture-sample/pkg/messaging"
	"microservice-architecture-sample/pkg/outbox"
	"microservice-architecture-sample/services/inventory/internal/application"
	"microservice-architecture-sample/services/inventory/internal/infrastructure"
	"microservice-architecture-sample/services/inventory/internal/interfaces"
)

func main() {
	// ロガー初期化
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Inventory Service")

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
	inventoryRepo := infrastructure.NewPostgresInventoryRepository(pool)
	reservationRepo := infrastructure.NewPostgresReservationRepository(pool)
	outboxPublisher := outbox.NewPublisher(pool, logger)
	idempotencyChecker := outbox.NewIdempotencyChecker(pool)
	useCase := application.NewInventoryUseCase(pool, inventoryRepo, reservationRepo, outboxPublisher, idempotencyChecker, logger)

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
	e := newHTTPServer(logger)

	handler := interfaces.NewHTTPHandler(useCase)
	handler.RegisterRoutes(e)

	// HTTPサーバー起動
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	// ctx を取り消すと、処理中の要求を最大10秒待って止まり、そのあと Start が戻る
	sc := echo.StartConfig{
		Address:    ":" + port,
		HideBanner: true,
		OnShutdownError: func(err error) {
			logger.Error("Failed to shutdown HTTP server", "error", err)
		},
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	serverDone := make(chan struct{})
	var startFailed atomic.Bool
	go func() {
		defer close(serverDone)
		if err := sc.Start(ctx, e); err != nil {
			logger.Error("HTTP server stopped", "error", err)
			startFailed.Store(true)
			// ポートの競合などで起動できないとき、HTTP を受けないまま process だけが残らないよう止める
			quit <- syscall.SIGTERM
		}
	}()
	logger.Info("HTTP server started", "port", port)

	// シャットダウンシグナル待機
	<-quit

	logger.Info("Shutting down Inventory Service...")
	cancel()

	<-serverDone

	logger.Info("Inventory Service stopped")
	// compose の restart: on-failure が再起動できるよう、起動の失敗は終了コードで伝える
	if startFailed.Load() {
		os.Exit(1)
	}
}

// newHTTPServer は共通のミドルウェアを付けた echo を返す。起動前に組み立ての panic をテストで捕まえるために切り出している
func newHTTPServer(logger *slog.Logger) *echo.Echo {
	e := echo.New()
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		HandleError: true,
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
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
	// v5 は許可元の指定がないと panic する。v4 の既定（全許可）と同じにする
	e.Use(middleware.CORS("*"))
	return e
}
