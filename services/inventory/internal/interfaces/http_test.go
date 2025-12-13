package interfaces

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/outbox"
	"github.com/yujiokamoto/microservice-architecture-sample/pkg/testutil"
	"github.com/yujiokamoto/microservice-architecture-sample/services/inventory/internal/application"
	"github.com/yujiokamoto/microservice-architecture-sample/services/inventory/internal/infrastructure"
)

// testLogger テスト用ロガー
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// setupTestHandler テスト用ハンドラをセットアップ
func setupTestHandler(t *testing.T) (*HTTPHandler, func()) {
	t.Helper()

	ctx := context.Background()
	db, err := testutil.GetInventoryTestDB(ctx)
	if err != nil {
		t.Fatalf("テストDB取得失敗: %v", err)
	}

	invRepo := infrastructure.NewPostgresInventoryRepository(db.Pool)
	resRepo := infrastructure.NewPostgresReservationRepository(db.Pool)
	publisher := outbox.NewPublisher(db.Pool, testLogger())
	checker := outbox.NewIdempotencyChecker(db.Pool)
	uc := application.NewInventoryUseCase(db.Pool, invRepo, resRepo, publisher, checker, testLogger())
	handler := NewHTTPHandler(uc)

	cleanup := func() {
		_ = db.CleanupInventoryDB(ctx)
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
		"GET /inventory/:product_id",
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

// TestHTTPHandler_GetInventory_正常系 在庫取得成功
func TestHTTPHandler_GetInventory_正常系(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/inventory/PROD-001", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("product_id")
	c.SetParamValues("PROD-001")

	if err := handler.GetInventory(c); err != nil {
		t.Fatalf("GetInventory失敗: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp InventoryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("レスポンスパース失敗: %v", err)
	}

	if resp.ProductID != "PROD-001" {
		t.Errorf("ProductID = %s, want PROD-001", resp.ProductID)
	}
}

// TestHTTPHandler_GetInventory_NotFound 存在しない商品
func TestHTTPHandler_GetInventory_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/inventory/PROD-NOTEXIST", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("product_id")
	c.SetParamValues("PROD-NOTEXIST")

	_ = handler.GetInventory(c)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestHTTPHandler_GetInventory_IDなし 商品IDなし
func TestHTTPHandler_GetInventory_IDなし(t *testing.T) {
	if testing.Short() {
		t.Skip("統合テストをスキップ")
	}

	handler, cleanup := setupTestHandler(t)
	defer cleanup()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/inventory/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("product_id")
	c.SetParamValues("")

	_ = handler.GetInventory(c)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// TestInventoryResponse レスポンス構造体テスト
func TestInventoryResponse(t *testing.T) {
	resp := InventoryResponse{
		ProductID:         "PROD-001",
		ProductName:       "Laptop",
		Quantity:          100,
		ReservedQuantity:  10,
		AvailableQuantity: 90,
	}

	data, _ := json.Marshal(resp)
	var parsed InventoryResponse
	_ = json.Unmarshal(data, &parsed)

	if parsed.ProductID != resp.ProductID {
		t.Errorf("ProductID = %s, want %s", parsed.ProductID, resp.ProductID)
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
