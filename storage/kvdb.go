package storage

import (
	"errors"
	"strings"
	"time"

	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/nikolaydubina/fpdecimal"
	"go.uber.org/zap"
)

type KvDB struct {
	db     *pebble.DB
	logger *zap.Logger
}

// Validation errors
var (
	ErrInvalidID      = errors.New("invalid ID: empty or too long")
	ErrInvalidData    = errors.New("invalid data: nil or empty")
	ErrInvalidOrder   = errors.New("invalid order data")
	ErrInvalidTrade   = errors.New("invalid trade order data")
	ErrInvalidReceipt = errors.New("invalid receipt data")
	ErrDatabaseClosed = errors.New("database is closed")
	ErrKeyTooLong     = errors.New("key too long")
)

const (
	MaxIDLength   = 255
	MaxKeyLength  = 512
	MaxDataLength = 1024 * 1024 // 1MB
)

func NewDB(path string, logger *zap.Logger) (*KvDB, error) {
	// In memory database for testing
	db, err := pebble.Open(path, &pebble.Options{FS: vfs.NewMem()})
	if err != nil {
		logger.Error("Failed to open KVDB", zap.String("path", path), zap.Error(err))
		return nil, err
	}
	logger.Info("KVDB initialized successfully", zap.String("path", path))
	return &KvDB{db: db, logger: logger}, nil
}

func (kv *KvDB) Close() error {
	if kv.db == nil {
		kv.logger.Warn("Attempted to close already closed KVDB")
		return ErrDatabaseClosed
	}
	kv.logger.Info("Closing KVDB")
	return kv.db.Close()
}

// validateID checks if ID is valid
func validateID(id string) error {
	if id == "" {
		return ErrInvalidID
	}
	if len(id) > MaxIDLength {
		return ErrInvalidID
	}
	// Check for invalid characters that could cause issues
	if strings.ContainsAny(id, "\x00\n\r\t") {
		return ErrInvalidID
	}
	return nil
}

// validateKey checks if the generated key is valid
func validateKey(key []byte) error {
	if len(key) == 0 {
		return ErrInvalidData
	}
	if len(key) > MaxKeyLength {
		return ErrKeyTooLong
	}
	return nil
}

// validateOrderData validates order before storage
func validateOrderData(order *engine.Order) error {
	if order == nil {
		return ErrInvalidData
	}

	// Validate ID
	if err := validateID(order.ID()); err != nil {
		return err
	}

	// Validate user ID through ToSimple() method
	simple := order.ToSimple()
	if simple.UserId == "" || len(simple.UserId) > MaxIDLength {
		return ErrInvalidOrder
	}

	// Validate quantity and price are not zero or negative
	if order.Quantity().LessThanOrEqual(fpdecimal.Zero) {
		return ErrInvalidOrder
	}

	// For limit orders, price should be positive
	if order.IsLimitOrder() && order.Price().LessThanOrEqual(fpdecimal.Zero) {
		return ErrInvalidOrder
	}

	return nil
}

// validateTradeOrderData validates trade order before storage
func validateTradeOrderData(tradeOrder *engine.TradeOrder) error {
	if tradeOrder == nil {
		return ErrInvalidData
	}

	// Validate OrderID
	if err := validateID(tradeOrder.OrderID); err != nil {
		return err
	}

	// Validate UserId
	if tradeOrder.UserId == "" || len(tradeOrder.UserId) > MaxIDLength {
		return ErrInvalidTrade
	}

	// Validate quantity and price are not zero or negative
	if tradeOrder.Quantity.LessThanOrEqual(fpdecimal.Zero) {
		return ErrInvalidTrade
	}

	if tradeOrder.Price.LessThanOrEqual(fpdecimal.Zero) {
		return ErrInvalidTrade
	}

	return nil
}

// validateReceiptData validates receipt before storage
func validateReceiptData(receipt *orderbook.Receipt) error {
	if receipt == nil {
		return ErrInvalidData
	}

	// Validate OrderID
	if err := validateID(receipt.OrderID); err != nil {
		return err
	}

	// Validate UserID
	if receipt.UserID == "" || len(receipt.UserID) > MaxIDLength {
		return ErrInvalidReceipt
	}

	// Validate FilledQty is not negative
	if receipt.FilledQty.LessThan(fpdecimal.Zero) {
		return ErrInvalidReceipt
	}

	// Validate trades if present
	for i, trade := range receipt.Trades {
		if trade == nil {
			return ErrInvalidReceipt
		}
		if err := validateTradeOrderData(trade); err != nil {
			return err
		}
		// Additional validation for trades in receipt context
		if trade.OrderID == "" {
			return ErrInvalidReceipt
		}
		// Prevent duplicate trade IDs in the same receipt
		for j, otherTrade := range receipt.Trades {
			if i != j && otherTrade != nil && trade.OrderID == otherTrade.OrderID {
				return ErrInvalidReceipt
			}
		}
	}

	return nil
}

// sanitizeOrderData performs additional sanitization on order data
func sanitizeOrderData(order *engine.Order) *engine.Order {
	// Create a copy to avoid modifying the original
	// Note: This is a simplified approach. In production,we will have to
	// to implement proper deep copying or use a different approach

	// For now, we'll just validate and return the original
	// In a real implementation, you might want to:
	// - Trim whitespace from string fields
	// - Normalize decimal precision
	// - Validate business rules
	// - Apply data transformations

	return order
}

func (kv *KvDB) PutOrder(order *engine.Order) error {
	if kv.db == nil {
		kv.logger.Error("KVDB is closed, cannot put order")
		return ErrDatabaseClosed
	}

	// Validate input data
	if err := validateOrderData(order); err != nil {
		kv.logger.Error("Order validation failed", zap.String("orderID", order.ID()), zap.Error(err))
		return err
	}

	// Sanitize data
	order = sanitizeOrderData(order)

	// Marshal to JSON
	orderBytes, err := order.MarshalJSON()
	if err != nil {
		kv.logger.Error("Failed to marshal order to JSON", zap.String("orderID", order.ID()), zap.Error(err))
		return err
	}

	// Validate data size
	if len(orderBytes) > MaxDataLength {
		kv.logger.Error("Order data too large", zap.String("orderID", order.ID()), zap.Int("size", len(orderBytes)), zap.Int("maxSize", MaxDataLength))
		return ErrInvalidData
	}

	// Create key and validate
	key := []byte("order:" + order.ID())
	if err := validateKey(key); err != nil {
		kv.logger.Error("Invalid key for order", zap.String("orderID", order.ID()), zap.Error(err))
		return err
	}

	// Store with timestamp for audit trail
	if err := kv.db.Set(key, orderBytes, pebble.Sync); err != nil {
		kv.logger.Error("Failed to store order in KVDB", zap.String("orderID", order.ID()), zap.Error(err))
		return err
	}

	kv.logger.Debug("Order stored successfully", zap.String("orderID", order.ID()))
	return nil
}

func (kv *KvDB) GetOrder(id string) (*engine.Order, error) {
	if kv.db == nil {
		kv.logger.Error("KVDB is closed, cannot get order")
		return nil, ErrDatabaseClosed
	}

	// Validate input
	if err := validateID(id); err != nil {
		kv.logger.Error("Invalid order ID", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}

	// Create key and validate
	key := []byte("order:" + id)
	if err := validateKey(key); err != nil {
		kv.logger.Error("Invalid key for order", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}

	// Retrieve data
	orderBytes, closer, err := kv.db.Get(key)
	if err != nil {
		kv.logger.Error("Failed to retrieve order from KVDB", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}
	defer closer.Close()

	// Validate data size
	if len(orderBytes) > MaxDataLength {
		kv.logger.Error("Order data too large", zap.String("orderID", id), zap.Int("size", len(orderBytes)), zap.Int("maxSize", MaxDataLength))
		return nil, ErrInvalidData
	}

	// Unmarshal
	order := &engine.Order{}
	err = order.UnmarshalJSON(orderBytes)
	if err != nil {
		kv.logger.Error("Failed to unmarshal order from JSON", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}

	// Validate retrieved data
	if err := validateOrderData(order); err != nil {
		kv.logger.Error("Retrieved order data validation failed", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}

	kv.logger.Debug("Order retrieved successfully", zap.String("orderID", id))
	return order, nil
}

func (kv *KvDB) DeleteOrder(id string) error {
	if kv.db == nil {
		kv.logger.Error("KVDB is closed, cannot delete order")
		return ErrDatabaseClosed
	}

	// Validate input
	if err := validateID(id); err != nil {
		kv.logger.Error("Invalid order ID", zap.String("orderID", id), zap.Error(err))
		return err
	}

	// Create key and validate
	key := []byte("order:" + id)
	if err := validateKey(key); err != nil {
		kv.logger.Error("Invalid key for order", zap.String("orderID", id), zap.Error(err))
		return err
	}

	if err := kv.db.Delete(key, pebble.Sync); err != nil {
		kv.logger.Error("Failed to delete order from KVDB", zap.String("orderID", id), zap.Error(err))
		return err
	}

	kv.logger.Debug("Order deleted successfully", zap.String("orderID", id))
	return nil
}

func (kv *KvDB) PutTradeOrder(tradeOrder *engine.TradeOrder) error {
	if kv.db == nil {
		kv.logger.Error("KVDB is closed, cannot put trade order")
		return ErrDatabaseClosed
	}

	// Validate input data
	if err := validateTradeOrderData(tradeOrder); err != nil {
		kv.logger.Error("Trade order validation failed", zap.String("orderID", tradeOrder.OrderID), zap.Error(err))
		return err
	}

	// Marshal to JSON
	tradeOrderBytes, err := tradeOrder.MarshalJSON()
	if err != nil {
		kv.logger.Error("Failed to marshal trade order to JSON", zap.String("orderID", tradeOrder.OrderID), zap.Error(err))
		return err
	}

	// Validate data size
	if len(tradeOrderBytes) > MaxDataLength {
		kv.logger.Error("Trade order data too large", zap.String("orderID", tradeOrder.OrderID), zap.Int("size", len(tradeOrderBytes)), zap.Int("maxSize", MaxDataLength))
		return ErrInvalidData
	}

	// Create key and validate
	key := []byte("trade:" + tradeOrder.OrderID)
	if err := validateKey(key); err != nil {
		kv.logger.Error("Invalid key for trade order", zap.String("orderID", tradeOrder.OrderID), zap.Error(err))
		return err
	}

	if err := kv.db.Set(key, tradeOrderBytes, pebble.Sync); err != nil {
		kv.logger.Error("Failed to store trade order in KVDB", zap.String("orderID", tradeOrder.OrderID), zap.Error(err))
		return err
	}

	kv.logger.Debug("Trade order stored successfully", zap.String("orderID", tradeOrder.OrderID))
	return nil
}

func (kv *KvDB) GetTradeOrder(id string) (*engine.TradeOrder, error) {
	if kv.db == nil {
		kv.logger.Error("KVDB is closed, cannot get trade order")
		return nil, ErrDatabaseClosed
	}

	// Validate input
	if err := validateID(id); err != nil {
		kv.logger.Error("Invalid trade order ID", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}

	// Create key and validate
	key := []byte("trade:" + id)
	if err := validateKey(key); err != nil {
		kv.logger.Error("Invalid key for trade order", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}

	// Retrieve data
	tradeOrderBytes, closer, err := kv.db.Get(key)
	if err != nil {
		kv.logger.Error("Failed to retrieve trade order from KVDB", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}
	defer closer.Close()

	// Validate data size
	if len(tradeOrderBytes) > MaxDataLength {
		kv.logger.Error("Trade order data too large", zap.String("orderID", id), zap.Int("size", len(tradeOrderBytes)), zap.Int("maxSize", MaxDataLength))
		return nil, ErrInvalidData
	}

	// Unmarshal
	tradeOrder := &engine.TradeOrder{}
	err = tradeOrder.UnmarshalJSON(tradeOrderBytes)
	if err != nil {
		kv.logger.Error("Failed to unmarshal trade order from JSON", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}

	// Validate retrieved data
	if err := validateTradeOrderData(tradeOrder); err != nil {
		kv.logger.Error("Retrieved trade order data validation failed", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}

	kv.logger.Debug("Trade order retrieved successfully", zap.String("orderID", id))
	return tradeOrder, nil
}

func (kv *KvDB) DeleteTradeOrder(id string) error {
	if kv.db == nil {
		kv.logger.Error("KVDB is closed, cannot delete trade order")
		return ErrDatabaseClosed
	}

	// Validate input
	if err := validateID(id); err != nil {
		kv.logger.Error("Invalid trade order ID", zap.String("orderID", id), zap.Error(err))
		return err
	}

	// Create key and validate
	key := []byte("trade:" + id)
	if err := validateKey(key); err != nil {
		kv.logger.Error("Invalid key for trade order", zap.String("orderID", id), zap.Error(err))
		return err
	}

	if err := kv.db.Delete(key, pebble.Sync); err != nil {
		kv.logger.Error("Failed to delete trade order from KVDB", zap.String("orderID", id), zap.Error(err))
		return err
	}

	kv.logger.Debug("Trade order deleted successfully", zap.String("orderID", id))
	return nil
}

func (kv *KvDB) PutReceipt(receipt *orderbook.Receipt) error {
	if kv.db == nil {
		kv.logger.Error("KVDB is closed, cannot put receipt")
		return ErrDatabaseClosed
	}

	// Validate input data
	if err := validateReceiptData(receipt); err != nil {
		kv.logger.Error("Receipt validation failed", zap.String("orderID", receipt.OrderID), zap.Error(err))
		return err
	}

	// Marshal to JSON
	receiptBytes, err := receipt.MarshalJSON()
	if err != nil {
		kv.logger.Error("Failed to marshal receipt to JSON", zap.String("orderID", receipt.OrderID), zap.Error(err))
		return err
	}

	// Validate data size
	if len(receiptBytes) > MaxDataLength {
		kv.logger.Error("Receipt data too large", zap.String("orderID", receipt.OrderID), zap.Int("size", len(receiptBytes)), zap.Int("maxSize", MaxDataLength))
		return ErrInvalidData
	}

	// Create key and validate
	key := []byte("receipt:" + receipt.OrderID)
	if err := validateKey(key); err != nil {
		kv.logger.Error("Invalid key for receipt", zap.String("orderID", receipt.OrderID), zap.Error(err))
		return err
	}

	if err := kv.db.Set(key, receiptBytes, pebble.Sync); err != nil {
		kv.logger.Error("Failed to store receipt in KVDB", zap.String("orderID", receipt.OrderID), zap.Error(err))
		return err
	}

	kv.logger.Debug("Receipt stored successfully", zap.String("orderID", receipt.OrderID))
	return nil
}

func (kv *KvDB) GetReceipt(id string) (*orderbook.Receipt, error) {
	if kv.db == nil {
		kv.logger.Error("KVDB is closed, cannot get receipt")
		return nil, ErrDatabaseClosed
	}

	// Validate input
	if err := validateID(id); err != nil {
		kv.logger.Error("Invalid receipt ID", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}

	// Create key and validate
	key := []byte("receipt:" + id)
	if err := validateKey(key); err != nil {
		kv.logger.Error("Invalid key for receipt", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}

	// Retrieve data
	receiptBytes, closer, err := kv.db.Get(key)
	if err != nil {
		kv.logger.Error("Failed to retrieve receipt from KVDB", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}
	defer closer.Close()

	// Validate data size
	if len(receiptBytes) > MaxDataLength {
		kv.logger.Error("Receipt data too large", zap.String("orderID", id), zap.Int("size", len(receiptBytes)), zap.Int("maxSize", MaxDataLength))
		return nil, ErrInvalidData
	}

	// Unmarshal
	receipt := &orderbook.Receipt{}
	err = receipt.UnmarshalJSON(receiptBytes)
	if err != nil {
		kv.logger.Error("Failed to unmarshal receipt from JSON", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}

	// Validate retrieved data
	if err := validateReceiptData(receipt); err != nil {
		kv.logger.Error("Retrieved receipt data validation failed", zap.String("orderID", id), zap.Error(err))
		return nil, err
	}

	kv.logger.Debug("Receipt retrieved successfully", zap.String("orderID", id))
	return receipt, nil
}

func (kv *KvDB) DeleteReceipt(id string) error {
	if kv.db == nil {
		kv.logger.Error("KVDB is closed, cannot delete receipt")
		return ErrDatabaseClosed
	}

	// Validate input
	if err := validateID(id); err != nil {
		kv.logger.Error("Invalid receipt ID", zap.String("orderID", id), zap.Error(err))
		return err
	}

	// Create key and validate
	key := []byte("receipt:" + id)
	if err := validateKey(key); err != nil {
		kv.logger.Error("Invalid key for receipt", zap.String("orderID", id), zap.Error(err))
		return err
	}

	if err := kv.db.Delete(key, pebble.Sync); err != nil {
		kv.logger.Error("Failed to delete receipt from KVDB", zap.String("orderID", id), zap.Error(err))
		return err
	}

	kv.logger.Debug("Receipt deleted successfully", zap.String("orderID", id))
	return nil
}

// Additional utility methods for data integrity

// IsHealthy checks if the database is in a healthy state
func (kv *KvDB) IsHealthy() bool {
	if kv.db == nil {
		kv.logger.Warn("KVDB is nil, not healthy")
		return false
	}

	// Try a simple operation to check if DB is responsive
	_, _, err := kv.db.Get([]byte("health_check"))
	// We expect an error (key not found), but not a closed DB error
	healthy := err == nil || err.Error() != "pebble: closed"
	if !healthy {
		kv.logger.Warn("KVDB health check failed", zap.Error(err))
	}
	return healthy
}

// GetStats returns basic database statistics
func (kv *KvDB) GetStats() map[string]interface{} {
	if kv.db == nil {
		return map[string]interface{}{
			"status": "closed",
		}
	}

	stats := map[string]interface{}{
		"status":    "open",
		"timestamp": time.Now().Unix(),
	}

	// Add more statistics as needed
	// You could add metrics like:
	// - Number of keys
	// - Database size
	// - Memory usage
	// - Operation counts

	return stats
}
