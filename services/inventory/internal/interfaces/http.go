// Package interfaces HTTPハンドラとリクエスト/レスポンス型
package interfaces

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/yujiokamoto/microservice-architecture-sample/services/inventory/internal/application"
)

// InventoryResponse APIレスポンス内の在庫
type InventoryResponse struct {
	ProductID         string `json:"product_id"`
	ProductName       string `json:"product_name"`
	Quantity          int    `json:"quantity"`
	ReservedQuantity  int    `json:"reserved_quantity"`
	AvailableQuantity int    `json:"available_quantity"`
}

// ErrorResponse エラーレスポンス
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HTTPHandler 在庫のHTTPリクエストを処理
type HTTPHandler struct {
	useCase *application.InventoryUseCase
}

// NewHTTPHandler 新規HTTPハンドラを生成
func NewHTTPHandler(useCase *application.InventoryUseCase) *HTTPHandler {
	return &HTTPHandler{useCase: useCase}
}

// RegisterRoutes HTTPルートを登録
func (h *HTTPHandler) RegisterRoutes(e *echo.Echo) {
	inventory := e.Group("/inventory")
	inventory.GET("/:product_id", h.GetInventory)

	// ヘルスチェック
	e.GET("/health", h.HealthCheck)
}

// GetInventory GET /inventory/:product_id を処理
func (h *HTTPHandler) GetInventory(c echo.Context) error {
	productID := c.Param("product_id")
	if productID == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Product ID is required",
		})
	}

	inventory, err := h.useCase.GetInventory(c.Request().Context(), productID)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Product not found",
		})
	}

	return c.JSON(http.StatusOK, InventoryResponse{
		ProductID:         inventory.ProductID,
		ProductName:       inventory.ProductName,
		Quantity:          inventory.Quantity,
		ReservedQuantity:  inventory.ReservedQuantity,
		AvailableQuantity: inventory.AvailableQuantity(),
	})
}

// HealthCheck GET /health を処理
func (h *HTTPHandler) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
}
