package network

import (
	"sync"

	"github.com/EggysOnCode/anomi/logger"
	"github.com/EggysOnCode/anomi/rpc"
	"github.com/EggysOnCode/anomi/storage"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// privatekey for singing, id, etc..
type NodeInfo struct {
	Id         string // from id for rpc msgs
	PrivateKey string
}

type MailboxConfig struct {
	P2pServer   *Server
	Amqp        *amqp.Connection
	NodeInfo    *NodeInfo
	RabbitMQCfg *storage.RabbitMQConfig
}

type Mailbox struct {
	outCh  chan []byte
	mu     *sync.RWMutex
	cfg    *MailboxConfig
	logger *zap.Logger
}

func NewMailbox(cfg *MailboxConfig) *Mailbox {
	out := make(chan []byte, 1000)
	mailboxLogger := logger.Get()

	box := &Mailbox{
		outCh:  out,
		cfg:    cfg,
		mu:     &sync.RWMutex{},
		logger: mailboxLogger,
	}

	go box.launchBroadcast() // consumes and saves to outch
	go box.broadcast()       // sends msgs to p2p server

	return box
}

// consumes from rabbitmq and broadcasts to the p2p network
func (m *Mailbox) launchBroadcast() error {
	// Create consumer
	consumer, err := storage.NewRabbitMQConsumer(m.cfg.Amqp, m.cfg.RabbitMQCfg)
	if err != nil {
		return err
	}

	// Setup queue
	if err := consumer.SetupQueue(false); err != nil {
		return err
	}

	// Start consuming in a goroutine
	go func() {
		defer consumer.Close()

		msgs, err := consumer.Consume()
		if err != nil {
			m.logger.Error("Failed to start consumer", zap.Error(err))
			return
		}

		m.logger.Info("Mailbox consumer started, waiting for messages...")

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

	// p2pserver will auto format intenral msgs to rpc msgs
	m.outCh <- msg
}

func (m *Mailbox) broadcast() {
	for msg := range m.outCh {
		internalMsg, err := rpc.FromBytes(msg)
		if err != nil {
			m.logger.Error("Failed to convert msg to internal message", zap.Error(err))
			continue
		}
		m.cfg.P2pServer.BroadcastMsg(internalMsg)
	}
}

func (m *Mailbox) Stop() {
	close(m.outCh)
}
