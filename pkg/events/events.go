// Package events マイクロサービス間で共有されるドメインイベント定義
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// イベントタイプ定数
const (
	EventTypeOrderCreated       = "order.created"
	EventTypeOrderCompleted     = "order.completed"
	EventTypeOrderCancelled     = "order.cancelled"
	EventTypeStockReserved      = "inventory.stock_reserved"
	EventTypeStockReserveFailed = "inventory.stock_reserve_failed"
	EventTypeStockReleased      = "inventory.stock_released"
	EventTypePaymentCompleted   = "payment.completed"
	EventTypePaymentFailed      = "payment.failed"
)

// BaseEvent 全イベント共通フィールド
type BaseEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
}

// NewBaseEvent IDを生成して新規BaseEventを作成
func NewBaseEvent(eventType string) BaseEvent {
	return BaseEvent{
		EventID:   uuid.New().String(),
		EventType: eventType,
		Timestamp: time.Now().UTC(),
	}
}

// OrderCreatedEvent 注文作成イベント
type OrderCreatedEvent struct {
	BaseEvent
	OrderID     string  `json:"order_id"`
	CustomerID  string  `json:"customer_id"`
	ProductID   string  `json:"product_id"`
	Quantity    int     `json:"quantity"`
	TotalAmount float64 `json:"total_amount"`
}

// NewOrderCreatedEvent OrderCreatedEventを生成
func NewOrderCreatedEvent(orderID, customerID, productID string, quantity int, totalAmount float64) OrderCreatedEvent {
	return OrderCreatedEvent{
		BaseEvent:   NewBaseEvent(EventTypeOrderCreated),
		OrderID:     orderID,
		CustomerID:  customerID,
		ProductID:   productID,
		Quantity:    quantity,
		TotalAmount: totalAmount,
	}
}

// OrderCompletedEvent 注文完了イベント
type OrderCompletedEvent struct {
	BaseEvent
	OrderID string `json:"order_id"`
}

// NewOrderCompletedEvent OrderCompletedEventを生成
func NewOrderCompletedEvent(orderID string) OrderCompletedEvent {
	return OrderCompletedEvent{
		BaseEvent: NewBaseEvent(EventTypeOrderCompleted),
		OrderID:   orderID,
	}
}

// OrderCancelledEvent 注文キャンセルイベント
type OrderCancelledEvent struct {
	BaseEvent
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// NewOrderCancelledEvent OrderCancelledEventを生成
func NewOrderCancelledEvent(orderID, reason string) OrderCancelledEvent {
	return OrderCancelledEvent{
		BaseEvent: NewBaseEvent(EventTypeOrderCancelled),
		OrderID:   orderID,
		Reason:    reason,
	}
}

// StockReservedEvent 在庫予約成功イベント
type StockReservedEvent struct {
	BaseEvent
	OrderID       string  `json:"order_id"`
	ProductID     string  `json:"product_id"`
	Quantity      int     `json:"quantity"`
	ReservationID string  `json:"reservation_id"`
	CustomerID    string  `json:"customer_id"`
	TotalAmount   float64 `json:"total_amount"`
}

// NewStockReservedEvent StockReservedEventを生成
func NewStockReservedEvent(orderID, productID string, quantity int, reservationID, customerID string, totalAmount float64) StockReservedEvent {
	return StockReservedEvent{
		BaseEvent:     NewBaseEvent(EventTypeStockReserved),
		OrderID:       orderID,
		ProductID:     productID,
		Quantity:      quantity,
		ReservationID: reservationID,
		CustomerID:    customerID,
		TotalAmount:   totalAmount,
	}
}

// StockReserveFailedEvent 在庫予約失敗イベント
type StockReserveFailedEvent struct {
	BaseEvent
	OrderID   string `json:"order_id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Reason    string `json:"reason"`
}

// NewStockReserveFailedEvent StockReserveFailedEventを生成
func NewStockReserveFailedEvent(orderID, productID string, quantity int, reason string) StockReserveFailedEvent {
	return StockReserveFailedEvent{
		BaseEvent: NewBaseEvent(EventTypeStockReserveFailed),
		OrderID:   orderID,
		ProductID: productID,
		Quantity:  quantity,
		Reason:    reason,
	}
}

// StockReleasedEvent 在庫解放イベント（補償トランザクション）
type StockReleasedEvent struct {
	BaseEvent
	OrderID       string `json:"order_id"`
	ProductID     string `json:"product_id"`
	Quantity      int    `json:"quantity"`
	ReservationID string `json:"reservation_id"`
}

// NewStockReleasedEvent StockReleasedEventを生成
func NewStockReleasedEvent(orderID, productID string, quantity int, reservationID string) StockReleasedEvent {
	return StockReleasedEvent{
		BaseEvent:     NewBaseEvent(EventTypeStockReleased),
		OrderID:       orderID,
		ProductID:     productID,
		Quantity:      quantity,
		ReservationID: reservationID,
	}
}

// PaymentCompletedEvent 決済完了イベント
type PaymentCompletedEvent struct {
	BaseEvent
	OrderID   string  `json:"order_id"`
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount"`
}

// NewPaymentCompletedEvent PaymentCompletedEventを生成
func NewPaymentCompletedEvent(orderID, paymentID string, amount float64) PaymentCompletedEvent {
	return PaymentCompletedEvent{
		BaseEvent: NewBaseEvent(EventTypePaymentCompleted),
		OrderID:   orderID,
		PaymentID: paymentID,
		Amount:    amount,
	}
}

// PaymentFailedEvent 決済失敗イベント
type PaymentFailedEvent struct {
	BaseEvent
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// NewPaymentFailedEvent PaymentFailedEventを生成
func NewPaymentFailedEvent(orderID, reason string) PaymentFailedEvent {
	return PaymentFailedEvent{
		BaseEvent: NewBaseEvent(EventTypePaymentFailed),
		OrderID:   orderID,
		Reason:    reason,
	}
}

// ToJSON イベントをJSONバイト列にシリアライズ
func ToJSON(event any) ([]byte, error) {
	return json.Marshal(event)
}

// ParseOrderCreatedEvent JSONをOrderCreatedEventにパース
func ParseOrderCreatedEvent(data []byte) (OrderCreatedEvent, error) {
	var event OrderCreatedEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

// ParseStockReservedEvent JSONをStockReservedEventにパース
func ParseStockReservedEvent(data []byte) (StockReservedEvent, error) {
	var event StockReservedEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

// ParseStockReserveFailedEvent JSONをStockReserveFailedEventにパース
func ParseStockReserveFailedEvent(data []byte) (StockReserveFailedEvent, error) {
	var event StockReserveFailedEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

// ParsePaymentCompletedEvent JSONをPaymentCompletedEventにパース
func ParsePaymentCompletedEvent(data []byte) (PaymentCompletedEvent, error) {
	var event PaymentCompletedEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

// ParsePaymentFailedEvent JSONをPaymentFailedEventにパース
func ParsePaymentFailedEvent(data []byte) (PaymentFailedEvent, error) {
	var event PaymentFailedEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

// ParseOrderCancelledEvent JSONをOrderCancelledEventにパース
func ParseOrderCancelledEvent(data []byte) (OrderCancelledEvent, error) {
	var event OrderCancelledEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

// ParseOrderCompletedEvent JSONをOrderCompletedEventにパース
func ParseOrderCompletedEvent(data []byte) (OrderCompletedEvent, error) {
	var event OrderCompletedEvent
	err := json.Unmarshal(data, &event)
	return event, err
}
