package handlers

import (
	"context"
	"fmt"

	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/EggysOnCode/anomi/storage"
	"github.com/nikolaydubina/fpdecimal"
)

// OrderHandler handles order-related business logic
type OrderHandler struct {
	orderbook   *engine.OrderBook
	kvdb        *storage.KvDB
	msgProducer MessageProducer
}

// MessageProducer interface for publishing messages
type MessageProducer interface {
	PublishOrderCreated(order *engine.Order) error
	PublishOrderUpdated(order *engine.Order) error
	PublishOrderDeleted(order *engine.Order) error
	PublishTradeExecuted(trade *engine.TradeOrder) error
	PublishReceiptGenerated(receipt *orderbook.Receipt) error
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderbook *engine.OrderBook, kvdb *storage.KvDB, msgProducer MessageProducer) *OrderHandler {
	return &OrderHandler{
		orderbook:   orderbook,
		kvdb:        kvdb,
		msgProducer: msgProducer,
	}
}

// HandlerResult represents the result from business handlers
type HandlerResult struct {
	Data    interface{}
	Error   error
	Message string
}

// CreateOrder handles order creation
func (h *OrderHandler) CreateOrder(ctx context.Context, order *engine.Order) *HandlerResult {
	// Validate order
	if err := h.validateOrder(order); err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Order validation failed",
		}
	}

	// Add order to orderbook
	result, err := h.orderbook.Process(order)
	if result == nil || err != nil {
		return &HandlerResult{
			Error:   fmt.Errorf("failed to add order to orderbook"),
			Message: "Order creation failed",
		}
	}

	// Store in KVDB
	if err := h.kvdb.PutOrder(order); err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Failed to store order in KVDB",
		}
	}

	// Publish message for PostgreSQL sync
	if err := h.msgProducer.PublishOrderCreated(order); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Warning: Failed to publish order created message: %v\n", err)
	}

	return &HandlerResult{
		Data:    result,
		Message: "Order created successfully",
	}
}

// UpdateOrder handles order updates
func (h *OrderHandler) UpdateOrder(ctx context.Context, order *engine.Order) *HandlerResult {
	// Validate order
	if err := h.validateOrder(order); err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Order validation failed",
		}
	}

	// Update order in orderbook
	if _, err := h.orderbook.Process(order); err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Failed to update order in orderbook",
		}
	}

	// Remove existing order from kvdbj
	if res := h.kvdb.DeleteOrder(order.ID()); res != nil {
		return &HandlerResult{
			Error:   fmt.Errorf("failed to remove order from kvdb"),
			Message: "Failed to remove order from kvdb",
		}
	}

	// Update in KVDB
	if err := h.kvdb.PutOrder(order); err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Failed to update order in KVDB",
		}
	}

	// Publish message for PostgreSQL sync
	if err := h.msgProducer.PublishOrderUpdated(order); err != nil {
		fmt.Printf("Warning: Failed to publish order updated message: %v\n", err)
	}

	return &HandlerResult{
		Data:    order,
		Message: "Order updated successfully",
	}
}

// CancelOrder handles order cancellation
func (h *OrderHandler) CancelOrder(ctx context.Context, orderID string) *HandlerResult {
	// Get order from orderbook
	order := h.orderbook.GetOrder(orderID)
	if order == nil {
		return &HandlerResult{
			Error:   fmt.Errorf("order not found"),
			Message: "Order cancellation failed",
		}
	}

	// Cancel order in orderbook
	if res := h.orderbook.CancelOrder(orderID); res != nil {
		return &HandlerResult{
			Error:   fmt.Errorf("failed to cancel order"),
			Message: "Failed to cancel order in orderbook",
		}
	}

	// Update in KVDB
	if err := h.kvdb.PutOrder(order); err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Failed to update order in KVDB",
		}
	}

	// Publish message for PostgreSQL sync
	if err := h.msgProducer.PublishOrderUpdated(order); err != nil {
		fmt.Printf("Warning: Failed to publish order updated message: %v\n", err)
	}

	return &HandlerResult{
		Data:    order,
		Message: "Order cancelled successfully",
	}
}

// GetOrder retrieves an order by ID
func (h *OrderHandler) GetOrder(ctx context.Context, orderID string) *HandlerResult {
	// Get order from orderbook
	order := h.orderbook.GetOrder(orderID)
	if order == nil {
		return &HandlerResult{
			Error:   fmt.Errorf("order not found"),
			Message: "Order retrieval failed",
		}
	}

	return &HandlerResult{
		Data:    order,
		Message: "Order retrieved successfully",
	}
}

// validateOrder validates an order
func (h *OrderHandler) validateOrder(order *engine.Order) error {
	if order == nil {
		return fmt.Errorf("order is nil")
	}

	if order.ID() == "" {
		return fmt.Errorf("order ID is empty")
	}

	if order.UserID() == "" {
		return fmt.Errorf("order user ID is empty")
	}

	if order.Quantity().Compare(fpdecimal.Zero) == 0 {
		return fmt.Errorf("order quantity is zero")
	}

	return nil
}
