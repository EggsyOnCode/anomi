package storage

import (
	"context"
	"database/sql"

	"github.com/EggysOnCode/anomi/storage/models"
	"github.com/EggysOnCode/anomi/storage/repositories"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"go.uber.org/zap"
)

type PgDB struct {
	db          *bun.DB
	amqp        *amqp.Connection
	rabbitmqCfg *RabbitMQConfig
	factory     repositories.RepositoryFactory
	handler     *PgSQLHandler
	logger      *zap.Logger
}

func NewPgDB(conn string, amqp *amqp.Connection, rabbitmqCfg *RabbitMQConfig, logger *zap.Logger) (*PgDB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(conn),
	))
	db := bun.NewDB(sqldb, pgdialect.New())

	pgdb := &PgDB{
		db:          db,
		amqp:        amqp,
		rabbitmqCfg: rabbitmqCfg,
		factory:     repositories.NewRepositoryFactory(db),
		handler:     NewPgSQLHandler(db, logger),
		logger:      logger,
	}

	if err := pgdb.setupDb(); err != nil {
		logger.Error("Failed to setup PostgreSQL database", zap.Error(err))
		return nil, err
	}
	if err := pgdb.launchConsumer(); err != nil {
		logger.Error("Failed to launch PostgreSQL consumer", zap.Error(err))
		return nil, err
	}

	logger.Info("PostgreSQL database initialized successfully")
	return pgdb, nil
}

func (pg *PgDB) setupDb() error {
	ctx := context.Background()

	// Use a transaction to ensure atomicity
	return pg.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		// Create tables
		_, err := tx.NewCreateTable().Model((*models.Order)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewCreateTable().Model((*models.Trade)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewCreateTable().Model((*models.Receipt)(nil)).IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for Order table
		_, err = tx.NewCreateIndex().Model((*models.Order)(nil)).Index("idx_orders_user_id").Column("user_id").IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewCreateIndex().Model((*models.Order)(nil)).Index("idx_orders_order_type").Column("order_type").IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewCreateIndex().Model((*models.Order)(nil)).Index("idx_orders_role").Column("role").IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewCreateIndex().Model((*models.Order)(nil)).Index("idx_orders_user_id_order_type").Column("user_id", "order_type").IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for Trade table
		_, err = tx.NewCreateIndex().Model((*models.Trade)(nil)).Index("idx_trades_user_id").Column("user_id").IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewCreateIndex().Model((*models.Trade)(nil)).Index("idx_trades_order_id").Column("order_id").IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		// Create indexes for Receipt table
		_, err = tx.NewCreateIndex().Model((*models.Receipt)(nil)).Index("idx_receipts_user_id").Column("user_id").IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewCreateIndex().Model((*models.Receipt)(nil)).Index("idx_receipts_order_id").Column("order_id").IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewCreateIndex().Model((*models.Receipt)(nil)).Index("idx_receipts_trade_id").Column("trade_id").IfNotExists().Exec(ctx)
		if err != nil {
			return err
		}

		pg.logger.Info("Database tables and indexes created successfully")
		return nil
	})
}

func (pg *PgDB) launchConsumer() error {
	// Create consumer
	consumer, err := NewRabbitMQConsumer(pg.amqp, pg.rabbitmqCfg)
	if err != nil {
		return err
	}

	// Setup queue
	if err := consumer.SetupQueue(true); err != nil {
		return err
	}

	// Start consuming in a goroutine
	go func() {
		defer consumer.Close()

		msgs, err := consumer.Consume()
		if err != nil {
			pg.logger.Error("Failed to start consumer", zap.Error(err))
			return
		}

		pg.logger.Info("PostgreSQL consumer started, waiting for messages...")

		for msg := range msgs {
			pg.logger.Info("Received message from RabbitMQ", zap.String("messageID", msg.MessageId), zap.String("routingKey", msg.RoutingKey))
			pg.handleMessage(msg)
		}
	}()

	return nil
}

func (pg *PgDB) handleMessage(msg amqp.Delivery) {
	pg.logger.Info("Processing message", zap.String("messageID", msg.MessageId), zap.String("body", string(msg.Body)))
	// Delegate to the handler
	if err := pg.handler.HandleMessage(msg); err != nil {
		pg.logger.Error("Failed to handle message", zap.Error(err))
	} else {
		pg.logger.Info("Successfully processed message", zap.String("messageID", msg.MessageId))
	}
}

// GetHandler returns the PostgreSQL handler
func (pg *PgDB) GetHandler() *PgSQLHandler {
	return pg.handler
}

// Close closes the database connection
func (pg *PgDB) Close() error {
	pg.logger.Info("Closing PostgreSQL database connection")
	return pg.db.Close()
}

// GetDB returns the Bun database instance
func (pg *PgDB) GetDB() *bun.DB {
	return pg.db
}

// GetFactory returns the repository factory
func (pg *PgDB) GetFactory() repositories.RepositoryFactory {
	return pg.factory
}

// Repository accessor methods for convenience
func (pg *PgDB) OrderRepository() repositories.OrderRepository {
	return pg.factory.NewOrderRepository()
}

func (pg *PgDB) TradeRepository() repositories.TradeRepository {
	return pg.factory.NewTradeRepository()
}

func (pg *PgDB) ReceiptRepository() repositories.ReceiptRepository {
	return pg.factory.NewReceiptRepository()
}
