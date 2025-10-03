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
	UserID    string `json:"userID" validate:"required"`
	Side      int    `json:"side" validate:"required,oneof=0 1"` // 0 = BUY, 1 = SELL
	IsQuote   bool   `json:"isQuote"`
	Quantity  string `json:"quantity" validate:"required"`
	Price     string `json:"price" validate:"required"`
	Stop      string `json:"stop"`
	TIF       string `json:"tif" validate:"required,oneof=GTC IOC FOK"`
	OCO       string `json:"oco"`
}

type UpdateOrderRequest struct {
	Quantity string `json:"quantity"`
	Price    string `json:"price"`
	Stop     string `json:"stop"`
}
