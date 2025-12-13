// Package domain リポジトリインターフェース定義
package domain

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// PaymentRepository 決済永続化操作
type PaymentRepository interface {
	// Save トランザクション内で新規決済を永続化
	Save(ctx context.Context, tx pgx.Tx, payment *Payment) error

	// Update トランザクション内で決済を更新
	Update(ctx context.Context, tx pgx.Tx, payment *Payment) error

	// FindByID IDで決済を取得
	FindByID(ctx context.Context, id string) (*Payment, error)

	// FindByOrderID 注文IDで決済を取得
	FindByOrderID(ctx context.Context, orderID string) (*Payment, error)

	// FindByOrderIDForUpdate 行ロック付きで注文IDから決済を取得
	FindByOrderIDForUpdate(ctx context.Context, tx pgx.Tx, orderID string) (*Payment, error)
}
