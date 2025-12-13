// Package domain 注文ドメインエンティティとビジネスロジック
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// 注文ステータス定数
const (
	OrderStatusPending   = "PENDING"
	OrderStatusConfirmed = "CONFIRMED"
	OrderStatusCompleted = "COMPLETED"
	OrderStatusCancelled = "CANCELLED"
)

// バリデーションエラー
var (
	ErrInvalidQuantity    = errors.New("quantity must be greater than 0")
	ErrInvalidTotalAmount = errors.New("total amount must be greater than 0")
	ErrInvalidCustomerID  = errors.New("customer ID is required")
	ErrInvalidProductID   = errors.New("product ID is required")
	ErrOrderNotFound      = errors.New("order not found")
	ErrInvalidTransition  = errors.New("invalid status transition")
)

// Order 注文集約ルート
type Order struct {
	ID          string
	CustomerID  string
	ProductID   string
	Quantity    int
	TotalAmount float64
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewOrder バリデーション付きで新規注文を生成
func NewOrder(customerID, productID string, quantity int, totalAmount float64) (*Order, error) {
	if customerID == "" {
		return nil, ErrInvalidCustomerID
	}
	if productID == "" {
		return nil, ErrInvalidProductID
	}
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}
	if totalAmount <= 0 {
		return nil, ErrInvalidTotalAmount
	}

	now := time.Now()
	return &Order{
		ID:          uuid.New().String(),
		CustomerID:  customerID,
		ProductID:   productID,
		Quantity:    quantity,
		TotalAmount: totalAmount,
		Status:      OrderStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Confirm 注文を確認済み（在庫予約完了）としてマーク
func (o *Order) Confirm() error {
	if o.Status != OrderStatusPending {
		return ErrInvalidTransition
	}
	o.Status = OrderStatusConfirmed
	o.UpdatedAt = time.Now()
	return nil
}

// Complete 注文を完了（決済成功）としてマーク
func (o *Order) Complete() error {
	if o.Status != OrderStatusConfirmed {
		return ErrInvalidTransition
	}
	o.Status = OrderStatusCompleted
	o.UpdatedAt = time.Now()
	return nil
}

// Cancel 注文をキャンセルとしてマーク
func (o *Order) Cancel() error {
	if o.Status == OrderStatusCompleted {
		return ErrInvalidTransition
	}
	o.Status = OrderStatusCancelled
	o.UpdatedAt = time.Now()
	return nil
}

// IsPending ペンディング状態かを返却
func (o *Order) IsPending() bool {
	return o.Status == OrderStatusPending
}

// IsCompleted 完了状態かを返却
func (o *Order) IsCompleted() bool {
	return o.Status == OrderStatusCompleted
}

// IsCancelled キャンセル状態かを返却
func (o *Order) IsCancelled() bool {
	return o.Status == OrderStatusCancelled
}
