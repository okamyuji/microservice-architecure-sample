package events

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNewBaseEvent BaseEventの生成テスト
func TestNewBaseEvent(t *testing.T) {
	eventType := "test.event"
	before := time.Now()
	event := NewBaseEvent(eventType)
	after := time.Now()

	// EventIDが生成されていること
	if event.EventID == "" {
		t.Error("EventID が空")
	}

	// EventTypeが設定されていること
	if event.EventType != eventType {
		t.Errorf("EventType = %s, want %s", event.EventType, eventType)
	}

	// Timestampが適切な範囲内であること
	if event.Timestamp.Before(before) || event.Timestamp.After(after) {
		t.Errorf("Timestamp = %v, want between %v and %v", event.Timestamp, before, after)
	}
}

// TestNewOrderCreatedEvent OrderCreatedEventの生成テスト
func TestNewOrderCreatedEvent(t *testing.T) {
	event := NewOrderCreatedEvent("order-1", "cust-1", "prod-1", 5, 100.50)

	if event.OrderID != "order-1" {
		t.Errorf("OrderID = %s, want order-1", event.OrderID)
	}
	if event.CustomerID != "cust-1" {
		t.Errorf("CustomerID = %s, want cust-1", event.CustomerID)
	}
	if event.ProductID != "prod-1" {
		t.Errorf("ProductID = %s, want prod-1", event.ProductID)
	}
	if event.Quantity != 5 {
		t.Errorf("Quantity = %d, want 5", event.Quantity)
	}
	if event.TotalAmount != 100.50 {
		t.Errorf("TotalAmount = %f, want 100.50", event.TotalAmount)
	}
	if event.EventType != EventTypeOrderCreated {
		t.Errorf("EventType = %s, want %s", event.EventType, EventTypeOrderCreated)
	}
}

// TestNewOrderCompletedEvent OrderCompletedEventの生成テスト
func TestNewOrderCompletedEvent(t *testing.T) {
	event := NewOrderCompletedEvent("order-1")

	if event.OrderID != "order-1" {
		t.Errorf("OrderID = %s, want order-1", event.OrderID)
	}
	if event.EventType != EventTypeOrderCompleted {
		t.Errorf("EventType = %s, want %s", event.EventType, EventTypeOrderCompleted)
	}
}

// TestNewOrderCancelledEvent OrderCancelledEventの生成テスト
func TestNewOrderCancelledEvent(t *testing.T) {
	event := NewOrderCancelledEvent("order-1", "payment failed")

	if event.OrderID != "order-1" {
		t.Errorf("OrderID = %s, want order-1", event.OrderID)
	}
	if event.Reason != "payment failed" {
		t.Errorf("Reason = %s, want payment failed", event.Reason)
	}
	if event.EventType != EventTypeOrderCancelled {
		t.Errorf("EventType = %s, want %s", event.EventType, EventTypeOrderCancelled)
	}
}

// TestNewStockReservedEvent StockReservedEventの生成テスト
func TestNewStockReservedEvent(t *testing.T) {
	event := NewStockReservedEvent("order-1", "prod-1", 10, "res-1", "cust-1", 500.00)

	if event.OrderID != "order-1" {
		t.Errorf("OrderID = %s, want order-1", event.OrderID)
	}
	if event.ProductID != "prod-1" {
		t.Errorf("ProductID = %s, want prod-1", event.ProductID)
	}
	if event.Quantity != 10 {
		t.Errorf("Quantity = %d, want 10", event.Quantity)
	}
	if event.ReservationID != "res-1" {
		t.Errorf("ReservationID = %s, want res-1", event.ReservationID)
	}
	if event.CustomerID != "cust-1" {
		t.Errorf("CustomerID = %s, want cust-1", event.CustomerID)
	}
	if event.TotalAmount != 500.00 {
		t.Errorf("TotalAmount = %f, want 500.00", event.TotalAmount)
	}
}

// TestNewStockReserveFailedEvent StockReserveFailedEventの生成テスト
func TestNewStockReserveFailedEvent(t *testing.T) {
	event := NewStockReserveFailedEvent("order-1", "prod-1", 10, "insufficient stock")

	if event.OrderID != "order-1" {
		t.Errorf("OrderID = %s, want order-1", event.OrderID)
	}
	if event.ProductID != "prod-1" {
		t.Errorf("ProductID = %s, want prod-1", event.ProductID)
	}
	if event.Quantity != 10 {
		t.Errorf("Quantity = %d, want 10", event.Quantity)
	}
	if event.Reason != "insufficient stock" {
		t.Errorf("Reason = %s, want insufficient stock", event.Reason)
	}
}

// TestNewStockReleasedEvent StockReleasedEventの生成テスト
func TestNewStockReleasedEvent(t *testing.T) {
	event := NewStockReleasedEvent("order-1", "prod-1", 5, "res-1")

	if event.OrderID != "order-1" {
		t.Errorf("OrderID = %s, want order-1", event.OrderID)
	}
	if event.ProductID != "prod-1" {
		t.Errorf("ProductID = %s, want prod-1", event.ProductID)
	}
	if event.Quantity != 5 {
		t.Errorf("Quantity = %d, want 5", event.Quantity)
	}
	if event.ReservationID != "res-1" {
		t.Errorf("ReservationID = %s, want res-1", event.ReservationID)
	}
}

// TestNewPaymentCompletedEvent PaymentCompletedEventの生成テスト
func TestNewPaymentCompletedEvent(t *testing.T) {
	event := NewPaymentCompletedEvent("order-1", "pay-1", 250.00)

	if event.OrderID != "order-1" {
		t.Errorf("OrderID = %s, want order-1", event.OrderID)
	}
	if event.PaymentID != "pay-1" {
		t.Errorf("PaymentID = %s, want pay-1", event.PaymentID)
	}
	if event.Amount != 250.00 {
		t.Errorf("Amount = %f, want 250.00", event.Amount)
	}
}

// TestNewPaymentFailedEvent PaymentFailedEventの生成テスト
func TestNewPaymentFailedEvent(t *testing.T) {
	event := NewPaymentFailedEvent("order-1", "card declined")

	if event.OrderID != "order-1" {
		t.Errorf("OrderID = %s, want order-1", event.OrderID)
	}
	if event.Reason != "card declined" {
		t.Errorf("Reason = %s, want card declined", event.Reason)
	}
}

// TestToJSON イベントのJSONシリアライズテスト
func TestToJSON(t *testing.T) {
	event := NewOrderCreatedEvent("order-1", "cust-1", "prod-1", 5, 100.50)

	data, err := ToJSON(event)
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	// JSONとしてパース可能であること
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	if result["order_id"] != "order-1" {
		t.Errorf("order_id = %v, want order-1", result["order_id"])
	}
}

// TestParseOrderCreatedEvent OrderCreatedEventのパーステスト
func TestParseOrderCreatedEvent(t *testing.T) {
	original := NewOrderCreatedEvent("order-1", "cust-1", "prod-1", 5, 100.50)
	data, _ := ToJSON(original)

	parsed, err := ParseOrderCreatedEvent(data)
	if err != nil {
		t.Fatalf("ParseOrderCreatedEvent error: %v", err)
	}

	if parsed.OrderID != original.OrderID {
		t.Errorf("OrderID = %s, want %s", parsed.OrderID, original.OrderID)
	}
	if parsed.CustomerID != original.CustomerID {
		t.Errorf("CustomerID = %s, want %s", parsed.CustomerID, original.CustomerID)
	}
}

// TestParseOrderCreatedEvent_InvalidJSON 不正なJSONのパーステスト
func TestParseOrderCreatedEvent_InvalidJSON(t *testing.T) {
	_, err := ParseOrderCreatedEvent([]byte("invalid json"))
	if err == nil {
		t.Error("不正なJSONでエラーが発生しない")
	}
}

// TestParseStockReservedEvent StockReservedEventのパーステスト
func TestParseStockReservedEvent(t *testing.T) {
	original := NewStockReservedEvent("order-1", "prod-1", 10, "res-1", "cust-1", 500.00)
	data, _ := ToJSON(original)

	parsed, err := ParseStockReservedEvent(data)
	if err != nil {
		t.Fatalf("ParseStockReservedEvent error: %v", err)
	}

	if parsed.OrderID != original.OrderID {
		t.Errorf("OrderID = %s, want %s", parsed.OrderID, original.OrderID)
	}
	if parsed.ReservationID != original.ReservationID {
		t.Errorf("ReservationID = %s, want %s", parsed.ReservationID, original.ReservationID)
	}
}

// TestParseStockReservedEvent_InvalidJSON 不正なJSONのパーステスト
func TestParseStockReservedEvent_InvalidJSON(t *testing.T) {
	_, err := ParseStockReservedEvent([]byte("invalid"))
	if err == nil {
		t.Error("不正なJSONでエラーが発生しない")
	}
}

// TestParseStockReserveFailedEvent StockReserveFailedEventのパーステスト
func TestParseStockReserveFailedEvent(t *testing.T) {
	original := NewStockReserveFailedEvent("order-1", "prod-1", 10, "insufficient stock")
	data, _ := ToJSON(original)

	parsed, err := ParseStockReserveFailedEvent(data)
	if err != nil {
		t.Fatalf("ParseStockReserveFailedEvent error: %v", err)
	}

	if parsed.Reason != original.Reason {
		t.Errorf("Reason = %s, want %s", parsed.Reason, original.Reason)
	}
}

// TestParseStockReserveFailedEvent_InvalidJSON 不正なJSONのパーステスト
func TestParseStockReserveFailedEvent_InvalidJSON(t *testing.T) {
	_, err := ParseStockReserveFailedEvent([]byte("invalid"))
	if err == nil {
		t.Error("不正なJSONでエラーが発生しない")
	}
}

// TestParsePaymentCompletedEvent PaymentCompletedEventのパーステスト
func TestParsePaymentCompletedEvent(t *testing.T) {
	original := NewPaymentCompletedEvent("order-1", "pay-1", 250.00)
	data, _ := ToJSON(original)

	parsed, err := ParsePaymentCompletedEvent(data)
	if err != nil {
		t.Fatalf("ParsePaymentCompletedEvent error: %v", err)
	}

	if parsed.PaymentID != original.PaymentID {
		t.Errorf("PaymentID = %s, want %s", parsed.PaymentID, original.PaymentID)
	}
}

// TestParsePaymentCompletedEvent_InvalidJSON 不正なJSONのパーステスト
func TestParsePaymentCompletedEvent_InvalidJSON(t *testing.T) {
	_, err := ParsePaymentCompletedEvent([]byte("invalid"))
	if err == nil {
		t.Error("不正なJSONでエラーが発生しない")
	}
}

// TestParsePaymentFailedEvent PaymentFailedEventのパーステスト
func TestParsePaymentFailedEvent(t *testing.T) {
	original := NewPaymentFailedEvent("order-1", "card declined")
	data, _ := ToJSON(original)

	parsed, err := ParsePaymentFailedEvent(data)
	if err != nil {
		t.Fatalf("ParsePaymentFailedEvent error: %v", err)
	}

	if parsed.Reason != original.Reason {
		t.Errorf("Reason = %s, want %s", parsed.Reason, original.Reason)
	}
}

// TestParsePaymentFailedEvent_InvalidJSON 不正なJSONのパーステスト
func TestParsePaymentFailedEvent_InvalidJSON(t *testing.T) {
	_, err := ParsePaymentFailedEvent([]byte("invalid"))
	if err == nil {
		t.Error("不正なJSONでエラーが発生しない")
	}
}

// TestParseOrderCancelledEvent OrderCancelledEventのパーステスト
func TestParseOrderCancelledEvent(t *testing.T) {
	original := NewOrderCancelledEvent("order-1", "payment failed")
	data, _ := ToJSON(original)

	parsed, err := ParseOrderCancelledEvent(data)
	if err != nil {
		t.Fatalf("ParseOrderCancelledEvent error: %v", err)
	}

	if parsed.Reason != original.Reason {
		t.Errorf("Reason = %s, want %s", parsed.Reason, original.Reason)
	}
}

// TestParseOrderCancelledEvent_InvalidJSON 不正なJSONのパーステスト
func TestParseOrderCancelledEvent_InvalidJSON(t *testing.T) {
	_, err := ParseOrderCancelledEvent([]byte("invalid"))
	if err == nil {
		t.Error("不正なJSONでエラーが発生しない")
	}
}

// TestParseOrderCompletedEvent OrderCompletedEventのパーステスト
func TestParseOrderCompletedEvent(t *testing.T) {
	original := NewOrderCompletedEvent("order-1")
	data, _ := ToJSON(original)

	parsed, err := ParseOrderCompletedEvent(data)
	if err != nil {
		t.Fatalf("ParseOrderCompletedEvent error: %v", err)
	}

	if parsed.OrderID != original.OrderID {
		t.Errorf("OrderID = %s, want %s", parsed.OrderID, original.OrderID)
	}
}

// TestParseOrderCompletedEvent_InvalidJSON 不正なJSONのパーステスト
func TestParseOrderCompletedEvent_InvalidJSON(t *testing.T) {
	_, err := ParseOrderCompletedEvent([]byte("invalid"))
	if err == nil {
		t.Error("不正なJSONでエラーが発生しない")
	}
}

// TestEventTypeConstants イベントタイプ定数のテスト
func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"OrderCreated", EventTypeOrderCreated, "order.created"},
		{"OrderCompleted", EventTypeOrderCompleted, "order.completed"},
		{"OrderCancelled", EventTypeOrderCancelled, "order.cancelled"},
		{"StockReserved", EventTypeStockReserved, "inventory.stock_reserved"},
		{"StockReserveFailed", EventTypeStockReserveFailed, "inventory.stock_reserve_failed"},
		{"StockReleased", EventTypeStockReleased, "inventory.stock_released"},
		{"PaymentCompleted", EventTypePaymentCompleted, "payment.completed"},
		{"PaymentFailed", EventTypePaymentFailed, "payment.failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("constant = %s, want %s", tt.constant, tt.want)
			}
		})
	}
}

// TestBaseEvent_UniqueEventID 各イベントのEventIDが一意であることを確認
func TestBaseEvent_UniqueEventID(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		event := NewBaseEvent("test")
		if ids[event.EventID] {
			t.Errorf("重複するEventID: %s", event.EventID)
		}
		ids[event.EventID] = true
	}
}
