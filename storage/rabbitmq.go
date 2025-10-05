package storage

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type RabbitMQClient struct {
	// TCP conncetion with rmqp server
	conn *amqp.Connection
	// channel is used to receive / produce messages
	ch     *amqp.Channel
	logger *zap.Logger
}

// TLS is unncessary since all the comms will be local
func CreateRmqpConnection(username, password, host, vhost string) (*amqp.Connection, error) {
	// Setup the Connection to RabbitMQ host using AMQPs
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s/%s", username, password, host, vhost))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func NewRabbitMQClient(conn *amqp.Connection, logger *zap.Logger) *RabbitMQClient {
	ch, err := conn.Channel()
	if err != nil {
		logger.Error("Failed to create RabbitMQ channel", zap.Error(err))
		panic(err)
	}

	if err := ch.Confirm(false); err != nil {
		logger.Error("Failed to set RabbitMQ channel confirmation", zap.Error(err))
		panic(err)
	}

	logger.Info("RabbitMQ client initialized successfully")
	return &RabbitMQClient{
		conn:   conn,
		ch:     ch,
		logger: logger,
	}
}

// close closes the one of hte multiplexed channels and not the underlying TCP connections
func (rc *RabbitMQClient) Close() error {
	rc.logger.Info("Closing RabbitMQ client")
	return rc.ch.Close()
}

// Conn exposes the underlying AMQP connection
func (rc *RabbitMQClient) Conn() *amqp.Connection {
	return rc.conn
}

// a wrapper method around amqp.Channel.QueueDeclare to avoid exposing the channel directly to the user
func (rc *RabbitMQClient) CreateQueue(name string, durable, autodelete bool) (amqp.Queue, error) {
	q, err := rc.ch.QueueDeclare(name, durable, autodelete, false, false, nil)
	if err != nil {
		return amqp.Queue{}, err
	}
	return q, err
}

// bind the queue with the specific binding key to an exchange
func (rc *RabbitMQClient) CreateBinding(queue_name, binding_key, exchange string) error {
	// leaving noWait to false, which will return an error if the channle fails to bind
	return rc.ch.QueueBind(queue_name, binding_key, exchange, false, nil)
}

// wrapper for publishing messages
func (rc *RabbitMQClient) Send(ctx context.Context, exchange, routingKey string, options amqp.Publishing) error {
	confirmation, err := rc.ch.PublishWithDeferredConfirmWithContext(ctx,
		exchange,
		routingKey,
		// mandatory flag is for receiving an error if exchange encoutners a failure to send msg
		true,
		false,
		options,
	)

	if err != nil {
		return err
	}

	confirmation.Wait()
	rc.logger.Debug("Message confirmation received")
	return nil
}

// wrapper for consuming msgs
func (rc *RabbitMQClient) Consume(queue, consumer string, autoAck bool) (<-chan amqp.Delivery, error) {
	return rc.ch.Consume(
		queue,
		consumer,
		// if the consuming microservice takes time to process the msgs and there's a possibiltiy of failure
		// better manually ACK then set autoAck because autoAck removes msg from queue as soon as its delivered
		autoAck,
		// exclusive flag should be set to true only if there's only one consumer consuming from the queue, otherwise its set to false
		// and server will use load-balancing to distribute msgs
		false,
		false,
		false,
		nil,
	)
}

// wrapper for QoS (Quality of Service) -- to avoid server form spamming consumers with unACK msgs
// prefetch count - an int on how many UNACK msg server can send
// prefetch size - an int of how many bytes of UNACK msg server can send
// global - a bool to set the QoS on the channel or the connection
func (rc *RabbitMQClient) ApplyQoS(count, size int, global bool) error {
	return rc.ch.Qos(count, size, global)
}
