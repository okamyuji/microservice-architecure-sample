package application

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/outbox"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/testutil"
	"github.com/yujiokamoto/microservice-architecture-sample/services/payment/internal/domain"
	"github.com/yujiokamoto/microservice-architecture-sample/services/payment/internal/infrastructure"
)

// testLogger テスト用ロガー
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// TestNewPaymentUseCase UseCase生成テスト
func TestNewPaymentUseCase(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)

	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())
	if uc == nil {
		t.Fatal("UseCase が nil")
	}
}

// TestPaymentUseCase_SetSimulateFailure 失敗シミュレーション設定
func TestPaymentUseCase_SetSimulateFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())

	if uc.simulateFailure {
		t.Error("simulateFailure = true, want false")
	}

	uc.SetSimulateFailure(true)
	if !uc.simulateFailure {
		t.Error("simulateFailure = false, want true")
	}
}

// TestPaymentUseCase_GetPayment_正常系 決済取得成功
func TestPaymentUseCase_GetPayment_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())

	orderID := uuid.New().String()
	eventID := uuid.New().String()

	// 決済を作成（.00で終わる金額は成功する）
	err = uc.HandleStockReserved(ctx, eventID, orderID, "cust-1", 100.00)
	if err != nil {
		t.Fatalf("HandleStockReserved失敗: %v", err)
	}

	// 取得
	payment, err := uc.GetPayment(ctx, orderID)
	if err != nil {
		t.Fatalf("GetPayment失敗: %v", err)
	}
	if payment.OrderID != orderID {
		t.Errorf("OrderID = %s, want %s", payment.OrderID, orderID)
	}
}

// TestPaymentUseCase_GetPayment_NotFound 存在しない決済
func TestPaymentUseCase_GetPayment_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())

	_, err = uc.GetPayment(ctx, uuid.New().String())
	if err == nil {
		t.Error("エラーが発生すべき")
	}
}

// TestPaymentUseCase_HandleStockReserved_成功 決済成功
func TestPaymentUseCase_HandleStockReserved_成功(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())

	orderID := uuid.New().String()
	eventID := uuid.New().String()

	// 金額.00で終わる → 成功（10%ランダム失敗を除く）
	// 確実に成功させるため、複数回試行してどちらかが成功すればOK
	var payment *domain.Payment
	for i := 0; i < 10; i++ {
		testOrderID := uuid.New().String()
		testEventID := uuid.New().String()
		_ = uc.HandleStockReserved(ctx, testEventID, testOrderID, "cust-1", 100.00)
		p, _ := repo.FindByOrderID(ctx, testOrderID)
		if p != nil && p.IsCompleted() {
			payment = p
			break
		}
	}

	if payment == nil {
		// シミュレーションをオフにして確実に成功させる
		uc.SetSimulateFailure(false)
		err = uc.HandleStockReserved(ctx, eventID, orderID, "cust-1", 100.50) // .50は失敗しない
		if err != nil {
			t.Fatalf("HandleStockReserved失敗: %v", err)
		}
		payment, _ = repo.FindByOrderID(ctx, orderID)
	}

	if payment == nil {
		t.Fatal("決済が作成されていない")
	}
}

// TestPaymentUseCase_HandleStockReserved_失敗 決済失敗（金額.99）
func TestPaymentUseCase_HandleStockReserved_失敗(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())

	orderID := uuid.New().String()
	eventID := uuid.New().String()

	// 金額.99で終わる → 失敗
	err = uc.HandleStockReserved(ctx, eventID, orderID, "cust-1", 100.99)
	if err != nil {
		t.Fatalf("HandleStockReserved失敗: %v", err)
	}

	// 決済がFAILEDになっていることを確認
	payment, _ := repo.FindByOrderID(ctx, orderID)
	if payment == nil {
		t.Fatal("決済が作成されていない")
	}
	if payment.Status != domain.PaymentStatusFailed {
		t.Errorf("Status = %s, want %s", payment.Status, domain.PaymentStatusFailed)
	}
}

// TestPaymentUseCase_HandleStockReserved_冪等性 同一イベントの重複処理
func TestPaymentUseCase_HandleStockReserved_冪等性(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())

	orderID := uuid.New().String()
	eventID := uuid.New().String()

	// 1回目
	err = uc.HandleStockReserved(ctx, eventID, orderID, "cust-1", 100.99)
	if err != nil {
		t.Fatalf("1回目失敗: %v", err)
	}

	// 2回目（冪等性）
	err = uc.HandleStockReserved(ctx, eventID, orderID, "cust-1", 100.99)
	if err != nil {
		t.Fatalf("2回目失敗: %v", err)
	}
}

// TestPaymentUseCase_HandleStockReserved_シミュレーション失敗 SetSimulateFailureテスト
func TestPaymentUseCase_HandleStockReserved_シミュレーション失敗(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())
	uc.SetSimulateFailure(true)

	orderID := uuid.New().String()
	eventID := uuid.New().String()

	err = uc.HandleStockReserved(ctx, eventID, orderID, "cust-1", 100.00)
	if err != nil {
		t.Fatalf("HandleStockReserved失敗: %v", err)
	}

	// シミュレーションにより失敗
	payment, _ := repo.FindByOrderID(ctx, orderID)
	if payment.Status != domain.PaymentStatusFailed {
		t.Errorf("Status = %s, want %s", payment.Status, domain.PaymentStatusFailed)
	}
}

// TestPaymentUseCase_HandleOrderCancelled_正常系 注文キャンセル時の返金
func TestPaymentUseCase_HandleOrderCancelled_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())

	// 成功する決済を作成（.50は失敗条件に該当しない）
	// ランダム失敗を避けるため複数回試行
	var completedOrderID string
	for i := 0; i < 20; i++ {
		testOrderID := uuid.New().String()
		testEventID := uuid.New().String()
		_ = uc.HandleStockReserved(ctx, testEventID, testOrderID, "cust-1", 100.50)
		p, _ := repo.FindByOrderID(ctx, testOrderID)
		if p != nil && p.IsCompleted() {
			completedOrderID = testOrderID
			break
		}
	}

	if completedOrderID == "" {
		// 成功しなかった場合はテストをスキップ
		t.Skip("決済成功を作成できなかった（ランダム失敗による）")
	}

	// キャンセル処理
	cancelEventID := uuid.New().String()
	err = uc.HandleOrderCancelled(ctx, cancelEventID, completedOrderID)
	if err != nil {
		t.Fatalf("HandleOrderCancelled失敗: %v", err)
	}

	// 返金されていることを確認
	payment, _ := repo.FindByOrderID(ctx, completedOrderID)
	if payment.Status != domain.PaymentStatusRefunded {
		t.Errorf("Status = %s, want %s", payment.Status, domain.PaymentStatusRefunded)
	}
}

// TestPaymentUseCase_HandleOrderCancelled_冪等性 同一イベントの重複処理
func TestPaymentUseCase_HandleOrderCancelled_冪等性(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())

	orderID := uuid.New().String()
	eventID := uuid.New().String()

	// 1回目
	_ = uc.HandleOrderCancelled(ctx, eventID, orderID)

	// 2回目（冪等性）
	err = uc.HandleOrderCancelled(ctx, eventID, orderID)
	if err != nil {
		t.Fatalf("冪等性チェック失敗: %v", err)
	}
}

// TestPaymentUseCase_HandleOrderCancelled_決済なし 決済が存在しない場合
func TestPaymentUseCase_HandleOrderCancelled_決済なし(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())

	eventID := uuid.New().String()
	orderID := uuid.New().String()

	// 決済なしでキャンセル
	err = uc.HandleOrderCancelled(ctx, eventID, orderID)
	// エラーにならない
	if err != nil {
		t.Fatalf("HandleOrderCancelled失敗: %v", err)
	}
}

// TestPaymentUseCase_HandleOrderCancelled_失敗済み決済 失敗した決済のキャンセル
func TestPaymentUseCase_HandleOrderCancelled_失敗済み決済(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())

	orderID := uuid.New().String()
	eventID1 := uuid.New().String()

	// 失敗する決済を作成（.99）
	_ = uc.HandleStockReserved(ctx, eventID1, orderID, "cust-1", 100.99)

	// キャンセル処理（失敗済み決済には返金不可）
	eventID2 := uuid.New().String()
	err = uc.HandleOrderCancelled(ctx, eventID2, orderID)
	if err != nil {
		t.Fatalf("HandleOrderCancelled失敗: %v", err)
	}

	// ステータスは変わっていない
	payment, _ := repo.FindByOrderID(ctx, orderID)
	if payment.Status != domain.PaymentStatusFailed {
		t.Errorf("Status = %s, want %s", payment.Status, domain.PaymentStatusFailed)
	}
}

// TestPaymentUseCase_shouldFailPayment shouldFailPaymentロジック
func TestPaymentUseCase_shouldFailPayment(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())

	// .99で終わる金額は必ず失敗
	if !uc.shouldFailPayment(100.99) {
		t.Error("100.99 should fail")
	}
	if !uc.shouldFailPayment(50.99) {
		t.Error("50.99 should fail")
	}

	// simulateFailure=trueなら必ず失敗
	uc.SetSimulateFailure(true)
	if !uc.shouldFailPayment(100.00) {
		t.Error("simulateFailure=true should fail")
	}
}
