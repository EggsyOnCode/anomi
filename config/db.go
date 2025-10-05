package config

const (
	Username    = "guest"
	Password    = "guest"
	Host        = "localhost"
	VHost       = "/"
	Exchange    = "amq.fanout"
	QueueName   = "orderbook.ops"
	RoutingKey  = "orderbook"
	BindingKey  = "orderbook"
	ConsumerTag = "orderbook"
	PostgresDB  = "postgres://guest:guest@localhost:5432/anomi?sslmode=disable"
)
