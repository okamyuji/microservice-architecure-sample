package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/testutil"
	"github.com/yujiokamoto/microservice-architecture-sample/services/payment/internal/domain"
)

// TestPostgresPaymentRepository_Save 決済保存テスト
func TestPostgresPaymentRepository_Save(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := NewPostgresPaymentRepository(db.Pool)

	payment := &domain.Payment{
		ID:         uuid.New().String(),
		OrderID:    uuid.New().String(),
		CustomerID: "cust-1",
		Amount:     100.50,
		Status:     domain.PaymentStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}

	err = repo.Save(ctx, tx, payment)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Save失敗: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 保存されていることを確認
	saved, err := repo.FindByOrderID(ctx, payment.OrderID)
	if err != nil {
		t.Fatalf("FindByOrderID失敗: %v", err)
	}
	if saved.Amount != payment.Amount {
		t.Errorf("Amount = %f, want %f", saved.Amount, payment.Amount)
	}
}

// TestPostgresPaymentRepository_Update 決済更新テスト
func TestPostgresPaymentRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := NewPostgresPaymentRepository(db.Pool)

	payment := &domain.Payment{
		ID:         uuid.New().String(),
		OrderID:    uuid.New().String(),
		CustomerID: "cust-1",
		Amount:     100.50,
		Status:     domain.PaymentStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Save(ctx, tx, payment); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Save失敗: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 更新
	payment.Status = domain.PaymentStatusCompleted
	payment.UpdatedAt = time.Now()

	tx2, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Update(ctx, tx2, payment); err != nil {
		_ = tx2.Rollback(ctx)
		t.Fatalf("Update失敗: %v", err)
	}
	if err := tx2.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 更新されていることを確認
	updated, err := repo.FindByOrderID(ctx, payment.OrderID)
	if err != nil {
		t.Fatalf("FindByOrderID失敗: %v", err)
	}
	if updated.Status != domain.PaymentStatusCompleted {
		t.Errorf("Status = %s, want %s", updated.Status, domain.PaymentStatusCompleted)
	}
}

// TestPostgresPaymentRepository_Update_WithFailureReason 失敗理由付き更新テスト
func TestPostgresPaymentRepository_Update_WithFailureReason(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := NewPostgresPaymentRepository(db.Pool)

	payment := &domain.Payment{
		ID:         uuid.New().String(),
		OrderID:    uuid.New().String(),
		CustomerID: "cust-1",
		Amount:     100.50,
		Status:     domain.PaymentStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Save(ctx, tx, payment); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Save失敗: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 失敗として更新
	payment.Status = domain.PaymentStatusFailed
	payment.FailureReason = "card declined"
	payment.UpdatedAt = time.Now()

	tx2, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Update(ctx, tx2, payment); err != nil {
		_ = tx2.Rollback(ctx)
		t.Fatalf("Update失敗: %v", err)
	}
	if err := tx2.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 更新されていることを確認
	updated, err := repo.FindByOrderID(ctx, payment.OrderID)
	if err != nil {
		t.Fatalf("FindByOrderID失敗: %v", err)
	}
	if updated.FailureReason != "card declined" {
		t.Errorf("FailureReason = %s, want 'card declined'", updated.FailureReason)
	}
}

// TestPostgresPaymentRepository_Update_NotFound 存在しない決済の更新
func TestPostgresPaymentRepository_Update_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresPaymentRepository(db.Pool)

	payment := &domain.Payment{
		ID:        uuid.New().String(),
		Status:    domain.PaymentStatusCompleted,
		UpdatedAt: time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = repo.Update(ctx, tx, payment)
	if err != domain.ErrPaymentNotFound {
		t.Errorf("err = %v, want ErrPaymentNotFound", err)
	}
}

// TestPostgresPaymentRepository_FindByID ID検索テスト
func TestPostgresPaymentRepository_FindByID(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := NewPostgresPaymentRepository(db.Pool)

	payment := &domain.Payment{
		ID:         uuid.New().String(),
		OrderID:    uuid.New().String(),
		CustomerID: "cust-1",
		Amount:     100.50,
		Status:     domain.PaymentStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Save(ctx, tx, payment); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Save失敗: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// ID検索
	found, err := repo.FindByID(ctx, payment.ID)
	if err != nil {
		t.Fatalf("FindByID失敗: %v", err)
	}
	if found.ID != payment.ID {
		t.Errorf("ID = %s, want %s", found.ID, payment.ID)
	}
}

// TestPostgresPaymentRepository_FindByID_NotFound 存在しない決済
func TestPostgresPaymentRepository_FindByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresPaymentRepository(db.Pool)

	_, err = repo.FindByID(ctx, uuid.New().String())
	if err != domain.ErrPaymentNotFound {
		t.Errorf("err = %v, want ErrPaymentNotFound", err)
	}
}

// TestPostgresPaymentRepository_FindByOrderID 注文ID検索テスト
func TestPostgresPaymentRepository_FindByOrderID(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := NewPostgresPaymentRepository(db.Pool)
	orderID := uuid.New().String()

	payment := &domain.Payment{
		ID:         uuid.New().String(),
		OrderID:    orderID,
		CustomerID: "cust-1",
		Amount:     100.50,
		Status:     domain.PaymentStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Save(ctx, tx, payment); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Save失敗: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 注文ID検索
	found, err := repo.FindByOrderID(ctx, orderID)
	if err != nil {
		t.Fatalf("FindByOrderID失敗: %v", err)
	}
	if found.OrderID != orderID {
		t.Errorf("OrderID = %s, want %s", found.OrderID, orderID)
	}
}

// TestPostgresPaymentRepository_FindByOrderID_NotFound 存在しない決済
func TestPostgresPaymentRepository_FindByOrderID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresPaymentRepository(db.Pool)

	_, err = repo.FindByOrderID(ctx, uuid.New().String())
	if err != domain.ErrPaymentNotFound {
		t.Errorf("err = %v, want ErrPaymentNotFound", err)
	}
}

// TestPostgresPaymentRepository_FindByOrderIDForUpdate 行ロック付き取得テスト
func TestPostgresPaymentRepository_FindByOrderIDForUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := NewPostgresPaymentRepository(db.Pool)
	orderID := uuid.New().String()

	payment := &domain.Payment{
		ID:         uuid.New().String(),
		OrderID:    orderID,
		CustomerID: "cust-1",
		Amount:     100.50,
		Status:     domain.PaymentStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Save(ctx, tx, payment); err != nil {
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

	found, err := repo.FindByOrderIDForUpdate(ctx, tx2, orderID)
	if err != nil {
		t.Fatalf("FindByOrderIDForUpdate失敗: %v", err)
	}
	if found.OrderID != orderID {
		t.Errorf("OrderID = %s, want %s", found.OrderID, orderID)
	}
}

// TestPostgresPaymentRepository_FindByOrderIDForUpdate_NotFound 存在しない場合
func TestPostgresPaymentRepository_FindByOrderIDForUpdate_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresPaymentRepository(db.Pool)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = repo.FindByOrderIDForUpdate(ctx, tx, uuid.New().String())
	if err != domain.ErrPaymentNotFound {
		t.Errorf("err = %v, want ErrPaymentNotFound", err)
	}
}

// TestNewPostgresPaymentRepository リポジトリ生成テスト
func TestNewPostgresPaymentRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresPaymentRepository(db.Pool)
	if repo == nil {
		t.Error("repo が nil")
	}
}

// TestPostgresPaymentRepository_Save_Rollback ロールバック時のテスト
func TestPostgresPaymentRepository_Save_Rollback(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupPaymentDB(ctx) }()

	repo := NewPostgresPaymentRepository(db.Pool)
	orderID := uuid.New().String()

	payment := &domain.Payment{
		ID:         uuid.New().String(),
		OrderID:    orderID,
		CustomerID: "cust-rollback",
		Amount:     100.50,
		Status:     domain.PaymentStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}

	if err := repo.Save(ctx, tx, payment); err != nil {
		t.Fatalf("Save失敗: %v", err)
	}

	// ロールバック
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("ロールバック失敗: %v", err)
	}

	// 保存されていないことを確認
	_, err = repo.FindByOrderID(ctx, orderID)
	if err != domain.ErrPaymentNotFound {
		t.Error("ロールバック後にデータが残っている")
	}
}
