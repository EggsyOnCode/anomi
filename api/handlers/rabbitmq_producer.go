package handlers

import (
	"fmt"

	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/EggysOnCode/anomi/rpc"
	"github.com/EggysOnCode/anomi/storage"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// RabbitMQMessageProducer implements MessageProducer using RabbitMQ
type RabbitMQMessageProducer struct {
	conn   *amqp.Connection
	cfg    *storage.RabbitMQConfig
	prod   *storage.RabbitMQProducer
	logger *zap.Logger
}

// NewRabbitMQMessageProducer creates a new producer instance
func NewRabbitMQMessageProducer(conn *amqp.Connection, cfg *storage.RabbitMQConfig, logger *zap.Logger) *RabbitMQMessageProducer {
	// Create a producer channel from the provided connection and cfg
	prod, err := storage.NewRabbitMQProducer(conn, cfg)
	if err != nil {
		logger.Error("Failed to create RabbitMQ producer", zap.Error(err))
		panic(err)
	}
	// Ensure exchange exists
	if err := prod.SetupExchange(); err != nil {
		logger.Error("Failed to setup RabbitMQ exchange", zap.Error(err))
		panic(err)
	}
	logger.Info("RabbitMQ message producer initialized successfully")
	return &RabbitMQMessageProducer{
		conn:   conn,
		cfg:    cfg,
		prod:   prod,
		logger: logger,
	}
}

func (p *RabbitMQMessageProducer) PublishOrderCreated(order *engine.Order) error {
	msg, err := rpc.NewOrderPutMessage(order)
	if err != nil {
		return err
	}
	return p.publishInternal(msg)
}

func (p *RabbitMQMessageProducer) PublishOrderUpdated(order *engine.Order) error {
	msg, err := rpc.NewOrderUpdateMessage(order)
	if err != nil {
		return err
	}
	return p.publishInternal(msg)
}

func (p *RabbitMQMessageProducer) PublishOrderDeleted(order *engine.Order) error {
	msg, err := rpc.NewOrderDeleteMessage(order)
	if err != nil {
		return err
	}
	return p.publishInternal(msg)
}

func (p *RabbitMQMessageProducer) PublishTradeExecuted(trade *engine.TradeOrder) error {
	msg, err := rpc.NewTradePutMessage(trade)
	if err != nil {
		return err
	}
	return p.publishInternal(msg)
}

func (p *RabbitMQMessageProducer) PublishReceiptGenerated(receipt *orderbook.Receipt) error {
	msg, err := rpc.NewReceiptPutMessage(receipt)
	if err != nil {
		return err
	}
	return p.publishInternal(msg)
}

func (p *RabbitMQMessageProducer) publishInternal(msg *rpc.InternalMessage) error {
	body, err := msg.ToBytes()
	if err != nil {
		return err
	}

	// Prefer producer wrapper; fallback to client if needed
	if p.prod != nil {
		return p.prod.Send(amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		}, "")
	} else {
		return fmt.Errorf("rabbitmq prodcuer not initalized")
	}
}
