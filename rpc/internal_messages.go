package rpc

import (
	"encoding/json"
	"time"

	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/google/uuid"
)

// Internal message types for service-to-service communication
type InternalMessageType byte

const (
	// Order operations
	ORDER_PUT InternalMessageType = iota
	ORDER_DELETE
	ORDER_UPDATE

	// Trade operations (put only)
	TRADE_PUT

	// Receipt operations (put only)
	RECEIPT_PUT
)

// Base internal message structure
type InternalMessage struct {
	ID        string              `json:"id"`
	Type      InternalMessageType `json:"type"`
	Timestamp time.Time           `json:"timestamp"`
	Data      json.RawMessage     `json:"data"`
}

// NewInternalMessage creates a new internal message
func NewInternalMessage(msgType InternalMessageType, data interface{}) (*InternalMessage, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &InternalMessage{
		ID:        generateMessageID(),
		Type:      msgType,
		Timestamp: time.Now(),
		Data:      jsonData,
	}, nil
}

// Helper functions for creating specific message types

// NewOrderPutMessage creates a message for putting an order
func NewOrderPutMessage(order *engine.Order) (*InternalMessage, error) {
	return NewInternalMessage(ORDER_PUT, order)
}

// NewOrderDeleteMessage creates a message for deleting an order
func NewOrderDeleteMessage(order *engine.Order) (*InternalMessage, error) {
	return NewInternalMessage(ORDER_DELETE, order)
}

// NewOrderUpdateMessage creates a message for updating an order
func NewOrderUpdateMessage(order *engine.Order) (*InternalMessage, error) {
	return NewInternalMessage(ORDER_UPDATE, order)
}

// NewTradePutMessage creates a message for putting a trade
func NewTradePutMessage(trade *engine.TradeOrder) (*InternalMessage, error) {
	return NewInternalMessage(TRADE_PUT, trade)
}

// NewReceiptPutMessage creates a message for putting a receipt
func NewReceiptPutMessage(receipt *orderbook.Receipt) (*InternalMessage, error) {
	return NewInternalMessage(RECEIPT_PUT, receipt)
}

// Message serialization methods

// ToBytes serializes the internal message to bytes
func (m *InternalMessage) ToBytes() ([]byte, error) {
	return json.Marshal(m)
}

// FromBytes deserializes bytes to internal message
func FromBytes(data []byte) (*InternalMessage, error) {
	var msg InternalMessage
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

// UnmarshalData unmarshals the message data into a specific type
func (m *InternalMessage) UnmarshalData(target interface{}) error {
	return json.Unmarshal(m.Data, target)
}

// Helper function to generate unique message IDs
func generateMessageID() string {
	return uuid.NewString()
}
