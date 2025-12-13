package interfaces

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"microservice-architecture-sample/pkg/outbox"
	"microservice-architecture-sample/pkg/testutil"
	"microservice-architecture-sample/services/payment/internal/application"
	"microservice-architecture-sample/services/payment/internal/infrastructure"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// testLogger テスト用ロガー
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// setupTestHandler テスト用ハンドラをセットアップ
func setupTestHandler(t *testing.T) (*HTTPHandler, *application.PaymentUseCase, func()) {
	t.Helper()

	ctx := context.Background()
	db, err := testutil.GetPaymentTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := infrastructure.NewPostgresPaymentRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := application.NewPaymentUseCase(db.Pool, repo, publisher, checker, testLogger())
	handler := NewHTTPHandler(uc)

	cleanup := func() {
		_ = db.CleanupPaymentDB(ctx)
	}

	return handler, uc, cleanup
}

// TestNewHTTPHandler ハンドラ生成テスト
func TestNewHTTPHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	if handler == nil {
		t.Fatal("handler が nil")
	}
}

// TestHTTPHandler_RegisterRoutes ルート登録テスト
func TestHTTPHandler_RegisterRoutes(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()
	handler.RegisterRoutes(e)

	routes := e.Routes()
	routeMap := make(map[string]bool)
	for _, r := range routes {
		routeMap[r.Method+" "+r.Path] = true
	}

	expectedRoutes := []string{
		"GET /payments/order/:order_id",
		"GET /health",
	}

	for _, expected := range expectedRoutes {
		if !routeMap[expected] {
			t.Errorf("ルート %s が登録されていない", expected)
		}
	}
}

// TestHTTPHandler_HealthCheck ヘルスチェック
func TestHTTPHandler_HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.HealthCheck(c); err != nil {
		t.Fatalf("HealthCheck失敗: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestHTTPHandler_GetPaymentByOrder_正常系 決済取得成功
func TestHTTPHandler_GetPaymentByOrder_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, uc, cleanup := setupTestHandler(t)
	defer cleanup()

	ctx := context.Background()
	orderID := uuid.New().String()
	eventID := uuid.New().String()

	// 決済を作成（.99で失敗）
	_ = uc.HandleStockReserved(ctx, eventID, orderID, "cust-1", 100.99)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/payments/order/"+orderID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("order_id")
	c.SetParamValues(orderID)

	if err := handler.GetPaymentByOrder(c); err != nil {
		t.Fatalf("GetPaymentByOrder失敗: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp PaymentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("レスポンスパース失敗: %v", err)
	}

	if resp.OrderID != orderID {
		t.Errorf("OrderID = %s, want %s", resp.OrderID, orderID)
	}
}

// TestHTTPHandler_GetPaymentByOrder_NotFound 存在しない決済
func TestHTTPHandler_GetPaymentByOrder_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/payments/order/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("order_id")
	c.SetParamValues(uuid.New().String())

	_ = handler.GetPaymentByOrder(c)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestHTTPHandler_GetPaymentByOrder_IDなし 注文IDなし
func TestHTTPHandler_GetPaymentByOrder_IDなし(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, _, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/payments/order/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("order_id")
	c.SetParamValues("")

	_ = handler.GetPaymentByOrder(c)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestPaymentResponse レスポンス構造体テスト
func TestPaymentResponse(t *testing.T) {
	resp := PaymentResponse{
		ID:            "pay-1",
		OrderID:       "order-1",
		CustomerID:    "cust-1",
		Amount:        100.50,
		Status:        "COMPLETED",
		FailureReason: "",
		CreatedAt:     "2025-01-01T00:00:00Z",
		UpdatedAt:     "2025-01-01T00:00:00Z",
	}

	data, _ := json.Marshal(resp)
	var parsed PaymentResponse
	_ = json.Unmarshal(data, &parsed)

	if parsed.ID != resp.ID {
		t.Errorf("ID = %s, want %s", parsed.ID, resp.ID)
	}
}

// TestErrorResponse エラーレスポンス構造体テスト
func TestErrorResponse(t *testing.T) {
	resp := ErrorResponse{
		Error:   "test_error",
		Message: "Test message",
	}

	data, _ := json.Marshal(resp)
	var parsed ErrorResponse
	_ = json.Unmarshal(data, &parsed)

	if parsed.Error != resp.Error {
		t.Errorf("Error = %s, want %s", parsed.Error, resp.Error)
	}
}
