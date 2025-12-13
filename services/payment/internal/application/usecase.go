// Package application 決済サービスのアプリケーション層
package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yujiokamoto/microservice-architecture-sample/pkg/events"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/outbox"
	"github.com/yujiokamoto/microservice-architecture-sample/services/payment/internal/domain"
)

// PaymentUseCase 決済関連のビジネスロジックを処理
type PaymentUseCase struct {
	pool               *pgxpool.Pool
	repo               domain.PaymentRepository
	outboxPublisher    outbox.EventPublisher
	idempotencyChecker outbox.EventIdempotencyChecker
	logger             *slog.Logger
	simulateFailure    bool // テスト用決済失敗シミュレーション
}

// NewPaymentUseCase PaymentUseCaseを生成
func NewPaymentUseCase(
	pool *pgxpool.Pool,
	repo domain.PaymentRepository,
	outboxPublisher outbox.EventPublisher,
	idempotencyChecker outbox.EventIdempotencyChecker,
	logger *slog.Logger,
) *PaymentUseCase {
	return &PaymentUseCase{
		pool:               pool,
		repo:               repo,
		outboxPublisher:    outboxPublisher,
		idempotencyChecker: idempotencyChecker,
		logger:             logger,
		simulateFailure:    false,
	}
}

// SetSimulateFailure テスト用決済失敗シミュレーションを有効化/無効化
func (uc *PaymentUseCase) SetSimulateFailure(simulate bool) {
	uc.simulateFailure = simulate
}

// GetPayment 注文IDで決済を取得
func (uc *PaymentUseCase) GetPayment(ctx context.Context, orderID string) (*domain.Payment, error) {
	return uc.repo.FindByOrderID(ctx, orderID)
}

// HandleStockReserved StockReservedイベントを処理し決済を実行
func (uc *PaymentUseCase) HandleStockReserved(ctx context.Context, eventID, orderID, customerID string, amount float64) error {
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

	// Create payment
	payment, err := domain.NewPayment(orderID, customerID, amount)
	if err != nil {
		return fmt.Errorf("create payment: %w", err)
	}

	// Simulate payment processing
	if uc.shouldFailPayment(amount) {
		// Payment failed
		if err := payment.Fail("Payment declined by provider"); err != nil {
			return fmt.Errorf("fail payment: %w", err)
		}

		if err := uc.repo.Save(ctx, tx, payment); err != nil {
			return fmt.Errorf("save payment: %w", err)
		}

		// Publish failure event
		failEvent := events.NewPaymentFailedEvent(orderID, payment.FailureReason)
		if err := uc.outboxPublisher.PublishInTx(ctx, tx, "Payment", orderID, events.EventTypePaymentFailed, failEvent); err != nil {
			return fmt.Errorf("publish failure event: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit transaction: %w", err)
		}

		uc.logger.Info("Payment failed",
			"payment_id", payment.ID,
			"order_id", orderID,
			"reason", payment.FailureReason,
		)
		return nil
	}

	// Payment successful
	if err := payment.Complete(); err != nil {
		return fmt.Errorf("complete payment: %w", err)
	}

	if err := uc.repo.Save(ctx, tx, payment); err != nil {
		return fmt.Errorf("save payment: %w", err)
	}

	// Publish success event
	successEvent := events.NewPaymentCompletedEvent(orderID, payment.ID, amount)
	if err := uc.outboxPublisher.PublishInTx(ctx, tx, "Payment", orderID, events.EventTypePaymentCompleted, successEvent); err != nil {
		return fmt.Errorf("publish success event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	uc.logger.Info("Payment completed",
		"payment_id", payment.ID,
		"order_id", orderID,
		"amount", amount,
	)

	return nil
}

// shouldFailPayment 金額またはランダムに基づき決済失敗をシミュレート
func (uc *PaymentUseCase) shouldFailPayment(amount float64) bool {
	if uc.simulateFailure {
		return true
	}

	// Fail payments with amount ending in .99 (for testing compensation)
	if int(amount*100)%100 == 99 {
		return true
	}

	// 10% random failure rate for testing
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Float64() < 0.1
}

// HandleOrderCancelled OrderCancelledイベントを処理し必要に応じて返金
func (uc *PaymentUseCase) HandleOrderCancelled(ctx context.Context, eventID, orderID string) error {
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

	// Find payment for this order
	payment, err := uc.repo.FindByOrderIDForUpdate(ctx, tx, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			// No payment to refund
			uc.logger.Info("No payment found for cancelled order", "order_id", orderID)
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return fmt.Errorf("commit transaction: %w", commitErr)
			}
			return nil
		}
		return fmt.Errorf("find payment: %w", err)
	}

	// Refund if completed
	if payment.IsCompleted() {
		if err := payment.Refund(); err != nil {
			uc.logger.Warn("Cannot refund payment", "payment_id", payment.ID, "status", payment.Status)
		} else {
			if err := uc.repo.Update(ctx, tx, payment); err != nil {
				return fmt.Errorf("update payment: %w", err)
			}
			uc.logger.Info("Payment refunded", "payment_id", payment.ID, "order_id", orderID)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
