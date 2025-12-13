package infrastructure

import (
	"context"
	"testing"
	"time"

	"microservice-architecture-sample/pkg/testutil"
	"microservice-architecture-sample/services/inventory/internal/domain"

	"github.com/google/uuid"
)

// TestPostgresInventoryRepository_FindByProductID 商品ID検索テスト
func TestPostgresInventoryRepository_FindByProductID(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresInventoryRepository(db.Pool)

	// 初期データのPROD-001を取得
	inv, err := repo.FindByProductID(ctx, "PROD-001")
	if err != nil {
		t.Fatalf("FindByProductID失敗: %v", err)
	}

	if inv.ProductID != "PROD-001" {
		t.Errorf("ProductID = %s, want PROD-001", inv.ProductID)
	}
	if inv.ProductName != "Laptop" {
		t.Errorf("ProductName = %s, want Laptop", inv.ProductName)
	}
}

// TestPostgresInventoryRepository_FindByProductID_NotFound 存在しない商品
func TestPostgresInventoryRepository_FindByProductID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresInventoryRepository(db.Pool)

	_, err = repo.FindByProductID(ctx, "PROD-NOTEXIST")
	if err != domain.ErrProductNotFound {
		t.Errorf("err = %v, want ErrProductNotFound", err)
	}
}

// TestPostgresInventoryRepository_FindByProductIDForUpdate 行ロック付き取得テスト
func TestPostgresInventoryRepository_FindByProductIDForUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresInventoryRepository(db.Pool)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	inv, err := repo.FindByProductIDForUpdate(ctx, tx, "PROD-002")
	if err != nil {
		t.Fatalf("FindByProductIDForUpdate失敗: %v", err)
	}

	if inv.ProductID != "PROD-002" {
		t.Errorf("ProductID = %s, want PROD-002", inv.ProductID)
	}
}

// TestPostgresInventoryRepository_FindByProductIDForUpdate_NotFound 存在しない場合
func TestPostgresInventoryRepository_FindByProductIDForUpdate_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresInventoryRepository(db.Pool)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = repo.FindByProductIDForUpdate(ctx, tx, "PROD-NOTEXIST")
	if err != domain.ErrProductNotFound {
		t.Errorf("err = %v, want ErrProductNotFound", err)
	}
}

// TestPostgresInventoryRepository_Update 在庫更新テスト
func TestPostgresInventoryRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	repo := NewPostgresInventoryRepository(db.Pool)

	// 現在の在庫を取得
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}

	inv, err := repo.FindByProductIDForUpdate(ctx, tx, "PROD-001")
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("FindByProductIDForUpdate失敗: %v", err)
	}

	// 予約数量を更新
	inv.ReservedQuantity = 10
	inv.UpdatedAt = time.Now()

	err = repo.Update(ctx, tx, inv)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Update失敗: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 更新されていることを確認
	updated, err := repo.FindByProductID(ctx, "PROD-001")
	if err != nil {
		t.Fatalf("FindByProductID失敗: %v", err)
	}
	if updated.ReservedQuantity != 10 {
		t.Errorf("ReservedQuantity = %d, want 10", updated.ReservedQuantity)
	}
}

// TestPostgresReservationRepository_Save 予約保存テスト
func TestPostgresReservationRepository_Save(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	repo := NewPostgresReservationRepository(db.Pool)

	reservation := &domain.Reservation{
		ID:        uuid.New().String(),
		OrderID:   uuid.New().String(),
		ProductID: "PROD-001",
		Quantity:  5,
		Status:    domain.ReservationStatusReserved,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}

	err = repo.Save(ctx, tx, reservation)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Save失敗: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 保存されていることを確認
	found, err := repo.FindByOrderID(ctx, reservation.OrderID)
	if err != nil {
		t.Fatalf("FindByOrderID失敗: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("found length = %d, want 1", len(found))
	}
	if found[0].ID != reservation.ID {
		t.Errorf("ID = %s, want %s", found[0].ID, reservation.ID)
	}
}

// TestPostgresReservationRepository_Update 予約更新テスト
func TestPostgresReservationRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	repo := NewPostgresReservationRepository(db.Pool)

	reservation := &domain.Reservation{
		ID:        uuid.New().String(),
		OrderID:   uuid.New().String(),
		ProductID: "PROD-001",
		Quantity:  5,
		Status:    domain.ReservationStatusReserved,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Save(ctx, tx, reservation); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("Save失敗: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 更新
	reservation.Status = domain.ReservationStatusReleased
	reservation.UpdatedAt = time.Now()

	tx2, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Update(ctx, tx2, reservation); err != nil {
		_ = tx2.Rollback(ctx)
		t.Fatalf("Update失敗: %v", err)
	}
	if err := tx2.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 更新されていることを確認
	found, err := repo.FindByOrderID(ctx, reservation.OrderID)
	if err != nil {
		t.Fatalf("FindByOrderID失敗: %v", err)
	}
	if found[0].Status != domain.ReservationStatusReleased {
		t.Errorf("Status = %s, want %s", found[0].Status, domain.ReservationStatusReleased)
	}
}

// TestPostgresReservationRepository_FindByOrderID 注文ID検索テスト
func TestPostgresReservationRepository_FindByOrderID(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	repo := NewPostgresReservationRepository(db.Pool)
	orderID := uuid.New().String()

	// 同じ注文に対して複数の予約を作成
	for i := 0; i < 3; i++ {
		reservation := &domain.Reservation{
			ID:        uuid.New().String(),
			OrderID:   orderID,
			ProductID: "PROD-001",
			Quantity:  i + 1,
			Status:    domain.ReservationStatusReserved,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		tx, err := db.Pool.Begin(ctx)
		if err != nil {
			t.Fatalf("トランザクション開始失敗: %v", err)
		}
		if err := repo.Save(ctx, tx, reservation); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("Save失敗: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("コミット失敗: %v", err)
		}
	}

	// 検索
	found, err := repo.FindByOrderID(ctx, orderID)
	if err != nil {
		t.Fatalf("FindByOrderID失敗: %v", err)
	}
	if len(found) != 3 {
		t.Errorf("found length = %d, want 3", len(found))
	}
}

// TestPostgresReservationRepository_FindByOrderID_Empty 空の結果
func TestPostgresReservationRepository_FindByOrderID_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresReservationRepository(db.Pool)

	found, err := repo.FindByOrderID(ctx, uuid.New().String())
	if err != nil {
		t.Fatalf("FindByOrderID失敗: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("found length = %d, want 0", len(found))
	}
}

// TestPostgresReservationRepository_FindByOrderIDForUpdate 行ロック付き取得テスト
func TestPostgresReservationRepository_FindByOrderIDForUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupInventoryDB(ctx) }()

	repo := NewPostgresReservationRepository(db.Pool)
	orderID := uuid.New().String()

	reservation := &domain.Reservation{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		ProductID: "PROD-001",
		Quantity:  5,
		Status:    domain.ReservationStatusReserved,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	if err := repo.Save(ctx, tx, reservation); err != nil {
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
	if len(found) != 1 {
		t.Errorf("found length = %d, want 1", len(found))
	}
}

// TestNewPostgresInventoryRepository リポジトリ生成テスト
func TestNewPostgresInventoryRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresInventoryRepository(db.Pool)
	if repo == nil {
		t.Error("repo が nil")
	}
}

// TestNewPostgresReservationRepository リポジトリ生成テスト
func TestNewPostgresReservationRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := NewPostgresReservationRepository(db.Pool)
	if repo == nil {
		t.Error("repo が nil")
	}
}
