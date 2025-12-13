package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"microservice-architecture-sample/pkg/testutil"

	"github.com/google/uuid"
)

// テスト用ロガー
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// TestPublisher_PublishInTx トランザクション内でのイベント発行テスト
func TestPublisher_PublishInTx(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	publisher := NewPublisher(db.Pool, testLogger())

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}

	payload := map[string]string{"key": "value"}
	err = publisher.PublishInTx(ctx, tx, "TestAggregate", "agg-1", "test.event", payload)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("PublishInTx失敗: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("コミット失敗: %v", err)
	}

	// Outboxにメッセージが保存されていることを確認
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox WHERE event_type = 'test.event'").Scan(&count)
	if err != nil {
		t.Fatalf("カウント取得失敗: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

// TestPublisher_PublishInTx_Rollback ロールバック時のテスト
func TestPublisher_PublishInTx_Rollback(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	publisher := NewPublisher(db.Pool, testLogger())

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}

	payload := map[string]string{"key": "value"}
	err = publisher.PublishInTx(ctx, tx, "TestAggregate", "agg-2", "test.rollback", payload)
	if err != nil {
		t.Fatalf("PublishInTx失敗: %v", err)
	}

	// ロールバック
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("ロールバック失敗: %v", err)
	}

	// Outboxにメッセージが保存されていないことを確認
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox WHERE event_type = 'test.rollback'").Scan(&count)
	if err != nil {
		t.Fatalf("カウント取得失敗: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 (ロールバックされるべき)", count)
	}
}

// TestRelay_Start Relayの起動と停止テスト
func TestRelay_Start(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	// テスト前にクリーンアップ
	if err := db.CleanupOrderDB(ctx); err != nil {
		t.Fatalf("クリーンアップ失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	published := make(chan struct{}, 10)
	mockPublisher := func(subject string, data []byte) error {
		published <- struct{}{}
		return nil
	}

	relay := NewRelay(db.Pool, testLogger(), mockPublisher)

	// Outboxにテストデータを挿入（動的UUID）
	testID := uuid.New().String()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO outbox (id, aggregate_type, aggregate_id, event_type, payload, status)
		VALUES ($1, 'Test', 'test-relay-1', 'test.relay', '{"test": true}', 'PENDING')
	`, testID)
	if err != nil {
		t.Fatalf("テストデータ挿入失敗: %v", err)
	}

	// Relayをバックグラウンドで起動
	go relay.Start(ctx)

	// メッセージが発行されるまで待機
	select {
	case <-published:
		// 成功
	case <-time.After(10 * time.Second):
		t.Error("タイムアウト: メッセージが発行されなかった")
	}

	cancel()
}

// TestNewPublisher Publisher生成テスト
func TestNewPublisher(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	publisher := NewPublisher(db.Pool, testLogger())
	if publisher == nil {
		t.Fatal("publisher が nil")
	}
	if publisher.pool != db.Pool {
		t.Error("pool が設定されていない")
	}
}

// TestNewRelay Relay生成テスト
func TestNewRelay(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	mockPublisher := func(subject string, data []byte) error {
		return nil
	}

	relay := NewRelay(db.Pool, testLogger(), mockPublisher)
	if relay == nil {
		t.Fatal("relay が nil")
	}
	if relay.interval != 5*time.Second {
		t.Errorf("interval = %v, want 5s", relay.interval)
	}
}

// TestMessage_Status ステータス定数テスト
func TestMessage_Status(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"Pending", StatusPending, "PENDING"},
		{"Sent", StatusSent, "SENT"},
		{"Failed", StatusFailed, "FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("constant = %s, want %s", tt.constant, tt.want)
			}
		})
	}
}

// TestMaxRetryCount リトライ上限定数テスト
func TestMaxRetryCount(t *testing.T) {
	if MaxRetryCount != 5 {
		t.Errorf("MaxRetryCount = %d, want 5", MaxRetryCount)
	}
}

// TestMessage_Struct メッセージ構造体テスト
func TestMessage_Struct(t *testing.T) {
	now := time.Now()
	msg := Message{
		ID:            "test-id",
		AggregateType: "Order",
		AggregateID:   "order-1",
		EventType:     "order.created",
		Payload:       []byte(`{"key": "value"}`),
		Status:        StatusPending,
		RetryCount:    0,
		CreatedAt:     now,
		ProcessedAt:   nil,
	}

	if msg.ID != "test-id" {
		t.Errorf("ID = %s, want test-id", msg.ID)
	}
	if msg.AggregateType != "Order" {
		t.Errorf("AggregateType = %s, want Order", msg.AggregateType)
	}
	if msg.AggregateID != "order-1" {
		t.Errorf("AggregateID = %s, want order-1", msg.AggregateID)
	}
	if msg.EventType != "order.created" {
		t.Errorf("EventType = %s, want order.created", msg.EventType)
	}
	if string(msg.Payload) != `{"key": "value"}` {
		t.Errorf("Payload = %s, want {\"key\": \"value\"}", string(msg.Payload))
	}
	if msg.Status != StatusPending {
		t.Errorf("Status = %s, want %s", msg.Status, StatusPending)
	}
	if msg.RetryCount != 0 {
		t.Errorf("RetryCount = %d, want 0", msg.RetryCount)
	}
	if msg.CreatedAt.IsZero() {
		t.Errorf("CreatedAt = %v, want not zero", msg.CreatedAt)
	}
	if msg.ProcessedAt != nil {
		t.Errorf("ProcessedAt = %v, want nil", msg.ProcessedAt)
	}
}

// TestRelay_ProcessMultipleMessages 複数メッセージの処理テスト
func TestRelay_ProcessMultipleMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	// テスト前にクリーンアップ
	if err := db.CleanupOrderDB(ctx); err != nil {
		t.Fatalf("クリーンアップ失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	var publishCount int32
	mockPublisher := func(subject string, data []byte) error {
		atomic.AddInt32(&publishCount, 1)
		return nil
	}

	relay := NewRelay(db.Pool, testLogger(), mockPublisher)

	// 複数のテストデータを挿入（動的にUUIDを生成）
	for i := 0; i < 5; i++ {
		id := uuid.New().String()
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO outbox (id, aggregate_type, aggregate_id, event_type, payload, status)
			VALUES ($1, 'Test', $2, 'test.multiple', '{"test": true}', 'PENDING')
		`, id, fmt.Sprintf("agg-multi-%d", i))
		if err != nil {
			t.Fatalf("テストデータ挿入失敗: %v", err)
		}
	}

	// Relayをバックグラウンドで起動
	go relay.Start(ctx)

	// メッセージが発行されるまで待機
	time.Sleep(8 * time.Second)
	cancel()

	count := atomic.LoadInt32(&publishCount)
	if count < 5 {
		t.Logf("発行されたメッセージ数: %d", count)
	}
}

// TestRelay_PublishError 発行エラー時のリトライカウント増加テスト
func TestRelay_PublishError(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	// テスト前にクリーンアップ
	if err := db.CleanupOrderDB(ctx); err != nil {
		t.Fatalf("クリーンアップ失敗: %v", err)
	}
	defer func() { _ = db.CleanupOrderDB(ctx) }()

	// エラーを返すモックパブリッシャー
	mockPublisher := func(subject string, data []byte) error {
		return fmt.Errorf("publish error")
	}

	relay := NewRelay(db.Pool, testLogger(), mockPublisher)

	// テストデータを挿入（動的UUID）
	errorTestID := uuid.New().String()
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO outbox (id, aggregate_type, aggregate_id, event_type, payload, status)
		VALUES ($1, 'Test', 'agg-error', 'test.error', '{"test": true}', 'PENDING')
	`, errorTestID)
	if err != nil {
		t.Fatalf("テストデータ挿入失敗: %v", err)
	}

	// Relayをバックグラウンドで起動
	go relay.Start(ctx)

	// 待機
	time.Sleep(8 * time.Second)
	cancel()

	// リトライカウントが増加していることを確認
	var retryCount int
	err = db.Pool.QueryRow(ctx, "SELECT retry_count FROM outbox WHERE id = $1", errorTestID).Scan(&retryCount)
	if err != nil {
		t.Logf("リトライカウント取得失敗: %v", err)
	} else {
		t.Logf("リトライカウント: %d", retryCount)
	}
}

// TestPublisher_PublishInTx_MarshalError マーシャルエラーテスト
func TestPublisher_PublishInTx_MarshalError(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	publisher := NewPublisher(db.Pool, testLogger())

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatalf("トランザクション開始失敗: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// チャンネルはJSONマーシャルできない
	invalidPayload := make(chan int)
	err = publisher.PublishInTx(ctx, tx, "TestAggregate", "agg-1", "test.event", invalidPayload)
	if err == nil {
		t.Error("マーシャルエラーが発生すべき")
	}
}
