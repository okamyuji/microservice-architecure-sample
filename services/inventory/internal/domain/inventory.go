// Package domain 在庫ドメインエンティティとビジネスロジック
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// 予約ステータス定数
const (
	ReservationStatusReserved  = "RESERVED"
	ReservationStatusReleased  = "RELEASED"
	ReservationStatusCommitted = "COMMITTED"
)

// バリデーションエラー
var (
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrProductNotFound     = errors.New("product not found")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrInvalidQuantity     = errors.New("quantity must be greater than 0")
)

// Inventory 商品在庫
type Inventory struct {
	ID               string
	ProductID        string
	ProductName      string
	Quantity         int
	ReservedQuantity int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// AvailableQuantity 予約可能数量を返却
func (i *Inventory) AvailableQuantity() int {
	return i.Quantity - i.ReservedQuantity
}

// CanReserve 指定数量を予約可能か確認
func (i *Inventory) CanReserve(quantity int) bool {
	return i.AvailableQuantity() >= quantity
}

// Reserve 指定数量を予約
func (i *Inventory) Reserve(quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}
	if !i.CanReserve(quantity) {
		return ErrInsufficientStock
	}
	i.ReservedQuantity += quantity
	i.UpdatedAt = time.Now()
	return nil
}

// Release 予約数量を解放
func (i *Inventory) Release(quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}
	if i.ReservedQuantity < quantity {
		// 可能な分だけ解放
		i.ReservedQuantity = 0
	} else {
		i.ReservedQuantity -= quantity
	}
	i.UpdatedAt = time.Now()
	return nil
}

// Commit 予約を確定し実在庫を減少
func (i *Inventory) Commit(quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}
	if i.ReservedQuantity < quantity {
		return ErrInsufficientStock
	}
	i.Quantity -= quantity
	i.ReservedQuantity -= quantity
	i.UpdatedAt = time.Now()
	return nil
}

// Reservation 注文に対する在庫予約
type Reservation struct {
	ID        string
	OrderID   string
	ProductID string
	Quantity  int
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewReservation 新規予約を生成
func NewReservation(orderID, productID string, quantity int) (*Reservation, error) {
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}

	now := time.Now()
	return &Reservation{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		ProductID: productID,
		Quantity:  quantity,
		Status:    ReservationStatusReserved,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Release 予約を解放済みとしてマーク
func (r *Reservation) Release() {
	r.Status = ReservationStatusReleased
	r.UpdatedAt = time.Now()
}

// Commit 予約を確定済みとしてマーク
func (r *Reservation) Commit() {
	r.Status = ReservationStatusCommitted
	r.UpdatedAt = time.Now()
}

// IsReserved 予約状態かを返却
func (r *Reservation) IsReserved() bool {
	return r.Status == ReservationStatusReserved
}
