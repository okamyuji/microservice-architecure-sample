// Package infrastructure インフラストラクチャ層実装
package infrastructure

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yujiokamoto/microservice-architecture-sample/services/payment/internal/domain"
)

// PostgresPaymentRepository PostgreSQLを使用したPaymentRepository実装
type PostgresPaymentRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresPaymentRepository 新規PostgreSQL決済リポジトリを生成
func NewPostgresPaymentRepository(pool *pgxpool.Pool) *PostgresPaymentRepository {
	return &PostgresPaymentRepository{pool: pool}
}

// Save トランザクション内で新規決済を永続化
func (r *PostgresPaymentRepository) Save(ctx context.Context, tx pgx.Tx, payment *domain.Payment) error {
	query := `
		INSERT INTO payments (id, order_id, customer_id, amount, status, failure_reason, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := tx.Exec(ctx, query,
		payment.ID,
		payment.OrderID,
		payment.CustomerID,
		payment.Amount,
		payment.Status,
		payment.FailureReason,
		payment.CreatedAt,
		payment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	return nil
}

// Update トランザクション内で決済を更新
func (r *PostgresPaymentRepository) Update(ctx context.Context, tx pgx.Tx, payment *domain.Payment) error {
	query := `
		UPDATE payments
		SET status = $1, failure_reason = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := tx.Exec(ctx, query, payment.Status, payment.FailureReason, payment.UpdatedAt, payment.ID)
	if err != nil {
		return fmt.Errorf("update payment: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrPaymentNotFound
	}

	return nil
}

// FindByID IDで決済を取得
func (r *PostgresPaymentRepository) FindByID(ctx context.Context, id string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, customer_id, amount, status, failure_reason, created_at, updated_at
		FROM payments
		WHERE id = $1
	`

	var payment domain.Payment
	var failureReason *string
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.CustomerID,
		&payment.Amount,
		&payment.Status,
		&failureReason,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("query payment: %w", err)
	}

	if failureReason != nil {
		payment.FailureReason = *failureReason
	}

	return &payment, nil
}

// FindByOrderID 注文IDで決済を取得
func (r *PostgresPaymentRepository) FindByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, customer_id, amount, status, failure_reason, created_at, updated_at
		FROM payments
		WHERE order_id = $1
	`

	var payment domain.Payment
	var failureReason *string
	err := r.pool.QueryRow(ctx, query, orderID).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.CustomerID,
		&payment.Amount,
		&payment.Status,
		&failureReason,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("query payment: %w", err)
	}

	if failureReason != nil {
		payment.FailureReason = *failureReason
	}

	return &payment, nil
}

// FindByOrderIDForUpdate 行ロック付きで注文IDから決済を取得
func (r *PostgresPaymentRepository) FindByOrderIDForUpdate(ctx context.Context, tx pgx.Tx, orderID string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, customer_id, amount, status, failure_reason, created_at, updated_at
		FROM payments
		WHERE order_id = $1
		FOR UPDATE
	`

	var payment domain.Payment
	var failureReason *string
	err := tx.QueryRow(ctx, query, orderID).Scan(
		&payment.ID,
		&payment.OrderID,
		&payment.CustomerID,
		&payment.Amount,
		&payment.Status,
		&failureReason,
		&payment.CreatedAt,
		&payment.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("query payment for update: %w", err)
	}

	if failureReason != nil {
		payment.FailureReason = *failureReason
	}

	return &payment, nil
}
