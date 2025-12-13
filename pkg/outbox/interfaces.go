// Package outbox Outboxパターンのインターフェース定義
package outbox

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// EventPublisher イベントをOutboxに書き込むインターフェース
type EventPublisher interface {
	// PublishInTx トランザクション内でイベントをOutboxに書き込む
	PublishInTx(ctx context.Context, tx pgx.Tx, aggregateType, aggregateID, eventType string, payload any) error
}

// EventIdempotencyChecker イベント冪等性チェックのインターフェース
type EventIdempotencyChecker interface {
	// CheckAndMark イベントが処理済みかチェックし、処理済みとしてマークする
	// 既に処理済みの場合は ErrEventAlreadyProcessed を返す
	CheckAndMark(ctx context.Context, tx pgx.Tx, eventID, eventType string) error

	// IsProcessed イベントが処理済みかチェックする（マークはしない）
	IsProcessed(ctx context.Context, eventID string) (bool, error)
}

// インターフェース実装の確認
var (
	_ EventPublisher          = (*Publisher)(nil)
	_ EventIdempotencyChecker = (*IdempotencyChecker)(nil)
)
