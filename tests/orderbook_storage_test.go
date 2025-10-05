package tests

import (
	"testing"

	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/EggysOnCode/anomi/storage"
	"github.com/nikolaydubina/fpdecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKvDB_OrderOperations(t *testing.T) {
	db, err := storage.NewDB("test_orders", nil)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Put and Get Order", func(t *testing.T) {
		// Create a limit order
		order := engine.NewLimitOrder(
			"order_1",
			engine.Buy,
			fpdecimal.FromInt(100), // price
			fpdecimal.FromInt(10),  // quantity
			engine.GTC,
			"", // oco
			"user_1",
		)

		// Store the order
		err := db.PutOrder(order)
		require.NoError(t, err)

		// Retrieve the order
		retrievedOrder, err := db.GetOrder("order_1")
		require.NoError(t, err)
		require.NotNil(t, retrievedOrder)

		// Verify the order data
		assert.Equal(t, order.ID(), retrievedOrder.ID())
		assert.Equal(t, order.Side(), retrievedOrder.Side())
		assert.Equal(t, order.Quantity(), retrievedOrder.Quantity())
		assert.Equal(t, order.Price(), retrievedOrder.Price())
		assert.Equal(t, order.ToSimple().UserId, retrievedOrder.ToSimple().UserId)
	})

	t.Run("Put and Get Market Order", func(t *testing.T) {
		// Create a market order
		order := engine.NewMarketOrder(
			"order_2",
			engine.Sell,
			fpdecimal.FromInt(5), // quantity
			"user_2",
		)

		// Store the order
		err := db.PutOrder(order)
		require.NoError(t, err)

		// Retrieve the order
		retrievedOrder, err := db.GetOrder("order_2")
		require.NoError(t, err)
		require.NotNil(t, retrievedOrder)

		// Verify the order data
		assert.Equal(t, order.ID(), retrievedOrder.ID())
		assert.Equal(t, order.Side(), retrievedOrder.Side())
		assert.Equal(t, order.Quantity(), retrievedOrder.Quantity())
		assert.True(t, retrievedOrder.IsMarketOrder())
	})

	t.Run("Put and Get Stop Limit Order", func(t *testing.T) {
		// Create a stop limit order
		order := engine.NewStopLimitOrder(
			"order_3",
			engine.Buy,
			fpdecimal.FromInt(95),  // stop price
			fpdecimal.FromInt(100), // limit price
			fpdecimal.FromInt(8),   // quantity
			"",                     // oco
			"user_3",
		)

		// Store the order
		err := db.PutOrder(order)
		require.NoError(t, err)

		// Retrieve the order
		retrievedOrder, err := db.GetOrder("order_3")
		require.NoError(t, err)
		require.NotNil(t, retrievedOrder)

		// Verify the order data
		assert.Equal(t, order.ID(), retrievedOrder.ID())
		assert.Equal(t, order.Side(), retrievedOrder.Side())
		assert.Equal(t, order.Quantity(), retrievedOrder.Quantity())
		assert.True(t, retrievedOrder.IsStopOrder())
	})

	t.Run("Get Non-existent Order", func(t *testing.T) {
		order, err := db.GetOrder("non_existent")
		assert.Error(t, err)
		assert.Nil(t, order)
	})

	t.Run("Delete Order", func(t *testing.T) {
		// Create and store an order
		order := engine.NewLimitOrder(
			"order_4",
			engine.Sell,
			fpdecimal.FromInt(200),
			fpdecimal.FromInt(3),
			engine.GTC,
			"",
			"user_4",
		)

		err := db.PutOrder(order)
		require.NoError(t, err)

		// Verify it exists
		retrievedOrder, err := db.GetOrder("order_4")
		require.NoError(t, err)
		require.NotNil(t, retrievedOrder)

		// Delete the order
		err = db.DeleteOrder("order_4")
		require.NoError(t, err)

		// Verify it's deleted
		deletedOrder, err := db.GetOrder("order_4")
		assert.Error(t, err)
		assert.Nil(t, deletedOrder)
	})

	t.Run("Update Order", func(t *testing.T) {
		// Create initial order
		order := engine.NewLimitOrder(
			"order_5",
			engine.Buy,
			fpdecimal.FromInt(150),
			fpdecimal.FromInt(7),
			engine.GTC,
			"",
			"user_5",
		)

		err := db.PutOrder(order)
		require.NoError(t, err)

		// Cancel the order (simulate update)
		order.Cancel()
		err = db.PutOrder(order)
		require.NoError(t, err)

		// Retrieve and verify it's canceled
		retrievedOrder, err := db.GetOrder("order_5")
		require.NoError(t, err)
		assert.True(t, retrievedOrder.IsCanceled())
	})
}

func TestKvDB_TradeOrderOperations(t *testing.T) {
	db, err := storage.NewDB("test_trades", nil)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Put and Get Trade Order", func(t *testing.T) {
		// Create a trade order
		tradeOrder := &engine.TradeOrder{
			OrderID:  "trade_1",
			UserId:   "user_1",
			Role:     engine.MAKER,
			Price:    fpdecimal.FromInt(100),
			IsQuote:  false,
			Quantity: fpdecimal.FromInt(5),
		}

		// Store the trade order
		err := db.PutTradeOrder(tradeOrder)
		require.NoError(t, err)

		// Retrieve the trade order
		retrievedTrade, err := db.GetTradeOrder("trade_1")
		require.NoError(t, err)
		require.NotNil(t, retrievedTrade)

		// Verify the trade order data
		assert.Equal(t, tradeOrder.OrderID, retrievedTrade.OrderID)
		assert.Equal(t, tradeOrder.UserId, retrievedTrade.UserId)
		assert.Equal(t, tradeOrder.Role, retrievedTrade.Role)
		assert.Equal(t, tradeOrder.Price, retrievedTrade.Price)
		assert.Equal(t, tradeOrder.IsQuote, retrievedTrade.IsQuote)
		assert.Equal(t, tradeOrder.Quantity, retrievedTrade.Quantity)
	})

	t.Run("Put and Get Taker Trade Order", func(t *testing.T) {
		// Create a taker trade order
		tradeOrder := &engine.TradeOrder{
			OrderID:  "trade_2",
			UserId:   "user_2",
			Role:     engine.TAKER,
			Price:    fpdecimal.FromInt(99),
			IsQuote:  true,
			Quantity: fpdecimal.FromInt(3),
		}

		// Store the trade order
		err := db.PutTradeOrder(tradeOrder)
		require.NoError(t, err)

		// Retrieve the trade order
		retrievedTrade, err := db.GetTradeOrder("trade_2")
		require.NoError(t, err)
		require.NotNil(t, retrievedTrade)

		// Verify the trade order data
		assert.Equal(t, tradeOrder.OrderID, retrievedTrade.OrderID)
		assert.Equal(t, tradeOrder.Role, retrievedTrade.Role)
		assert.True(t, retrievedTrade.IsQuote)
	})

	t.Run("Get Non-existent Trade Order", func(t *testing.T) {
		trade, err := db.GetTradeOrder("non_existent_trade")
		assert.Error(t, err)
		assert.Nil(t, trade)
	})

	t.Run("Delete Trade Order", func(t *testing.T) {
		// Create and store a trade order
		tradeOrder := &engine.TradeOrder{
			OrderID:  "trade_3",
			UserId:   "user_3",
			Role:     engine.MAKER,
			Price:    fpdecimal.FromInt(101),
			IsQuote:  false,
			Quantity: fpdecimal.FromInt(2),
		}

		err := db.PutTradeOrder(tradeOrder)
		require.NoError(t, err)

		// Verify it exists
		retrievedTrade, err := db.GetTradeOrder("trade_3")
		require.NoError(t, err)
		require.NotNil(t, retrievedTrade)

		// Delete the trade order
		err = db.DeleteTradeOrder("trade_3")
		require.NoError(t, err)

		// Verify it's deleted
		deletedTrade, err := db.GetTradeOrder("trade_3")
		assert.Error(t, err)
		assert.Nil(t, deletedTrade)
	})
}

func TestKvDB_ReceiptOperations(t *testing.T) {
	db, err := storage.NewDB("test_receipts", nil)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Put and Get Receipt", func(t *testing.T) {
		// Create trade orders for the receipt
		trade1 := &engine.TradeOrder{
			OrderID:  "trade_1",
			UserId:   "user_1",
			Role:     engine.MAKER,
			Price:    fpdecimal.FromInt(100),
			IsQuote:  false,
			Quantity: fpdecimal.FromInt(3),
		}

		trade2 := &engine.TradeOrder{
			OrderID:  "trade_2",
			UserId:   "user_2",
			Role:     engine.TAKER,
			Price:    fpdecimal.FromInt(100),
			IsQuote:  false,
			Quantity: fpdecimal.FromInt(2),
		}

		// Create a receipt
		receipt := &orderbook.Receipt{
			UserID:    "user_1",
			OrderID:   "order_1",
			Trades:    []*engine.TradeOrder{trade1, trade2},
			FilledQty: fpdecimal.FromInt(5),
		}

		// Store the receipt
		err := db.PutReceipt(receipt)
		require.NoError(t, err)

		// Retrieve the receipt
		retrievedReceipt, err := db.GetReceipt("order_1")
		require.NoError(t, err)
		require.NotNil(t, retrievedReceipt)

		// Verify the receipt data
		assert.Equal(t, receipt.UserID, retrievedReceipt.UserID)
		assert.Equal(t, receipt.OrderID, retrievedReceipt.OrderID)
		assert.Equal(t, receipt.FilledQty, retrievedReceipt.FilledQty)
		assert.Len(t, retrievedReceipt.Trades, 2)
		assert.Equal(t, trade1.OrderID, retrievedReceipt.Trades[0].OrderID)
		assert.Equal(t, trade2.OrderID, retrievedReceipt.Trades[1].OrderID)
	})

	t.Run("Put and Get Receipt with Partial Fill", func(t *testing.T) {
		// Create a receipt with partial fill
		receipt := &orderbook.Receipt{
			UserID:    "user_2",
			OrderID:   "order_2",
			Trades:    []*engine.TradeOrder{},
			FilledQty: fpdecimal.FromInt(2), // partial fill
		}

		// Store the receipt
		err := db.PutReceipt(receipt)
		require.NoError(t, err)

		// Retrieve the receipt
		retrievedReceipt, err := db.GetReceipt("order_2")
		require.NoError(t, err)
		require.NotNil(t, retrievedReceipt)

		// Verify the receipt data
		assert.Equal(t, receipt.UserID, retrievedReceipt.UserID)
		assert.Equal(t, receipt.OrderID, retrievedReceipt.OrderID)
		assert.Equal(t, receipt.FilledQty, retrievedReceipt.FilledQty)
		assert.Len(t, retrievedReceipt.Trades, 0)
	})

	t.Run("Get Non-existent Receipt", func(t *testing.T) {
		receipt, err := db.GetReceipt("non_existent_receipt")
		assert.Error(t, err)
		assert.Nil(t, receipt)
	})

	t.Run("Delete Receipt", func(t *testing.T) {
		// Create and store a receipt
		receipt := &orderbook.Receipt{
			UserID:    "user_3",
			OrderID:   "order_3",
			Trades:    []*engine.TradeOrder{},
			FilledQty: fpdecimal.FromInt(1),
		}

		err := db.PutReceipt(receipt)
		require.NoError(t, err)

		// Verify it exists
		retrievedReceipt, err := db.GetReceipt("order_3")
		require.NoError(t, err)
		require.NotNil(t, retrievedReceipt)

		// Delete the receipt
		err = db.DeleteReceipt("order_3")
		require.NoError(t, err)

		// Verify it's deleted
		deletedReceipt, err := db.GetReceipt("order_3")
		assert.Error(t, err)
		assert.Nil(t, deletedReceipt)
	})
}

func TestKvDB_DataIntegrity(t *testing.T) {
	db, err := storage.NewDB("test_integrity", nil)
	require.NoError(t, err)
	defer db.Close()

	t.Run("Decimal Precision Preservation", func(t *testing.T) {
		// Test with decimal values that might lose precision
		price, err := fpdecimal.FromString("123.456789")
		require.NoError(t, err)
		quantity, err := fpdecimal.FromString("0.001")
		require.NoError(t, err)
		order := engine.NewLimitOrder(
			"precision_test",
			engine.Buy,
			quantity,
			price,
			engine.GTC,
			"",
			"user_precision",
		)

		err = db.PutOrder(order)
		require.NoError(t, err)

		retrievedOrder, err := db.GetOrder("precision_test")
		require.NoError(t, err)

		// Verify precision is preserved
		assert.Equal(t, order.Price().String(), retrievedOrder.Price().String())
		assert.Equal(t, order.Quantity().String(), retrievedOrder.Quantity().String())
	})

	t.Run("Complex Trade Order with All Fields", func(t *testing.T) {
		price, err := fpdecimal.FromString("999.999999")
		require.NoError(t, err)
		quantity, err := fpdecimal.FromString("0.123456")
		require.NoError(t, err)
		tradeOrder := &engine.TradeOrder{
			OrderID:  "complex_trade",
			UserId:   "complex_user",
			Role:     engine.MAKER,
			Price:    price,
			IsQuote:  true,
			Quantity: quantity,
		}

		err = db.PutTradeOrder(tradeOrder)
		require.NoError(t, err)

		retrievedTrade, err := db.GetTradeOrder("complex_trade")
		require.NoError(t, err)

		// Verify all fields are preserved
		assert.Equal(t, tradeOrder.OrderID, retrievedTrade.OrderID)
		assert.Equal(t, tradeOrder.UserId, retrievedTrade.UserId)
		assert.Equal(t, tradeOrder.Role, retrievedTrade.Role)
		assert.Equal(t, tradeOrder.Price.String(), retrievedTrade.Price.String())
		assert.Equal(t, tradeOrder.IsQuote, retrievedTrade.IsQuote)
		assert.Equal(t, tradeOrder.Quantity.String(), retrievedTrade.Quantity.String())
	})

	t.Run("Receipt with Multiple Trades", func(t *testing.T) {
		// Create multiple trades
		trades := []*engine.TradeOrder{
			{
				OrderID:  "trade_1",
				UserId:   "user_1",
				Role:     engine.MAKER,
				Price:    fpdecimal.FromInt(100),
				IsQuote:  false,
				Quantity: fpdecimal.FromInt(2),
			},
			{
				OrderID:  "trade_2",
				UserId:   "user_2",
				Role:     engine.TAKER,
				Price:    fpdecimal.FromInt(101),
				IsQuote:  false,
				Quantity: fpdecimal.FromInt(1),
			},
			{
				OrderID:  "trade_3",
				UserId:   "user_3",
				Role:     engine.MAKER,
				Price:    fpdecimal.FromInt(102),
				IsQuote:  true,
				Quantity: fpdecimal.FromInt(3),
			},
		}

		receipt := &orderbook.Receipt{
			UserID:    "receipt_user",
			OrderID:   "multi_trade_order",
			Trades:    trades,
			FilledQty: fpdecimal.FromInt(6),
		}

		err := db.PutReceipt(receipt)
		require.NoError(t, err)

		retrievedReceipt, err := db.GetReceipt("multi_trade_order")
		require.NoError(t, err)

		// Verify all trades are preserved
		assert.Len(t, retrievedReceipt.Trades, 3)
		for i, expectedTrade := range trades {
			actualTrade := retrievedReceipt.Trades[i]
			assert.Equal(t, expectedTrade.OrderID, actualTrade.OrderID)
			assert.Equal(t, expectedTrade.UserId, actualTrade.UserId)
			assert.Equal(t, expectedTrade.Role, actualTrade.Role)
			assert.Equal(t, expectedTrade.Price.String(), actualTrade.Price.String())
			assert.Equal(t, expectedTrade.IsQuote, actualTrade.IsQuote)
			assert.Equal(t, expectedTrade.Quantity.String(), actualTrade.Quantity.String())
		}
	})
}

func TestKvDB_ErrorHandling(t *testing.T) {
	db, err := storage.NewDB("test_errors", nil)
	require.NoError(t, err)

	t.Run("Database Operations After Close", func(t *testing.T) {
		// Close the database
		err := db.Close()
		require.NoError(t, err)

		// Try to perform operations after close
		order := engine.NewLimitOrder("test", engine.Buy, fpdecimal.FromInt(100), fpdecimal.FromInt(1), engine.GTC, "", "user")

		// Use defer/recover to catch panics and convert to errors
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Panic is expected, test passes
				}
			}()
			db.PutOrder(order)
			t.Error("Expected panic but got no error")
		}()

		func() {
			defer func() {
				if r := recover(); r != nil {
					// Panic is expected, test passes
				}
			}()
			db.GetOrder("test")
			t.Error("Expected panic but got no error")
		}()

		func() {
			defer func() {
				if r := recover(); r != nil {
					// Panic is expected, test passes
				}
			}()
			db.DeleteOrder("test")
			t.Error("Expected panic but got no error")
		}()
	})
}
