package node

import "github.com/EggysOnCode/anomi/storage"

type OrderBookCfg struct {
	Base  string
	Quote string
}

type NodeConfig struct {
	books          []OrderBookCfg
	httpServerPort string
	dbConn         string
	kvdbPath       string
	rabbitmqCfg    storage.RabbitMQConfig
}
