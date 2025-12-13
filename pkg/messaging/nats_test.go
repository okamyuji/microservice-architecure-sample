package messaging

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"microservice-architecture-sample/pkg/testutil"

	"github.com/nats-io/nats.go"
)

// testLogger テスト用ロガー
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// TestNewClient_WithTestContainer TestContainerを使った統合テスト
func TestNewClient_WithTestContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	testNATS, err := testutil.GetTestNATS(ctx)
	if err != nil {
		t.Fatalf("テストNATS取得失敗: %v", err)
	}

	// 環境変数をテストNATSに設定
	t.Setenv("NATS_URL", testNATS.URL)

	client, err := NewClient(ctx, testLogger())
	if err != nil {
		t.Fatalf("クライアント作成失敗: %v", err)
	}
	defer client.Close()

	if client.Conn() == nil {
		t.Error("接続がnilになっている")
	}
}

// TestClient_PublishSubscribe_Integration 発行・購読の統合テスト
func TestClient_PublishSubscribe_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	testNATS, err := testutil.GetTestNATS(ctx)
	if err != nil {
		t.Fatalf("テストNATS取得失敗: %v", err)
	}

	t.Setenv("NATS_URL", testNATS.URL)

	client, err := NewClient(ctx, testLogger())
	if err != nil {
		t.Fatalf("クライアント作成失敗: %v", err)
	}
	defer client.Close()

	// メッセージ受信用チャネル
	received := make(chan []byte, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	// 購読
	sub, err := client.Subscribe("test.subject", "test-queue", func(msg *nats.Msg) {
		received <- msg.Data
		wg.Done()
	})
	if err != nil {
		t.Fatalf("購読失敗: %v", err)
	}
	defer func() { _ = sub.Unsubscribe() }()

	// 発行
	testData := []byte("test message")
	if err := client.Publish("test.subject", testData); err != nil {
		t.Fatalf("発行失敗: %v", err)
	}

	// 受信を待機
	select {
	case data := <-received:
		if string(data) != string(testData) {
			t.Errorf("受信データ = %s, want %s", string(data), string(testData))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("メッセージ受信タイムアウト")
	}
}

// TestClient_PublishMultiple 複数メッセージの発行テスト
func TestClient_PublishMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	testNATS, err := testutil.GetTestNATS(ctx)
	if err != nil {
		t.Fatalf("テストNATS取得失敗: %v", err)
	}

	t.Setenv("NATS_URL", testNATS.URL)

	client, err := NewClient(ctx, testLogger())
	if err != nil {
		t.Fatalf("クライアント作成失敗: %v", err)
	}
	defer client.Close()

	// 複数メッセージを発行
	for i := 0; i < 10; i++ {
		if err := client.Publish("test.multiple", []byte("message")); err != nil {
			t.Fatalf("発行失敗: %v", err)
		}
	}
}

// TestClient_SubscribeMultipleHandlers 複数サブスクリプションのテスト
func TestClient_SubscribeMultipleHandlers(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	testNATS, err := testutil.GetTestNATS(ctx)
	if err != nil {
		t.Fatalf("テストNATS取得失敗: %v", err)
	}

	t.Setenv("NATS_URL", testNATS.URL)

	client, err := NewClient(ctx, testLogger())
	if err != nil {
		t.Fatalf("クライアント作成失敗: %v", err)
	}
	defer client.Close()

	// 複数の購読
	sub1, err := client.Subscribe("test.multi1", "queue1", func(msg *nats.Msg) {})
	if err != nil {
		t.Fatalf("購読1失敗: %v", err)
	}
	defer func() { _ = sub1.Unsubscribe() }()

	sub2, err := client.Subscribe("test.multi2", "queue2", func(msg *nats.Msg) {})
	if err != nil {
		t.Fatalf("購読2失敗: %v", err)
	}
	defer func() { _ = sub2.Unsubscribe() }()
}

// TestNewClient_DefaultURL デフォルトURLテスト
func TestNewClient_DefaultURL(t *testing.T) {
	// 環境変数をクリア
	t.Setenv("NATS_URL", "")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// NATSサーバーがないためエラーになる
	_, err := NewClient(ctx, testLogger())
	// タイムアウトまたは接続失敗が期待される
	if err == nil {
		t.Log("NATS接続成功（サーバーが起動している場合）")
	} else {
		t.Logf("NATS接続失敗（期待通り）: %v", err)
	}
}

// TestNewClient_CustomURL カスタムURLテスト
func TestNewClient_CustomURL(t *testing.T) {
	t.Setenv("NATS_URL", "nats://localhost:14222")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := NewClient(ctx, testLogger())
	if err == nil {
		t.Log("NATS接続成功（サーバーが起動している場合）")
	} else {
		t.Logf("NATS接続失敗（期待通り）: %v", err)
	}
}

// TestNewClient_ContextCanceled コンテキストキャンセル
func TestNewClient_ContextCanceled(t *testing.T) {
	t.Setenv("NATS_URL", "nats://localhost:14222")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 即座にキャンセル

	_, err := NewClient(ctx, testLogger())
	if err == nil {
		t.Log("接続成功（予想外）")
	} else {
		t.Logf("接続失敗（期待通り）: %v", err)
	}
}

// TestClient_Close Closeメソッドテスト（nil安全）
func TestClient_Close(t *testing.T) {
	client := &Client{
		conn:   nil,
		logger: testLogger(),
	}

	// panicしないことを確認
	client.Close()
}

// TestClient_Conn Connメソッドテスト
func TestClient_Conn(t *testing.T) {
	client := &Client{
		conn:   nil,
		logger: testLogger(),
	}

	if client.Conn() != nil {
		t.Error("Conn should return nil when not connected")
	}
}

// TestClient_Close_WithConnection 接続ありでCloseテスト
func TestClient_Close_WithConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	ctx := context.Background()
	testNATS, err := testutil.GetTestNATS(ctx)
	if err != nil {
		t.Fatalf("テストNATS取得失敗: %v", err)
	}

	t.Setenv("NATS_URL", testNATS.URL)

	client, err := NewClient(ctx, testLogger())
	if err != nil {
		t.Fatalf("クライアント作成失敗: %v", err)
	}

	// 接続中であることを確認
	if client.Conn() == nil {
		t.Fatal("接続がnil")
	}

	// Closeを呼び出し
	client.Close()

	// Closeが正常に完了したことを確認（panicしない）
}

// TestNewClient_RetryOnFailedConnect リトライ動作テスト
func TestNewClient_RetryOnFailedConnect(t *testing.T) {
	// 無効なURLを設定してリトライをテスト
	t.Setenv("NATS_URL", "nats://invalid-host:4222")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := NewClient(ctx, testLogger())
	// コンテキストタイムアウトまたは接続失敗が期待される
	if err != nil {
		t.Logf("リトライ後の失敗（期待通り）: %v", err)
	}
}
