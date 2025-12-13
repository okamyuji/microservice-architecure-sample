// Package domain 決済ドメインエンティティとビジネスロジック
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// 決済ステータス定数
const (
	PaymentStatusPending   = "PENDING"
	PaymentStatusCompleted = "COMPLETED"
	PaymentStatusFailed    = "FAILED"
	PaymentStatusRefunded  = "REFUNDED"
)

// バリデーションエラー
var (
	ErrInvalidAmount     = errors.New("amount must be greater than 0")
	ErrPaymentNotFound   = errors.New("payment not found")
	ErrInvalidTransition = errors.New("invalid status transition")
)

// Payment 決済集約ルート
type Payment struct {
	ID            string
	OrderID       string
	CustomerID    string
	Amount        float64
	Status        string
	FailureReason string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewPayment バリデーション付きで新規決済を生成
func NewPayment(orderID, customerID string, amount float64) (*Payment, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	now := time.Now()
	return &Payment{
		ID:         uuid.New().String(),
		OrderID:    orderID,
		CustomerID: customerID,
		Amount:     amount,
		Status:     PaymentStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

// Complete 決済を完了としてマーク
func (p *Payment) Complete() error {
	if p.Status != PaymentStatusPending {
		return ErrInvalidTransition
	}
	p.Status = PaymentStatusCompleted
	p.UpdatedAt = time.Now()
	return nil
}

// Fail 決済を失敗としてマークし理由を記録
func (p *Payment) Fail(reason string) error {
	if p.Status != PaymentStatusPending {
		return ErrInvalidTransition
	}
	p.Status = PaymentStatusFailed
	p.FailureReason = reason
	p.UpdatedAt = time.Now()
	return nil
}

// Refund 決済を返金済みとしてマーク
func (p *Payment) Refund() error {
	if p.Status != PaymentStatusCompleted {
		return ErrInvalidTransition
	}
	p.Status = PaymentStatusRefunded
	p.UpdatedAt = time.Now()
	return nil
}

// IsCompleted 完了状態かを返却
func (p *Payment) IsCompleted() bool {
	return p.Status == PaymentStatusCompleted
}

// IsFailed 失敗状態かを返却
func (p *Payment) IsFailed() bool {
	return p.Status == PaymentStatusFailed
}
