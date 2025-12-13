// Package testutil テスト用ユーティリティ
package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDB テスト用データベース接続
type TestDB struct {
	Pool      *pgxpool.Pool
	Container testcontainers.Container
}

// TestNATS テスト用NATS接続
type TestNATS struct {
	Conn      *nats.Conn
	Container testcontainers.Container
	URL       string
}

var (
	orderDB     *TestDB
	inventoryDB *TestDB
	paymentDB   *TestDB
	testNATS    *TestNATS
	orderOnce   sync.Once
	invOnce     sync.Once
	payOnce     sync.Once
	natsOnce    sync.Once
)

// GetOrderTestDB Order Service用テストDBを取得（シングルトン）
func GetOrderTestDB(ctx context.Context) (*TestDB, error) {
	var err error
	orderOnce.Do(func() {
		orderDB, err = createTestDB(ctx, "order", "init-order-db.sql")
	})
	if err != nil {
		return nil, err
	}
	return orderDB, nil
}

// GetInventoryTestDB Inventory Service用テストDBを取得（シングルトン）
func GetInventoryTestDB(ctx context.Context) (*TestDB, error) {
	var err error
	invOnce.Do(func() {
		inventoryDB, err = createTestDB(ctx, "inventory", "init-inventory-db.sql")
	})
	if err != nil {
		return nil, err
	}
	return inventoryDB, nil
}

// GetPaymentTestDB Payment Service用テストDBを取得（シングルトン）
func GetPaymentTestDB(ctx context.Context) (*TestDB, error) {
	var err error
	payOnce.Do(func() {
		paymentDB, err = createTestDB(ctx, "payment", "init-payment-db.sql")
	})
	if err != nil {
		return nil, err
	}
	return paymentDB, nil
}

// createTestDB TestContainerでPostgreSQLを起動
func createTestDB(ctx context.Context, name, initScript string) (*TestDB, error) {
	// scriptsディレクトリのパスを取得
	_, currentFile, _, _ := runtime.Caller(0)
	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
	scriptPath := filepath.Join(projectRoot, "scripts", initScript)

	// 初期化スクリプトを読み込み
	initSQL, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("read init script %s: %w", scriptPath, err)
	}

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       name,
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("get host: %w", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("get port: %w", err)
	}

	connStr := fmt.Sprintf("postgres://test:test@%s:%s/%s?sslmode=disable", host, port.Port(), name)

	// 接続を待機
	var pool *pgxpool.Pool
	for i := 0; i < 30; i++ {
		pool, err = pgxpool.New(ctx, connStr)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				break
			}
			pool.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	if pool == nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	// 初期化スクリプト実行
	if _, err := pool.Exec(ctx, string(initSQL)); err != nil {
		pool.Close()
		return nil, fmt.Errorf("execute init script: %w", err)
	}

	return &TestDB{
		Pool:      pool,
		Container: container,
	}, nil
}

// CleanupOrderDB ordersテーブルをクリーンアップ
func (db *TestDB) CleanupOrderDB(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM processed_events;
		DELETE FROM outbox;
		DELETE FROM orders;
	`)
	return err
}

// CleanupInventoryDB inventoryテーブルをクリーンアップ
func (db *TestDB) CleanupInventoryDB(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM processed_events;
		DELETE FROM outbox;
		DELETE FROM reservations;
		UPDATE inventory SET reserved_quantity = 0;
	`)
	return err
}

// CleanupPaymentDB paymentsテーブルをクリーンアップ
func (db *TestDB) CleanupPaymentDB(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM processed_events;
		DELETE FROM outbox;
		DELETE FROM payments;
	`)
	return err
}

// Close コンテナを停止
func (db *TestDB) Close(ctx context.Context) error {
	if db.Pool != nil {
		db.Pool.Close()
	}
	if db.Container != nil {
		return db.Container.Terminate(ctx)
	}
	return nil
}

// GetTestNATS テスト用NATSサーバーを取得（シングルトン）
func GetTestNATS(ctx context.Context) (*TestNATS, error) {
	var err error
	natsOnce.Do(func() {
		testNATS, err = createTestNATS(ctx)
	})
	if err != nil {
		return nil, err
	}
	return testNATS, nil
}

// createTestNATS TestContainerでNATSを起動
func createTestNATS(ctx context.Context) (*TestNATS, error) {
	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		ExposedPorts: []string{"4222/tcp"},
		WaitingFor: wait.ForLog("Server is ready").
			WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("start nats container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("get nats host: %w", err)
	}

	port, err := container.MappedPort(ctx, "4222")
	if err != nil {
		return nil, fmt.Errorf("get nats port: %w", err)
	}

	natsURL := fmt.Sprintf("nats://%s:%s", host, port.Port())

	// NATS接続を待機
	var conn *nats.Conn
	for i := 0; i < 30; i++ {
		conn, err = nats.Connect(natsURL)
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if conn == nil {
		return nil, fmt.Errorf("connect to nats: %w", err)
	}

	return &TestNATS{
		Conn:      conn,
		Container: container,
		URL:       natsURL,
	}, nil
}

// Close NATSコンテナを停止
func (n *TestNATS) Close(ctx context.Context) error {
	if n.Conn != nil {
		n.Conn.Close()
	}
	if n.Container != nil {
		return n.Container.Terminate(ctx)
	}
	return nil
}
