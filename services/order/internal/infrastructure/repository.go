// Package infrastructure インフラストラクチャ層実装
package infrastructure

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"microservice-architecture-sample/services/order/internal/domain"
)

// PostgresOrderRepository PostgreSQLを使用したOrderRepository実装
type PostgresOrderRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresOrderRepository 新規PostgreSQL注文リポジトリを生成
func NewPostgresOrderRepository(pool *pgxpool.Pool) *PostgresOrderRepository {
	return &PostgresOrderRepository{pool: pool}
}

// Save トランザクション内で新規注文を永続化
func (r *PostgresOrderRepository) Save(ctx context.Context, tx pgx.Tx, order *domain.Order) error {
	query := `
		INSERT INTO orders (id, customer_id, product_id, quantity, total_amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := tx.Exec(ctx, query,
		order.ID,
		order.CustomerID,
		order.ProductID,
		order.Quantity,
		order.TotalAmount,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	return nil
}

// Update トランザクション内で既存注文を更新
func (r *PostgresOrderRepository) Update(ctx context.Context, tx pgx.Tx, order *domain.Order) error {
	query := `
		UPDATE orders
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := tx.Exec(ctx, query, order.Status, order.UpdatedAt, order.ID)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrOrderNotFound
	}

	return nil
}

// FindByID IDで注文を取得
func (r *PostgresOrderRepository) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	query := `
		SELECT id, customer_id, product_id, quantity, total_amount, status, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	var order domain.Order
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.CustomerID,
		&order.ProductID,
		&order.Quantity,
		&order.TotalAmount,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("query order: %w", err)
	}

	return &order, nil
}

// FindByIDForUpdate 行ロック付きでIDから注文を取得
func (r *PostgresOrderRepository) FindByIDForUpdate(ctx context.Context, tx pgx.Tx, id string) (*domain.Order, error) {
	query := `
		SELECT id, customer_id, product_id, quantity, total_amount, status, created_at, updated_at
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`

	var order domain.Order
	err := tx.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.CustomerID,
		&order.ProductID,
		&order.Quantity,
		&order.TotalAmount,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("query order for update: %w", err)
	}

	return &order, nil
}
