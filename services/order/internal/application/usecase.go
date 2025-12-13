// Package application 注文サービスのアプリケーション層
package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yujiokamoto/microservice-architecture-sample/pkg/events"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/outbox"
	"github.com/yujiokamoto/microservice-architecture-sample/services/order/internal/domain"
)

// CreateOrderInput 注文作成の入力
type CreateOrderInput struct {
	CustomerID  string
	ProductID   string
	Quantity    int
	TotalAmount float64
}

// CreateOrderOutput 注文作成の出力
type CreateOrderOutput struct {
	OrderID string
	Status  string
}

// OrderUseCase 注文関連のビジネスロジックを処理
type OrderUseCase struct {
	pool               *pgxpool.Pool
	repo               domain.OrderRepository
	outboxPublisher    outbox.EventPublisher
	idempotencyChecker outbox.EventIdempotencyChecker
	logger             *slog.Logger
}

// NewOrderUseCase OrderUseCaseを生成
func NewOrderUseCase(
	pool *pgxpool.Pool,
	repo domain.OrderRepository,
	outboxPublisher outbox.EventPublisher,
	idempotencyChecker outbox.EventIdempotencyChecker,
	logger *slog.Logger,
) *OrderUseCase {
	return &OrderUseCase{
		pool:               pool,
		repo:               repo,
		outboxPublisher:    outboxPublisher,
		idempotencyChecker: idempotencyChecker,
		logger:             logger,
	}
}

// CreateOrder 新規注文を作成しOrderCreatedイベントを発行
func (uc *OrderUseCase) CreateOrder(ctx context.Context, input CreateOrderInput) (*CreateOrderOutput, error) {
	order, err := domain.NewOrder(input.CustomerID, input.ProductID, input.Quantity, input.TotalAmount)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Save order
	if err := uc.repo.Save(ctx, tx, order); err != nil {
		return nil, fmt.Errorf("save order: %w", err)
	}

	// Create and publish event to outbox
	event := events.NewOrderCreatedEvent(
		order.ID,
		order.CustomerID,
		order.ProductID,
		order.Quantity,
		order.TotalAmount,
	)

	if err := uc.outboxPublisher.PublishInTx(ctx, tx, "Order", order.ID, events.EventTypeOrderCreated, event); err != nil {
		return nil, fmt.Errorf("publish event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	uc.logger.Info("Order created",
		"order_id", order.ID,
		"customer_id", order.CustomerID,
		"product_id", order.ProductID,
	)

	return &CreateOrderOutput{
		OrderID: order.ID,
		Status:  order.Status,
	}, nil
}

// GetOrder IDで注文を取得
func (uc *OrderUseCase) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	order, err := uc.repo.FindByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("find order: %w", err)
	}
	return order, nil
}

// HandleStockReserved StockReservedイベントを処理し注文を確認
func (uc *OrderUseCase) HandleStockReserved(ctx context.Context, eventID, orderID string) error {
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Check idempotency
	if err := uc.idempotencyChecker.CheckAndMark(ctx, tx, eventID, events.EventTypeStockReserved); err != nil {
		if errors.Is(err, outbox.ErrEventAlreadyProcessed) {
			uc.logger.Info("Event already processed, skipping", "event_id", eventID)
			return nil
		}
		return fmt.Errorf("check idempotency: %w", err)
	}

	order, err := uc.repo.FindByIDForUpdate(ctx, tx, orderID)
	if err != nil {
		return fmt.Errorf("find order: %w", err)
	}

	if err := order.Confirm(); err != nil {
		uc.logger.Warn("Cannot confirm order", "order_id", orderID, "status", order.Status)
		return nil // Not an error, just skip
	}

	if err := uc.repo.Update(ctx, tx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	uc.logger.Info("Order confirmed (stock reserved)", "order_id", orderID)
	return nil
}

// HandlePaymentCompleted PaymentCompletedイベントを処理し注文を完了
func (uc *OrderUseCase) HandlePaymentCompleted(ctx context.Context, eventID, orderID string) error {
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Check idempotency
	if err := uc.idempotencyChecker.CheckAndMark(ctx, tx, eventID, events.EventTypePaymentCompleted); err != nil {
		if errors.Is(err, outbox.ErrEventAlreadyProcessed) {
			uc.logger.Info("Event already processed, skipping", "event_id", eventID)
			return nil
		}
		return fmt.Errorf("check idempotency: %w", err)
	}

	order, err := uc.repo.FindByIDForUpdate(ctx, tx, orderID)
	if err != nil {
		return fmt.Errorf("find order: %w", err)
	}

	if err := order.Complete(); err != nil {
		uc.logger.Warn("Cannot complete order", "order_id", orderID, "status", order.Status)
		return nil // Not an error, just skip
	}

	if err := uc.repo.Update(ctx, tx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	// Publish OrderCompleted event
	event := events.NewOrderCompletedEvent(orderID)
	if err := uc.outboxPublisher.PublishInTx(ctx, tx, "Order", orderID, events.EventTypeOrderCompleted, event); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	uc.logger.Info("Order completed", "order_id", orderID)
	return nil
}

// HandlePaymentFailed PaymentFailedイベントを処理し注文をキャンセル、補償をトリガー
func (uc *OrderUseCase) HandlePaymentFailed(ctx context.Context, eventID, orderID, reason string) error {
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Check idempotency
	if err := uc.idempotencyChecker.CheckAndMark(ctx, tx, eventID, events.EventTypePaymentFailed); err != nil {
		if errors.Is(err, outbox.ErrEventAlreadyProcessed) {
			uc.logger.Info("Event already processed, skipping", "event_id", eventID)
			return nil
		}
		return fmt.Errorf("check idempotency: %w", err)
	}

	order, err := uc.repo.FindByIDForUpdate(ctx, tx, orderID)
	if err != nil {
		return fmt.Errorf("find order: %w", err)
	}

	if err := order.Cancel(); err != nil {
		uc.logger.Warn("Cannot cancel order", "order_id", orderID, "status", order.Status)
		return nil
	}

	if err := uc.repo.Update(ctx, tx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	// Publish OrderCancelled event for compensation
	event := events.NewOrderCancelledEvent(orderID, reason)
	if err := uc.outboxPublisher.PublishInTx(ctx, tx, "Order", orderID, events.EventTypeOrderCancelled, event); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	uc.logger.Info("Order cancelled due to payment failure", "order_id", orderID, "reason", reason)
	return nil
}

// HandleStockReserveFailed StockReserveFailedイベントを処理し注文をキャンセル
func (uc *OrderUseCase) HandleStockReserveFailed(ctx context.Context, eventID, orderID, reason string) error {
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Check idempotency
	if err := uc.idempotencyChecker.CheckAndMark(ctx, tx, eventID, events.EventTypeStockReserveFailed); err != nil {
		if errors.Is(err, outbox.ErrEventAlreadyProcessed) {
			uc.logger.Info("Event already processed, skipping", "event_id", eventID)
			return nil
		}
		return fmt.Errorf("check idempotency: %w", err)
	}

	order, err := uc.repo.FindByIDForUpdate(ctx, tx, orderID)
	if err != nil {
		return fmt.Errorf("find order: %w", err)
	}

	if err := order.Cancel(); err != nil {
		uc.logger.Warn("Cannot cancel order", "order_id", orderID, "status", order.Status)
		return nil
	}

	if err := uc.repo.Update(ctx, tx, order); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	uc.logger.Info("Order cancelled due to stock reservation failure", "order_id", orderID, "reason", reason)
	return nil
}
