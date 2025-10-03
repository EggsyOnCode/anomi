package handlers

import (
	"context"
	"fmt"

	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/storage"
)

// ReceiptHandler handles receipt-related business logic
type ReceiptHandler struct {
	kvdb        *storage.KvDB
	msgProducer MessageProducer
}

// NewReceiptHandler creates a new receipt handler
func NewReceiptHandler(kvdb *storage.KvDB, msgProducer MessageProducer) *ReceiptHandler {
	return &ReceiptHandler{
		kvdb:        kvdb,
		msgProducer: msgProducer,
	}
}

// CreateReceipt handles receipt creation
func (h *ReceiptHandler) CreateReceipt(ctx context.Context, receipt *orderbook.Receipt) *HandlerResult {
	// Validate receipt
	if err := h.validateReceipt(receipt); err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Receipt validation failed",
		}
	}

	// Store in KVDB
	if err := h.kvdb.PutReceipt(receipt); err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Failed to store receipt in KVDB",
		}
	}

	// Publish message for PostgreSQL sync
	if err := h.msgProducer.PublishReceiptGenerated(receipt); err != nil {
		fmt.Printf("Warning: Failed to publish receipt generated message: %v\n", err)
	}

	return &HandlerResult{
		Data:    receipt,
		Message: "Receipt created successfully",
	}
}

// GetReceipt retrieves a receipt by ID
func (h *ReceiptHandler) GetReceipt(ctx context.Context, receiptID string) *HandlerResult {
	// Get receipt from KVDB
	receipt, err := h.kvdb.GetReceipt(receiptID)
	if err != nil {
		return &HandlerResult{
			Error:   err,
			Message: "Receipt retrieval failed",
		}
	}

	return &HandlerResult{
		Data:    receipt,
		Message: "Receipt retrieved successfully",
	}
}

// GetReceiptsByUser retrieves receipts for a specific user
func (h *ReceiptHandler) GetReceiptsByUser(ctx context.Context, userID string) *HandlerResult {
	// TODO: Implement GetReceiptsByUser in KVDB
	return &HandlerResult{
		Error:   fmt.Errorf("not implemented"),
		Message: "GetReceiptsByUser not yet implemented",
	}
}

// GetReceiptsByOrder retrieves receipts for a specific order
func (h *ReceiptHandler) GetReceiptsByOrder(ctx context.Context, orderID string) *HandlerResult {
	// TODO: Implement GetReceiptsByOrder in KVDB
	return &HandlerResult{
		Error:   fmt.Errorf("not implemented"),
		Message: "GetReceiptsByOrder not yet implemented",
	}
}

// validateReceipt validates a receipt
func (h *ReceiptHandler) validateReceipt(receipt *orderbook.Receipt) error {
	if receipt == nil {
		return fmt.Errorf("receipt is nil")
	}

	if receipt.UserID == "" {
		return fmt.Errorf("receipt user ID is empty")
	}

	if receipt.OrderID == "" {
		return fmt.Errorf("receipt order ID is empty")
	}

	if len(receipt.Trades) == 0 {
		return fmt.Errorf("receipt has no trades")
	}

	return nil
}
