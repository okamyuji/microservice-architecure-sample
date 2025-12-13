// Package interfaces イベントコンシューマ
package interfaces

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"

	"microservice-architecture-sample/pkg/events"
	"microservice-architecture-sample/pkg/messaging"
	"microservice-architecture-sample/services/order/internal/application"
)

// EventConsumer NATSからイベントを消費
type EventConsumer struct {
	client  *messaging.Client
	useCase *application.OrderUseCase
	logger  *slog.Logger
	subs    []*nats.Subscription
}

// NewEventConsumer 新規イベントコンシューマを生成
func NewEventConsumer(client *messaging.Client, useCase *application.OrderUseCase, logger *slog.Logger) *EventConsumer {
	return &EventConsumer{
		client:  client,
		useCase: useCase,
		logger:  logger,
	}
}

// Start 関連イベントを購読開始
func (c *EventConsumer) Start(ctx context.Context) error {
	// StockReservedイベントを購読
	sub1, err := c.client.Subscribe(events.EventTypeStockReserved, "order-service", func(msg *nats.Msg) {
		c.handleStockReserved(ctx, msg)
	})
	if err != nil {
		return err
	}
	c.subs = append(c.subs, sub1)

	// StockReserveFailedイベントを購読
	sub2, err := c.client.Subscribe(events.EventTypeStockReserveFailed, "order-service", func(msg *nats.Msg) {
		c.handleStockReserveFailed(ctx, msg)
	})
	if err != nil {
		return err
	}
	c.subs = append(c.subs, sub2)

	// PaymentCompletedイベントを購読
	sub3, err := c.client.Subscribe(events.EventTypePaymentCompleted, "order-service", func(msg *nats.Msg) {
		c.handlePaymentCompleted(ctx, msg)
	})
	if err != nil {
		return err
	}
	c.subs = append(c.subs, sub3)

	// PaymentFailedイベントを購読
	sub4, err := c.client.Subscribe(events.EventTypePaymentFailed, "order-service", func(msg *nats.Msg) {
		c.handlePaymentFailed(ctx, msg)
	})
	if err != nil {
		return err
	}
	c.subs = append(c.subs, sub4)

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

	if err := c.useCase.HandleStockReserved(ctx, event.EventID, event.OrderID); err != nil {
		c.logger.Error("Failed to handle StockReserved event",
			"event_id", event.EventID,
			"order_id", event.OrderID,
			"error", err,
		)
	}
}

func (c *EventConsumer) handleStockReserveFailed(ctx context.Context, msg *nats.Msg) {
	event, err := events.ParseStockReserveFailedEvent(msg.Data)
	if err != nil {
		c.logger.Error("Failed to parse StockReserveFailed event", "error", err)
		return
	}

	c.logger.Info("Received StockReserveFailed event",
		"event_id", event.EventID,
		"order_id", event.OrderID,
	)

	if err := c.useCase.HandleStockReserveFailed(ctx, event.EventID, event.OrderID, event.Reason); err != nil {
		c.logger.Error("Failed to handle StockReserveFailed event",
			"event_id", event.EventID,
			"order_id", event.OrderID,
			"error", err,
		)
	}
}

func (c *EventConsumer) handlePaymentCompleted(ctx context.Context, msg *nats.Msg) {
	event, err := events.ParsePaymentCompletedEvent(msg.Data)
	if err != nil {
		c.logger.Error("Failed to parse PaymentCompleted event", "error", err)
		return
	}

	c.logger.Info("Received PaymentCompleted event",
		"event_id", event.EventID,
		"order_id", event.OrderID,
	)

	if err := c.useCase.HandlePaymentCompleted(ctx, event.EventID, event.OrderID); err != nil {
		c.logger.Error("Failed to handle PaymentCompleted event",
			"event_id", event.EventID,
			"order_id", event.OrderID,
			"error", err,
		)
	}
}

func (c *EventConsumer) handlePaymentFailed(ctx context.Context, msg *nats.Msg) {
	event, err := events.ParsePaymentFailedEvent(msg.Data)
	if err != nil {
		c.logger.Error("Failed to parse PaymentFailed event", "error", err)
		return
	}

	c.logger.Info("Received PaymentFailed event",
		"event_id", event.EventID,
		"order_id", event.OrderID,
	)

	if err := c.useCase.HandlePaymentFailed(ctx, event.EventID, event.OrderID, event.Reason); err != nil {
		c.logger.Error("Failed to handle PaymentFailed event",
			"event_id", event.EventID,
			"order_id", event.OrderID,
			"error", err,
		)
	}
}
