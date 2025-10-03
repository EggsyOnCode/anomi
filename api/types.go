package api

import (
	"time"
)

// ResponseWrapper is a common response wrapper for all mediums
type ResponseWrapper struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// NewSuccessResponse creates a success response
func NewSuccessResponse(data interface{}, message string) *ResponseWrapper {
	return &ResponseWrapper{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(err string, message string) *ResponseWrapper {
	return &ResponseWrapper{
		Success:   false,
		Error:     err,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// RequestContext holds common request context information
type RequestContext struct {
	UserID    string
	RequestID string
	Source    string // "http", "p2p", etc.
}

// HandlerResult represents the result from business handlers
type HandlerResult struct {
	Data    interface{}
	Error   error
	Message string
}
