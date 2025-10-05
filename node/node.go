package node

import (
	"context"
	"path/filepath"

	"github.com/EggysOnCode/anomi/api"
	"github.com/EggysOnCode/anomi/api/controllers/p2p"
	"github.com/EggysOnCode/anomi/api/handlers"
	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/crypto"
	"github.com/EggysOnCode/anomi/logger"
	"github.com/EggysOnCode/anomi/network"
	"github.com/EggysOnCode/anomi/rpc"
	"github.com/EggysOnCode/anomi/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Node struct {
	id        string
	cfg       *NodeConfig
	db        *storage.PgDB
	kvdb      *storage.KvDB
	api       *api.Server
	p2pServer *network.Server
	pk        crypto.PrivateKey
	mailbox   *network.Mailbox
	logger    *zap.Logger
}

func NewNode(cfg *NodeConfig) *Node {
	if len(cfg.Books) == 0 {
		return nil // we need at least one book to run the node
	}

	// Initialize logger with node ID (will be set after node creation)
	nodeLogger := logger.Get()

	var books []*orderbook.OrderBook

	// create orderbooks based on cfg
	for _, book := range cfg.Books {
		b, err := orderbook.NewOrderBook(book.Base, book.Quote, 0)
		if err != nil {
			nodeLogger.Error("Failed to create orderbook", zap.String("base", book.Base), zap.String("quote", book.Quote), zap.Error(err))
			panic(err)
		}
		books = append(books, b)
	}

	// init kvdb
	path := filepath.Join(cfg.KvdbPath, "kvdb")
	kvdb, err := storage.NewDB(path, nodeLogger)
	if err != nil {
		nodeLogger.Error("Failed to initialize KVDB", zap.String("path", path), zap.Error(err))
		return nil
	}

	// init rabbitmq conn
	amqpConn, err := storage.CreateRmqpConnection(cfg.RabbitmqCfg.Username, cfg.RabbitmqCfg.Password, cfg.RabbitmqCfg.Host, cfg.RabbitmqCfg.VHost)
	if err != nil {
		nodeLogger.Error("Failed to create RabbitMQ connection", zap.Error(err))
		return nil
	}

	// init postgres
	pgdb, err := storage.NewPgDB(cfg.DbConn, amqpConn, &cfg.RabbitmqCfg, nodeLogger)
	if err != nil {
		nodeLogger.Error("Failed to initialize PostgreSQL", zap.Error(err))
		return nil
	}

	// setup p2pHandlers and controllers
	msgProducer := handlers.NewRabbitMQMessageProducer(amqpConn, &cfg.RabbitmqCfg, nodeLogger)
	orderHandler := handlers.NewOrderHandler(books, kvdb, msgProducer, nodeLogger)
	tradeHandler := handlers.NewTradeHandler(kvdb, msgProducer, nodeLogger)
	receiptHandler := handlers.NewReceiptHandler(kvdb, msgProducer, nodeLogger)
	p2pController := p2p.NewP2PController(orderHandler, tradeHandler, receiptHandler, nodeLogger)

	// setup lib p2p transport layer
	pk := crypto.GeneratePrivateKey()
	p2pServerOpts := &network.ServerOpts{
		ListenAddr:     cfg.ListenAddr,
		CodecType:      rpc.JsonCodec,
		BootstrapNodes: cfg.BootStrapNodes,
		PrivateKey:     pk,
	}
	p2pServer := network.NewServer(p2pServerOpts, p2pController)

	// init mailbox
	mailBoxCfg := &network.MailboxConfig{
		P2pServer: p2pServer,
		Amqp:      amqpConn,
		NodeInfo: &network.NodeInfo{
			Id:         pk.PublicKey().String(),
			PrivateKey: pk.Key().X.String(),
		},
		RabbitMQCfg: &cfg.RabbitmqCfg,
	}

	mailbox := network.NewMailbox(mailBoxCfg)

	// http port
	var p string
	if cfg.HttpServerPort == "" {
		p = "8080"
	} else {
		p = cfg.HttpServerPort
	}

	// init api server
	apiCfg := &api.Config{
		Port:        p,
		Orderbooks:  books,
		KvDB:        kvdb,
		MsgProducer: msgProducer,
		Logger:      nodeLogger,
	}
	apiServer := api.NewServer(apiCfg)

	nodeID := uuid.NewString()
	return &Node{
		id:        nodeID,
		cfg:       cfg,
		db:        pgdb,
		kvdb:      kvdb,
		api:       apiServer,
		p2pServer: p2pServer,
		mailbox:   mailbox,
		pk:        *pk,
		logger:    logger.GetWithNodeID(nodeID),
	}
}

// Start starts the node and all its components
func (n *Node) Start(ctx context.Context) error {
	n.logger.Info("Starting Anomi node...")

	// Start P2P server in a goroutine
	go func() {
		n.p2pServer.Start()
	}()

	// Start API server in a goroutine
	go func() {
		if err := n.api.Start(n.cfg.HttpServerPort); err != nil {
			n.logger.Error("Failed to start API server", zap.Error(err))
		}
	}()

	// Wait for context cancellation or timeout
	<-ctx.Done()
	return ctx.Err()
}

// Stop stops the node and all its components
func (n *Node) Stop() error {
	n.logger.Info("Stopping Anomi node...")

	// Stop P2P server
	n.p2pServer.Stop()

	// Stop API server
	if err := n.api.Stop(); err != nil {
		n.logger.Error("Failed to stop API server", zap.Error(err))
	}

	// Stop mailbox
	n.mailbox.Stop()

	// Close database connections
	if n.db != nil {
		// Add database close method if available
	}

	// Close KVDB
	if n.kvdb != nil {
		// Add KVDB close method if available
	}

	n.logger.Info("Anomi node stopped")
	return nil
}

// GetKVDB returns the KVDB instance
func (n *Node) GetKVDB() *storage.KvDB {
	return n.kvdb
}

// GetDB returns the PostgreSQL database instance
func (n *Node) GetDB() *storage.PgDB {
	return n.db
}

// GetP2PServer returns the P2P server instance
func (n *Node) GetP2PServer() *network.Server {
	return n.p2pServer
}

// GetAPIServer returns the API server instance
func (n *Node) GetAPIServer() *api.Server {
	return n.api
}

// GetMailbox returns the mailbox instance
func (n *Node) GetMailbox() *network.Mailbox {
	return n.mailbox
}

// GetLogger returns the logger instance
func (n *Node) GetLogger() *zap.Logger {
	return n.logger
}

// GetConfig returns the node configuration
func (n *Node) GetConfig() *NodeConfig {
	return n.cfg
}

func (n *Node) ID() string {
	return n.id
}
