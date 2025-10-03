package handlers

import (
	"context"
	"fmt"

	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/EggysOnCode/anomi/storage"
	"github.com/nikolaydubina/fpdecimal"
)

// TradeHandler handles trade-related business logic
type TradeHandler struct {
	kvdb        *storage.KvDB
	msgProducer MessageProducer
}

// NewTradeHandler creates a new trade handler
func NewTradeHandler(kvdb *storage.KvDB, msgProducer MessageProducer) *TradeHandler {
	return &TradeHandler{
		kvdb:        kvdb,
		msgProducer: msgProducer,
	}
}

// CreateTrade handles trade creation
func (h *TradeHandler) CreateTrade(ctx context.Context, trade *engine.TradeOrder) *HandlerResult {
	// Validate trade
	if err := h.validateTrade(trade); err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Trade validation failed",
		}
	}

	// Store in KVDB
	if err := h.kvdb.PutTradeOrder(trade); err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Failed to store trade in KVDB",
		}
	}

	// Publish message for PostgreSQL sync
	if err := h.msgProducer.PublishTradeExecuted(trade); err != nil {
		fmt.Printf("Warning: Failed to publish trade executed message: %v\n", err)
	}

	return &HandlerResult{
		Data:    trade,
		Message: "Trade created successfully",
	}
}

// GetTrade retrieves a trade by ID
func (h *TradeHandler) GetTrade(ctx context.Context, tradeID string) *HandlerResult {
	// Get trade from KVDB
	trade, err := h.kvdb.GetTradeOrder(tradeID)
	if err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Trade retrieval failed",
		}
	}

	return &HandlerResult{
		Data:    trade,
		Message: "Trade retrieved successfully",
	}
}

// GetTradesByUser retrieves trades for a specific user
func (h *TradeHandler) GetTradesByUser(ctx context.Context, userID string) *HandlerResult {
	// TODO: Implement GetTradesByUser in KVDB
	return &HandlerResult{
		Error:   fmt.Errorf("not implemented"),
		Message: "GetTradesByUser not yet implemented",
	}
}

// GetTradesByOrder retrieves trades for a specific order
func (h *TradeHandler) GetTradesByOrder(ctx context.Context, orderID string) *HandlerResult {
	// TODO: Implement GetTradesByOrder in KVDB
	return &HandlerResult{
		Error:   fmt.Errorf("not implemented"),
		Message: "GetTradesByOrder not yet implemented",
	}
}

// validateTrade validates a trade
func (h *TradeHandler) validateTrade(trade *engine.TradeOrder) error {
	if trade == nil {
		return fmt.Errorf("trade is nil")
	}

	if trade.OrderID == "" {
		return fmt.Errorf("trade order ID is empty")
	}

	if trade.UserId == "" {
		return fmt.Errorf("trade user ID is empty")
	}

	if trade.Quantity.Compare(fpdecimal.Zero) == 0 {
		return fmt.Errorf("trade quantity is zero")
	}

	return nil
}
