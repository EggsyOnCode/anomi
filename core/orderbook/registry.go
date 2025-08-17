package orderbook

import (
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	lru "github.com/hashicorp/golang-lru"
	"github.com/nikolaydubina/fpdecimal"
)

// BuyerPos, orignial order amt, array of trades that have filled the bid, left
type BuyerPos struct {
	Order  *engine.Order        // Original limit order
	Trades []*engine.TradeOrder // Trades that have filled this order
	Left   fpdecimal.Decimal    // Remaining quantity to fill
}

type BuyerCache struct {
	cache *lru.Cache // LRU cache keyed by OrderID (orderID -> BuyerPos)
}

func NewBuyerCache(size int) (*BuyerCache, error) {
	c, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &BuyerCache{cache: c}, nil
}

func (b *BuyerCache) Get(orderID string) (*BuyerPos, bool) {
	pos, ok := b.cache.Get(orderID)
	if !ok {
		return nil, false
	}
	return pos.(*BuyerPos), true
}

func (b *BuyerCache) Set(orderID string, pos *BuyerPos) {
	b.cache.Add(orderID, pos)
}

func (b *BuyerCache) Remove(orderID string) {
	b.cache.Remove(orderID)
}
