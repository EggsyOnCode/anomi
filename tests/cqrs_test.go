package tests

import (
	"context"
	"encoding/json"
	"log"
	"testing"
	"time"

	"github.com/EggysOnCode/anomi/config"
	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/EggysOnCode/anomi/storage"
	"github.com/nikolaydubina/fpdecimal"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCQRSE2E tests the complete CQRS workflow:
// 1. Initialize KvDB (Command side)
// 2. Initialize RabbitMQ connection
// 3. Initialize PostgreSQL with RabbitMQ consumer (Query side)
// 4. Create orders and save to KvDB
// 5. Publish events to RabbitMQ
// 6. Verify PostgreSQL receives and processes events
func TestCQRSE2E(t *testing.T) {
	// Skip if running in CI or if services are not available
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	_ = context.Background()

	// Step 1: Initialize KvDB (Command side)
	t.Log("Initializing KvDB...")
	kvdb, err := storage.NewDB("test_cqrs_orders")
	require.NoError(t, err)
	defer kvdb.Close()

	// Step 2: Initialize RabbitMQ connection
	t.Log("Initializing RabbitMQ connection...")
	amqpConn, err := storage.CreateRmqpConnection(
		config.Username,
		config.Password,
		config.Host+":5672",
		config.VHost,
	)
	require.NoError(t, err)
	defer amqpConn.Close()

	// Step 3: Initialize PostgreSQL with RabbitMQ consumer (Query side)
	t.Log("Initializing PostgreSQL with RabbitMQ consumer...")
	pgdb, err := storage.NewPgDB(config.PostgresDB, amqpConn)
	require.NoError(t, err)
	defer pgdb.Close()

	// Wait a moment for consumer to start
	time.Sleep(2 * time.Second)

	// Step 4: Create test orders and save to KvDB
	t.Log("Creating test orders...")
	testOrders := createTestOrders(t)

	// Save orders to KvDB
	for _, order := range testOrders {
		err := kvdb.PutOrder(order)
		require.NoError(t, err)
		t.Logf("Saved order %s to KvDB", order.ID())
	}

	// Step 5: Create test trades and receipts
	t.Log("Creating test trades and receipts...")
	testTrades := createTestTrades(t, testOrders)
	testReceipts := createTestReceipts(t, testOrders, testTrades)

	// Save trades and receipts to KvDB
	for _, trade := range testTrades {
		err := kvdb.PutTradeOrder(trade)
		require.NoError(t, err)
		t.Logf("Saved trade %s to KvDB", trade.OrderID)
	}

	for _, receipt := range testReceipts {
		err := kvdb.PutReceipt(receipt)
		require.NoError(t, err)
		t.Logf("Saved receipt %s to KvDB", receipt.OrderID)
	}

	// Step 6: Publish events to RabbitMQ
	t.Log("Publishing events to RabbitMQ...")
	err = publishEventsToRabbitMQ(amqpConn, testOrders, testTrades, testReceipts)
	require.NoError(t, err)

	// Step 7: Wait for events to be processed
	t.Log("Waiting for events to be processed...")
	time.Sleep(5 * time.Second)

	// Step 8: Verify PostgreSQL received and processed events
	t.Log("Verifying PostgreSQL data...")
	verifyPostgreSQLData(t, pgdb, testOrders, testTrades, testReceipts)

	t.Log("CQRS E2E test completed successfully!")
}

// createTestOrders creates sample orders for testing
func createTestOrders(t *testing.T) []*engine.Order {
	orders := []*engine.Order{}

	// Create limit orders
	limitOrder1 := engine.NewLimitOrder(
		"order_limit_1",
		engine.Buy,
		fpdecimal.FromInt(100),
		fpdecimal.FromInt(10),
		engine.GTC,
		"",
		"user_1",
	)
	orders = append(orders, limitOrder1)

	limitOrder2 := engine.NewLimitOrder(
		"order_limit_2",
		engine.Sell,
		fpdecimal.FromInt(105),
		fpdecimal.FromInt(5),
		engine.GTC,
		"",
		"user_2",
	)
	orders = append(orders, limitOrder2)

	// Create market orders
	marketOrder1 := engine.NewMarketOrder(
		"order_market_1",
		engine.Buy,
		fpdecimal.FromInt(20),
		"user_1",
	)
	orders = append(orders, marketOrder1)

	// Create stop-limit orders
	stopOrder1 := engine.NewStopLimitOrder(
		"order_stop_1",
		engine.Sell,
		fpdecimal.FromInt(15),
		fpdecimal.FromInt(95),
		fpdecimal.FromInt(100),
		"",
		"user_3",
	)
	orders = append(orders, stopOrder1)

	return orders
}

// createTestTrades creates sample trades for testing
func createTestTrades(t *testing.T, orders []*engine.Order) []*engine.TradeOrder {
	trades := []*engine.TradeOrder{}

	for i, order := range orders {
		if i >= 2 { // Only create trades for first 2 orders
			break
		}

		trade := &engine.TradeOrder{
			OrderID:  order.ID(),
			UserId:   order.UserID(),
			Role:     engine.MAKER,
			Price:    order.Price(),
			IsQuote:  order.IsQuote(),
			Quantity: order.Quantity().Div(fpdecimal.FromInt(2)), // Half quantity
		}
		trades = append(trades, trade)
	}

	return trades
}

// createTestReceipts creates sample receipts for testing
func createTestReceipts(t *testing.T, orders []*engine.Order, trades []*engine.TradeOrder) []*orderbook.Receipt {
	receipts := []*orderbook.Receipt{}

	for _, trade := range trades {
		receipt := &orderbook.Receipt{
			UserID:    trade.UserId,
			OrderID:   trade.OrderID,
			Trades:    []*engine.TradeOrder{trade},
			FilledQty: trade.Quantity,
		}
		receipts = append(receipts, receipt)
	}

	return receipts
}

// publishEventsToRabbitMQ publishes events to RabbitMQ
func publishEventsToRabbitMQ(conn *amqp.Connection, orders []*engine.Order, trades []*engine.TradeOrder, receipts []*orderbook.Receipt) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Declare exchange
	err = ch.ExchangeDeclare(
		config.Exchange, // name
		"direct",        // type
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		return err
	}

	// Publish order events
	for _, order := range orders {
		orderData, err := json.Marshal(order)
		if err != nil {
			return err
		}

		err = ch.Publish(
			config.Exchange,
			config.RoutingKey,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        orderData,
				Headers: amqp.Table{
					"event_type": "order.created",
					"order_id":   order.ID(),
					"user_id":    order.UserID(),
				},
			},
		)
		if err != nil {
			return err
		}

		log.Printf("Published order event for %s", order.ID())
	}

	// Publish trade events
	for _, trade := range trades {
		tradeData, err := json.Marshal(trade)
		if err != nil {
			return err
		}

		err = ch.Publish(
			config.Exchange,
			config.RoutingKey,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        tradeData,
				Headers: amqp.Table{
					"event_type": "trade.executed",
					"order_id":   trade.OrderID,
					"user_id":    trade.UserId,
				},
			},
		)
		if err != nil {
			return err
		}

		log.Printf("Published trade event for %s", trade.OrderID)
	}

	// Publish receipt events
	for _, receipt := range receipts {
		receiptData, err := json.Marshal(receipt)
		if err != nil {
			return err
		}

		err = ch.Publish(
			config.Exchange,
			config.RoutingKey,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        receiptData,
				Headers: amqp.Table{
					"event_type": "receipt.created",
					"order_id":   receipt.OrderID,
					"user_id":    receipt.UserID,
				},
			},
		)
		if err != nil {
			return err
		}

		log.Printf("Published receipt event for %s", receipt.OrderID)
	}

	return nil
}

// verifyPostgreSQLData verifies that PostgreSQL received and processed the events
func verifyPostgreSQLData(t *testing.T, pgdb *storage.PgDB, orders []*engine.Order, trades []*engine.TradeOrder, receipts []*orderbook.Receipt) {
	ctx := context.Background()

	// Get repositories
	orderRepo := pgdb.OrderRepository()
	tradeRepo := pgdb.TradeRepository()
	receiptRepo := pgdb.ReceiptRepository()

	// Verify orders were created
	t.Log("Verifying orders in PostgreSQL...")
	for _, expectedOrder := range orders {
		order, err := orderRepo.GetByID(ctx, expectedOrder.ID())
		require.NoError(t, err, "Order %s should exist in PostgreSQL", expectedOrder.ID())
		require.NotNil(t, order, "Order %s should not be nil", expectedOrder.ID())

		assert.Equal(t, expectedOrder.ID(), (order).ID)
		assert.Equal(t, expectedOrder.UserID(), (order).UserID)
		assert.Equal(t, int(expectedOrder.Side()), (order).Side)
		assert.Equal(t, expectedOrder.Quantity().String(), (order).Quantity)
		assert.Equal(t, expectedOrder.Price().String(), (order).Price)

		t.Logf("✓ Verified order %s in PostgreSQL", expectedOrder.ID())
	}

	// Verify trades were created
	t.Log("Verifying trades in PostgreSQL...")
	for _, expectedTrade := range trades {
		trades, err := tradeRepo.GetByOrderID(ctx, expectedTrade.OrderID)
		require.NoError(t, err, "Trades for order %s should exist in PostgreSQL", expectedTrade.OrderID)
		require.NotEmpty(t, trades, "Trades for order %s should not be empty", expectedTrade.OrderID)

		trade := trades[0] // Get first trade
		assert.Equal(t, expectedTrade.OrderID, trade.OrderID)
		assert.Equal(t, expectedTrade.UserId, trade.UserID)
		assert.Equal(t, string(expectedTrade.Role), trade.Role)
		assert.Equal(t, expectedTrade.Price.String(), trade.Price)
		assert.Equal(t, expectedTrade.Quantity.String(), trade.Quantity)

		t.Logf("✓ Verified trade for order %s in PostgreSQL", expectedTrade.OrderID)
	}

	// Verify receipts were created
	t.Log("Verifying receipts in PostgreSQL...")
	for _, expectedReceipt := range receipts {
		receipts, err := receiptRepo.GetByOrderID(ctx, expectedReceipt.OrderID)
		require.NoError(t, err, "Receipts for order %s should exist in PostgreSQL", expectedReceipt.OrderID)
		require.NotEmpty(t, receipts, "Receipts for order %s should not be empty", expectedReceipt.OrderID)

		receipt := receipts[0] // Get first receipt
		assert.Equal(t, expectedReceipt.UserID, receipt.UserID)
		assert.Equal(t, expectedReceipt.OrderID, receipt.OrderID)
		assert.Equal(t, expectedReceipt.FilledQty.String(), receipt.FilledQty)

		t.Logf("✓ Verified receipt for order %s in PostgreSQL", expectedReceipt.OrderID)
	}

	// Test some repository methods
	t.Log("Testing repository methods...")

	// Test GetByUserID
	user1Orders, err := orderRepo.GetByUserID(ctx, "user_1")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(user1Orders), 2, "User 1 should have at least 2 orders")

	// Test GetActiveOrders
	activeOrders, err := orderRepo.GetActiveOrders(ctx, "user_1")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(activeOrders), 2, "User 1 should have at least 2 active orders")

	// Test GetByTimeRange
	now := time.Now()
	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)
	recentOrders, err := orderRepo.GetByTimeRange(ctx, start, end)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(recentOrders), len(orders), "Should have recent orders")

	t.Log("✓ All PostgreSQL verifications passed!")
}

// TestCQRSWorkflowIntegration tests the integration between components
func TestCQRSWorkflowIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Initialize all components
	kvdb, err := storage.NewDB("test_integration")
	require.NoError(t, err)
	defer kvdb.Close()

	amqpConn, err := storage.CreateRmqpConnection(
		config.Username,
		config.Password,
		config.Host+":5672",
		config.VHost,
	)
	require.NoError(t, err)
	defer amqpConn.Close()

	pgdb, err := storage.NewPgDB(config.PostgresDB, amqpConn)
	require.NoError(t, err)
	defer pgdb.Close()

	// Wait for consumer to start
	time.Sleep(2 * time.Second)

	// Create a complex order scenario
	order := engine.NewLimitOrder(
		"integration_test_order",
		engine.Buy,
		fpdecimal.FromInt(150),
		fpdecimal.FromInt(25),
		engine.GTC,
		"",
		"integration_user",
	)

	// Save to KvDB
	err = kvdb.PutOrder(order)
	require.NoError(t, err)

	// Create trade
	trade := &engine.TradeOrder{
		OrderID:  order.ID(),
		UserId:   order.UserID(),
		Role:     engine.TAKER,
		Price:    order.Price(),
		IsQuote:  order.IsQuote(),
		Quantity: order.Quantity().Div(fpdecimal.FromInt(2)),
	}

	err = kvdb.PutTradeOrder(trade)
	require.NoError(t, err)

	// Create receipt
	receipt := &orderbook.Receipt{
		UserID:    order.UserID(),
		OrderID:   order.ID(),
		Trades:    []*engine.TradeOrder{trade},
		FilledQty: trade.Quantity,
	}

	err = kvdb.PutReceipt(receipt)
	require.NoError(t, err)

	// Publish events
	err = publishEventsToRabbitMQ(amqpConn, []*engine.Order{order}, []*engine.TradeOrder{trade}, []*orderbook.Receipt{receipt})
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(3 * time.Second)

	// Verify end-to-end data consistency
	orderRepo := pgdb.OrderRepository()
	tradeRepo := pgdb.TradeRepository()
	receiptRepo := pgdb.ReceiptRepository()

	// Verify order
	dbOrder, err := orderRepo.GetByID(ctx, order.ID())
	require.NoError(t, err)
	assert.Equal(t, order.ID(), (dbOrder).ID)

	// Verify trade
	dbTrades, err := tradeRepo.GetByOrderID(ctx, order.ID())
	require.NoError(t, err)
	require.Len(t, dbTrades, 1)
	assert.Equal(t, trade.OrderID, dbTrades[0].OrderID)

	// Verify receipt
	dbReceipts, err := receiptRepo.GetByOrderID(ctx, order.ID())
	require.NoError(t, err)
	require.Len(t, dbReceipts, 1)
	assert.Equal(t, receipt.OrderID, dbReceipts[0].OrderID)

	t.Log("✓ Integration test completed successfully!")
}
