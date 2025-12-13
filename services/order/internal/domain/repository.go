// Package domain リポジトリインターフェース定義
package domain

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// OrderRepository 注文永続化操作
type OrderRepository interface {
	// Save トランザクション内で新規注文を永続化
	Save(ctx context.Context, tx pgx.Tx, order *Order) error

	// Update トランザクション内で既存注文を更新
	Update(ctx context.Context, tx pgx.Tx, order *Order) error

	// FindByID IDで注文を取得
	FindByID(ctx context.Context, id string) (*Order, error)

	// FindByIDForUpdate 行ロック付きでIDから注文を取得
	FindByIDForUpdate(ctx context.Context, tx pgx.Tx, id string) (*Order, error)
}
