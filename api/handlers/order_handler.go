package handlers

import (
	"context"
	"fmt"

	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/EggysOnCode/anomi/storage"
	"github.com/nikolaydubina/fpdecimal"
	"go.uber.org/zap"
)

// OrderHandler handles order-related business logic
type OrderHandler struct {
	orderbooks  map[string]*orderbook.OrderBook
	kvdb        *storage.KvDB
	msgProducer MessageProducer
	logger      *zap.Logger
}

// NewOrderHandler creates a new order handler that supports multiple orderbooks keyed by symbol "BASE/QUOTE"
func NewOrderHandler(books []*orderbook.OrderBook, kvdb *storage.KvDB, msgProducer MessageProducer, logger *zap.Logger) *OrderHandler {
	index := make(map[string]*orderbook.OrderBook, len(books))
	for _, b := range books {
		if b == nil {
			continue
		}
		// Symbol() returns "BASE/QUOTE"
		sym := b.Symbol()
		if sym == "" {
			continue
		}
		index[sym] = b
	}
	return &OrderHandler{
		orderbooks:  index,
		kvdb:        kvdb,
		msgProducer: msgProducer,
		logger:      logger,
	}
}

// getBook returns the appropriate orderbook for a symbol. If symbol is empty, it falls back to the default single book.
func (h *OrderHandler) getBook(symbol string) (*orderbook.OrderBook, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	ob, ok := h.orderbooks[symbol]
	if !ok || ob == nil {
		return nil, fmt.Errorf("orderbook not found for symbol %s", symbol)
	}
	return ob, nil
}

// HandlerResult represents the result from business handlers
type HandlerResult struct {
	Data    interface{}
	Error   error
	Message string
}

// CreateOrder handles order creation
// CreateOrder handles order creation for a specific symbol ("BASE/QUOTE")
func (h *OrderHandler) CreateOrder(ctx context.Context, symbol string, order *engine.Order) *HandlerResult {
	h.logger.Info("Creating order", zap.String("orderID", order.ID()), zap.String("symbol", symbol), zap.String("side", order.Side().String()))

	// Validate order
	if err := h.validateOrder(order); err != nil {
		h.logger.Error("Order validation failed", zap.String("orderID", order.ID()), zap.Error(err))
		return &HandlerResult{
			Error:   err,
			Message: "Order validation failed",
		}
	}

	ob, err := h.getBook(symbol)
	if err != nil {
		h.logger.Error("Failed to get orderbook", zap.String("symbol", symbol), zap.Error(err))
		return &HandlerResult{Error: err, Message: "Order creation failed"}
	}

	h.logger.Info("Processing order in orderbook", zap.String("orderID", order.ID()), zap.String("symbol", symbol))
	// Add order to orderbook
	result, err := ob.Process(order)
	if result == nil || err != nil {
		h.logger.Error("Failed to process order in orderbook", zap.String("orderID", order.ID()), zap.Error(err))
		return &HandlerResult{
			Error:   fmt.Errorf("failed to add order to orderbook"),
			Message: "Order creation failed",
		}
	}

	h.logger.Info("Storing order in KVDB", zap.String("orderID", order.ID()))
	// Store in KVDB
	if err := h.kvdb.PutOrder(order); err != nil {
		h.logger.Error("Failed to store order in KVDB", zap.String("orderID", order.ID()), zap.Error(err))
		return &HandlerResult{
			Error:   err,
			Message: "Failed to store order in KVDB",
		}
	}

	h.logger.Info("Publishing order to RabbitMQ", zap.String("orderID", order.ID()))
	// Publish message for PostgreSQL sync
	if err := h.msgProducer.PublishOrderCreated(order); err != nil {
		// Log error but don't fail the operation
		h.logger.Warn("Failed to publish order created message", zap.String("orderID", order.ID()), zap.Error(err))
	} else {
		h.logger.Info("Successfully published order to RabbitMQ", zap.String("orderID", order.ID()))
	}

	h.logger.Info("Order created successfully", zap.String("orderID", order.ID()))
	return &HandlerResult{
		Data:    result,
		Message: "Order created successfully",
	}
}

// UpdateOrder handles order updates
// UpdateOrder handles order updates for a specific symbol
func (h *OrderHandler) UpdateOrder(ctx context.Context, symbol string, order *engine.Order) *HandlerResult {
	// Validate order
	if err := h.validateOrder(order); err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Order validation failed",
		}
	}

	// Update order in orderbook
	ob, err := h.getBook(symbol)
	if err != nil {
		return &HandlerResult{Error: err, Message: "Failed to update order in orderbook"}
	}
	if _, err := ob.Process(order); err != nil {
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
		h.logger.Warn("Failed to publish order updated message", zap.String("orderID", order.ID()), zap.Error(err))
	}

	return &HandlerResult{
		Data:    order,
		Message: "Order updated successfully",
	}
}

// CancelOrder handles order cancellation
// CancelOrder handles order cancellation for a specific symbol
func (h *OrderHandler) CancelOrder(ctx context.Context, symbol string, orderID string) *HandlerResult {
	// Get order from orderbook
	ob, err := h.getBook(symbol)
	if err != nil {
		return &HandlerResult{Error: err, Message: "Order cancellation failed"}
	}
	order := ob.GetOrder(orderID)
	if order == nil {
		return &HandlerResult{
			Error:   fmt.Errorf("order not found"),
			Message: "Order cancellation failed",
		}
	}

	// Cancel order in orderbook
	if res := ob.CancelOrder(orderID); res != nil {
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
		h.logger.Warn("Failed to publish order updated message", zap.String("orderID", order.ID()), zap.Error(err))
	}

	return &HandlerResult{
		Data:    order,
		Message: "Order cancelled successfully",
	}
}

// GetOrder retrieves an order by ID
// GetOrder retrieves an order by ID from a specific symbol's orderbook
func (h *OrderHandler) GetOrder(ctx context.Context, symbol string, orderID string) *HandlerResult {
	// Get order from orderbook
	ob, err := h.getBook(symbol)
	if err != nil {
		return &HandlerResult{Error: err, Message: "Order retrieval failed"}
	}
	order := ob.GetOrder(orderID)
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
