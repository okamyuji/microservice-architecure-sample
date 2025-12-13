package outbox

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/testutil"
)

// TestIdempotencyChecker_CheckAndMark 冪等性チェックテスト
func TestIdempotencyChecker_CheckAndMark(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	checker := NewIdempotencyChecker(db.Pool)
	eventID := uuid.New().String()

	// 最初の呼び出しは成功するはず
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}

	err = checker.CheckAndMark(ctx, tx, eventID, "test.event")
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("CheckAndMark失敗: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 2回目の呼び出しはErrEventAlreadyProcessedになるはず
	tx2, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}

	err = checker.CheckAndMark(ctx, tx2, eventID, "test.event")
	_ = tx2.Rollback(ctx)

	if !errors.Is(err, ErrEventAlreadyProcessed) {
		t.Errorf("err = %v, want ErrEventAlreadyProcessed", err)
	}
}

// TestIdempotencyChecker_CheckAndMark_DifferentEvents 異なるイベントIDテスト
func TestIdempotencyChecker_CheckAndMark_DifferentEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	checker := NewIdempotencyChecker(db.Pool)

	// 2つの異なるイベントIDで処理
	for i := 0; i < 2; i++ {
		eventID := uuid.New().String()
		tx, err := db.Pool.Begin(ctx)
		if err != nil {
			t.Fatalf("トランザクション開始失敗: %v", err)
		}

		err = checker.CheckAndMark(ctx, tx, eventID, "test.event")
		if err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("CheckAndMark失敗 (i=%d): %v", i, err)
		}

		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("コミット失敗: %v", err)
		}
	}
}

// TestIdempotencyChecker_IsProcessed 処理済みチェックテスト
func TestIdempotencyChecker_IsProcessed(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	checker := NewIdempotencyChecker(db.Pool)
	eventID := uuid.New().String()

	// 処理前は false
	processed, err := checker.IsProcessed(ctx, eventID)
	if err != nil {
		t.Fatalf("IsProcessed失敗: %v", err)
	}
	if processed {
		t.Error("processed = true, want false")
	}

	// イベントを処理済みにする
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	err = checker.CheckAndMark(ctx, tx, eventID, "test.event")
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("CheckAndMark失敗: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// 処理後は true
	processed, err = checker.IsProcessed(ctx, eventID)
	if err != nil {
		t.Fatalf("IsProcessed失敗: %v", err)
	}
	if !processed {
		t.Error("processed = false, want true")
	}
}

// TestIdempotencyChecker_IsProcessed_NotFound 存在しないイベントテスト
func TestIdempotencyChecker_IsProcessed_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	checker := NewIdempotencyChecker(db.Pool)

	// 存在しないイベントID（UUIDフォーマット）
	processed, err := checker.IsProcessed(ctx, uuid.New().String())
	if err != nil {
		t.Fatalf("IsProcessed失敗: %v", err)
	}
	if processed {
		t.Error("processed = true, want false")
	}
}

// TestNewIdempotencyChecker IdempotencyChecker生成テスト
func TestNewIdempotencyChecker(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	checker := NewIdempotencyChecker(db.Pool)
	if checker == nil {
		t.Fatal("checker が nil")
	}
	if checker.pool != db.Pool {
		t.Error("pool が設定されていない")
	}
}

// TestErrEventAlreadyProcessed エラー定義テスト
func TestErrEventAlreadyProcessed(t *testing.T) {
	if ErrEventAlreadyProcessed == nil {
		t.Error("ErrEventAlreadyProcessed が nil")
	}
	if ErrEventAlreadyProcessed.Error() != "event already processed" {
		t.Errorf("error message = %s, want 'event already processed'", ErrEventAlreadyProcessed.Error())
	}
}

// TestIdempotencyChecker_CheckAndMark_Rollback ロールバック時のテスト
func TestIdempotencyChecker_CheckAndMark_Rollback(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	checker := NewIdempotencyChecker(db.Pool)
	eventID := uuid.New().String()

	// トランザクション内でマークするがロールバック
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}

	err = checker.CheckAndMark(ctx, tx, eventID, "test.event")
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("CheckAndMark失敗: %v", err)
	}

	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("ロールバック失敗: %v", err)
	}

	// ロールバック後は処理済みになっていないはず
	processed, err := checker.IsProcessed(ctx, eventID)
	if err != nil {
		t.Fatalf("IsProcessed失敗: %v", err)
	}
	if processed {
		t.Error("ロールバック後も処理済みになっている")
	}
}
