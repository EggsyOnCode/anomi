package storage

import (
	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

type KvDB struct {
	db *pebble.DB
}

func NewDB(path string) (*KvDB, error) {
	// In memory database for testing
	db, err := pebble.Open(path, &pebble.Options{FS: vfs.NewMem()})
	if err != nil {
		return nil, err
	}
	return &KvDB{db: db}, nil
}

func (kv *KvDB) Close() error {
	return kv.db.Close()
}

func (kv *KvDB) PutOrder(order *engine.Order) error {
	orderBytes, err := order.MarshalJSON()
	if err != nil {
		return err
	}
	key := []byte("order:" + order.ID())
	return kv.db.Set(key, orderBytes, pebble.Sync)
}

func (kv *KvDB) GetOrder(id string) (*engine.Order, error) {
	key := []byte("order:" + id)
	orderBytes, closer, err := kv.db.Get(key)
	if err != nil {
		return nil, err
	}
	order := &engine.Order{}
	err = order.UnmarshalJSON(orderBytes)
	if err != nil {
		return nil, err
	}
	closer.Close()
	return order, nil
}

func (kv *KvDB) DeleteOrder(id string) error {
	key := []byte("order:" + id)
	return kv.db.Delete(key, pebble.Sync)
}

func (kv *KvDB) PutTradeOrder(tradeOrder *engine.TradeOrder) error {
	tradeOrderBytes, err := tradeOrder.MarshalJSON()
	if err != nil {
		return err
	}
	key := []byte("trade:" + tradeOrder.OrderID)
	return kv.db.Set(key, tradeOrderBytes, pebble.Sync)
}

func (kv *KvDB) GetTradeOrder(id string) (*engine.TradeOrder, error) {
	key := []byte("trade:" + id)
	tradeOrderBytes, closer, err := kv.db.Get(key)
	if err != nil {
		return nil, err
	}
	tradeOrder := &engine.TradeOrder{}
	err = tradeOrder.UnmarshalJSON(tradeOrderBytes)
	if err != nil {
		return nil, err
	}
	closer.Close()
	return tradeOrder, nil
}

func (kv *KvDB) DeleteTradeOrder(id string) error {
	key := []byte("trade:" + id)
	return kv.db.Delete(key, pebble.Sync)
}

func (kv *KvDB) PutReceipt(receipt *orderbook.Receipt) error {
	receiptBytes, err := receipt.MarshalJSON()
	if err != nil {
		return err
	}
	key := []byte("receipt:" + receipt.OrderID)
	return kv.db.Set(key, receiptBytes, pebble.Sync)
}

func (kv *KvDB) GetReceipt(id string) (*orderbook.Receipt, error) {
	key := []byte("receipt:" + id)
	receiptBytes, closer, err := kv.db.Get(key)
	if err != nil {
		return nil, err
	}
	receipt := &orderbook.Receipt{}
	err = receipt.UnmarshalJSON(receiptBytes)
	if err != nil {
		return nil, err
	}
	closer.Close()
	return receipt, nil
}

func (kv *KvDB) DeleteReceipt(id string) error {
	key := []byte("receipt:" + id)
	return kv.db.Delete(key, pebble.Sync)
}
