package application

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"microservice-architecture-sample/pkg/outbox"
	"microservice-architecture-sample/pkg/testutil"
	"microservice-architecture-sample/services/inventory/internal/domain"
	"microservice-architecture-sample/services/inventory/internal/infrastructure"

	"github.com/google/uuid"
)

// testLogger テスト用ロガー
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// TestNewInventoryUseCase UseCase生成テスト
func TestNewInventoryUseCase(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)

	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())
	if uc == nil {
		t.Fatal("UseCase が nil")
	}
}

// TestInventoryUseCase_GetInventory_正常系 在庫取得成功
func TestInventoryUseCase_GetInventory_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	inv, err := uc.GetInventory(ctx, "PROD-001")
	if err != nil {
		t.Fatalf("GetInventory失敗: %v", err)
	}
	if inv.ProductID != "PROD-001" {
		t.Errorf("ProductID = %s, want PROD-001", inv.ProductID)
	}
}

// TestInventoryUseCase_GetInventory_NotFound 存在しない商品
func TestInventoryUseCase_GetInventory_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	_, err = uc.GetInventory(ctx, "PROD-NOTEXIST")
	if err == nil {
		t.Error("エラーが発生すべき")
	}
}

// TestInventoryUseCase_HandleOrderCreated_正常系 注文作成イベント処理成功
func TestInventoryUseCase_HandleOrderCreated_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	eventID := uuid.New().String()
	orderID := uuid.New().String()

	err = uc.HandleOrderCreated(ctx, eventID, orderID, "PROD-001", "cust-1", 5, 100.00)
	if err != nil {
		t.Fatalf("HandleOrderCreated失敗: %v", err)
	}

	// 予約が作成されていることを確認
	reservations, err := resRepo.FindByOrderID(ctx, orderID)
	if err != nil {
		t.Fatalf("FindByOrderID失敗: %v", err)
	}
	if len(reservations) != 1 {
		t.Fatalf("予約数 = %d, want 1", len(reservations))
	}
	if reservations[0].Quantity != 5 {
		t.Errorf("Quantity = %d, want 5", reservations[0].Quantity)
	}
}

// TestInventoryUseCase_HandleOrderCreated_冪等性 同一イベントの重複処理
func TestInventoryUseCase_HandleOrderCreated_冪等性(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	eventID := uuid.New().String()
	orderID := uuid.New().String()

	// 1回目
	err = uc.HandleOrderCreated(ctx, eventID, orderID, "PROD-001", "cust-1", 5, 100.00)
	if err != nil {
		t.Fatalf("1回目失敗: %v", err)
	}

	// 2回目（冪等性）
	err = uc.HandleOrderCreated(ctx, eventID, orderID, "PROD-001", "cust-1", 5, 100.00)
	if err != nil {
		t.Fatalf("2回目失敗: %v", err)
	}

	// 予約は1つだけ
	reservations, _ := resRepo.FindByOrderID(ctx, orderID)
	if len(reservations) != 1 {
		t.Errorf("予約数 = %d, want 1", len(reservations))
	}
}

// TestInventoryUseCase_HandleOrderCreated_商品なし 存在しない商品
func TestInventoryUseCase_HandleOrderCreated_商品なし(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	eventID := uuid.New().String()
	orderID := uuid.New().String()

	// 存在しない商品
	err = uc.HandleOrderCreated(ctx, eventID, orderID, "PROD-NOTEXIST", "cust-1", 5, 100.00)
	// エラーにならず、失敗イベントが発行される
	if err != nil {
		t.Fatalf("HandleOrderCreated失敗: %v", err)
	}
}

// TestInventoryUseCase_HandleOrderCreated_在庫不足 在庫不足
func TestInventoryUseCase_HandleOrderCreated_在庫不足(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	eventID := uuid.New().String()
	orderID := uuid.New().String()

	// 大量注文（在庫不足）
	err = uc.HandleOrderCreated(ctx, eventID, orderID, "PROD-001", "cust-1", 99999, 100000.00)
	// エラーにならず、失敗イベントが発行される
	if err != nil {
		t.Fatalf("HandleOrderCreated失敗: %v", err)
	}
}

// TestInventoryUseCase_HandleOrderCancelled_正常系 注文キャンセルイベント処理成功
func TestInventoryUseCase_HandleOrderCancelled_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	eventID1 := uuid.New().String()
	orderID := uuid.New().String()

	// まず予約を作成
	err = uc.HandleOrderCreated(ctx, eventID1, orderID, "PROD-001", "cust-1", 5, 100.00)
	if err != nil {
		t.Fatalf("HandleOrderCreated失敗: %v", err)
	}

	// キャンセル処理
	eventID2 := uuid.New().String()
	err = uc.HandleOrderCancelled(ctx, eventID2, orderID)
	if err != nil {
		t.Fatalf("HandleOrderCancelled失敗: %v", err)
	}

	// 予約がRELEASEDになっていることを確認
	reservations, _ := resRepo.FindByOrderID(ctx, orderID)
	if len(reservations) != 1 {
		t.Fatalf("予約数 = %d, want 1", len(reservations))
	}
	if reservations[0].Status != domain.ReservationStatusReleased {
		t.Errorf("Status = %s, want %s", reservations[0].Status, domain.ReservationStatusReleased)
	}
}

// TestInventoryUseCase_HandleOrderCancelled_冪等性 同一イベントの重複処理
func TestInventoryUseCase_HandleOrderCancelled_冪等性(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	orderID := uuid.New().String()
	_ = uc.HandleOrderCreated(ctx, uuid.New().String(), orderID, "PROD-001", "cust-1", 5, 100.00)

	eventID := uuid.New().String()

	// 1回目
	_ = uc.HandleOrderCancelled(ctx, eventID, orderID)

	// 2回目（冪等性）
	err = uc.HandleOrderCancelled(ctx, eventID, orderID)
	if err != nil {
		t.Fatalf("冪等性チェック失敗: %v", err)
	}
}

// TestInventoryUseCase_HandleOrderCancelled_予約なし 予約が存在しない場合
func TestInventoryUseCase_HandleOrderCancelled_予約なし(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	eventID := uuid.New().String()
	orderID := uuid.New().String()

	// 予約なしでキャンセル
	err = uc.HandleOrderCancelled(ctx, eventID, orderID)
	// エラーにならない
	if err != nil {
		t.Fatalf("HandleOrderCancelled失敗: %v", err)
	}
}

// TestInventoryUseCase_HandleOrderCompleted_正常系 注文完了イベント処理成功
func TestInventoryUseCase_HandleOrderCompleted_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	eventID1 := uuid.New().String()
	orderID := uuid.New().String()

	// 予約を作成
	err = uc.HandleOrderCreated(ctx, eventID1, orderID, "PROD-001", "cust-1", 5, 100.00)
	if err != nil {
		t.Fatalf("HandleOrderCreated失敗: %v", err)
	}

	// 完了処理
	eventID2 := uuid.New().String()
	err = uc.HandleOrderCompleted(ctx, eventID2, orderID)
	if err != nil {
		t.Fatalf("HandleOrderCompleted失敗: %v", err)
	}

	// 予約がCOMMITTEDになっていることを確認
	reservations, _ := resRepo.FindByOrderID(ctx, orderID)
	if len(reservations) != 1 {
		t.Fatalf("予約数 = %d, want 1", len(reservations))
	}
	if reservations[0].Status != domain.ReservationStatusCommitted {
		t.Errorf("Status = %s, want %s", reservations[0].Status, domain.ReservationStatusCommitted)
	}
}

// TestInventoryUseCase_HandleOrderCompleted_冪等性 同一イベントの重複処理
func TestInventoryUseCase_HandleOrderCompleted_冪等性(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	orderID := uuid.New().String()
	_ = uc.HandleOrderCreated(ctx, uuid.New().String(), orderID, "PROD-001", "cust-1", 5, 100.00)

	eventID := uuid.New().String()

	// 1回目
	_ = uc.HandleOrderCompleted(ctx, eventID, orderID)

	// 2回目（冪等性）
	err = uc.HandleOrderCompleted(ctx, eventID, orderID)
	if err != nil {
		t.Fatalf("冪等性チェック失敗: %v", err)
	}
}

// TestInventoryUseCase_HandleOrderCompleted_予約なし 予約が存在しない場合
func TestInventoryUseCase_HandleOrderCompleted_予約なし(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	eventID := uuid.New().String()
	orderID := uuid.New().String()

	// 予約なしで完了
	err = uc.HandleOrderCompleted(ctx, eventID, orderID)
	// エラーにならない
	if err != nil {
		t.Fatalf("HandleOrderCompleted失敗: %v", err)
	}
}

// TestInventoryUseCase_HandleOrderCancelled_既にリリース済み 既にリリースされた予約
func TestInventoryUseCase_HandleOrderCancelled_既にリリース済み(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())

	orderID := uuid.New().String()
	_ = uc.HandleOrderCreated(ctx, uuid.New().String(), orderID, "PROD-001", "cust-1", 5, 100.00)

	// 一度キャンセル
	_ = uc.HandleOrderCancelled(ctx, uuid.New().String(), orderID)

	// 再度キャンセル（異なるイベントID）
	err = uc.HandleOrderCancelled(ctx, uuid.New().String(), orderID)
	// エラーにならない（既にリリース済みはスキップ）
	if err != nil {
		t.Fatalf("HandleOrderCancelled失敗: %v", err)
	}
}
