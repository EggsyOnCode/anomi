package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/EggysOnCode/anomi/api"
	"github.com/EggysOnCode/anomi/api/handlers"
	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/EggysOnCode/anomi/rpc"
)

// MessageParser handles parsing of RPC messages from P2P network
type MessageParser struct{}

// NewMessageParser creates a new message parser
func NewMessageParser() *MessageParser {
	return &MessageParser{}
}

// ParseRPCMessage parses an RPC message and extracts the business data
func (p *MessageParser) ParseRPCMessage(rpcMsg *rpc.RPCMessage) (*api.RequestContext, interface{}, error) {
	// Parse the internal message from RPC payload
	internalMsg, err := rpc.FromBytes(rpcMsg.Payload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse internal message: %w", err)
	}

	// Create request context
	ctx := &api.RequestContext{
		UserID:    rpcMsg.FromID,
		RequestID: internalMsg.ID,
		Source:    "p2p",
	}

	// Parse data based on message type
	var businessData interface{}
	switch internalMsg.Type {
	case rpc.ORDER_PUT, rpc.ORDER_UPDATE, rpc.ORDER_DELETE:
		order, err := p.parseOrderMessage(internalMsg)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse order message: %w", err)
		}
		businessData = order

	case rpc.TRADE_PUT:
		trade, err := p.parseTradeMessage(internalMsg)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse trade message: %w", err)
		}
		businessData = trade

	case rpc.RECEIPT_PUT:
		receipt, err := p.parseReceiptMessage(internalMsg)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse receipt message: %w", err)
		}
		businessData = receipt

	default:
		return nil, nil, fmt.Errorf("unsupported message type: %v", internalMsg.Type)
	}

	return ctx, businessData, nil
}

// parseOrderMessage parses order-related messages
func (p *MessageParser) parseOrderMessage(msg *rpc.InternalMessage) (*engine.Order, error) {
	var order engine.Order
	if err := msg.UnmarshalData(&order); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order data: %w", err)
	}
	return &order, nil
}

// parseTradeMessage parses trade-related messages
func (p *MessageParser) parseTradeMessage(msg *rpc.InternalMessage) (*engine.TradeOrder, error) {
	var trade engine.TradeOrder
	if err := msg.UnmarshalData(&trade); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trade data: %w", err)
	}
	return &trade, nil
}

// parseReceiptMessage parses receipt-related messages
func (p *MessageParser) parseReceiptMessage(msg *rpc.InternalMessage) (*orderbook.Receipt, error) {
	var receipt orderbook.Receipt
	if err := msg.UnmarshalData(&receipt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal receipt data: %w", err)
	}
	return &receipt, nil
}

// CreateRPCResponse creates an RPC response from handler result
func (p *MessageParser) CreateRPCResponse(result *handlers.HandlerResult, requestID string) (*rpc.RPCMessage, error) {
	// Create response wrapper
	response := api.NewSuccessResponse(result.Data, result.Message)
	if result.Error != nil {
		response = api.NewErrorResponse(result.Error.Error(), result.Message)
	}

	// Serialize response
	responseData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	// Create RPC message
	rpcMsg := &rpc.RPCMessage{
		FromID:  "api-server", // This should be the server ID
		Payload: responseData,
	}

	return rpcMsg, nil
}

// ValidateRPCMessage validates an RPC message
func (p *MessageParser) ValidateRPCMessage(rpcMsg *rpc.RPCMessage) error {
	if rpcMsg == nil {
		return fmt.Errorf("RPC message is nil")
	}

	if rpcMsg.FromID == "" {
		return fmt.Errorf("RPC message FromID is empty")
	}

	if len(rpcMsg.Payload) == 0 {
		return fmt.Errorf("RPC message payload is empty")
	}

	return nil
}
