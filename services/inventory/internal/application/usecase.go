// Package application 在庫サービスのアプリケーション層
package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"microservice-architecture-sample/pkg/events"
	"microservice-architecture-sample/pkg/outbox"
	"microservice-architecture-sample/services/inventory/internal/domain"
)

// InventoryUseCase 在庫関連のビジネスロジックを処理
type InventoryUseCase struct {
	pool               *pgxpool.Pool
	inventoryRepo      domain.InventoryRepository
	reservationRepo    domain.ReservationRepository
	outboxPublisher    outbox.EventPublisher
	idempotencyChecker outbox.EventIdempotencyChecker
	logger             *slog.Logger
}

// NewInventoryUseCase InventoryUseCaseを生成
func NewInventoryUseCase(
	pool *pgxpool.Pool,
	inventoryRepo domain.InventoryRepository,
	reservationRepo domain.ReservationRepository,
	outboxPublisher outbox.EventPublisher,
	idempotencyChecker outbox.EventIdempotencyChecker,
	logger *slog.Logger,
) *InventoryUseCase {
	return &InventoryUseCase{
		pool:               pool,
		inventoryRepo:      inventoryRepo,
		reservationRepo:    reservationRepo,
		outboxPublisher:    outboxPublisher,
		idempotencyChecker: idempotencyChecker,
		logger:             logger,
	}
}

// GetInventory 商品の在庫を取得
func (uc *InventoryUseCase) GetInventory(ctx context.Context, productID string) (*domain.Inventory, error) {
	return uc.inventoryRepo.FindByProductID(ctx, productID)
}

// HandleOrderCreated OrderCreatedイベントを処理し在庫を予約
func (uc *InventoryUseCase) HandleOrderCreated(ctx context.Context, eventID, orderID, productID, customerID string, quantity int, totalAmount float64) error {
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Check idempotency
	if err := uc.idempotencyChecker.CheckAndMark(ctx, tx, eventID, events.EventTypeOrderCreated); err != nil {
		if errors.Is(err, outbox.ErrEventAlreadyProcessed) {
			uc.logger.Info("Event already processed, skipping", "event_id", eventID)
			return nil
		}
		return fmt.Errorf("check idempotency: %w", err)
	}

	// Get inventory with lock
	inventory, err := uc.inventoryRepo.FindByProductIDForUpdate(ctx, tx, productID)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			// Publish failure event
			failEvent := events.NewStockReserveFailedEvent(orderID, productID, quantity, "product not found")
			if pubErr := uc.outboxPublisher.PublishInTx(ctx, tx, "Inventory", orderID, events.EventTypeStockReserveFailed, failEvent); pubErr != nil {
				return fmt.Errorf("publish failure event: %w", pubErr)
			}
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return fmt.Errorf("commit transaction: %w", commitErr)
			}
			return nil
		}
		return fmt.Errorf("find inventory: %w", err)
	}

	// Try to reserve
	if err := inventory.Reserve(quantity); err != nil {
		// Publish failure event
		failEvent := events.NewStockReserveFailedEvent(orderID, productID, quantity, err.Error())
		if pubErr := uc.outboxPublisher.PublishInTx(ctx, tx, "Inventory", orderID, events.EventTypeStockReserveFailed, failEvent); pubErr != nil {
			return fmt.Errorf("publish failure event: %w", pubErr)
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return fmt.Errorf("commit transaction: %w", commitErr)
		}
		uc.logger.Warn("Stock reservation failed", "order_id", orderID, "product_id", productID, "reason", err.Error())
		return nil
	}

	// Update inventory
	if err := uc.inventoryRepo.Update(ctx, tx, inventory); err != nil {
		return fmt.Errorf("update inventory: %w", err)
	}

	// Create reservation
	reservation, err := domain.NewReservation(orderID, productID, quantity)
	if err != nil {
		return fmt.Errorf("create reservation: %w", err)
	}

	if err := uc.reservationRepo.Save(ctx, tx, reservation); err != nil {
		return fmt.Errorf("save reservation: %w", err)
	}

	// Publish success event
	successEvent := events.NewStockReservedEvent(orderID, productID, quantity, reservation.ID, customerID, totalAmount)
	if err := uc.outboxPublisher.PublishInTx(ctx, tx, "Inventory", orderID, events.EventTypeStockReserved, successEvent); err != nil {
		return fmt.Errorf("publish success event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	uc.logger.Info("Stock reserved",
		"order_id", orderID,
		"product_id", productID,
		"quantity", quantity,
		"reservation_id", reservation.ID,
	)

	return nil
}

// HandleOrderCancelled OrderCancelledイベントを処理し予約在庫を解放（補償トランザクション）
func (uc *InventoryUseCase) HandleOrderCancelled(ctx context.Context, eventID, orderID string) error {
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Check idempotency
	if err := uc.idempotencyChecker.CheckAndMark(ctx, tx, eventID, events.EventTypeOrderCancelled); err != nil {
		if errors.Is(err, outbox.ErrEventAlreadyProcessed) {
			uc.logger.Info("Event already processed, skipping", "event_id", eventID)
			return nil
		}
		return fmt.Errorf("check idempotency: %w", err)
	}

	// Get reservations for this order
	reservations, err := uc.reservationRepo.FindByOrderIDForUpdate(ctx, tx, orderID)
	if err != nil {
		return fmt.Errorf("find reservations: %w", err)
	}

	for _, reservation := range reservations {
		if !reservation.IsReserved() {
			continue
		}

		// Get inventory
		inventory, err := uc.inventoryRepo.FindByProductIDForUpdate(ctx, tx, reservation.ProductID)
		if err != nil {
			uc.logger.Error("Failed to find inventory for release", "product_id", reservation.ProductID, "error", err)
			continue
		}

		// Release reservation
		if err := inventory.Release(reservation.Quantity); err != nil {
			uc.logger.Error("Failed to release stock", "product_id", reservation.ProductID, "error", err)
			continue
		}

		// Update inventory
		if err := uc.inventoryRepo.Update(ctx, tx, inventory); err != nil {
			uc.logger.Error("Failed to update inventory", "product_id", reservation.ProductID, "error", err)
			continue
		}

		// Mark reservation as released
		reservation.Release()
		if err := uc.reservationRepo.Update(ctx, tx, reservation); err != nil {
			uc.logger.Error("Failed to update reservation", "reservation_id", reservation.ID, "error", err)
			continue
		}

		uc.logger.Info("Stock released (compensation)",
			"order_id", orderID,
			"product_id", reservation.ProductID,
			"quantity", reservation.Quantity,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// HandleOrderCompleted OrderCompletedイベントを処理し予約を確定
func (uc *InventoryUseCase) HandleOrderCompleted(ctx context.Context, eventID, orderID string) error {
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Check idempotency
	if err := uc.idempotencyChecker.CheckAndMark(ctx, tx, eventID, events.EventTypeOrderCompleted); err != nil {
		if errors.Is(err, outbox.ErrEventAlreadyProcessed) {
			uc.logger.Info("Event already processed, skipping", "event_id", eventID)
			return nil
		}
		return fmt.Errorf("check idempotency: %w", err)
	}

	// Get reservations for this order
	reservations, err := uc.reservationRepo.FindByOrderIDForUpdate(ctx, tx, orderID)
	if err != nil {
		return fmt.Errorf("find reservations: %w", err)
	}

	for _, reservation := range reservations {
		if !reservation.IsReserved() {
			continue
		}

		// Get inventory
		inventory, err := uc.inventoryRepo.FindByProductIDForUpdate(ctx, tx, reservation.ProductID)
		if err != nil {
			uc.logger.Error("Failed to find inventory for commit", "product_id", reservation.ProductID, "error", err)
			continue
		}

		// Commit reservation (reduce actual quantity)
		if err := inventory.Commit(reservation.Quantity); err != nil {
			uc.logger.Error("Failed to commit stock", "product_id", reservation.ProductID, "error", err)
			continue
		}

		// Update inventory
		if err := uc.inventoryRepo.Update(ctx, tx, inventory); err != nil {
			uc.logger.Error("Failed to update inventory", "product_id", reservation.ProductID, "error", err)
			continue
		}

		// Mark reservation as committed
		reservation.Commit()
		if err := uc.reservationRepo.Update(ctx, tx, reservation); err != nil {
			uc.logger.Error("Failed to update reservation", "reservation_id", reservation.ID, "error", err)
			continue
		}

		uc.logger.Info("Stock committed",
			"order_id", orderID,
			"product_id", reservation.ProductID,
			"quantity", reservation.Quantity,
		)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
