package orderbook

import (
	"encoding/json"

	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/nikolaydubina/fpdecimal"
)

type Receipt struct {
	UserID    string
	OrderID   string
	Trades    []*engine.TradeOrder
	FilledQty fpdecimal.Decimal
}

func (r *Receipt) MarshalJSON() ([]byte, error) {
	customStruct := struct {
		UserID    string               `json:"userID"`
		OrderID   string               `json:"orderID"`
		Trades    []*engine.TradeOrder `json:"trades"`
		FilledQty string               `json:"filledQty"`
	}{
		UserID:    r.UserID,
		OrderID:   r.OrderID,
		Trades:    r.Trades,
		FilledQty: r.FilledQty.String(),
	}
	return json.Marshal(customStruct)
}

func (r *Receipt) UnmarshalJSON(data []byte) error {
	customStruct := struct {
		UserID    string               `json:"userID"`
		OrderID   string               `json:"orderID"`
		Trades    []*engine.TradeOrder `json:"trades"`
		FilledQty string               `json:"filledQty"`
	}{}
	err := json.Unmarshal(data, &customStruct)
	if err != nil {
		return err
	}
	
	// Set basic fields
	r.UserID = customStruct.UserID
	r.OrderID = customStruct.OrderID
	r.Trades = customStruct.Trades
	
	// Parse decimal field
	filledQty, err := fpdecimal.FromString(customStruct.FilledQty)
	if err != nil {
		return err
	}
	r.FilledQty = filledQty
	return nil
}