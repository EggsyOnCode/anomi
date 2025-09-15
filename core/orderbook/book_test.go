package orderbook

import (
	"testing"

	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/google/uuid"
	"github.com/nikolaydubina/fpdecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func d(n int) fpdecimal.Decimal { return fpdecimal.FromInt(n) }

func mkLimit(id string, side engine.Side, qty, px int) *engine.Order {
	// TIF: use zero value since we don't need IOC/FOK in these tests.
	userId := uuid.New()
	return engine.NewLimitOrder(id, side, d(qty), d(px), engine.TIF(""), "", userId.String())
}
func mkMarket(id string, side engine.Side, qty int) *engine.Order {
	userId := uuid.New()
	return engine.NewMarketOrder(id, side, d(qty), userId.String())
}

// newBook creates a new order book with a large cache for testing
func newBook(t *testing.T) *OrderBook {
	ob, err := NewOrderBook("BTC", "PKR", 0)
	require.NoError(t, err)
	return ob
}

// newBookSmallCache creates an order book with a specific cache size for LRU tests
func newBookSmallCache(t *testing.T, n int) *OrderBook {
	ob, err := NewOrderBook("BTC", "PKR", n)
	require.NoError(t, err)
	return ob
}

// Test_BuyMarket_FullFill tests a market buy order that is fully filled
func Test_BuyMarket_FullFill(t *testing.T) {
	ob := newBook(t)

	// Seed asks: 5 @ 100
	_, _, err := ob.AddOrder(mkLimit("ask1", engine.Sell, 5, 100))
	require.NoError(t, err)

	// Market buy 5 → full fill
	done, receipts, err := ob.AddOrder(mkMarket("mb1", engine.Buy, 5))
	require.NoError(t, err)

	require.NotNil(t, done)
	assert.True(t, done.Processed.Equal(d(5)))
	require.Len(t, receipts, 1)
	assert.Equal(t, "mb1", receipts[0].OrderID)
	assert.True(t, receipts[0].FilledQty.Equal(d(5)))
}

// Test_BuyMarket_PartialFill tests a market buy order that is partially filled
func Test_BuyMarket_PartialFill(t *testing.T) {
	ob := newBook(t)

	// Only 3 available
	_, _, _ = ob.AddOrder(mkLimit("ask1", engine.Sell, 3, 100))

	// Market buy 10 → partial fill 3, left 7
	done, receipts, err := ob.AddOrder(mkMarket("mb2", engine.Buy, 10))
	require.NoError(t, err)

	require.NotNil(t, done)
	assert.True(t, done.Processed.Equal(d(3)))
	require.Len(t, receipts, 1) // you always emit a receipt for BUY-MARKET
	assert.True(t, receipts[0].FilledQty.Equal(d(3)))
}

// Test_BuyLimit_NoLiquidity_Cached tests a limit buy order that is not filled at all and is cached
func Test_BuyLimit_NoLiquidity_Cached(t *testing.T) {
	ob := newBook(t)

	// Bid 10 @ 100, no asks
	done, receipts, err := ob.AddOrder(mkLimit("bid1", engine.Buy, 10, 100))
	require.NoError(t, err)

	// No trade, order stored, cached
	require.NotNil(t, done)
	assert.True(t, done.Stored)
	assert.Empty(t, receipts)

	// Internals: buyer cache should have Left == 10
	pos, ok := ob.bc.Get("bid1")
	require.True(t, ok)
	assert.True(t, pos.Left.Equal(d(10)))
}

// Test_BuyLimit_PartialFill_ThenComplete tests a limit buy order that is partially filled, then completed
func Test_BuyLimit_PartialFill_ThenComplete(t *testing.T) {
	ob := newBook(t)

	// Place bid 10 @ 100
	_, _, _ = ob.AddOrder(mkLimit("bid2", engine.Buy, 10, 100))

	// Sell 3 @ 100 → partial fill
	_, receipts, err := ob.AddOrder(mkLimit("askA", engine.Sell, 3, 100))
	require.NoError(t, err)
	assert.Empty(t, receipts)

	pos, ok := ob.bc.Get("bid2")
	require.True(t, ok)
	assert.True(t, pos.Left.Equal(d(7)))

	// Sell 7 @ 100 → completes buyer
	_, receipts, err = ob.AddOrder(mkLimit("askB", engine.Sell, 7, 100))
	require.NoError(t, err)

	require.Len(t, receipts, 1)
	assert.Equal(t, "bid2", receipts[0].OrderID)
	assert.True(t, receipts[0].FilledQty.Equal(d(10)))

	// Evicted from cache
	_, ok = ob.bc.Get("bid2")
	assert.False(t, ok)
}

// Test_SellTaker_HitsMultipleBuyers tests a single sell order filling multiple resting buy orders
func Test_SellTaker_HitsMultipleBuyers(t *testing.T) {
	ob := newBook(t)

	// Two buyers resting
	_, _, _ = ob.AddOrder(mkLimit("bidA", engine.Buy, 3, 100))
	_, _, _ = ob.AddOrder(mkLimit("bidB", engine.Buy, 2, 100))

	// One sell 5 @ 100 should complete both
	_, receipts, err := ob.AddOrder(mkLimit("askCombo", engine.Sell, 5, 100))
	require.NoError(t, err)

	// Two receipts (for buyers) in any order
	require.Len(t, receipts, 2)
	var ids = map[string]bool{receipts[0].OrderID: true, receipts[1].OrderID: true}
	assert.True(t, ids["bidA"])
	assert.True(t, ids["bidB"])
}

// Test_RemoveOrder_FullFill_EvictsBuyerCache tests removing a fully filled order
func Test_RemoveOrder_FullFill_EvictsBuyerCache(t *testing.T) {
	ob := newBook(t)

	// Resting buyer
	_, _, _ = ob.AddOrder(mkLimit("bidX", engine.Buy, 4, 100))

	// Cancel -> must evict cache
	ord, err := ob.RemoveOrder("bidX")
	require.NotNil(t, ord)
	require.NoError(t, err)

	_, ok := ob.bc.Get("bidX")
	assert.False(t, ok)
}

// Test_RemoveOrder_PartialFill_Fails tests that a partially filled order cannot be removed
func Test_RemoveOrder_PartialFill_Fails(t *testing.T) {
	ob := newBook(t)

	// Place a resting bid for 10
	bidID := "partially-filled-bid"
	_, _, err := ob.AddOrder(mkLimit(bidID, engine.Buy, 10, 100))
	require.NoError(t, err)

	// Partially fill the bid with a sell for 5
	_, _, err = ob.AddOrder(mkLimit("partial-sell", engine.Sell, 5, 100))
	require.NoError(t, err)

	// Check the buyer cache state: should be partially filled
	pos, ok := ob.bc.Get(bidID)
	require.True(t, ok)
	assert.True(t, pos.Left.Equal(d(5)))

	// Attempt to remove the partially filled order and assert that it fails
	_, err = ob.RemoveOrder(bidID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can't remove order now since its partially filled")

	// Assert that the order is still present in both the engine and the buyer cache
	_, ok = ob.bc.Get(bidID)
	assert.True(t, ok, "Order should still be in BuyerCache")

	orderInEngine := ob.OrderBook.GetOrder(bidID)
	assert.NotNil(t, orderInEngine, "Order should still be in the engine's orderbook")
	assert.Equal(t, bidID, orderInEngine.ID(), "Order ID should match")
}

// Test_BuyerCache_LRU_EvictsOldest tests the LRU cache behavior
func Test_BuyerCache_LRU_EvictsOldest(t *testing.T) {
	ob := newBookSmallCache(t, 2)

	_, _, _ = ob.AddOrder(mkLimit("b1", engine.Buy, 1, 10))
	_, _, _ = ob.AddOrder(mkLimit("b2", engine.Buy, 1, 10))
	_, _, _ = ob.AddOrder(mkLimit("b3", engine.Buy, 1, 10)) // should evict b1

	_, ok1 := ob.bc.Get("b1")
	_, ok2 := ob.bc.Get("b2")
	_, ok3 := ob.bc.Get("b3")

	assert.False(t, ok1)
	assert.True(t, ok2)
	assert.True(t, ok3)
}
