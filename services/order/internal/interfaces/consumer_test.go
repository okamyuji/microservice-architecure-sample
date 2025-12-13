package interfaces

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"microservice-architecture-sample/pkg/events"
	"microservice-architecture-sample/pkg/messaging"
	"microservice-architecture-sample/pkg/outbox"
	"microservice-architecture-sample/pkg/testutil"
	"microservice-architecture-sample/services/order/internal/application"
	"microservice-architecture-sample/services/order/internal/infrastructure"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// setupTestConsumer テスト用コンシューマをセットアップ（NATSなし）
func setupTestConsumer(t *testing.T) (*EventConsumer, *application.OrderUseCase, func()) {
	t.Helper()

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, logger)
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := application.NewOrderUseCase(db.Pool, repo, publisher, checker, logger)

	consumer := &EventConsumer{
		client:  nil, // NATSなし
		useCase: uc,
		logger:  logger,
		subs:    nil,
	}

	cleanup := func() {
		_ = db.CleanupOrderDB(ctx)
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

	// panicしないことを確認
	consumer.Stop()
}

// TestEventConsumer_handleStockReserved StockReservedイベントハンドラ
func TestEventConsumer_handleStockReserved(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, uc, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// 注文を作成
	output, err := uc.CreateOrder(ctx, application.CreateOrderInput{
		CustomerID:  "cust-1",
		ProductID:   "prod-1",
		Quantity:    5,
		TotalAmount: 100.00,
	})
	if err != nil {
		t.Fatalf("CreateOrder失敗: %v", err)
	}

	// イベントを作成
	event := events.NewStockReservedEvent(output.OrderID, "prod-1", 5, "res-1", "cust-1", 100.00)
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// ハンドラを呼び出し
	consumer.handleStockReserved(ctx, msg)

	// 注文がCONFIRMEDになっていることを確認
	order, _ := uc.GetOrder(ctx, output.OrderID)
	if order.Status != "CONFIRMED" {
		t.Errorf("Status = %s, want CONFIRMED", order.Status)
	}
}

// TestEventConsumer_handleStockReserved_InvalidJSON 不正なJSONの処理
func TestEventConsumer_handleStockReserved_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, _, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()
	msg := &nats.Msg{Data: []byte("invalid json")}

	// panicしないことを確認
	consumer.handleStockReserved(ctx, msg)
}

// TestEventConsumer_handleStockReserveFailed StockReserveFailedイベントハンドラ
func TestEventConsumer_handleStockReserveFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, uc, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// 注文を作成
	output, err := uc.CreateOrder(ctx, application.CreateOrderInput{
		CustomerID:  "cust-1",
		ProductID:   "prod-1",
		Quantity:    5,
		TotalAmount: 100.00,
	})
	if err != nil {
		t.Fatalf("CreateOrder失敗: %v", err)
	}

	// イベントを作成
	event := events.NewStockReserveFailedEvent(output.OrderID, "prod-1", 5, "insufficient stock")
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// ハンドラを呼び出し
	consumer.handleStockReserveFailed(ctx, msg)

	// 注文がCANCELLEDになっていることを確認
	order, _ := uc.GetOrder(ctx, output.OrderID)
	if order.Status != "CANCELLED" {
		t.Errorf("Status = %s, want CANCELLED", order.Status)
	}
}

// TestEventConsumer_handleStockReserveFailed_InvalidJSON 不正なJSONの処理
func TestEventConsumer_handleStockReserveFailed_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, _, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()
	msg := &nats.Msg{Data: []byte("invalid json")}

	// panicしないことを確認
	consumer.handleStockReserveFailed(ctx, msg)
}

// TestEventConsumer_handlePaymentCompleted PaymentCompletedイベントハンドラ
func TestEventConsumer_handlePaymentCompleted(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, uc, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// 注文を作成→確認
	output, _ := uc.CreateOrder(ctx, application.CreateOrderInput{
		CustomerID:  "cust-1",
		ProductID:   "prod-1",
		Quantity:    5,
		TotalAmount: 100.00,
	})
	_ = uc.HandleStockReserved(ctx, uuid.New().String(), output.OrderID)

	// イベントを作成
	event := events.NewPaymentCompletedEvent(output.OrderID, "pay-1", 100.00)
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// ハンドラを呼び出し
	consumer.handlePaymentCompleted(ctx, msg)

	// 注文がCOMPLETEDになっていることを確認
	order, _ := uc.GetOrder(ctx, output.OrderID)
	if order.Status != "COMPLETED" {
		t.Errorf("Status = %s, want COMPLETED", order.Status)
	}
}

// TestEventConsumer_handlePaymentCompleted_InvalidJSON 不正なJSONの処理
func TestEventConsumer_handlePaymentCompleted_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, _, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()
	msg := &nats.Msg{Data: []byte("invalid json")}

	// panicしないことを確認
	consumer.handlePaymentCompleted(ctx, msg)
}

// TestEventConsumer_handlePaymentFailed PaymentFailedイベントハンドラ
func TestEventConsumer_handlePaymentFailed(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, uc, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// 注文を作成→確認
	output, _ := uc.CreateOrder(ctx, application.CreateOrderInput{
		CustomerID:  "cust-1",
		ProductID:   "prod-1",
		Quantity:    5,
		TotalAmount: 100.00,
	})
	_ = uc.HandleStockReserved(ctx, uuid.New().String(), output.OrderID)

	// イベントを作成
	event := events.NewPaymentFailedEvent(output.OrderID, "card declined")
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// ハンドラを呼び出し
	consumer.handlePaymentFailed(ctx, msg)

	// 注文がCANCELLEDになっていることを確認
	order, _ := uc.GetOrder(ctx, output.OrderID)
	if order.Status != "CANCELLED" {
		t.Errorf("Status = %s, want CANCELLED", order.Status)
	}
}

// TestEventConsumer_handlePaymentFailed_InvalidJSON 不正なJSONの処理
func TestEventConsumer_handlePaymentFailed_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, _, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()
	msg := &nats.Msg{Data: []byte("invalid json")}

	// panicしないことを確認
	consumer.handlePaymentFailed(ctx, msg)
}

// TestEventConsumer_Start_正常系 NATS接続ありでのStart
func TestEventConsumer_Start_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	testNATS, err := testutil.GetTestNATS(ctx)
	if err != nil {
		t.Fatalf("テストNATS取得失敗: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, logger)
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := application.NewOrderUseCase(db.Pool, repo, publisher, checker, logger)

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
	if len(consumer.subs) != 4 {
		t.Errorf("subs数 = %d, want 4", len(consumer.subs))
	}
}

// TestEventConsumer_handleStockReserved_HandleError ハンドラがエラーを返す場合
func TestEventConsumer_handleStockReserved_HandleError(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, _, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// 存在しない注文へのイベント
	event := events.NewStockReservedEvent(uuid.New().String(), "prod-1", 5, "res-1", "cust-1", 100.00)
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// エラーが発生してもpanicしないことを確認
	consumer.handleStockReserved(ctx, msg)
}

// TestEventConsumer_handlePaymentCompleted_HandleError ハンドラがエラーを返す場合
func TestEventConsumer_handlePaymentCompleted_HandleError(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, _, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// 存在しない注文へのイベント
	event := events.NewPaymentCompletedEvent(uuid.New().String(), "pay-1", 100.00)
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// エラーが発生してもpanicしないことを確認
	consumer.handlePaymentCompleted(ctx, msg)
}

// TestEventConsumer_handlePaymentFailed_HandleError ハンドラがエラーを返す場合
func TestEventConsumer_handlePaymentFailed_HandleError(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, _, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// 存在しない注文へのイベント
	event := events.NewPaymentFailedEvent(uuid.New().String(), "card declined")
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// エラーが発生してもpanicしないことを確認
	consumer.handlePaymentFailed(ctx, msg)
}

// TestEventConsumer_handleStockReserveFailed_HandleError ハンドラがエラーを返す場合
func TestEventConsumer_handleStockReserveFailed_HandleError(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, _, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// 存在しない注文へのイベント
	event := events.NewStockReserveFailedEvent(uuid.New().String(), "prod-1", 5, "insufficient stock")
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// エラーが発生してもpanicしないことを確認
	consumer.handleStockReserveFailed(ctx, msg)
}
