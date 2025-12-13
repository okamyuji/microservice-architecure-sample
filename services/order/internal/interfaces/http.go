// Package interfaces HTTPハンドラとリクエスト/レスポンス型
package interfaces

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/yujiokamoto/microservice-architecture-sample/services/order/internal/application"
)

// CreateOrderRequest 注文作成リクエストボディ
type CreateOrderRequest struct {
	CustomerID  string  `json:"customer_id" validate:"required"`
	ProductID   string  `json:"product_id" validate:"required"`
	Quantity    int     `json:"quantity" validate:"required,gt=0"`
	TotalAmount float64 `json:"total_amount" validate:"required,gt=0"`
}

// CreateOrderResponse 注文作成レスポンス
type CreateOrderResponse struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

// OrderResponse APIレスポンス内の注文
type OrderResponse struct {
	ID          string  `json:"id"`
	CustomerID  string  `json:"customer_id"`
	ProductID   string  `json:"product_id"`
	Quantity    int     `json:"quantity"`
	TotalAmount float64 `json:"total_amount"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// ErrorResponse エラーレスポンス
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HTTPHandler 注文のHTTPリクエストを処理
type HTTPHandler struct {
	useCase *application.OrderUseCase
}

// NewHTTPHandler 新規HTTPハンドラを生成
func NewHTTPHandler(useCase *application.OrderUseCase) *HTTPHandler {
	return &HTTPHandler{useCase: useCase}
}

// RegisterRoutes HTTPルートを登録
func (h *HTTPHandler) RegisterRoutes(e *echo.Echo) {
	orders := e.Group("/orders")
	orders.POST("", h.CreateOrder)
	orders.GET("/:id", h.GetOrder)

	// ヘルスチェック
	e.GET("/health", h.HealthCheck)
}

// CreateOrder POST /orders を処理
func (h *HTTPHandler) CreateOrder(c echo.Context) error {
	var req CreateOrderRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
	}

	if req.CustomerID == "" || req.ProductID == "" || req.Quantity <= 0 || req.TotalAmount <= 0 {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "All fields are required and must be valid",
		})
	}

	output, err := h.useCase.CreateOrder(c.Request().Context(), application.CreateOrderInput{
		CustomerID:  req.CustomerID,
		ProductID:   req.ProductID,
		Quantity:    req.Quantity,
		TotalAmount: req.TotalAmount,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, CreateOrderResponse{
		OrderID: output.OrderID,
		Status:  output.Status,
	})
}

// GetOrder GET /orders/:id を処理
func (h *HTTPHandler) GetOrder(c echo.Context) error {
	orderID := c.Param("id")
	if orderID == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Order ID is required",
		})
	}

	order, err := h.useCase.GetOrder(c.Request().Context(), orderID)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Order not found",
		})
	}

	return c.JSON(http.StatusOK, OrderResponse{
		ID:          order.ID,
		CustomerID:  order.CustomerID,
		ProductID:   order.ProductID,
		Quantity:    order.Quantity,
		TotalAmount: order.TotalAmount,
		Status:      order.Status,
		CreatedAt:   order.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   order.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// HealthCheck GET /health を処理
func (h *HTTPHandler) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
}
