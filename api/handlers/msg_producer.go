package handlers

import (
	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
)

// MessageProducer interface for publishing messages
type MessageProducer interface {
	PublishOrderCreated(order *engine.Order) error
	PublishOrderUpdated(order *engine.Order) error
	PublishOrderDeleted(order *engine.Order) error
	PublishTradeExecuted(trade *engine.TradeOrder) error
	PublishReceiptGenerated(receipt *orderbook.Receipt) error
}


