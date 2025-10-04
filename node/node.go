package node

import (
	"path/filepath"

	"github.com/EggysOnCode/anomi/api"
	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/crypto"
	"github.com/EggysOnCode/anomi/network"
	"github.com/EggysOnCode/anomi/rpc"
	"github.com/EggysOnCode/anomi/storage"
)

type Node struct {
	cfg             *NodeConfig
	db              *storage.PgDB
	kvdb            *storage.KvDB
	api             *api.Server
	mailbox         *network.Mailbox
	rpcCh           chan *rpc.RPCMessage
	libp2pTransport *network.LibP2pTransport
	pk              crypto.PrivateKey
}

func NewNode(cfg *NodeConfig) *Node {
	if cfg.books == nil || len(cfg.books) == 0 {
		return nil // we need at least one book to run the node
	}

	// create orderbooks based on cfg
	for _, book := range cfg.books {
		orderbook.NewOrderBook(book.Base, book.Quote, 0)
	}

	// init kvdb
	path := filepath.Join(cfg.kvdbPath, "kvdb")
	kvdb, err := storage.NewDB(path)
	if err != nil {
		return nil
	}

	// init rabbitmq conn
	amqpConn, err := storage.CreateRmqpConnection(cfg.rabbitmqCfg.Username, cfg.rabbitmqCfg.Password, cfg.rabbitmqCfg.Host, cfg.rabbitmqCfg.VHost)
	if err != nil {
		return nil
	}

	// init postgres
	pgdb, err := storage.NewPgDB(cfg.dbConn, amqpConn)
	if err != nil {
		return nil
	}

	// setup lib p2p transport layer
	rpcCh := make(chan *rpc.RPCMessage, 1000)
	pk := crypto.GeneratePrivateKey()
	libp2pTransport := network.NewLibp2pTransport(rpcCh, &rpc.JSONCodec{}, *pk)

	// init mailbox


	return nil
}
