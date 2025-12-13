// Package interfaces イベントコンシューマ
package interfaces

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/yujiokamoto/microservice-architecture-sample/pkg/events"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/messaging"
	"github.com/yujiokamoto/microservice-architecture-sample/services/inventory/internal/application"
)

// EventConsumer NATSからイベントを消費
type EventConsumer struct {
	client  *messaging.Client
	useCase *application.InventoryUseCase
	logger  *slog.Logger
	subs    []*nats.Subscription
}

// NewEventConsumer 新規イベントコンシューマを生成
func NewEventConsumer(client *messaging.Client, useCase *application.InventoryUseCase, logger *slog.Logger) *EventConsumer {
	return &EventConsumer{
		client:  client,
		useCase: useCase,
		logger:  logger,
	}
}

// Start 関連イベントを購読開始
func (c *EventConsumer) Start(ctx context.Context) error {
	// OrderCreatedイベントを購読
	sub1, err := c.client.Subscribe(events.EventTypeOrderCreated, "inventory-service", func(msg *nats.Msg) {
		c.handleOrderCreated(ctx, msg)
	})
	if err != nil {
		return err
	}
	c.subs = append(c.subs, sub1)

	// OrderCancelledイベントを購読
	sub2, err := c.client.Subscribe(events.EventTypeOrderCancelled, "inventory-service", func(msg *nats.Msg) {
		c.handleOrderCancelled(ctx, msg)
	})
	if err != nil {
		return err
	}
	c.subs = append(c.subs, sub2)

	// OrderCompletedイベントを購読
	sub3, err := c.client.Subscribe(events.EventTypeOrderCompleted, "inventory-service", func(msg *nats.Msg) {
		c.handleOrderCompleted(ctx, msg)
	})
	if err != nil {
		return err
	}
	c.subs = append(c.subs, sub3)

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

func (c *EventConsumer) handleOrderCreated(ctx context.Context, msg *nats.Msg) {
	event, err := events.ParseOrderCreatedEvent(msg.Data)
	if err != nil {
		c.logger.Error("Failed to parse OrderCreated event", "error", err)
		return
	}

	c.logger.Info("Received OrderCreated event",
		"event_id", event.EventID,
		"order_id", event.OrderID,
		"product_id", event.ProductID,
	)

	if err := c.useCase.HandleOrderCreated(ctx, event.EventID, event.OrderID, event.ProductID, event.CustomerID, event.Quantity, event.TotalAmount); err != nil {
		c.logger.Error("Failed to handle OrderCreated event",
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

func (c *EventConsumer) handleOrderCompleted(ctx context.Context, msg *nats.Msg) {
	event, err := events.ParseOrderCompletedEvent(msg.Data)
	if err != nil {
		c.logger.Error("Failed to parse OrderCompleted event", "error", err)
		return
	}

	c.logger.Info("Received OrderCompleted event",
		"event_id", event.EventID,
		"order_id", event.OrderID,
	)

	if err := c.useCase.HandleOrderCompleted(ctx, event.EventID, event.OrderID); err != nil {
		c.logger.Error("Failed to handle OrderCompleted event",
			"event_id", event.EventID,
			"order_id", event.OrderID,
			"error", err,
		)
	}
}
