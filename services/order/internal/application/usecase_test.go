package application

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/outbox"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/testutil"
	"github.com/yujiokamoto/microservice-architecture-sample/services/order/internal/domain"
	"github.com/yujiokamoto/microservice-architecture-sample/services/order/internal/infrastructure"
)

// testLogger テスト用ロガー
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// TestNewOrderUseCase UseCase生成テスト
func TestNewOrderUseCase(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)

	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())
	if uc == nil {
		t.Fatal("UseCase が nil")
	}
}

// TestOrderUseCase_CreateOrder_正常系 注文作成成功
func TestOrderUseCase_CreateOrder_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	input := CreateOrderInput{
		CustomerID:  "cust-1",
		ProductID:   "prod-1",
		Quantity:    5,
		TotalAmount: 100.50,
	}

	output, err := uc.CreateOrder(ctx, input)
	if err != nil {
		t.Fatalf("CreateOrder失敗: %v", err)
	}

	if output.OrderID == "" {
		t.Error("OrderID が空")
	}
	if output.Status != domain.OrderStatusPending {
		t.Errorf("Status = %s, want %s", output.Status, domain.OrderStatusPending)
	}

	// DBに保存されていることを確認
	order, err := repo.FindByID(ctx, output.OrderID)
	if err != nil {
		t.Fatalf("FindByID失敗: %v", err)
	}
	if order.CustomerID != input.CustomerID {
		t.Errorf("CustomerID = %s, want %s", order.CustomerID, input.CustomerID)
	}
}

// TestOrderUseCase_CreateOrder_バリデーションエラー 無効な入力
func TestOrderUseCase_CreateOrder_バリデーションエラー(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	tests := []struct {
		name  string
		input CreateOrderInput
	}{
		{
			name: "CustomerID空",
			input: CreateOrderInput{
				CustomerID:  "",
				ProductID:   "prod-1",
				Quantity:    5,
				TotalAmount: 100,
			},
		},
		{
			name: "ProductID空",
			input: CreateOrderInput{
				CustomerID:  "cust-1",
				ProductID:   "",
				Quantity:    5,
				TotalAmount: 100,
			},
		},
		{
			name: "Quantity0",
			input: CreateOrderInput{
				CustomerID:  "cust-1",
				ProductID:   "prod-1",
				Quantity:    0,
				TotalAmount: 100,
			},
		},
		{
			name: "TotalAmount0",
			input: CreateOrderInput{
				CustomerID:  "cust-1",
				ProductID:   "prod-1",
				Quantity:    5,
				TotalAmount: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.CreateOrder(ctx, tt.input)
			if err == nil {
				t.Error("エラーが発生すべき")
			}
		})
	}
}

// TestOrderUseCase_GetOrder_正常系 注文取得成功
func TestOrderUseCase_GetOrder_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	// 注文作成
	input := CreateOrderInput{
		CustomerID:  "cust-get",
		ProductID:   "prod-get",
		Quantity:    3,
		TotalAmount: 50.00,
	}
	output, err := uc.CreateOrder(ctx, input)
	if err != nil {
		t.Fatalf("CreateOrder失敗: %v", err)
	}

	// 取得
	order, err := uc.GetOrder(ctx, output.OrderID)
	if err != nil {
		t.Fatalf("GetOrder失敗: %v", err)
	}
	if order.ID != output.OrderID {
		t.Errorf("ID = %s, want %s", order.ID, output.OrderID)
	}
}

// TestOrderUseCase_GetOrder_NotFound 存在しない注文
func TestOrderUseCase_GetOrder_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	_, err = uc.GetOrder(ctx, uuid.New().String())
	if err == nil {
		t.Error("エラーが発生すべき")
	}
	if !errors.Is(err, domain.ErrOrderNotFound) {
		t.Errorf("err = %v, want ErrOrderNotFound wrapped", err)
	}
}

// TestOrderUseCase_HandleStockReserved_正常系 在庫確保イベント処理
func TestOrderUseCase_HandleStockReserved_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	// 注文作成
	output, err := uc.CreateOrder(ctx, CreateOrderInput{
		CustomerID:  "cust-stock",
		ProductID:   "prod-stock",
		Quantity:    2,
		TotalAmount: 200.00,
	})
	if err != nil {
		t.Fatalf("CreateOrder失敗: %v", err)
	}

	// StockReservedイベント処理
	eventID := uuid.New().String()
	err = uc.HandleStockReserved(ctx, eventID, output.OrderID)
	if err != nil {
		t.Fatalf("HandleStockReserved失敗: %v", err)
	}

	// 注文がCONFIRMEDになっていることを確認
	order, err := repo.FindByID(ctx, output.OrderID)
	if err != nil {
		t.Fatalf("FindByID失敗: %v", err)
	}
	if order.Status != domain.OrderStatusConfirmed {
		t.Errorf("Status = %s, want %s", order.Status, domain.OrderStatusConfirmed)
	}
}

// TestOrderUseCase_HandleStockReserved_冪等性 同一イベントの重複処理
func TestOrderUseCase_HandleStockReserved_冪等性(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	output, err := uc.CreateOrder(ctx, CreateOrderInput{
		CustomerID:  "cust-idem",
		ProductID:   "prod-idem",
		Quantity:    1,
		TotalAmount: 10.00,
	})
	if err != nil {
		t.Fatalf("CreateOrder失敗: %v", err)
	}

	eventID := uuid.New().String()

	// 1回目の処理
	err = uc.HandleStockReserved(ctx, eventID, output.OrderID)
	if err != nil {
		t.Fatalf("1回目のHandleStockReserved失敗: %v", err)
	}

	// 2回目の処理（冪等性により成功すべき）
	err = uc.HandleStockReserved(ctx, eventID, output.OrderID)
	if err != nil {
		t.Fatalf("2回目のHandleStockReserved失敗: %v", err)
	}
}

// TestOrderUseCase_HandlePaymentCompleted_正常系 決済完了イベント処理
func TestOrderUseCase_HandlePaymentCompleted_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	// 注文作成→在庫確保
	output, _ := uc.CreateOrder(ctx, CreateOrderInput{
		CustomerID:  "cust-pay",
		ProductID:   "prod-pay",
		Quantity:    1,
		TotalAmount: 50.00,
	})
	_ = uc.HandleStockReserved(ctx, uuid.New().String(), output.OrderID)

	// PaymentCompletedイベント処理
	eventID := uuid.New().String()
	err = uc.HandlePaymentCompleted(ctx, eventID, output.OrderID)
	if err != nil {
		t.Fatalf("HandlePaymentCompleted失敗: %v", err)
	}

	// 注文がCOMPLETEDになっていることを確認
	order, _ := repo.FindByID(ctx, output.OrderID)
	if order.Status != domain.OrderStatusCompleted {
		t.Errorf("Status = %s, want %s", order.Status, domain.OrderStatusCompleted)
	}
}

// TestOrderUseCase_HandlePaymentFailed_正常系 決済失敗イベント処理
func TestOrderUseCase_HandlePaymentFailed_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	// 注文作成→在庫確保
	output, _ := uc.CreateOrder(ctx, CreateOrderInput{
		CustomerID:  "cust-fail",
		ProductID:   "prod-fail",
		Quantity:    1,
		TotalAmount: 50.00,
	})
	_ = uc.HandleStockReserved(ctx, uuid.New().String(), output.OrderID)

	// PaymentFailedイベント処理
	eventID := uuid.New().String()
	err = uc.HandlePaymentFailed(ctx, eventID, output.OrderID, "card declined")
	if err != nil {
		t.Fatalf("HandlePaymentFailed失敗: %v", err)
	}

	// 注文がCANCELLEDになっていることを確認
	order, _ := repo.FindByID(ctx, output.OrderID)
	if order.Status != domain.OrderStatusCancelled {
		t.Errorf("Status = %s, want %s", order.Status, domain.OrderStatusCancelled)
	}
}

// TestOrderUseCase_HandleStockReserveFailed_正常系 在庫確保失敗イベント処理
func TestOrderUseCase_HandleStockReserveFailed_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	// 注文作成
	output, _ := uc.CreateOrder(ctx, CreateOrderInput{
		CustomerID:  "cust-stock-fail",
		ProductID:   "prod-stock-fail",
		Quantity:    1,
		TotalAmount: 50.00,
	})

	// StockReserveFailedイベント処理
	eventID := uuid.New().String()
	err = uc.HandleStockReserveFailed(ctx, eventID, output.OrderID, "insufficient stock")
	if err != nil {
		t.Fatalf("HandleStockReserveFailed失敗: %v", err)
	}

	// 注文がCANCELLEDになっていることを確認
	order, _ := repo.FindByID(ctx, output.OrderID)
	if order.Status != domain.OrderStatusCancelled {
		t.Errorf("Status = %s, want %s", order.Status, domain.OrderStatusCancelled)
	}
}

// TestOrderUseCase_HandleStockReserved_NotFound 存在しない注文
func TestOrderUseCase_HandleStockReserved_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	eventID := uuid.New().String()
	err = uc.HandleStockReserved(ctx, eventID, uuid.New().String())
	if err == nil {
		t.Error("エラーが発生すべき")
	}
}

// TestOrderUseCase_HandlePaymentCompleted_冪等性 同一イベントの重複処理
func TestOrderUseCase_HandlePaymentCompleted_冪等性(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	output, _ := uc.CreateOrder(ctx, CreateOrderInput{
		CustomerID:  "cust-pay-idem",
		ProductID:   "prod-pay-idem",
		Quantity:    1,
		TotalAmount: 50.00,
	})
	_ = uc.HandleStockReserved(ctx, uuid.New().String(), output.OrderID)

	eventID := uuid.New().String()

	// 1回目
	_ = uc.HandlePaymentCompleted(ctx, eventID, output.OrderID)

	// 2回目（冪等性）
	err = uc.HandlePaymentCompleted(ctx, eventID, output.OrderID)
	if err != nil {
		t.Fatalf("冪等性チェック失敗: %v", err)
	}
}

// TestOrderUseCase_HandlePaymentFailed_冪等性 同一イベントの重複処理
func TestOrderUseCase_HandlePaymentFailed_冪等性(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	output, _ := uc.CreateOrder(ctx, CreateOrderInput{
		CustomerID:  "cust-fail-idem",
		ProductID:   "prod-fail-idem",
		Quantity:    1,
		TotalAmount: 50.00,
	})
	_ = uc.HandleStockReserved(ctx, uuid.New().String(), output.OrderID)

	eventID := uuid.New().String()

	// 1回目
	_ = uc.HandlePaymentFailed(ctx, eventID, output.OrderID, "reason")

	// 2回目（冪等性）
	err = uc.HandlePaymentFailed(ctx, eventID, output.OrderID, "reason")
	if err != nil {
		t.Fatalf("冪等性チェック失敗: %v", err)
	}
}

// TestOrderUseCase_HandleStockReserveFailed_冪等性 同一イベントの重複処理
func TestOrderUseCase_HandleStockReserveFailed_冪等性(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	output, _ := uc.CreateOrder(ctx, CreateOrderInput{
		CustomerID:  "cust-srf-idem",
		ProductID:   "prod-srf-idem",
		Quantity:    1,
		TotalAmount: 50.00,
	})

	eventID := uuid.New().String()

	// 1回目
	_ = uc.HandleStockReserveFailed(ctx, eventID, output.OrderID, "reason")

	// 2回目（冪等性）
	err = uc.HandleStockReserveFailed(ctx, eventID, output.OrderID, "reason")
	if err != nil {
		t.Fatalf("冪等性チェック失敗: %v", err)
	}
}

// TestCreateOrderInput 入力構造体テスト
func TestCreateOrderInput(t *testing.T) {
	input := CreateOrderInput{
		CustomerID:  "cust-1",
		ProductID:   "prod-1",
		Quantity:    5,
		TotalAmount: 100.50,
	}

	if input.CustomerID != "cust-1" {
		t.Errorf("CustomerID = %s, want cust-1", input.CustomerID)
	}
	if input.ProductID != "prod-1" {
		t.Errorf("ProductID = %s, want prod-1", input.ProductID)
	}
	if input.Quantity != 5 {
		t.Errorf("Quantity = %d, want 5", input.Quantity)
	}
	if input.TotalAmount != 100.50 {
		t.Errorf("TotalAmount = %f, want 100.50", input.TotalAmount)
	}
}

// TestCreateOrderOutput 出力構造体テスト
func TestCreateOrderOutput(t *testing.T) {
	output := CreateOrderOutput{
		OrderID: "order-1",
		Status:  "PENDING",
	}

	if output.OrderID != "order-1" {
		t.Errorf("OrderID = %s, want order-1", output.OrderID)
	}
	if output.Status != "PENDING" {
		t.Errorf("Status = %s, want PENDING", output.Status)
	}
}

// TestOrderUseCase_HandleStockReserved_InvalidTransition 既にConfirmedの場合
func TestOrderUseCase_HandleStockReserved_InvalidTransition(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	// 注文作成
	output, _ := uc.CreateOrder(ctx, CreateOrderInput{
		CustomerID:  "cust-inv",
		ProductID:   "prod-inv",
		Quantity:    1,
		TotalAmount: 10.00,
	})

	// 一度Confirmする
	eventID1 := uuid.New().String()
	_ = uc.HandleStockReserved(ctx, eventID1, output.OrderID)

	// 再度Confirmしようとする（異なるイベントID）
	eventID2 := uuid.New().String()
	err = uc.HandleStockReserved(ctx, eventID2, output.OrderID)
	// エラーにならず、スキップされる
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
}

// TestOrderUseCase_HandlePaymentCompleted_InvalidTransition PENDINGの場合
func TestOrderUseCase_HandlePaymentCompleted_InvalidTransition(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	// 注文作成（CONFIRMEDにしない）
	output, _ := uc.CreateOrder(ctx, CreateOrderInput{
		CustomerID:  "cust-inv-pay",
		ProductID:   "prod-inv-pay",
		Quantity:    1,
		TotalAmount: 10.00,
	})

	// PaymentCompletedを処理しようとする
	eventID := uuid.New().String()
	err = uc.HandlePaymentCompleted(ctx, eventID, output.OrderID)
	// エラーにならず、スキップされる
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}

	// ステータスは変わっていない
	order, _ := repo.FindByID(ctx, output.OrderID)
	if order.Status != domain.OrderStatusPending {
		t.Errorf("Status = %s, want %s", order.Status, domain.OrderStatusPending)
	}
}

// TestOrderUseCase_HandlePaymentFailed_InvalidTransition 既にCANCELLEDの場合
func TestOrderUseCase_HandlePaymentFailed_InvalidTransition(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())

	// 注文作成→キャンセル
	output, _ := uc.CreateOrder(ctx, CreateOrderInput{
		CustomerID:  "cust-inv-fail",
		ProductID:   "prod-inv-fail",
		Quantity:    1,
		TotalAmount: 10.00,
	})

	// 直接DBでキャンセル状態にする
	order, _ := repo.FindByID(ctx, output.OrderID)
	order.Status = domain.OrderStatusCancelled
	order.UpdatedAt = time.Now()
	tx, _ := db.Pool.Begin(ctx)
	_ = repo.Update(ctx, tx, order)
	_ = tx.Commit(ctx)

	// PaymentFailedを処理しようとする
	eventID := uuid.New().String()
	err = uc.HandlePaymentFailed(ctx, eventID, output.OrderID, "reason")
	// エラーにならず、スキップされる
	if err != nil {
		t.Errorf("予期しないエラー: %v", err)
	}
}
