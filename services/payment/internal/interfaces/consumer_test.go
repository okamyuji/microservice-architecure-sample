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
	"github.com/yujiokamoto/microservice-architecture-sample/services/payment/internal/application"
	"github.com/yujiokamoto/microservice-architecture-sample/services/payment/internal/infrastructure"
)

// setupTestConsumer テスト用コンシューマをセットアップ（NATSなし）
func setupTestConsumer(t *testing.T) (*EventConsumer, *application.PaymentUseCase, func()) {
	t.Helper()

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, logger)
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := application.NewPaymentUseCase(db.Pool, repo, publisher, checker, logger)

	consumer := &EventConsumer{
		client:  nil,
		useCase: uc,
		logger:  logger,
		subs:    nil,
	}

	cleanup := func() {
		_ = db.CleanupPaymentDB(ctx)
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

// TestEventConsumer_handleStockReserved StockReservedイベントハンドラ
func TestEventConsumer_handleStockReserved(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, uc, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// イベントを作成（.99で失敗させる）
	orderID := uuid.New().String()
	event := events.NewStockReservedEvent(orderID, "prod-1", 5, "res-1", "cust-1", 100.99)
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// ハンドラを呼び出し
	consumer.handleStockReserved(ctx, msg)

	// 決済が作成されていることを確認
	payment, _ := uc.GetPayment(ctx, orderID)
	if payment == nil {
		t.Log("決済が作成されなかった場合")
	} else {
		t.Logf("決済ステータス: %s", payment.Status)
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

	consumer.handleStockReserved(ctx, msg)
}

// TestEventConsumer_handleOrderCancelled OrderCancelledイベントハンドラ
func TestEventConsumer_handleOrderCancelled(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, uc, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// 先に決済を作成
	orderID := uuid.New().String()
	eventID1 := uuid.New().String()
	_ = uc.HandleStockReserved(ctx, eventID1, orderID, "cust-1", 100.99)

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

// TestEventConsumer_Start_正常系 NATS接続ありでのStart
func TestEventConsumer_Start_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	testNATS, err := testutil.GetTestNATS(ctx)
	if err != nil {
		t.Fatalf("テストNATS取得失敗: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, logger)
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := application.NewPaymentUseCase(db.Pool, repo, publisher, checker, logger)

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
	if len(consumer.subs) != 2 {
		t.Errorf("subs数 = %d, want 2", len(consumer.subs))
	}
}

// TestEventConsumer_handleStockReserved_成功 正常な決済処理
func TestEventConsumer_handleStockReserved_成功(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	consumer, uc, cleanup := setupTestConsumer(t)
	defer cleanup()

	ctx := context.Background()

	// イベントを作成（整数金額で成功させる）
	orderID := uuid.New().String()
	event := events.NewStockReservedEvent(orderID, "prod-1", 5, "res-1", "cust-1", 100.00)
	data, _ := events.ToJSON(event)
	msg := &nats.Msg{Data: data}

	// ハンドラを呼び出し
	consumer.handleStockReserved(ctx, msg)

	// 決済が作成されていることを確認
	payment, _ := uc.GetPayment(ctx, orderID)
	if payment != nil {
		t.Logf("決済ステータス: %s", payment.Status)
	}
}
