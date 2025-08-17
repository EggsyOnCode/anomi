package orderbook

import (
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/nikolaydubina/fpdecimal"
)

type Receipt struct {
	UserID    string
	OrderID   string
	Trades    []*engine.TradeOrder
	FilledQty fpdecimal.Decimal
}
