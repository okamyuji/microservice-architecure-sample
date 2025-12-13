package interfaces

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/outbox"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/testutil"
	"github.com/yujiokamoto/microservice-architecture-sample/services/order/internal/application"
	"github.com/yujiokamoto/microservice-architecture-sample/services/order/internal/infrastructure"
)

// testLogger テスト用ロガー
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// setupTestHandler テスト用ハンドラをセットアップ
func setupTestHandler(t *testing.T) (*HTTPHandler, func()) {
	t.Helper()

	ctx := context.Background()
	db, err := testutil.GetOrderTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	repo := infrastructure.NewPostgresOrderRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := application.NewOrderUseCase(db.Pool, repo, publisher, checker, testLogger())
	handler := NewHTTPHandler(uc)

	cleanup := func() {
		_ = db.CleanupOrderDB(ctx)
	}

	return handler, cleanup
}

// TestNewHTTPHandler ハンドラ生成テスト
func TestNewHTTPHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, cleanup := setupTestHandler(t)
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

	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()
	handler.RegisterRoutes(e)

	routes := e.Routes()
	routeMap := make(map[string]bool)
	for _, r := range routes {
		routeMap[r.Method+" "+r.Path] = true
	}

	expectedRoutes := []string{
		"POST /orders",
		"GET /orders/:id",
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

	handler, cleanup := setupTestHandler(t)
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

// TestHTTPHandler_CreateOrder_正常系 注文作成成功
func TestHTTPHandler_CreateOrder_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()
	body := CreateOrderRequest{
		CustomerID:  "cust-1",
		ProductID:   "prod-1",
		Quantity:    5,
		TotalAmount: 100.50,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.CreateOrder(c); err != nil {
		t.Fatalf("CreateOrder失敗: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var resp CreateOrderResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("レスポンスパース失敗: %v", err)
	}

	if resp.OrderID == "" {
		t.Error("OrderID が空")
	}
	if resp.Status != "PENDING" {
		t.Errorf("Status = %s, want PENDING", resp.Status)
	}
}

// TestHTTPHandler_CreateOrder_バリデーションエラー バリデーション失敗
func TestHTTPHandler_CreateOrder_バリデーションエラー(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	tests := []struct {
		name string
		body CreateOrderRequest
	}{
		{
			name: "CustomerID空",
			body: CreateOrderRequest{
				CustomerID:  "",
				ProductID:   "prod-1",
				Quantity:    5,
				TotalAmount: 100,
			},
		},
		{
			name: "ProductID空",
			body: CreateOrderRequest{
				CustomerID:  "cust-1",
				ProductID:   "",
				Quantity:    5,
				TotalAmount: 100,
			},
		},
		{
			name: "Quantity0",
			body: CreateOrderRequest{
				CustomerID:  "cust-1",
				ProductID:   "prod-1",
				Quantity:    0,
				TotalAmount: 100,
			},
		},
		{
			name: "TotalAmount0",
			body: CreateOrderRequest{
				CustomerID:  "cust-1",
				ProductID:   "prod-1",
				Quantity:    5,
				TotalAmount: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(bodyBytes))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = handler.CreateOrder(c)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

// TestHTTPHandler_CreateOrder_不正なJSON 不正なJSONリクエスト
func TestHTTPHandler_CreateOrder_不正なJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader([]byte("invalid json")))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.CreateOrder(c)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestHTTPHandler_GetOrder_正常系 注文取得成功
func TestHTTPHandler_GetOrder_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()

	// まず注文を作成
	body := CreateOrderRequest{
		CustomerID:  "cust-get",
		ProductID:   "prod-get",
		Quantity:    3,
		TotalAmount: 50.00,
	}
	bodyBytes, _ := json.Marshal(body)
	createReq := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(bodyBytes))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	createC := e.NewContext(createReq, createRec)
	_ = handler.CreateOrder(createC)

	var createResp CreateOrderResponse
	_ = json.Unmarshal(createRec.Body.Bytes(), &createResp)

	// 取得
	getReq := httptest.NewRequest(http.MethodGet, "/orders/"+createResp.OrderID, nil)
	getRec := httptest.NewRecorder()
	getC := e.NewContext(getReq, getRec)
	getC.SetParamNames("id")
	getC.SetParamValues(createResp.OrderID)

	if err := handler.GetOrder(getC); err != nil {
		t.Fatalf("GetOrder失敗: %v", err)
	}

	if getRec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", getRec.Code, http.StatusOK)
	}

	var resp OrderResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("レスポンスパース失敗: %v", err)
	}

	if resp.ID != createResp.OrderID {
		t.Errorf("ID = %s, want %s", resp.ID, createResp.OrderID)
	}
}

// TestHTTPHandler_GetOrder_NotFound 存在しない注文
func TestHTTPHandler_GetOrder_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/orders/"+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.New().String())

	_ = handler.GetOrder(c)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestHTTPHandler_GetOrder_IDなし IDパラメータなし
func TestHTTPHandler_GetOrder_IDなし(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/orders/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("")

	_ = handler.GetOrder(c)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestCreateOrderRequest リクエスト構造体テスト
func TestCreateOrderRequest(t *testing.T) {
	req := CreateOrderRequest{
		CustomerID:  "cust-1",
		ProductID:   "prod-1",
		Quantity:    5,
		TotalAmount: 100.50,
	}

	data, _ := json.Marshal(req)
	var parsed CreateOrderRequest
	_ = json.Unmarshal(data, &parsed)

	if parsed.CustomerID != req.CustomerID {
		t.Errorf("CustomerID = %s, want %s", parsed.CustomerID, req.CustomerID)
	}
}

// TestCreateOrderResponse レスポンス構造体テスト
func TestCreateOrderResponse(t *testing.T) {
	resp := CreateOrderResponse{
		OrderID: "order-1",
		Status:  "PENDING",
	}

	data, _ := json.Marshal(resp)
	var parsed CreateOrderResponse
	_ = json.Unmarshal(data, &parsed)

	if parsed.OrderID != resp.OrderID {
		t.Errorf("OrderID = %s, want %s", parsed.OrderID, resp.OrderID)
	}
}

// TestOrderResponse レスポンス構造体テスト
func TestOrderResponse(t *testing.T) {
	resp := OrderResponse{
		ID:          "id-1",
		CustomerID:  "cust-1",
		ProductID:   "prod-1",
		Quantity:    5,
		TotalAmount: 100.50,
		Status:      "PENDING",
		CreatedAt:   "2025-01-01T00:00:00Z",
		UpdatedAt:   "2025-01-01T00:00:00Z",
	}

	data, _ := json.Marshal(resp)
	var parsed OrderResponse
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
