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
	QueueName  string // Store the actual queue name used
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
func (c *RabbitMQConsumer) SetupQueue(db bool) error {
	// Declare exchange
	err := c.Channel.ExchangeDeclare(
		c.Config.Exchange, // name
		"fanout",          // type
		true,              // durable
		false,             // auto-deleted
		false,             // internal
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		return err
	}

	var queue string
	if db {
		queue = c.Config.QueueName + "db"
	} else {
		queue = c.Config.QueueName + "mailbox"
	}

	// Store the queue name for later use
	c.QueueName = queue

	// Declare queue
	_, err = c.Channel.QueueDeclare(
		queue, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	// Bind queue to exchange (for fanout, routing key is ignored)
	err = c.Channel.QueueBind(
		queue,             // queue name
		"",                // routing key (ignored for fanout)
		c.Config.Exchange, // exchange
		false,             // no-wait
		nil,               // arguments
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
		c.QueueName,          // queue (use the actual queue name)
		c.Config.ConsumerTag, // consumer
		false,                // auto-ack
		false,                // exclusive
		false,                // no-local
		false,                // no-wait
		nil,                  // args
	)
}

// RabbitMQProducer holds connection and configuration for a producer
type RabbitMQProducer struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
	Config     *RabbitMQConfig
}

// NewRabbitMQProducer creates a new producer with the given configuration
func NewRabbitMQProducer(conn *amqp.Connection, config *RabbitMQConfig) (*RabbitMQProducer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Enable publisher confirms on channel
	if err := ch.Confirm(false); err != nil {
		return nil, err
	}

	return &RabbitMQProducer{
		Connection: conn,
		Channel:    ch,
		Config:     config,
	}, nil
}

// SetupExchange ensures the exchange exists
func (p *RabbitMQProducer) SetupExchange() error {
	return p.Channel.ExchangeDeclare(
		p.Config.Exchange, // name
		"fanout",          // type
		true,              // durable
		false,             // auto-deleted
		false,             // internal
		false,             // no-wait
		nil,               // arguments
	)
}

// ApplyQoS sets the QoS on the producer channel
func (p *RabbitMQProducer) ApplyQoS(count, size int, global bool) error {
	return p.Channel.Qos(count, size, global)
}

// Close closes the producer channel
func (p *RabbitMQProducer) Close() error {
	return p.Channel.Close()
}

// Send publishes a message to the configured exchange and routing key
func (p *RabbitMQProducer) Send(ctx amqp.Publishing, routingKeyOverride string) error {
	rk := ""
	if routingKeyOverride != "" {
		rk = routingKeyOverride
	}

	// Use basic publish with confirm deferred
	confirmation, err := p.Channel.PublishWithDeferredConfirm(
		p.Config.Exchange,
		rk,    // empty routing key for fanout exchange
		true,  // mandatory
		false, // immediate (deprecated, must be false)
		ctx,
	)
	if err != nil {
		return err
	}
	// fire-and-forget semantics: we don't block on confirm; still wait quickly to flush
	confirmation.Wait()
	return nil
}
