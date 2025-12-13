package domain

import (
	"testing"
)

// TestInventory_AvailableQuantity 利用可能数量計算テスト
func TestInventory_AvailableQuantity(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 30,
	}

	if inv.AvailableQuantity() != 70 {
		t.Errorf("AvailableQuantity = %d, want 70", inv.AvailableQuantity())
	}
}

// TestInventory_AvailableQuantity_全て予約済み 全て予約済みの場合
func TestInventory_AvailableQuantity_全て予約済み(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 100,
	}

	if inv.AvailableQuantity() != 0 {
		t.Errorf("AvailableQuantity = %d, want 0", inv.AvailableQuantity())
	}
}

// TestInventory_CanReserve 予約可能チェックテスト
func TestInventory_CanReserve(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 30,
	}

	tests := []struct {
		quantity int
		want     bool
	}{
		{70, true},   // ちょうど利用可能
		{69, true},   // 利用可能未満
		{1, true},    // 最小
		{71, false},  // 超過
		{100, false}, // 全体数量
	}

	for _, tt := range tests {
		if got := inv.CanReserve(tt.quantity); got != tt.want {
			t.Errorf("CanReserve(%d) = %v, want %v", tt.quantity, got, tt.want)
		}
	}
}

// TestInventory_Reserve_正常系 予約成功テスト
func TestInventory_Reserve_正常系(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 30,
	}

	err := inv.Reserve(20)

	if err != nil {
		t.Fatalf("Reserve失敗: %v", err)
	}
	if inv.ReservedQuantity != 50 {
		t.Errorf("ReservedQuantity = %d, want 50", inv.ReservedQuantity)
	}
}

// TestInventory_Reserve_在庫不足 在庫不足時のエラー
func TestInventory_Reserve_在庫不足(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 90,
	}

	err := inv.Reserve(20)

	if err != ErrInsufficientStock {
		t.Errorf("err = %v, want ErrInsufficientStock", err)
	}
	// ReservedQuantityが変わっていないこと
	if inv.ReservedQuantity != 90 {
		t.Errorf("ReservedQuantity = %d, want 90", inv.ReservedQuantity)
	}
}

// TestInventory_Reserve_数量0 数量0のエラー
func TestInventory_Reserve_数量0(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 0,
	}

	err := inv.Reserve(0)

	if err != ErrInvalidQuantity {
		t.Errorf("err = %v, want ErrInvalidQuantity", err)
	}
}

// TestInventory_Reserve_数量負 負の数量のエラー
func TestInventory_Reserve_数量負(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 0,
	}

	err := inv.Reserve(-10)

	if err != ErrInvalidQuantity {
		t.Errorf("err = %v, want ErrInvalidQuantity", err)
	}
}

// TestInventory_Release_正常系 解放成功テスト
func TestInventory_Release_正常系(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 50,
	}

	err := inv.Release(30)

	if err != nil {
		t.Fatalf("Release失敗: %v", err)
	}
	if inv.ReservedQuantity != 20 {
		t.Errorf("ReservedQuantity = %d, want 20", inv.ReservedQuantity)
	}
}

// TestInventory_Release_予約超過 予約数を超える解放
func TestInventory_Release_予約超過(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 10,
	}

	err := inv.Release(30)

	if err != nil {
		t.Fatalf("Release失敗: %v", err)
	}
	// 0になること
	if inv.ReservedQuantity != 0 {
		t.Errorf("ReservedQuantity = %d, want 0", inv.ReservedQuantity)
	}
}

// TestInventory_Release_数量0 数量0のエラー
func TestInventory_Release_数量0(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 50,
	}

	err := inv.Release(0)

	if err != ErrInvalidQuantity {
		t.Errorf("err = %v, want ErrInvalidQuantity", err)
	}
}

// TestInventory_Release_数量負 負の数量のエラー
func TestInventory_Release_数量負(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 50,
	}

	err := inv.Release(-10)

	if err != ErrInvalidQuantity {
		t.Errorf("err = %v, want ErrInvalidQuantity", err)
	}
}

// TestInventory_Commit_正常系 コミット成功テスト
func TestInventory_Commit_正常系(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 30,
	}

	err := inv.Commit(20)

	if err != nil {
		t.Fatalf("Commit失敗: %v", err)
	}
	if inv.Quantity != 80 {
		t.Errorf("Quantity = %d, want 80", inv.Quantity)
	}
	if inv.ReservedQuantity != 10 {
		t.Errorf("ReservedQuantity = %d, want 10", inv.ReservedQuantity)
	}
}

// TestInventory_Commit_予約不足 予約数不足時のエラー
func TestInventory_Commit_予約不足(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 10,
	}

	err := inv.Commit(20)

	if err != ErrInsufficientStock {
		t.Errorf("err = %v, want ErrInsufficientStock", err)
	}
}

// TestInventory_Commit_数量0 数量0のエラー
func TestInventory_Commit_数量0(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 30,
	}

	err := inv.Commit(0)

	if err != ErrInvalidQuantity {
		t.Errorf("err = %v, want ErrInvalidQuantity", err)
	}
}

// TestNewReservation_正常系 予約生成テスト
func TestNewReservation_正常系(t *testing.T) {
	res, err := NewReservation("order-1", "prod-1", 5)

	if err != nil {
		t.Fatalf("NewReservation失敗: %v", err)
	}
	if res.ID == "" {
		t.Error("ID が空")
	}
	if res.OrderID != "order-1" {
		t.Errorf("OrderID = %s, want order-1", res.OrderID)
	}
	if res.ProductID != "prod-1" {
		t.Errorf("ProductID = %s, want prod-1", res.ProductID)
	}
	if res.Quantity != 5 {
		t.Errorf("Quantity = %d, want 5", res.Quantity)
	}
	if res.Status != ReservationStatusReserved {
		t.Errorf("Status = %s, want %s", res.Status, ReservationStatusReserved)
	}
}

// TestNewReservation_数量0 数量0のエラー
func TestNewReservation_数量0(t *testing.T) {
	_, err := NewReservation("order-1", "prod-1", 0)

	if err != ErrInvalidQuantity {
		t.Errorf("err = %v, want ErrInvalidQuantity", err)
	}
}

// TestNewReservation_数量負 負の数量のエラー
func TestNewReservation_数量負(t *testing.T) {
	_, err := NewReservation("order-1", "prod-1", -5)

	if err != ErrInvalidQuantity {
		t.Errorf("err = %v, want ErrInvalidQuantity", err)
	}
}

// TestReservation_Release 予約解放テスト
func TestReservation_Release(t *testing.T) {
	res, _ := NewReservation("order-1", "prod-1", 5)

	res.Release()

	if res.Status != ReservationStatusReleased {
		t.Errorf("Status = %s, want %s", res.Status, ReservationStatusReleased)
	}
}

// TestReservation_Commit 予約確定テスト
func TestReservation_Commit(t *testing.T) {
	res, _ := NewReservation("order-1", "prod-1", 5)

	res.Commit()

	if res.Status != ReservationStatusCommitted {
		t.Errorf("Status = %s, want %s", res.Status, ReservationStatusCommitted)
	}
}

// TestReservation_IsReserved 予約状態チェックテスト
func TestReservation_IsReserved(t *testing.T) {
	res, _ := NewReservation("order-1", "prod-1", 5)

	if !res.IsReserved() {
		t.Error("IsReserved = false, want true")
	}

	res.Release()
	if res.IsReserved() {
		t.Error("IsReserved = true, want false (after release)")
	}
}

// TestReservationStatusConstants ステータス定数テスト
func TestReservationStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"Reserved", ReservationStatusReserved, "RESERVED"},
		{"Released", ReservationStatusReleased, "RELEASED"},
		{"Committed", ReservationStatusCommitted, "COMMITTED"},
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
		{"ErrInsufficientStock", ErrInsufficientStock, "insufficient stock"},
		{"ErrProductNotFound", ErrProductNotFound, "product not found"},
		{"ErrReservationNotFound", ErrReservationNotFound, "reservation not found"},
		{"ErrInvalidQuantity", ErrInvalidQuantity, "quantity must be greater than 0"},
	}

	for _, e := range errors {
		t.Run(e.name, func(t *testing.T) {
			if e.err.Error() != e.msg {
				t.Errorf("error message = %s, want %s", e.err.Error(), e.msg)
			}
		})
	}
}

// TestInventory_境界値_利用可能ちょうど 境界値テスト
func TestInventory_境界値_利用可能ちょうど(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 50,
	}

	// ちょうど利用可能な数量を予約
	err := inv.Reserve(50)

	if err != nil {
		t.Fatalf("Reserve失敗: %v", err)
	}
	if inv.AvailableQuantity() != 0 {
		t.Errorf("AvailableQuantity = %d, want 0", inv.AvailableQuantity())
	}
}

// TestInventory_境界値_1つ超過 境界値テスト（1つ超過）
func TestInventory_境界値_1つ超過(t *testing.T) {
	inv := &Inventory{
		Quantity:         100,
		ReservedQuantity: 50,
	}

	err := inv.Reserve(51)

	if err != ErrInsufficientStock {
		t.Errorf("err = %v, want ErrInsufficientStock", err)
	}
}
