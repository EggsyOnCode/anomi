package repositories

import (
	"github.com/uptrace/bun"
)

// RepositoryFactory defines the interface for creating repository instances
type RepositoryFactory interface {
	NewOrderRepository() OrderRepository
	NewTradeRepository() TradeRepository
	NewReceiptRepository() ReceiptRepository
}

// RepositoryFactoryImpl implements RepositoryFactory interface
type RepositoryFactoryImpl struct {
	db *bun.DB
}

// NewRepositoryFactory creates a new repository factory
func NewRepositoryFactory(db *bun.DB) RepositoryFactory {
	return &RepositoryFactoryImpl{
		db: db,
	}
}

// NewOrderRepository creates a new order repository
func (f *RepositoryFactoryImpl) NewOrderRepository() OrderRepository {
	return NewOrderRepository(f.db)
}

// NewTradeRepository creates a new trade repository
func (f *RepositoryFactoryImpl) NewTradeRepository() TradeRepository {
	return NewTradeRepository(f.db)
}

// NewReceiptRepository creates a new receipt repository
func (f *RepositoryFactoryImpl) NewReceiptRepository() ReceiptRepository {
	return NewReceiptRepository(f.db)
}
