// Package interfaces イベントコンシューマ
package interfaces

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/yujiokamoto/microservice-architecture-sample/pkg/events"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/messaging"
	"github.com/yujiokamoto/microservice-architecture-sample/services/payment/internal/application"
)

// EventConsumer NATSからイベントを消費
type EventConsumer struct {
	client  *messaging.Client
	useCase *application.PaymentUseCase
	logger  *slog.Logger
	subs    []*nats.Subscription
}

// NewEventConsumer 新規イベントコンシューマを生成
func NewEventConsumer(client *messaging.Client, useCase *application.PaymentUseCase, logger *slog.Logger) *EventConsumer {
	return &EventConsumer{
		client:  client,
		useCase: useCase,
		logger:  logger,
	}
}

// Start 関連イベントを購読開始
func (c *EventConsumer) Start(ctx context.Context) error {
	// StockReservedイベントを購読
	sub1, err := c.client.Subscribe(events.EventTypeStockReserved, "payment-service", func(msg *nats.Msg) {
		c.handleStockReserved(ctx, msg)
	})
	if err != nil {
		return err
	}
	c.subs = append(c.subs, sub1)

	// OrderCancelledイベントを購読
	sub2, err := c.client.Subscribe(events.EventTypeOrderCancelled, "payment-service", func(msg *nats.Msg) {
		c.handleOrderCancelled(ctx, msg)
	})
	if err != nil {
		return err
	}
	c.subs = append(c.subs, sub2)

	c.logger.Info("Event consumer started")
	return nil
}

// Stop 全イベント購読を解除
func (c *EventConsumer) Stop() {
	for _, sub := range c.subs {
		_ = sub.Unsubscribe()
	}
	c.logger.Info("Event consumer stopped")
}

func (c *EventConsumer) handleStockReserved(ctx context.Context, msg *nats.Msg) {
	event, err := events.ParseStockReservedEvent(msg.Data)
	if err != nil {
		c.logger.Error("Failed to parse StockReserved event", "error", err)
		return
	}

	c.logger.Info("Received StockReserved event",
		"event_id", event.EventID,
		"order_id", event.OrderID,
	)

	if err := c.useCase.HandleStockReserved(ctx, event.EventID, event.OrderID, event.CustomerID, event.TotalAmount); err != nil {
		c.logger.Error("Failed to handle StockReserved event",
			"event_id", event.EventID,
			"order_id", event.OrderID,
			"error", err,
		)
	}
}

func (c *EventConsumer) handleOrderCancelled(ctx context.Context, msg *nats.Msg) {
	event, err := events.ParseOrderCancelledEvent(msg.Data)
	if err != nil {
		c.logger.Error("Failed to parse OrderCancelled event", "error", err)
		return
	}

	c.logger.Info("Received OrderCancelled event",
		"event_id", event.EventID,
		"order_id", event.OrderID,
	)

	if err := c.useCase.HandleOrderCancelled(ctx, event.EventID, event.OrderID); err != nil {
		c.logger.Error("Failed to handle OrderCancelled event",
			"event_id", event.EventID,
			"order_id", event.OrderID,
			"error", err,
		)
	}
}
