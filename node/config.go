package node

import (
	"fmt"

	"github.com/EggysOnCode/anomi/storage"
)

type OrderBookCfg struct {
	Base  string
	Quote string
}

func (ob *OrderBookCfg) Symbol() string {
	return fmt.Sprintf("%s/%s", ob.Base, ob.Quote)
}

type NodeConfig struct {
	Books          []OrderBookCfg
	HttpServerPort string
	DbConn         string
	KvdbPath       string
	RabbitmqCfg    storage.RabbitMQConfig
	ListenAddr     string   // for p2p
	BootStrapNodes []string // for p2p
}
