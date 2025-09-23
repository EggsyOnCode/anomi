package storage

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQConfig holds configuration for RabbitMQ connections
type RabbitMQConfig struct {
	Username    string
	Password    string
	Host        string
	VHost       string
	Exchange    string
	QueueName   string
	RoutingKey  string
	BindingKey  string
	ConsumerTag string
}

// RabbitMQConsumer holds connection and configuration for a consumer
type RabbitMQConsumer struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
	Config     *RabbitMQConfig
}

// NewRabbitMQConsumer creates a new consumer with the given configuration
func NewRabbitMQConsumer(conn *amqp.Connection, config *RabbitMQConfig) (*RabbitMQConsumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Set QoS for consumer
	if err := ch.Qos(1, 0, false); err != nil {
		return nil, err
	}

	return &RabbitMQConsumer{
		Connection: conn,
		Channel:    ch,
		Config:     config,
	}, nil
}

// SetupQueue creates and binds the queue for this consumer
func (c *RabbitMQConsumer) SetupQueue() error {
	// Declare exchange
	err := c.Channel.ExchangeDeclare(
		c.Config.Exchange, // name
		"direct",           // type
		true,              // durable
		false,             // auto-deleted
		false,             // internal
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		return err
	}

	// Declare queue
	_, err = c.Channel.QueueDeclare(
		c.Config.QueueName, // name
		true,               // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		return err
	}

	// Bind queue to exchange
	err = c.Channel.QueueBind(
		c.Config.QueueName,  // queue name
		c.Config.BindingKey, // routing key
		c.Config.Exchange,   // exchange
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return err
	}

	return nil
}

// Close closes the consumer channel
func (c *RabbitMQConsumer) Close() error {
	return c.Channel.Close()
}

// Consume starts consuming messages from the queue
func (c *RabbitMQConsumer) Consume() (<-chan amqp.Delivery, error) {
	return c.Channel.Consume(
		c.Config.QueueName,   // queue
		c.Config.ConsumerTag, // consumer
		false,                // auto-ack
		false,                // exclusive
		false,                // no-local
		false,                // no-wait
		nil,                  // args
	)
}
