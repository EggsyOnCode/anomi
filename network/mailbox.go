package network

import (
	"log"
	"sync"

	"github.com/EggysOnCode/anomi/config"
	"github.com/EggysOnCode/anomi/storage"
	amqp "github.com/rabbitmq/amqp091-go"
)

// privatekey for singing, id, etc..
type NodeInfo struct {
}

type MailboxConfig struct {
	Transport *LibP2pTransport
	Amqp      *amqp.Connection
	ApiServer string // port
	NodeInfo  *NodeInfo
}

type Mailbox struct {
	outCh chan []byte
	mu    *sync.RWMutex
	cfg   *MailboxConfig
}

func NewMailbox(cfg *MailboxConfig) *Mailbox {
	out := make(chan []byte, 1000)

	box := &Mailbox{
		outCh: out,
		cfg:   cfg,
	}

	go box.launchBroadcast()
	// go box.broadcast()
	go box.listen()

	return box
}

// consumes from rabbitmq and broadcasts to the p2p network
func (m *Mailbox) launchBroadcast() error {
	// Create consumer configuration
	config := &storage.RabbitMQConfig{
		Username:    config.Username,
		Password:    config.Password,
		Host:        config.Host,
		VHost:       config.VHost,
		Exchange:    config.Exchange,
		QueueName:   config.QueueName,
		RoutingKey:  config.RoutingKey,
		BindingKey:  config.BindingKey,
		ConsumerTag: config.ConsumerTag,
	}

	// Create consumer
	consumer, err := storage.NewRabbitMQConsumer(m.cfg.Amqp, config)
	if err != nil {
		return err
	}

	// Setup queue
	if err := consumer.SetupQueue(); err != nil {
		return err
	}

	// Start consuming in a goroutine
	go func() {
		defer consumer.Close()

		msgs, err := consumer.Consume()
		if err != nil {
			log.Printf("Failed to start consumer: %v", err)
			return
		}

		log.Println("Mailbox consumer started, waiting for messages...")

		for msg := range msgs {
			m.Out(msg.Body) // forward the msg to the api server
		}
	}()

	return nil
}

// called when the caller wishes to send a msg to the p2p network
// use the msg as payload to constrcut RPC.msg
func (m *Mailbox) Out(msg []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.outCh <- msg
}

func (m *Mailbox) broadcast() {
	for msg := range m.outCh {
		m.cfg.Transport.Broadcast(msg)
	}
}

func (m *Mailbox) listen() {
	msgs := m.cfg.Transport.ConsumeMsgs()
	for msg := range msgs {
		// TODO: replace this with api gateway or something else that should be handling the msg
		m.outCh <- msg.Payload
	}
}

func (m *Mailbox) Stop() {
	close(m.outCh)
}
