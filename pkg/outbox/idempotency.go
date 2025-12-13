// Package outbox イベント処理の冪等性サポート
package outbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrEventAlreadyProcessed イベントが既に処理済みであることを示すエラー
var ErrEventAlreadyProcessed = errors.New("event already processed")

// IdempotencyChecker 処理済みイベントのチェックと記録
type IdempotencyChecker struct {
	pool *pgxpool.Pool
}

// NewIdempotencyChecker 新規IdempotencyCheckerを生成
func NewIdempotencyChecker(pool *pgxpool.Pool) *IdempotencyChecker {
	return &IdempotencyChecker{pool: pool}
}

// CheckAndMark トランザクション内でイベント処理済みかチェックしマーク
// 既に処理済みの場合は ErrEventAlreadyProcessed を返却
func (c *IdempotencyChecker) CheckAndMark(ctx context.Context, tx pgx.Tx, eventID, eventType string) error {
	// イベントID挿入を試行
	query := `
		INSERT INTO processed_events (event_id, event_type)
		VALUES ($1, $2)
		ON CONFLICT (event_id) DO NOTHING
	`

	result, err := tx.Exec(ctx, query, eventID, eventType)
	if err != nil {
		return fmt.Errorf("insert processed event: %w", err)
	}

	// 影響行数が0の場合は既に処理済み
	if result.RowsAffected() == 0 {
		return ErrEventAlreadyProcessed
	}

	return nil
}

// IsProcessed イベントが処理済みかチェック（マークはしない）
func (c *IdempotencyChecker) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = $1)`

	if err := c.pool.QueryRow(ctx, query, eventID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check processed event: %w", err)
	}

	return exists, nil
}
