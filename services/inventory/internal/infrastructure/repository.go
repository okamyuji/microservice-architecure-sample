// Package infrastructure インフラストラクチャ層実装
package infrastructure

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yujiokamoto/microservice-architecture-sample/services/inventory/internal/domain"
)

// PostgresInventoryRepository PostgreSQLを使用したInventoryRepository実装
type PostgresInventoryRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresInventoryRepository 新規PostgreSQL在庫リポジトリを生成
func NewPostgresInventoryRepository(pool *pgxpool.Pool) *PostgresInventoryRepository {
	return &PostgresInventoryRepository{pool: pool}
}

// FindByProductID 商品IDで在庫を取得
func (r *PostgresInventoryRepository) FindByProductID(ctx context.Context, productID string) (*domain.Inventory, error) {
	query := `
		SELECT id, product_id, product_name, quantity, reserved_quantity, created_at, updated_at
		FROM inventory
		WHERE product_id = $1
	`

	var inv domain.Inventory
	err := r.pool.QueryRow(ctx, query, productID).Scan(
		&inv.ID,
		&inv.ProductID,
		&inv.ProductName,
		&inv.Quantity,
		&inv.ReservedQuantity,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("query inventory: %w", err)
	}

	return &inv, nil
}

// FindByProductIDForUpdate 行ロック付きで在庫を取得
func (r *PostgresInventoryRepository) FindByProductIDForUpdate(ctx context.Context, tx pgx.Tx, productID string) (*domain.Inventory, error) {
	query := `
		SELECT id, product_id, product_name, quantity, reserved_quantity, created_at, updated_at
		FROM inventory
		WHERE product_id = $1
		FOR UPDATE
	`

	var inv domain.Inventory
	err := tx.QueryRow(ctx, query, productID).Scan(
		&inv.ID,
		&inv.ProductID,
		&inv.ProductName,
		&inv.Quantity,
		&inv.ReservedQuantity,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrProductNotFound
		}
		return nil, fmt.Errorf("query inventory for update: %w", err)
	}

	return &inv, nil
}

// Update トランザクション内で在庫を更新
func (r *PostgresInventoryRepository) Update(ctx context.Context, tx pgx.Tx, inventory *domain.Inventory) error {
	query := `
		UPDATE inventory
		SET quantity = $1, reserved_quantity = $2, updated_at = $3
		WHERE id = $4
	`

	result, err := tx.Exec(ctx, query, inventory.Quantity, inventory.ReservedQuantity, inventory.UpdatedAt, inventory.ID)
	if err != nil {
		return fmt.Errorf("update inventory: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrProductNotFound
	}

	return nil
}

// PostgresReservationRepository PostgreSQLを使用したReservationRepository実装
type PostgresReservationRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresReservationRepository 新規PostgreSQL予約リポジトリを生成
func NewPostgresReservationRepository(pool *pgxpool.Pool) *PostgresReservationRepository {
	return &PostgresReservationRepository{pool: pool}
}

// Save トランザクション内で新規予約を永続化
func (r *PostgresReservationRepository) Save(ctx context.Context, tx pgx.Tx, reservation *domain.Reservation) error {
	query := `
		INSERT INTO reservations (id, order_id, product_id, quantity, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := tx.Exec(ctx, query,
		reservation.ID,
		reservation.OrderID,
		reservation.ProductID,
		reservation.Quantity,
		reservation.Status,
		reservation.CreatedAt,
		reservation.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert reservation: %w", err)
	}

	return nil
}

// Update トランザクション内で予約を更新
func (r *PostgresReservationRepository) Update(ctx context.Context, tx pgx.Tx, reservation *domain.Reservation) error {
	query := `
		UPDATE reservations
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := tx.Exec(ctx, query, reservation.Status, reservation.UpdatedAt, reservation.ID)
	if err != nil {
		return fmt.Errorf("update reservation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrReservationNotFound
	}

	return nil
}

// FindByOrderID 注文IDで予約を取得
func (r *PostgresReservationRepository) FindByOrderID(ctx context.Context, orderID string) ([]*domain.Reservation, error) {
	query := `
		SELECT id, order_id, product_id, quantity, status, created_at, updated_at
		FROM reservations
		WHERE order_id = $1
	`

	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("query reservations: %w", err)
	}
	defer rows.Close()

	var reservations []*domain.Reservation
	for rows.Next() {
		var res domain.Reservation
		if err := rows.Scan(
			&res.ID,
			&res.OrderID,
			&res.ProductID,
			&res.Quantity,
			&res.Status,
			&res.CreatedAt,
			&res.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reservation: %w", err)
		}
		reservations = append(reservations, &res)
	}

	return reservations, nil
}

// FindByOrderIDForUpdate 行ロック付きで予約を取得
func (r *PostgresReservationRepository) FindByOrderIDForUpdate(ctx context.Context, tx pgx.Tx, orderID string) ([]*domain.Reservation, error) {
	query := `
		SELECT id, order_id, product_id, quantity, status, created_at, updated_at
		FROM reservations
		WHERE order_id = $1
		FOR UPDATE
	`

	rows, err := tx.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("query reservations for update: %w", err)
	}
	defer rows.Close()

	var reservations []*domain.Reservation
	for rows.Next() {
		var res domain.Reservation
		if err := rows.Scan(
			&res.ID,
			&res.OrderID,
			&res.ProductID,
			&res.Quantity,
			&res.Status,
			&res.CreatedAt,
			&res.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reservation: %w", err)
		}
		reservations = append(reservations, &res)
	}

	return reservations, nil
}
