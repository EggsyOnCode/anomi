package p2p

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/EggysOnCode/anomi/api"
	"github.com/EggysOnCode/anomi/api/handlers"
	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/EggysOnCode/anomi/rpc"
)

// P2PController handles P2P network messages
type P2PController struct {
	parser         *MessageParser
	orderHandler   *handlers.OrderHandler
	tradeHandler   *handlers.TradeHandler
	receiptHandler *handlers.ReceiptHandler
}

// NewP2PController creates a new P2P controller
func NewP2PController(
	orderHandler *handlers.OrderHandler,
	tradeHandler *handlers.TradeHandler,
	receiptHandler *handlers.ReceiptHandler,
) *P2PController {
	return &P2PController{
		parser:         NewMessageParser(),
		orderHandler:   orderHandler,
		tradeHandler:   tradeHandler,
		receiptHandler: receiptHandler,
	}
}

// HandleMessage handles incoming P2P messages
func (c *P2PController) HandleMessage(rpcMsg *rpc.RPCMessage) (*rpc.RPCMessage, error) {
	// Validate RPC message
	if err := c.parser.ValidateRPCMessage(rpcMsg); err != nil {
		return c.createErrorResponse(fmt.Sprintf("Invalid RPC message: %v", err), "validation_failed")
	}

	// Parse the message
	ctx, businessData, err := c.parser.ParseRPCMessage(rpcMsg)
	if err != nil {
		return c.createErrorResponse(fmt.Sprintf("Failed to parse message: %v", err), "parse_error")
	}

	// Process the message based on data type
	var result *handlers.HandlerResult
	switch data := businessData.(type) {
	case *engine.Order:
		result = c.handleOrderMessage(ctx, data, rpcMsg)
	case *engine.TradeOrder:
		result = c.handleTradeMessage(ctx, data)
	case *orderbook.Receipt:
		result = c.handleReceiptMessage(ctx, data)
	default:
		return c.createErrorResponse("Unknown message type", "unknown_type")
	}

	// Create RPC response
	response, err := c.parser.CreateRPCResponse(result, ctx.RequestID)
	if err != nil {
		return c.createErrorResponse(fmt.Sprintf("Failed to create response: %v", err), "response_error")
	}

	return response, nil
}

// handleOrderMessage handles order-related messages
func (c *P2PController) handleOrderMessage(ctx *api.RequestContext, order *engine.Order, rpcMsg *rpc.RPCMessage) *handlers.HandlerResult {
	// Parse the internal message to determine operation type
	internalMsg, err := rpc.FromBytes(rpcMsg.Payload)
	if err != nil {
		return &handlers.HandlerResult{
			Error:   err,
			Message: "Failed to parse internal message",
		}
	}

	// Route to appropriate handler method
	switch internalMsg.Type {
	case rpc.ORDER_PUT:
		return c.orderHandler.CreateOrder(context.Background(), order)
	case rpc.ORDER_UPDATE:
		return c.orderHandler.UpdateOrder(context.Background(), order)
	case rpc.ORDER_DELETE:
		return c.orderHandler.CancelOrder(context.Background(), order.ID())
	default:
		return &handlers.HandlerResult{
			Error:   fmt.Errorf("unsupported order operation: %v", internalMsg.Type),
			Message: "Unsupported operation",
		}
	}
}

// handleTradeMessage handles trade-related messages
func (c *P2PController) handleTradeMessage(ctx *api.RequestContext, trade *engine.TradeOrder) *handlers.HandlerResult {
	return c.tradeHandler.CreateTrade(context.Background(), trade)
}

// handleReceiptMessage handles receipt-related messages
func (c *P2PController) handleReceiptMessage(ctx *api.RequestContext, receipt *orderbook.Receipt) *handlers.HandlerResult {
	return c.receiptHandler.CreateReceipt(context.Background(), receipt)
}

// createErrorResponse creates an error response
func (c *P2PController) createErrorResponse(errorMsg, errorType string) (*rpc.RPCMessage, error) {
	response := api.NewErrorResponse(errorMsg, errorType)

	responseData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal error response: %w", err)
	}

	return &rpc.RPCMessage{
		FromID:  "api-server",
		Payload: responseData,
	}, nil
}
