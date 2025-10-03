package storage

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/EggysOnCode/anomi/rpc"
	"github.com/EggysOnCode/anomi/storage/models"
	"github.com/EggysOnCode/anomi/storage/repositories"
	"github.com/nikolaydubina/fpdecimal"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/uptrace/bun"
)

// PgSQLHandler handles internal messages from RabbitMQ and performs database operations
type PgSQLHandler struct {
	db          *bun.DB
	factory     repositories.RepositoryFactory
	orderRepo   repositories.OrderRepository
	tradeRepo   repositories.TradeRepository
	receiptRepo repositories.ReceiptRepository
}

// NewPgSQLHandler creates a new PostgreSQL handler
func NewPgSQLHandler(db *bun.DB) *PgSQLHandler {
	factory := repositories.NewRepositoryFactory(db)

	return &PgSQLHandler{
		db:          db,
		factory:     factory,
		orderRepo:   factory.NewOrderRepository(),
		tradeRepo:   factory.NewTradeRepository(),
		receiptRepo: factory.NewReceiptRepository(),
	}
}

// HandleMessage processes incoming RabbitMQ messages
func (h *PgSQLHandler) HandleMessage(msg amqp.Delivery) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Log the received message
	log.Printf("Received message: %s", string(msg.Body))

	// Parse the internal message
	internalMsg, err := rpc.FromBytes(msg.Body)
	if err != nil {
		log.Printf("Failed to decode internal message: %v", err)
		return h.ackMessage(msg, false) // Nack the message
	}

	// Validate the message
	if err := h.validateMessage(internalMsg); err != nil {
		log.Printf("Message validation failed: %v", err)
		return h.ackMessage(msg, false) // Nack the message
	}

	// Process the message based on type
	if err := h.processMessage(ctx, internalMsg); err != nil {
		log.Printf("Failed to process message: %v", err)
		return h.ackMessage(msg, false) // Nack the message
	}

	// Acknowledge successful processing
	return h.ackMessage(msg, true)
}

// validateMessage validates the internal message structure
func (h *PgSQLHandler) validateMessage(msg *rpc.InternalMessage) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	if msg.ID == "" {
		return fmt.Errorf("message ID is empty")
	}

	if len(msg.Data) == 0 {
		return fmt.Errorf("message data is empty")
	}

	// Validate message type
	switch msg.Type {
	case rpc.ORDER_PUT, rpc.ORDER_DELETE, rpc.ORDER_UPDATE:
		// Valid order operations
	case rpc.TRADE_PUT:
		// Valid trade operation
	case rpc.RECEIPT_PUT:
		// Valid receipt operation
	default:
		return fmt.Errorf("unknown message type: %v", msg.Type)
	}

	return nil
}

// processMessage processes the internal message based on its type
func (h *PgSQLHandler) processMessage(ctx context.Context, msg *rpc.InternalMessage) error {
	switch msg.Type {
	case rpc.ORDER_PUT:
		return h.handleOrderPut(ctx, msg)
	case rpc.ORDER_DELETE:
		return h.handleOrderDelete(ctx, msg)
	case rpc.ORDER_UPDATE:
		return h.handleOrderUpdate(ctx, msg)
	case rpc.TRADE_PUT:
		return h.handleTradePut(ctx, msg)
	case rpc.RECEIPT_PUT:
		return h.handleReceiptPut(ctx, msg)
	default:
		return fmt.Errorf("unsupported message type: %v", msg.Type)
	}
}

// handleOrderPut processes ORDER_PUT messages
func (h *PgSQLHandler) handleOrderPut(ctx context.Context, msg *rpc.InternalMessage) error {
	// Unmarshal the order data
	var order engine.Order
	if err := msg.UnmarshalData(&order); err != nil {
		return fmt.Errorf("failed to unmarshal order: %w", err)
	}

	// Validate the order
	if err := h.validateOrder(&order); err != nil {
		return fmt.Errorf("order validation failed: %w", err)
	}

	// Convert to storage model
	modelOrder := models.NewOrderFromEngine(&order)

	// Check if order already exists
	existingOrder, err := h.orderRepo.GetByID(ctx, modelOrder.ID)
	if err == nil && existingOrder.ID != "" {
		log.Printf("Order %s already exists, skipping insert", modelOrder.ID)
		return nil
	}

	// Create the order
	if err := h.orderRepo.Create(ctx, *modelOrder); err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	log.Printf("Successfully created order: %s", modelOrder.ID)
	return nil
}

// handleOrderDelete processes ORDER_DELETE messages
func (h *PgSQLHandler) handleOrderDelete(ctx context.Context, msg *rpc.InternalMessage) error {
	// Unmarshal the order data
	var order engine.Order
	if err := msg.UnmarshalData(&order); err != nil {
		return fmt.Errorf("failed to unmarshal order: %w", err)
	}

	// Validate the order ID
	if order.ID() == "" {
		return fmt.Errorf("order ID is empty")
	}

	// Check if order exists
	_, err := h.orderRepo.GetByID(ctx, order.ID())
	if err != nil {
		log.Printf("Order %s not found for deletion: %v", order.ID(), err)
		return nil // Don't treat as error if order doesn't exist
	}

	// Delete the order
	if err := h.orderRepo.Delete(ctx, order.ID()); err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	log.Printf("Successfully deleted order: %s", order.ID())
	return nil
}

// handleOrderUpdate processes ORDER_UPDATE messages
func (h *PgSQLHandler) handleOrderUpdate(ctx context.Context, msg *rpc.InternalMessage) error {
	// Unmarshal the order data
	var order engine.Order
	if err := msg.UnmarshalData(&order); err != nil {
		return fmt.Errorf("failed to unmarshal order: %w", err)
	}

	// Validate the order
	if err := h.validateOrder(&order); err != nil {
		return fmt.Errorf("order validation failed: %w", err)
	}

	// Convert to storage model
	modelOrder := models.NewOrderFromEngine(&order)

	// Check if order exists
	_, err := h.orderRepo.GetByID(ctx, modelOrder.ID)
	if err != nil {
		return fmt.Errorf("order %s not found for update: %w", modelOrder.ID, err)
	}

	// Update the order
	if err := h.orderRepo.Update(ctx, *modelOrder); err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	log.Printf("Successfully updated order: %s", modelOrder.ID)
	return nil
}

// handleTradePut processes TRADE_PUT messages
func (h *PgSQLHandler) handleTradePut(ctx context.Context, msg *rpc.InternalMessage) error {
	// Unmarshal the trade data
	var trade engine.TradeOrder
	if err := msg.UnmarshalData(&trade); err != nil {
		return fmt.Errorf("failed to unmarshal trade: %w", err)
	}

	// Validate the trade
	if err := h.validateTrade(&trade); err != nil {
		return fmt.Errorf("trade validation failed: %w", err)
	}

	// Convert to storage model
	modelTrade := models.NewTradeFromEngine(&trade)

	// Check if trade already exists (by ID)
	existingTrade, err := h.tradeRepo.GetByID(ctx, modelTrade.ID)
	if err == nil && existingTrade.ID != "" {
		log.Printf("Trade %s already exists, skipping insert", modelTrade.ID)
		return nil
	}

	// Create the trade
	if err := h.tradeRepo.Create(ctx, modelTrade); err != nil {
		return fmt.Errorf("failed to create trade: %w", err)
	}

	log.Printf("Successfully created trade: %s", modelTrade.ID)
	return nil
}

// handleReceiptPut processes RECEIPT_PUT messages
func (h *PgSQLHandler) handleReceiptPut(ctx context.Context, msg *rpc.InternalMessage) error {
	// Unmarshal the receipt data
	var receipt orderbook.Receipt
	if err := msg.UnmarshalData(&receipt); err != nil {
		return fmt.Errorf("failed to unmarshal receipt: %w", err)
	}

	// Validate the receipt
	if err := h.validateReceipt(&receipt); err != nil {
		return fmt.Errorf("receipt validation failed: %w", err)
	}

	// Convert to storage models (one receipt per trade)
	modelReceipts := models.CreateReceiptsFromEngine(&receipt)

	// Create all receipts
	for _, modelReceipt := range modelReceipts {
		// Check if receipt already exists
		existingReceipt, err := h.receiptRepo.GetByID(ctx, modelReceipt.ID)
		if err == nil && existingReceipt.ID != "" {
			log.Printf("Receipt %s already exists, skipping insert", modelReceipt.ID)
			continue
		}

		// Create the receipt
		if err := h.receiptRepo.Create(ctx, modelReceipt); err != nil {
			return fmt.Errorf("failed to create receipt %s: %w", modelReceipt.ID, err)
		}

		log.Printf("Successfully created receipt: %s", modelReceipt.ID)
	}

	return nil
}

// validateOrder validates an engine.Order
func (h *PgSQLHandler) validateOrder(order *engine.Order) error {
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

// validateTrade validates an engine.TradeOrder
func (h *PgSQLHandler) validateTrade(trade *engine.TradeOrder) error {
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

// validateReceipt validates an orderbook.Receipt
func (h *PgSQLHandler) validateReceipt(receipt *orderbook.Receipt) error {
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

// ackMessage acknowledges or nacks a RabbitMQ message
func (h *PgSQLHandler) ackMessage(msg amqp.Delivery, ack bool) error {
	if ack {
		return msg.Ack(false)
	} else {
		return msg.Nack(false, true) // Requeue the message
	}
}

// GetStats returns handler statistics
func (h *PgSQLHandler) GetStats(ctx context.Context) (map[string]interface{}, error) {
	orderCount, err := h.orderRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get order count: %w", err)
	}

	tradeCount, err := h.tradeRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade count: %w", err)
	}

	receiptCount, err := h.receiptRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt count: %w", err)
	}

	return map[string]interface{}{
		"orders":    orderCount,
		"trades":    tradeCount,
		"receipts":  receiptCount,
		"timestamp": time.Now(),
	}, nil
}

// Close closes the handler and its resources
func (h *PgSQLHandler) Close() error {
	// Close database connection if needed
	return h.db.Close()
}
