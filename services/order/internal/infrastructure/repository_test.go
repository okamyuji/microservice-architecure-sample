package infrastructure

import (
	"context"
	"testing"
	"time"

	"microservice-architecture-sample/pkg/testutil"
	"microservice-architecture-sample/services/order/internal/domain"

	"github.com/google/uuid"
)

// TestPostgresOrderRepository_Save 注文保存テスト
func TestPostgresOrderRepository_Save(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := NewPostgresOrderRepository(db.Pool)

	order := &domain.Order{
		ID:          uuid.New().String(),
		CustomerID:  "cust-1",
		ProductID:   "prod-1",
		Quantity:    5,
		TotalAmount: 100.50,
		Status:      domain.OrderStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}

	err = repo.Save(ctx, tx, order)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Save失敗: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 保存されていることを確認
	saved, err := repo.FindByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("FindByID失敗: %v", err)
	}
	if saved.CustomerID != order.CustomerID {
		t.Errorf("CustomerID = %s, want %s", saved.CustomerID, order.CustomerID)
	}
}

// TestPostgresOrderRepository_Update 注文更新テスト
func TestPostgresOrderRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := NewPostgresOrderRepository(db.Pool)

	// 注文を作成
	order := &domain.Order{
		ID:          uuid.New().String(),
		CustomerID:  "cust-1",
		ProductID:   "prod-1",
		Quantity:    5,
		TotalAmount: 100.50,
		Status:      domain.OrderStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Save(ctx, tx, order); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Save失敗: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 更新
	order.Status = domain.OrderStatusConfirmed
	order.UpdatedAt = time.Now()

	tx2, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Update(ctx, tx2, order); err != nil {
		_ = tx2.Rollback(ctx)
		t.Fatalf("Update失敗: %v", err)
	}
	if err := tx2.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 更新されていることを確認
	updated, err := repo.FindByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("FindByID失敗: %v", err)
	}
	if updated.Status != domain.OrderStatusConfirmed {
		t.Errorf("Status = %s, want %s", updated.Status, domain.OrderStatusConfirmed)
	}
}

// TestPostgresOrderRepository_Update_NotFound 存在しない注文の更新
func TestPostgresOrderRepository_Update_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresOrderRepository(db.Pool)

	order := &domain.Order{
		ID:        uuid.New().String(),
		Status:    domain.OrderStatusPending,
		UpdatedAt: time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = repo.Update(ctx, tx, order)
	if err != domain.ErrOrderNotFound {
		t.Errorf("err = %v, want ErrOrderNotFound", err)
	}
}

// TestPostgresOrderRepository_FindByID 注文取得テスト
func TestPostgresOrderRepository_FindByID(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := NewPostgresOrderRepository(db.Pool)

	// 注文を作成
	order := &domain.Order{
		ID:          uuid.New().String(),
		CustomerID:  "cust-find",
		ProductID:   "prod-find",
		Quantity:    3,
		TotalAmount: 50.00,
		Status:      domain.OrderStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Save(ctx, tx, order); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Save失敗: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 取得
	found, err := repo.FindByID(ctx, order.ID)
	if err != nil {
		t.Fatalf("FindByID失敗: %v", err)
	}

	if found.ID != order.ID {
		t.Errorf("ID = %s, want %s", found.ID, order.ID)
	}
	if found.CustomerID != order.CustomerID {
		t.Errorf("CustomerID = %s, want %s", found.CustomerID, order.CustomerID)
	}
	if found.ProductID != order.ProductID {
		t.Errorf("ProductID = %s, want %s", found.ProductID, order.ProductID)
	}
	if found.Quantity != order.Quantity {
		t.Errorf("Quantity = %d, want %d", found.Quantity, order.Quantity)
	}
}

// TestPostgresOrderRepository_FindByID_NotFound 存在しない注文の取得
func TestPostgresOrderRepository_FindByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresOrderRepository(db.Pool)

	_, err = repo.FindByID(ctx, uuid.New().String())
	if err != domain.ErrOrderNotFound {
		t.Errorf("err = %v, want ErrOrderNotFound", err)
	}
}

// TestPostgresOrderRepository_FindByIDForUpdate 行ロック付き取得テスト
func TestPostgresOrderRepository_FindByIDForUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := NewPostgresOrderRepository(db.Pool)

	// 注文を作成
	order := &domain.Order{
		ID:          uuid.New().String(),
		CustomerID:  "cust-lock",
		ProductID:   "prod-lock",
		Quantity:    1,
		TotalAmount: 10.00,
		Status:      domain.OrderStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Save(ctx, tx, order); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Save失敗: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 行ロック付きで取得
	tx2, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	defer func() { _ = tx2.Rollback(ctx) }()

	found, err := repo.FindByIDForUpdate(ctx, tx2, order.ID)
	if err != nil {
		t.Fatalf("FindByIDForUpdate失敗: %v", err)
	}

	if found.ID != order.ID {
		t.Errorf("ID = %s, want %s", found.ID, order.ID)
	}
}

// TestPostgresOrderRepository_FindByIDForUpdate_NotFound 存在しない場合
func TestPostgresOrderRepository_FindByIDForUpdate_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresOrderRepository(db.Pool)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = repo.FindByIDForUpdate(ctx, tx, uuid.New().String())
	if err != domain.ErrOrderNotFound {
		t.Errorf("err = %v, want ErrOrderNotFound", err)
	}
}

// TestNewPostgresOrderRepository リポジトリ生成テスト
func TestNewPostgresOrderRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresOrderRepository(db.Pool)
	if repo == nil {
		t.Error("repo が nil")
	}
}

// TestPostgresOrderRepository_Save_Rollback ロールバック時のテスト
func TestPostgresOrderRepository_Save_Rollback(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := NewPostgresOrderRepository(db.Pool)

	order := &domain.Order{
		ID:          uuid.New().String(),
		CustomerID:  "cust-rollback",
		ProductID:   "prod-rollback",
		Quantity:    1,
		TotalAmount: 10.00,
		Status:      domain.OrderStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}

	if err := repo.Save(ctx, tx, order); err != nil {
		t.Fatalf("Save失敗: %v", err)
	}

	// ロールバック
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("ロールバック失敗: %v", err)
	}

	// 保存されていないことを確認
	_, err = repo.FindByID(ctx, order.ID)
	if err != domain.ErrOrderNotFound {
		t.Error("ロールバック後にデータが残っている")
	}
}

// TestPostgresOrderRepository_MultipleOrders 複数注文の操作テスト
func TestPostgresOrderRepository_MultipleOrders(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	repo := NewPostgresOrderRepository(db.Pool)

	// 複数の注文を作成
	orderIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		order := &domain.Order{
			ID:          uuid.New().String(),
			CustomerID:  "cust-multi",
			ProductID:   "prod-multi",
			Quantity:    i + 1,
			TotalAmount: float64(i+1) * 10.0,
			Status:      domain.OrderStatusPending,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		orderIDs[i] = order.ID

		tx, err := db.Pool.Begin(ctx)
		if err != nil {
			t.Fatalf("トランザクション開始失敗: %v", err)
		}
		if err := repo.Save(ctx, tx, order); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("Save失敗: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("コミット失敗: %v", err)
		}
	}

	// 全て取得できることを確認
	for i, id := range orderIDs {
		found, err := repo.FindByID(ctx, id)
		if err != nil {
			t.Errorf("注文 %d の取得に失敗: %v", i, err)
		}
		if found.Quantity != i+1 {
			t.Errorf("Quantity = %d, want %d", found.Quantity, i+1)
		}
	}
}
