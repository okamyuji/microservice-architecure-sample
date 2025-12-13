// Package domain リポジトリインターフェース定義
package domain

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// InventoryRepository 在庫永続化操作
type InventoryRepository interface {
	// FindByProductID 商品IDで在庫を取得
	FindByProductID(ctx context.Context, productID string) (*Inventory, error)

	// FindByProductIDForUpdate 行ロック付きで在庫を取得
	FindByProductIDForUpdate(ctx context.Context, tx pgx.Tx, productID string) (*Inventory, error)

	// Update トランザクション内で在庫を更新
	Update(ctx context.Context, tx pgx.Tx, inventory *Inventory) error
}

// ReservationRepository 予約永続化操作
type ReservationRepository interface {
	// Save トランザクション内で新規予約を永続化
	Save(ctx context.Context, tx pgx.Tx, reservation *Reservation) error

	// Update トランザクション内で予約を更新
	Update(ctx context.Context, tx pgx.Tx, reservation *Reservation) error

	// FindByOrderID 注文IDで予約を取得
	FindByOrderID(ctx context.Context, orderID string) ([]*Reservation, error)

	// FindByOrderIDForUpdate 行ロック付きで予約を取得
	FindByOrderIDForUpdate(ctx context.Context, tx pgx.Tx, orderID string) ([]*Reservation, error)
}
