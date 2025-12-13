// Package interfaces HTTPハンドラとリクエスト/レスポンス型
package interfaces

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/yujiokamoto/microservice-architecture-sample/services/payment/internal/application"
)

// PaymentResponse APIレスポンス内の決済
type PaymentResponse struct {
	ID            string  `json:"id"`
	OrderID       string  `json:"order_id"`
	CustomerID    string  `json:"customer_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	FailureReason string  `json:"failure_reason,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// ErrorResponse エラーレスポンス
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HTTPHandler 決済のHTTPリクエストを処理
type HTTPHandler struct {
	useCase *application.PaymentUseCase
}

// NewHTTPHandler 新規HTTPハンドラを生成
func NewHTTPHandler(useCase *application.PaymentUseCase) *HTTPHandler {
	return &HTTPHandler{useCase: useCase}
}

// RegisterRoutes HTTPルートを登録
func (h *HTTPHandler) RegisterRoutes(e *echo.Echo) {
	payments := e.Group("/payments")
	payments.GET("/order/:order_id", h.GetPaymentByOrder)

	// ヘルスチェック
	e.GET("/health", h.HealthCheck)
}

// GetPaymentByOrder GET /payments/order/:order_id を処理
func (h *HTTPHandler) GetPaymentByOrder(c echo.Context) error {
	orderID := c.Param("order_id")
	if orderID == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Order ID is required",
		})
	}

	payment, err := h.useCase.GetPayment(c.Request().Context(), orderID)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Payment not found",
		})
	}

	return c.JSON(http.StatusOK, PaymentResponse{
		ID:            payment.ID,
		OrderID:       payment.OrderID,
		CustomerID:    payment.CustomerID,
		Amount:        payment.Amount,
		Status:        payment.Status,
		FailureReason: payment.FailureReason,
		CreatedAt:     payment.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     payment.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// HealthCheck GET /health を処理
func (h *HTTPHandler) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
}
