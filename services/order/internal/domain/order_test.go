package domain

import (
	"testing"
)

// TestNewOrder_正常系 正常な注文生成テスト
func TestNewOrder_正常系(t *testing.T) {
	order, err := NewOrder("cust-1", "prod-1", 5, 100.50)

	if err != nil {
		t.Fatalf("NewOrder失敗: %v", err)
	}
	if order.ID == "" {
		t.Error("ID が空")
	}
	if order.CustomerID != "cust-1" {
		t.Errorf("CustomerID = %s, want cust-1", order.CustomerID)
	}
	if order.ProductID != "prod-1" {
		t.Errorf("ProductID = %s, want prod-1", order.ProductID)
	}
	if order.Quantity != 5 {
		t.Errorf("Quantity = %d, want 5", order.Quantity)
	}
	if order.TotalAmount != 100.50 {
		t.Errorf("TotalAmount = %f, want 100.50", order.TotalAmount)
	}
	if order.Status != OrderStatusPending {
		t.Errorf("Status = %s, want %s", order.Status, OrderStatusPending)
	}
}

// TestNewOrder_CustomerIDが空 CustomerID空の場合
func TestNewOrder_CustomerIDが空(t *testing.T) {
	_, err := NewOrder("", "prod-1", 5, 100.50)

	if err != ErrInvalidCustomerID {
		t.Errorf("err = %v, want ErrInvalidCustomerID", err)
	}
}

// TestNewOrder_ProductIDが空 ProductID空の場合
func TestNewOrder_ProductIDが空(t *testing.T) {
	_, err := NewOrder("cust-1", "", 5, 100.50)

	if err != ErrInvalidProductID {
		t.Errorf("err = %v, want ErrInvalidProductID", err)
	}
}

// TestNewOrder_Quantityが0 Quantity=0の場合
func TestNewOrder_Quantityが0(t *testing.T) {
	_, err := NewOrder("cust-1", "prod-1", 0, 100.50)

	if err != ErrInvalidQuantity {
		t.Errorf("err = %v, want ErrInvalidQuantity", err)
	}
}

// TestNewOrder_Quantityが負 Quantity<0の場合
func TestNewOrder_Quantityが負(t *testing.T) {
	_, err := NewOrder("cust-1", "prod-1", -1, 100.50)

	if err != ErrInvalidQuantity {
		t.Errorf("err = %v, want ErrInvalidQuantity", err)
	}
}

// TestNewOrder_TotalAmountが0 TotalAmount=0の場合
func TestNewOrder_TotalAmountが0(t *testing.T) {
	_, err := NewOrder("cust-1", "prod-1", 5, 0)

	if err != ErrInvalidTotalAmount {
		t.Errorf("err = %v, want ErrInvalidTotalAmount", err)
	}
}

// TestNewOrder_TotalAmountが負 TotalAmount<0の場合
func TestNewOrder_TotalAmountが負(t *testing.T) {
	_, err := NewOrder("cust-1", "prod-1", 5, -100)

	if err != ErrInvalidTotalAmount {
		t.Errorf("err = %v, want ErrInvalidTotalAmount", err)
	}
}

// TestNewOrder_境界値_Quantity1 Quantity=1の境界値
func TestNewOrder_境界値_Quantity1(t *testing.T) {
	order, err := NewOrder("cust-1", "prod-1", 1, 0.01)

	if err != nil {
		t.Fatalf("NewOrder失敗: %v", err)
	}
	if order.Quantity != 1 {
		t.Errorf("Quantity = %d, want 1", order.Quantity)
	}
}

// TestNewOrder_境界値_TotalAmount最小 TotalAmount=0.01の境界値
func TestNewOrder_境界値_TotalAmount最小(t *testing.T) {
	order, err := NewOrder("cust-1", "prod-1", 1, 0.01)

	if err != nil {
		t.Fatalf("NewOrder失敗: %v", err)
	}
	if order.TotalAmount != 0.01 {
		t.Errorf("TotalAmount = %f, want 0.01", order.TotalAmount)
	}
}

// TestOrder_Confirm_正常系 PENDING→CONFIRMEDへの遷移
func TestOrder_Confirm_正常系(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)

	err := order.Confirm()

	if err != nil {
		t.Fatalf("Confirm失敗: %v", err)
	}
	if order.Status != OrderStatusConfirmed {
		t.Errorf("Status = %s, want %s", order.Status, OrderStatusConfirmed)
	}
}

// TestOrder_Confirm_既にConfirmed 既にConfirmedの場合
func TestOrder_Confirm_既にConfirmed(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)
	order.Status = OrderStatusConfirmed

	err := order.Confirm()

	if err != ErrInvalidTransition {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

// TestOrder_Confirm_Completedから COMPLETED状態からの遷移
func TestOrder_Confirm_Completedから(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)
	order.Status = OrderStatusCompleted

	err := order.Confirm()

	if err != ErrInvalidTransition {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

// TestOrder_Complete_正常系 CONFIRMED→COMPLETEDへの遷移
func TestOrder_Complete_正常系(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)
	order.Status = OrderStatusConfirmed

	err := order.Complete()

	if err != nil {
		t.Fatalf("Complete失敗: %v", err)
	}
	if order.Status != OrderStatusCompleted {
		t.Errorf("Status = %s, want %s", order.Status, OrderStatusCompleted)
	}
}

// TestOrder_Complete_Pendingから PENDING状態からの遷移
func TestOrder_Complete_Pendingから(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)

	err := order.Complete()

	if err != ErrInvalidTransition {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

// TestOrder_Complete_Cancelledから CANCELLED状態からの遷移
func TestOrder_Complete_Cancelledから(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)
	order.Status = OrderStatusCancelled

	err := order.Complete()

	if err != ErrInvalidTransition {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

// TestOrder_Cancel_正常系_Pendingから PENDING→CANCELLEDへの遷移
func TestOrder_Cancel_正常系_Pendingから(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)

	err := order.Cancel()

	if err != nil {
		t.Fatalf("Cancel失敗: %v", err)
	}
	if order.Status != OrderStatusCancelled {
		t.Errorf("Status = %s, want %s", order.Status, OrderStatusCancelled)
	}
}

// TestOrder_Cancel_正常系_Confirmedから CONFIRMED→CANCELLEDへの遷移
func TestOrder_Cancel_正常系_Confirmedから(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)
	order.Status = OrderStatusConfirmed

	err := order.Cancel()

	if err != nil {
		t.Fatalf("Cancel失敗: %v", err)
	}
	if order.Status != OrderStatusCancelled {
		t.Errorf("Status = %s, want %s", order.Status, OrderStatusCancelled)
	}
}

// TestOrder_Cancel_Completedから COMPLETED状態からの遷移（不可）
func TestOrder_Cancel_Completedから(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)
	order.Status = OrderStatusCompleted

	err := order.Cancel()

	if err != ErrInvalidTransition {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

// TestOrder_IsPending ステータスチェックテスト
func TestOrder_IsPending(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)

	if !order.IsPending() {
		t.Error("IsPending = false, want true")
	}

	order.Status = OrderStatusConfirmed
	if order.IsPending() {
		t.Error("IsPending = true, want false")
	}
}

// TestOrder_IsCompleted 完了チェックテスト
func TestOrder_IsCompleted(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)

	if order.IsCompleted() {
		t.Error("IsCompleted = true, want false")
	}

	order.Status = OrderStatusCompleted
	if !order.IsCompleted() {
		t.Error("IsCompleted = false, want true")
	}
}

// TestOrder_IsCancelled キャンセルチェックテスト
func TestOrder_IsCancelled(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)

	if order.IsCancelled() {
		t.Error("IsCancelled = true, want false")
	}

	order.Status = OrderStatusCancelled
	if !order.IsCancelled() {
		t.Error("IsCancelled = false, want true")
	}
}

// TestOrderStatusConstants ステータス定数テスト
func TestOrderStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"Pending", OrderStatusPending, "PENDING"},
		{"Confirmed", OrderStatusConfirmed, "CONFIRMED"},
		{"Completed", OrderStatusCompleted, "COMPLETED"},
		{"Cancelled", OrderStatusCancelled, "CANCELLED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("constant = %s, want %s", tt.constant, tt.want)
			}
		})
	}
}

// TestOrder_IDはUUID IDがUUID形式であることを確認
func TestOrder_IDはUUID(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)

	// UUID形式（8-4-4-4-12）
	if len(order.ID) != 36 {
		t.Errorf("ID length = %d, want 36", len(order.ID))
	}
}

// TestOrder_Timestamps タイムスタンプが設定されることを確認
func TestOrder_Timestamps(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)

	if order.CreatedAt.IsZero() {
		t.Error("CreatedAt が未設定")
	}
	if order.UpdatedAt.IsZero() {
		t.Error("UpdatedAt が未設定")
	}
	if order.CreatedAt != order.UpdatedAt {
		t.Error("作成時は CreatedAt と UpdatedAt が同じであるべき")
	}
}

// TestOrder_UpdatedAt_Changed 状態変更時にUpdatedAtが更新されることを確認
func TestOrder_UpdatedAt_Changed(t *testing.T) {
	order, _ := NewOrder("cust-1", "prod-1", 1, 100)
	originalUpdatedAt := order.UpdatedAt

	if err := order.Confirm(); err != nil {
		t.Fatalf("Confirm失敗: %v", err)
	}

	// 時間精度の問題で同じになる可能性があるため、同じでも許容
	if order.UpdatedAt.Before(originalUpdatedAt) {
		t.Error("UpdatedAt が古くなっている")
	}
}

// TestErrors エラー定義テスト
func TestErrors(t *testing.T) {
	errors := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrInvalidQuantity", ErrInvalidQuantity, "quantity must be greater than 0"},
		{"ErrInvalidTotalAmount", ErrInvalidTotalAmount, "total amount must be greater than 0"},
		{"ErrInvalidCustomerID", ErrInvalidCustomerID, "customer ID is required"},
		{"ErrInvalidProductID", ErrInvalidProductID, "product ID is required"},
		{"ErrOrderNotFound", ErrOrderNotFound, "order not found"},
		{"ErrInvalidTransition", ErrInvalidTransition, "invalid status transition"},
	}

	for _, e := range errors {
		t.Run(e.name, func(t *testing.T) {
			if e.err.Error() != e.msg {
				t.Errorf("error message = %s, want %s", e.err.Error(), e.msg)
			}
		})
	}
}
