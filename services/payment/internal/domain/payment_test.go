package domain

import (
	"testing"
)

// TestNewPayment_正常系 決済生成テスト
func TestNewPayment_正常系(t *testing.T) {
	payment, err := NewPayment("order-1", "cust-1", 100.50)

	if err != nil {
		t.Fatalf("NewPayment失敗: %v", err)
	}
	if payment.ID == "" {
		t.Error("ID が空")
	}
	if payment.OrderID != "order-1" {
		t.Errorf("OrderID = %s, want order-1", payment.OrderID)
	}
	if payment.CustomerID != "cust-1" {
		t.Errorf("CustomerID = %s, want cust-1", payment.CustomerID)
	}
	if payment.Amount != 100.50 {
		t.Errorf("Amount = %f, want 100.50", payment.Amount)
	}
	if payment.Status != PaymentStatusPending {
		t.Errorf("Status = %s, want %s", payment.Status, PaymentStatusPending)
	}
	if payment.FailureReason != "" {
		t.Errorf("FailureReason = %s, want empty", payment.FailureReason)
	}
}

// TestNewPayment_Amount0 金額0のエラー
func TestNewPayment_Amount0(t *testing.T) {
	_, err := NewPayment("order-1", "cust-1", 0)

	if err != ErrInvalidAmount {
		t.Errorf("err = %v, want ErrInvalidAmount", err)
	}
}

// TestNewPayment_Amount負 負の金額のエラー
func TestNewPayment_Amount負(t *testing.T) {
	_, err := NewPayment("order-1", "cust-1", -100)

	if err != ErrInvalidAmount {
		t.Errorf("err = %v, want ErrInvalidAmount", err)
	}
}

// TestNewPayment_境界値_最小金額 最小金額テスト
func TestNewPayment_境界値_最小金額(t *testing.T) {
	payment, err := NewPayment("order-1", "cust-1", 0.01)

	if err != nil {
		t.Fatalf("NewPayment失敗: %v", err)
	}
	if payment.Amount != 0.01 {
		t.Errorf("Amount = %f, want 0.01", payment.Amount)
	}
}

// TestPayment_Complete_正常系 決済完了テスト
func TestPayment_Complete_正常系(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)

	err := payment.Complete()

	if err != nil {
		t.Fatalf("Complete失敗: %v", err)
	}
	if payment.Status != PaymentStatusCompleted {
		t.Errorf("Status = %s, want %s", payment.Status, PaymentStatusCompleted)
	}
}

// TestPayment_Complete_既にCompleted 既にCompletedの場合
func TestPayment_Complete_既にCompleted(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)
	payment.Status = PaymentStatusCompleted

	err := payment.Complete()

	if err != ErrInvalidTransition {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

// TestPayment_Complete_Failedから Failed状態からの遷移
func TestPayment_Complete_Failedから(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)
	payment.Status = PaymentStatusFailed

	err := payment.Complete()

	if err != ErrInvalidTransition {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

// TestPayment_Fail_正常系 決済失敗テスト
func TestPayment_Fail_正常系(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)

	err := payment.Fail("card declined")

	if err != nil {
		t.Fatalf("Fail失敗: %v", err)
	}
	if payment.Status != PaymentStatusFailed {
		t.Errorf("Status = %s, want %s", payment.Status, PaymentStatusFailed)
	}
	if payment.FailureReason != "card declined" {
		t.Errorf("FailureReason = %s, want 'card declined'", payment.FailureReason)
	}
}

// TestPayment_Fail_既にFailed 既にFailedの場合
func TestPayment_Fail_既にFailed(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)
	payment.Status = PaymentStatusFailed

	err := payment.Fail("another reason")

	if err != ErrInvalidTransition {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

// TestPayment_Fail_Completedから Completed状態からの遷移
func TestPayment_Fail_Completedから(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)
	payment.Status = PaymentStatusCompleted

	err := payment.Fail("some reason")

	if err != ErrInvalidTransition {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

// TestPayment_Refund_正常系 返金テスト
func TestPayment_Refund_正常系(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)
	payment.Status = PaymentStatusCompleted

	err := payment.Refund()

	if err != nil {
		t.Fatalf("Refund失敗: %v", err)
	}
	if payment.Status != PaymentStatusRefunded {
		t.Errorf("Status = %s, want %s", payment.Status, PaymentStatusRefunded)
	}
}

// TestPayment_Refund_Pendingから Pending状態からの遷移
func TestPayment_Refund_Pendingから(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)

	err := payment.Refund()

	if err != ErrInvalidTransition {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

// TestPayment_Refund_Failedから Failed状態からの遷移
func TestPayment_Refund_Failedから(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)
	payment.Status = PaymentStatusFailed

	err := payment.Refund()

	if err != ErrInvalidTransition {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}

// TestPayment_IsCompleted 完了チェックテスト
func TestPayment_IsCompleted(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)

	if payment.IsCompleted() {
		t.Error("IsCompleted = true, want false")
	}

	payment.Status = PaymentStatusCompleted
	if !payment.IsCompleted() {
		t.Error("IsCompleted = false, want true")
	}
}

// TestPayment_IsFailed 失敗チェックテスト
func TestPayment_IsFailed(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)

	if payment.IsFailed() {
		t.Error("IsFailed = true, want false")
	}

	payment.Status = PaymentStatusFailed
	if !payment.IsFailed() {
		t.Error("IsFailed = false, want true")
	}
}

// TestPaymentStatusConstants ステータス定数テスト
func TestPaymentStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"Pending", PaymentStatusPending, "PENDING"},
		{"Completed", PaymentStatusCompleted, "COMPLETED"},
		{"Failed", PaymentStatusFailed, "FAILED"},
		{"Refunded", PaymentStatusRefunded, "REFUNDED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("constant = %s, want %s", tt.constant, tt.want)
			}
		})
	}
}

// TestErrors エラー定義テスト
func TestErrors(t *testing.T) {
	errors := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrInvalidAmount", ErrInvalidAmount, "amount must be greater than 0"},
		{"ErrPaymentNotFound", ErrPaymentNotFound, "payment not found"},
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

// TestPayment_IDはUUID IDがUUID形式であることを確認
func TestPayment_IDはUUID(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)

	// UUID形式（8-4-4-4-12）
	if len(payment.ID) != 36 {
		t.Errorf("ID length = %d, want 36", len(payment.ID))
	}
}

// TestPayment_Timestamps タイムスタンプが設定されることを確認
func TestPayment_Timestamps(t *testing.T) {
	payment, _ := NewPayment("order-1", "cust-1", 100)

	if payment.CreatedAt.IsZero() {
		t.Error("CreatedAt が未設定")
	}
	if payment.UpdatedAt.IsZero() {
		t.Error("UpdatedAt が未設定")
	}
}

// TestPayment_状態遷移_全パターン 全状態遷移パターンテスト
func TestPayment_状態遷移_全パターン(t *testing.T) {
	tests := []struct {
		name       string
		initial    string
		action     string
		wantErr    bool
		wantStatus string
	}{
		{"PENDING->COMPLETED", PaymentStatusPending, "complete", false, PaymentStatusCompleted},
		{"PENDING->FAILED", PaymentStatusPending, "fail", false, PaymentStatusFailed},
		{"COMPLETED->REFUNDED", PaymentStatusCompleted, "refund", false, PaymentStatusRefunded},
		{"COMPLETED->COMPLETED", PaymentStatusCompleted, "complete", true, ""},
		{"COMPLETED->FAILED", PaymentStatusCompleted, "fail", true, ""},
		{"FAILED->COMPLETED", PaymentStatusFailed, "complete", true, ""},
		{"FAILED->REFUNDED", PaymentStatusFailed, "refund", true, ""},
		{"REFUNDED->COMPLETED", PaymentStatusRefunded, "complete", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, _ := NewPayment("order-1", "cust-1", 100)
			payment.Status = tt.initial

			var err error
			switch tt.action {
			case "complete":
				err = payment.Complete()
			case "fail":
				err = payment.Fail("reason")
			case "refund":
				err = payment.Refund()
			}

			if tt.wantErr && err == nil {
				t.Error("エラーが発生すべき")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("予期しないエラー: %v", err)
			}
			if !tt.wantErr && payment.Status != tt.wantStatus {
				t.Errorf("Status = %s, want %s", payment.Status, tt.wantStatus)
			}
		})
	}
}
