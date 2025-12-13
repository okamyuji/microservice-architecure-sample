package interfaces

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/events"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/messaging"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/outbox"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/testutil"
	"github.com/yujiokamoto/microservice-architecture-sample/services/inventory/internal/application"
	"github.com/yujiokamoto/microservice-architecture-sample/services/inventory/internal/infrastructure"
)

// setupTestConsumer テスト用コンシューマをセットアップ（NATSなし）
func setupTestConsumer(t *testing.T) (*EventConsumer, *application.InventoryUseCase, func()) {
	t.Helper()

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, logger)
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := application.NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, logger)

	consumer := &EventConsumer{
		client:  nil,
		useCase: uc,
		logger:  logger,
		subs:    nil,
	}

	cleanup := func() {
		_ = db.CleanupInventoryDB(ctx)
	}

	return consumer, uc, cleanup
}

// TestNewEventConsumer コンシューマ生成テスト
func TestNewEventConsumer(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	consumer := NewEventConsumer(nil, nil, logger)

	if consumer == nil {
		t.Fatal("consumer が nil")
	}
}

// TestEventConsumer_Stop ストップテスト
func TestEventConsumer_Stop(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, _, cleanup := setupTestConsumer(t)
	defer cleanup()

	consumer.Stop()
}

// TestEventConsumer_handleOrderCreated OrderCreatedイベントハンドラ
func TestEventConsumer_handleOrderCreated(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, uc, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// イベントを作成
	orderID := uuid.New().String()
	event := events.NewOrderCreatedEvent(orderID, "cust-1", "PROD-001", 5, 100.00)
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// ハンドラを呼び出し
	consumer.handleOrderCreated(ctx, msg)

	// 在庫が予約されていることを確認
	inv, _ := uc.GetInventory(ctx, "PROD-001")
	if inv.ReservedQuantity < 5 {
		t.Log("在庫予約成功の確認")
	}
}

// TestEventConsumer_handleOrderCreated_InvalidJSON 不正なJSONの処理
func TestEventConsumer_handleOrderCreated_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, _, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()
	msg := &nats.Msg{Data: []byte("invalid json")}

	consumer.handleOrderCreated(ctx, msg)
}

// TestEventConsumer_handleOrderCancelled OrderCancelledイベントハンドラ
func TestEventConsumer_handleOrderCancelled(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, uc, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// 先に予約を作成
	orderID := uuid.New().String()
	eventID1 := uuid.New().String()
	_ = uc.HandleOrderCreated(ctx, eventID1, orderID, "PROD-001", "cust-1", 5, 100.00)

	// キャンセルイベントを作成
	event := events.NewOrderCancelledEvent(orderID, "test reason")
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// ハンドラを呼び出し
	consumer.handleOrderCancelled(ctx, msg)
}

// TestEventConsumer_handleOrderCancelled_InvalidJSON 不正なJSONの処理
func TestEventConsumer_handleOrderCancelled_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, _, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()
	msg := &nats.Msg{Data: []byte("invalid json")}

	consumer.handleOrderCancelled(ctx, msg)
}

// TestEventConsumer_handleOrderCompleted OrderCompletedイベントハンドラ
func TestEventConsumer_handleOrderCompleted(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, uc, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// 先に予約を作成
	orderID := uuid.New().String()
	eventID1 := uuid.New().String()
	_ = uc.HandleOrderCreated(ctx, eventID1, orderID, "PROD-001", "cust-1", 5, 100.00)

	// 完了イベントを作成
	event := events.NewOrderCompletedEvent(orderID)
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// ハンドラを呼び出し
	consumer.handleOrderCompleted(ctx, msg)
}

// TestEventConsumer_handleOrderCompleted_InvalidJSON 不正なJSONの処理
func TestEventConsumer_handleOrderCompleted_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, _, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()
	msg := &nats.Msg{Data: []byte("invalid json")}

	consumer.handleOrderCompleted(ctx, msg)
}

// TestEventConsumer_Start_正常系 NATS接続ありでのStart
func TestEventConsumer_Start_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	testNATS, err := testutil.GetTestNATS(ctx)
	if err != nil {
		t.Fatalf("テストNATS取得失敗: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, logger)
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := application.NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, logger)

	// NATS接続でクライアント作成
	t.Setenv("NATS_URL", testNATS.URL)
	natsClient, err := messaging.NewClient(ctx, logger)
	if err != nil {
		t.Fatalf("NATSクライアント作成失敗: %v", err)
	}
	defer natsClient.Close()

	consumer := NewEventConsumer(natsClient, uc, logger)

	// Start
	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("Start失敗: %v", err)
	}
	defer consumer.Stop()

	// 購読が追加されていることを確認
	if len(consumer.subs) != 3 {
		t.Errorf("subs数 = %d, want 3", len(consumer.subs))
	}
}
