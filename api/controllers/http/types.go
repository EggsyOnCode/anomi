package http

import (
	"time"
)

// ResponseWrapper is a common response wrapper for HTTP responses
type ResponseWrapper struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// NewSuccessResponse creates a success response
func NewSuccessResponse(data interface{}, message string) *ResponseWrapper {
	return &ResponseWrapper{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(err string, message string) *ResponseWrapper {
	return &ResponseWrapper{
		Success:   false,
		Error:     err,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// Request/Response DTOs

type CreateOrderRequest struct {
	OrderType string `json:"orderType" validate:"required,oneof=MARKET LIMIT STOP-LIMIT"`
	UserID    string `json:"userID" validate:"required,min=1"`
	Side      int    `json:"side" validate:"required,oneof=0 1"` // 0 = BUY, 1 = SELL
	IsQuote   bool   `json:"isQuote"`
	Quantity  string `json:"quantity" validate:"required,min=1"`
	Price     string `json:"price"` // Required for LIMIT and STOP-LIMIT orders
	Stop      string `json:"stop"`  // Required for STOP-LIMIT orders
	TIF       string `json:"tif"`   // Required for LIMIT orders
	OCO       string `json:"oco"`   // Optional OCO reference
}

type UpdateOrderRequest struct {
	Quantity string `json:"quantity" validate:"omitempty,min=1"`
	Price    string `json:"price" validate:"omitempty,min=1"`
	Stop     string `json:"stop" validate:"omitempty,min=1"`
}

type OrderResponse struct {
	ID          string `json:"id"`
	OrderType   string `json:"orderType"`
	UserID      string `json:"userID"`
	Side        string `json:"side"`
	IsQuote     bool   `json:"isQuote"`
	Quantity    string `json:"quantity"`
	OriginalQty string `json:"originalQty"`
	Price       string `json:"price"`
	Stop        string `json:"stop"`
	Canceled    bool   `json:"canceled"`
	Role        string `json:"role"`
	TIF         string `json:"tif"`
	OCO         string `json:"oco"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}
