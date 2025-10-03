package models

import (
	"fmt"
	"time"

	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/google/uuid"
	"github.com/nikolaydubina/fpdecimal"
	"github.com/uptrace/bun"
)

// Order model for PostgreSQL
type Order struct {
	bun.BaseModel `bun:"table:orders,alias:o"`

	ID          string    `bun:"id,pk" json:"id"`
	OrderType   string    `bun:"order_type,notnull" json:"orderType"` // engine.OrderType as string
	UserID      string    `bun:"user_id,notnull" json:"userID"`
	Side        int       `bun:"side,notnull" json:"side"` // engine.Side as int
	IsQuote     bool      `bun:"is_quote,notnull,default:false" json:"isQuote"`
	Quantity    string    `bun:"quantity,notnull" json:"quantity"` // fpdecimal.Decimal as string
	OriginalQty string    `bun:"original_qty,notnull" json:"originalQty"`
	Price       string    `bun:"price,notnull" json:"price"`
	Canceled    bool      `bun:"canceled,notnull,default:false" json:"canceled"`
	Role        string    `bun:"role" json:"role"` // engine.Role as string
	Stop        string    `bun:"stop" json:"stop"` // fpdecimal.Decimal as string
	TIF         string    `bun:"tif" json:"tif"`   // engine.TIF as string
	OCO         string    `bun:"oco" json:"oco"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:now()" json:"createdAt"`
	UpdatedAt   time.Time `bun:"updated_at,notnull,default:now()" json:"updatedAt"`
}

// Trade model for PostgreSQL (with separate ID as PK)
type Trade struct {
	bun.BaseModel `bun:"table:trades,alias:t"`

	ID        string    `bun:"id,pk" json:"id"` // Separate ID for trade
	OrderID   string    `bun:"order_id,notnull" json:"orderID"`
	UserID    string    `bun:"user_id,notnull" json:"userID"`
	Role      string    `bun:"role,notnull" json:"role"`   // engine.Role as string
	Price     string    `bun:"price,notnull" json:"price"` // fpdecimal.Decimal as string
	IsQuote   bool      `bun:"is_quote,notnull,default:false" json:"isQuote"`
	Quantity  string    `bun:"quantity,notnull" json:"quantity"` // fpdecimal.Decimal as string
	CreatedAt time.Time `bun:"created_at,notnull,default:now()" json:"createdAt"`
}

// Receipt model for PostgreSQL (one receipt per trade order)
type Receipt struct {
	bun.BaseModel `bun:"table:receipts,alias:r"`

	ID        string    `bun:"id,pk" json:"id"` // Separate ID for receipt
	UserID    string    `bun:"user_id,notnull" json:"userID"`
	OrderID   string    `bun:"order_id,notnull" json:"orderID"`
	TradeID   string    `bun:"trade_id,notnull" json:"tradeID"`     // Reference to the specific trade
	FilledQty string    `bun:"filled_qty,notnull" json:"filledQty"` // fpdecimal.Decimal as string
	CreatedAt time.Time `bun:"created_at,notnull,default:now()" json:"createdAt"`
}

// Conversion functions from engine types to Bun models

// NewOrderFromEngine converts engine.Order to storage.Order
func NewOrderFromEngine(order *engine.Order) *Order {
	return &Order{
		ID:          order.ID(),
		OrderType:   string(order.OrderType()),
		UserID:      order.UserID(),
		Side:        int(order.Side()),
		IsQuote:     order.IsQuote(),
		Quantity:    order.Quantity().String(),
		OriginalQty: order.OriginalQty().String(),
		Price:       order.Price().String(),
		Canceled:    order.IsCanceled(),
		Role:        string(order.Role()),
		Stop:        order.StopPrice().String(),
		TIF:         string(order.TIF()),
		OCO:         order.OCO(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ToEngine converts storage.Order to engine.Order
func (o *Order) ToEngine() (*engine.Order, error) {
	// Parse decimal fields
	quantity, err := fpdecimal.FromString(o.Quantity)
	if err != nil {
		return nil, err
	}
	_, err = fpdecimal.FromString(o.OriginalQty)
	if err != nil {
		return nil, err
	}
	price, err := fpdecimal.FromString(o.Price)
	if err != nil {
		return nil, err
	}
	stop, err := fpdecimal.FromString(o.Stop)
	if err != nil {
		return nil, err
	}

	// Create engine.Order based on type
	var order *engine.Order
	switch o.OrderType {
	case "MARKET":
		if o.IsQuote {
			order = engine.NewMarketQuoteOrder(o.ID, engine.Side(o.Side), quantity, o.UserID)
		} else {
			order = engine.NewMarketOrder(o.ID, engine.Side(o.Side), quantity, o.UserID)
		}
	case "LIMIT":
		order = engine.NewLimitOrder(o.ID, engine.Side(o.Side), quantity, price, engine.TIF(o.TIF), o.OCO, o.UserID)
	case "STOP-LIMIT":
		order = engine.NewStopLimitOrder(o.ID, engine.Side(o.Side), quantity, price, stop, o.OCO, o.UserID)
	}

	if order == nil {
		return nil, fmt.Errorf("unknown order type: %s", o.OrderType)
	}

	// Set additional fields
	if o.Canceled {
		order.Cancel()
	}
	if o.Role == "MAKER" {
		order.SetMaker()
	} else {
		order.SetTaker()
	}

	return order, nil
}

// NewTradeFromEngine converts engine.TradeOrder to storage.Trade
func NewTradeFromEngine(tradeOrder *engine.TradeOrder) *Trade {
	return &Trade{
		ID:        uuid.NewString(),
		OrderID:   tradeOrder.OrderID,
		UserID:    tradeOrder.UserId,
		Role:      string(tradeOrder.Role),
		Price:     tradeOrder.Price.String(),
		IsQuote:   tradeOrder.IsQuote,
		Quantity:  tradeOrder.Quantity.String(),
		CreatedAt: time.Now(),
	}
}

// ToEngine converts storage.Trade to engine.TradeOrder
func (t *Trade) ToEngine() (*engine.TradeOrder, error) {
	price, err := fpdecimal.FromString(t.Price)
	if err != nil {
		return nil, err
	}
	quantity, err := fpdecimal.FromString(t.Quantity)
	if err != nil {
		return nil, err
	}

	return &engine.TradeOrder{
		OrderID:  t.OrderID,
		UserId:   t.UserID,
		Role:     engine.Role(t.Role),
		Price:    price,
		IsQuote:  t.IsQuote,
		Quantity: quantity,
	}, nil
}

// NewReceiptFromEngine converts orderbook.Receipt to storage.Receipt
// This creates one receipt per trade order as requested
func NewReceiptFromEngine(receipt *orderbook.Receipt, receiptID string, tradeID string) *Receipt {
	return &Receipt{
		ID:        receiptID,
		UserID:    receipt.UserID,
		OrderID:   receipt.OrderID,
		TradeID:   tradeID,
		FilledQty: receipt.FilledQty.String(),
		CreatedAt: time.Now(),
	}
}

// ToEngine converts storage.Receipt to orderbook.Receipt
func (r *Receipt) ToEngine() (*orderbook.Receipt, error) {
	filledQty, err := fpdecimal.FromString(r.FilledQty)
	if err != nil {
		return nil, err
	}

	return &orderbook.Receipt{
		UserID:    r.UserID,
		OrderID:   r.OrderID,
		Trades:    []*engine.TradeOrder{}, // Will be populated separately
		FilledQty: filledQty,
	}, nil
}

// IMP: receipt from a biz pov is a superstruct that holds lots of Trades.. But for db, we need to create one receipt per trade
// a receipt is when a user say posted a big ask for 10 BTc that will be 
// eaten by lots of bid trades.. but only one receipt will be minted to the seller
// Helper function to create multiple receipts from orderbook.Receipt
func CreateReceiptsFromEngine(receipt *orderbook.Receipt) []*Receipt {
	receipts := make([]*Receipt, len(receipt.Trades))

	for i, trade := range receipt.Trades {
		receiptID := uuid.NewString()
		tradeID := uuid.NewString()

		receipts[i] = &Receipt{
			ID:        receiptID,
			UserID:    receipt.UserID,
			OrderID:   receipt.OrderID,
			TradeID:   tradeID,
			FilledQty: trade.Quantity.String(), // Use trade quantity as filled quantity
			CreatedAt: time.Now(),
		}
	}

	return receipts
}
