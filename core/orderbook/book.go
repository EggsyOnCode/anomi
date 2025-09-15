package orderbook

import (
	"fmt"

	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/ethereum/go-ethereum/log"
	"github.com/nikolaydubina/fpdecimal"
)

const (
	BCSIZE int = 50000
)

type OrderBook struct {
	*engine.OrderBook
	Base  Asset
	Quote Asset
	bc    *BuyerCache
}

func NewOrderBook(base, quote string, bcSize int) (*OrderBook, error) {
	if !IsAllowedAsset(base) || !IsAllowedAsset(quote) {
		return nil, fmt.Errorf("unsupported asset")
	}

	var size int
	if bcSize == 0 {
		size = BCSIZE
	} else {
		size = bcSize
	}
	bc, err := NewBuyerCache(size)
	if err != nil {
		return nil, err
	}

	return &OrderBook{
		engine.NewOrderBook(),
		Asset(base),
		Asset(quote),
		bc,
	}, nil
}

func (o *OrderBook) AddOrder(order *engine.Order) (*engine.Done, []*Receipt, error) {
	done, err := o.Process(order)
	if err != nil {
		return done, nil, err
	}

	var receipts []*Receipt

	// -------------------
	// Case 0: BUY MARKET
	// -------------------
	if order.Side() == engine.Buy && order.IsMarketOrder() {
		filled := done.Processed
		left := order.OriginalQty().Sub(filled)

		// Always generate a receipt for the buyer
		receipts = append(receipts, &Receipt{
			UserID:    order.ToSimple().UserId,
			OrderID:   order.ID(),
			Trades:    done.Trades,
			FilledQty: filled,
		})

		// Monitor low liquidity if partially filled
		if left.GreaterThan(fpdecimal.Zero) {
			//TODO: setup a proper logger for anomi
			log.Warn("Market BUY partially filled due to low liquidity",
				"user", order.ToSimple().UserId,
				"orderID", order.ID(),
				"requestedQty", order.OriginalQty(),
				"filledQty", filled,
				"leftQty", left,
			)
		}

		return done, receipts, nil
	}

	// -------------------
	// Case 1: BUY LIMIT
	// -------------------
	if order.Side() == engine.Buy && order.IsLimitOrder() {
		pos, exists := o.bc.Get(order.ID())
		if !exists {
			pos = &BuyerPos{
				Order:  order,
				Trades: []*engine.TradeOrder{},
				Left:   order.OriginalQty(),
			}
		}

		// Update pos with trades
		for _, trade := range done.Trades {
			if trade.OrderID != order.ID() { // matched against a seller
				pos.Trades = append(pos.Trades, trade)
				pos.Left = pos.Left.Sub(trade.Quantity)
			}
		}

		if pos.Left.GreaterThan(fpdecimal.Zero) {
			o.bc.Set(order.ID(), pos) // still active
		} else {
			o.bc.Remove(order.ID())
			receipts = append(receipts, &Receipt{
				UserID:    order.ToSimple().UserId,
				OrderID:   order.ID(),
				Trades:    pos.Trades,
				FilledQty: order.OriginalQty(),
			})
		}
	}

	// -------------------
	// Case 2: SELL (ASK)
	// -------------------
	if order.Side() == engine.Sell {
		for _, trade := range done.Trades {
			if makerPos, exists := o.bc.Get(trade.OrderID); exists {
				makerPos.Trades = append(makerPos.Trades, trade)
				makerPos.Left = makerPos.Left.Sub(trade.Quantity)

				if makerPos.Left.GreaterThan(fpdecimal.Zero) {
					o.bc.Set(trade.OrderID, makerPos)
				} else {
					o.bc.Remove(trade.OrderID)
					receipts = append(receipts, &Receipt{
						UserID:    makerPos.Order.ToSimple().UserId,
						OrderID:   makerPos.Order.ID(),
						Trades:    makerPos.Trades,
						FilledQty: makerPos.Order.OriginalQty(),
					})
				}
			}
		}
	}

	return done, receipts, nil
}

func (o *OrderBook) RemoveOrder(id string) (*engine.Order, error) {
	// remove from cache
	pos, ok := o.bc.Get(id)
	if ok {
		// partial fill detected
		// pos.left != pos.order.originalQnt
		if !pos.Left.Equal(pos.Order.OriginalQty()) {
			// we can't allow removal since settlement will be difficult
			return nil, fmt.Errorf("can't remove order now since its partially filled, settlement at this stage would be difficult")
		}
	}
	o.bc.Remove(id)
	return o.CancelOrder(id), nil
}

func (o *OrderBook) Symbol() string {
	return fmt.Sprintf("%s/%s", o.Base, o.Quote)
}
